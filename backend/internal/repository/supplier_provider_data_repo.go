package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type supplierProviderDataRepository struct {
	db *sql.DB
}

func NewSupplierProviderDataRepository(db *sql.DB) service.SupplierProviderDataRepository {
	return &supplierProviderDataRepository{db: db}
}

func (r *supplierProviderDataRepository) UpdateAccountRateSnapshot(ctx context.Context, providerID int64, upstreamKey string, rate float64, syncedAt time.Time) (bool, error) {
	var accountID int64
	err := r.db.QueryRowContext(ctx, `
UPDATE supplier_provider_accounts
SET rate_multiplier=$3, rate_sync_status='success', rate_sync_message='',
    last_rate_sync_at=$4, updated_at=$4
WHERE provider_id=$1 AND upstream_account_key=$2
RETURNING id`, providerID, strings.TrimSpace(upstreamKey), rate, syncedAt).Scan(&accountID)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("update supplier account rate snapshot: %w", err)
	}
	return true, nil
}

func (r *supplierProviderDataRepository) SaveMonitorSnapshot(ctx context.Context, providerID int64, monitors []service.SupplierProviderMonitorItem, seenAt time.Time) ([]service.SupplierProviderMonitorBinding, error) {
	if providerID <= 0 {
		return []service.SupplierProviderMonitorBinding{}, nil
	}
	if seenAt.IsZero() {
		seenAt = time.Now().UTC()
	} else {
		seenAt = seenAt.UTC()
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin supplier provider monitor snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	targetKeys := make([]string, 0, len(monitors))
	seenTargetKeys := make(map[string]struct{}, len(monitors))
	for _, monitor := range monitors {
		targetKey := strings.TrimSpace(monitor.Key)
		if targetKey == "" {
			targetKey = strings.TrimSpace(monitor.Name)
		}
		if targetKey == "" {
			continue
		}
		if _, exists := seenTargetKeys[targetKey]; !exists {
			seenTargetKeys[targetKey] = struct{}{}
			targetKeys = append(targetKeys, targetKey)
		}

		var targetID int64
		if err := tx.QueryRowContext(ctx, `
INSERT INTO supplier_provider_monitor_targets (
  provider_id, monitor_key, monitor_name, monitor_provider, primary_model,
  availability_7d, active, first_seen_at, last_seen_at, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, TRUE, $7, $7, NOW(), NOW())
ON CONFLICT (provider_id, monitor_key) DO UPDATE SET
  monitor_name = EXCLUDED.monitor_name,
  monitor_provider = EXCLUDED.monitor_provider,
  primary_model = EXCLUDED.primary_model,
  availability_7d = EXCLUDED.availability_7d,
  active = TRUE,
  last_seen_at = EXCLUDED.last_seen_at,
  updated_at = NOW()
RETURNING id`, providerID, targetKey, strings.TrimSpace(monitor.Name), strings.TrimSpace(monitor.Provider), strings.TrimSpace(monitor.PrimaryModel), monitor.Availability7D, seenAt).Scan(&targetID); err != nil {
			return nil, fmt.Errorf("upsert supplier provider monitor target: %w", err)
		}

		points := monitor.Timeline
		if len(points) == 0 {
			points = []service.SupplierProviderMonitorPoint{{Status: monitor.PrimaryStatus, LatencyMS: monitor.PrimaryLatencyMS, PingLatencyMS: monitor.PrimaryPingLatencyMS, CheckedAt: seenAt}}
		}
		for _, point := range points {
			checkedAt := point.CheckedAt
			if checkedAt.IsZero() {
				checkedAt = seenAt
			}
			checkedAt = checkedAt.UTC()
			if _, err := tx.ExecContext(ctx, `
INSERT INTO supplier_provider_monitor_samples (
  monitor_target_id, checked_at, status, raw_status, latency_ms, ping_latency_ms, availability_7d, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
ON CONFLICT (monitor_target_id, checked_at) DO UPDATE SET
  status = EXCLUDED.status,
  raw_status = EXCLUDED.raw_status,
  latency_ms = EXCLUDED.latency_ms,
  ping_latency_ms = EXCLUDED.ping_latency_ms,
  availability_7d = EXCLUDED.availability_7d`, targetID, checkedAt, normalizeSupplierProviderMonitorStatus(point.Status), strings.TrimSpace(point.Status), point.LatencyMS, point.PingLatencyMS, monitor.Availability7D); err != nil {
				return nil, fmt.Errorf("upsert supplier provider monitor sample: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit supplier provider monitor snapshot: %w", err)
	}
	if len(targetKeys) == 0 {
		return []service.SupplierProviderMonitorBinding{}, nil
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT t.id, t.provider_id, t.monitor_key, t.monitor_name,
       CASE WHEN local_account.id IS NULL THEN 0 ELSE b.local_account_id END AS local_account_id,
       COALESCE(local_account.name, '') AS local_account_name,
       COALESCE((
         SELECT jsonb_agg(
           jsonb_build_object(
             'id', local_group.id,
             'name', local_group.name,
             'platform', COALESCE(local_group.platform, ''),
             'rate_multiplier', COALESCE(local_group.rate_multiplier, 0),
             'subscription_type', COALESCE(local_group.subscription_type, '')
           )
           ORDER BY local_group.id
         )
         FROM account_groups account_group
         JOIN groups local_group ON local_group.id = account_group.group_id AND local_group.deleted_at IS NULL
         WHERE account_group.account_id = local_account.id
       ), '[]'::jsonb) AS binding_groups
FROM supplier_provider_monitor_targets t
LEFT JOIN supplier_provider_monitor_bindings b
  ON b.provider_id = t.provider_id
 AND b.monitor_target_id = t.id
 AND b.match_status = 'active'
LEFT JOIN accounts local_account
  ON local_account.id = b.local_account_id
 AND local_account.deleted_at IS NULL
WHERE t.provider_id = $1
  AND t.monitor_key = ANY($2)
ORDER BY t.id`, providerID, pq.Array(targetKeys))
	if err != nil {
		return nil, fmt.Errorf("query supplier provider monitor bindings: %w", err)
	}
	defer rows.Close()

	bindings := make([]service.SupplierProviderMonitorBinding, 0)
	for rows.Next() {
		var binding service.SupplierProviderMonitorBinding
		var localAccountID int64
		var bindingGroupsJSON []byte
		if err := rows.Scan(&binding.MonitorTargetID, &binding.ProviderID, &binding.MonitorKey, &binding.MonitorName, &localAccountID, &binding.LocalAccountName, &bindingGroupsJSON); err != nil {
			return nil, fmt.Errorf("scan supplier provider monitor binding: %w", err)
		}
		binding.LocalAccountID = localAccountID
		if err := json.Unmarshal(bindingGroupsJSON, &binding.BindingGroups); err != nil {
			return nil, fmt.Errorf("decode supplier provider monitor binding groups: %w", err)
		}
		bindings = append(bindings, binding)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier provider monitor bindings: %w", err)
	}
	return bindings, nil
}

func (r *supplierProviderDataRepository) ListMonitorTargets(ctx context.Context, params service.SupplierProviderMonitorTargetListParams) (service.SupplierProviderMonitorTargetListResult, error) {
	params = normalizeSupplierProviderMonitorTargetListParams(params)
	where := []string{"TRUE"}
	args := make([]any, 0, 3)
	if params.ProviderID > 0 {
		args = append(args, params.ProviderID)
		where = append(where, fmt.Sprintf("t.provider_id = $%d", len(args)))
	}
	if params.Active != nil {
		args = append(args, *params.Active)
		where = append(where, fmt.Sprintf("t.active = $%d", len(args)))
	}
	if search := strings.TrimSpace(params.Search); search != "" {
		args = append(args, "%"+search+"%")
		where = append(where, fmt.Sprintf("(t.monitor_key ILIKE $%d OR t.monitor_name ILIKE $%d OR t.primary_model ILIKE $%d)", len(args), len(args), len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM supplier_provider_monitor_targets t WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return service.SupplierProviderMonitorTargetListResult{}, fmt.Errorf("count supplier provider monitor targets: %w", err)
	}

	queryArgs := append(append([]any{}, args...), params.PageSize, (params.Page-1)*params.PageSize)
	rows, err := r.db.QueryContext(ctx, `
SELECT t.id, t.provider_id, p.name, t.monitor_key, t.monitor_name, t.monitor_provider,
       t.primary_model, t.availability_7d, t.active, t.last_seen_at,
       CASE WHEN local_account.id IS NULL THEN 0 ELSE b.local_account_id END AS local_account_id,
       COALESCE(local_account.name, '') AS local_account_name,
       COALESCE((
         SELECT jsonb_agg(
           jsonb_build_object(
             'id', local_group.id,
             'name', local_group.name,
             'platform', COALESCE(local_group.platform, ''),
             'rate_multiplier', COALESCE(local_group.rate_multiplier, 0),
             'subscription_type', COALESCE(local_group.subscription_type, '')
           )
           ORDER BY local_group.id
         )
         FROM account_groups account_group
         JOIN groups local_group ON local_group.id = account_group.group_id AND local_group.deleted_at IS NULL
         WHERE account_group.account_id = local_account.id
       ), '[]'::jsonb) AS binding_groups
FROM supplier_provider_monitor_targets t
JOIN supplier_providers p ON p.id = t.provider_id
LEFT JOIN supplier_provider_monitor_bindings b
  ON b.provider_id = t.provider_id
 AND b.monitor_target_id = t.id
 AND b.match_status = 'active'
LEFT JOIN accounts local_account
  ON local_account.id = b.local_account_id
 AND local_account.deleted_at IS NULL
WHERE `+whereSQL+`
ORDER BY t.last_seen_at DESC, t.id DESC
LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2), queryArgs...)
	if err != nil {
		return service.SupplierProviderMonitorTargetListResult{}, fmt.Errorf("query supplier provider monitor targets: %w", err)
	}
	defer rows.Close()

	items := make([]service.SupplierProviderMonitorTarget, 0)
	for rows.Next() {
		item, err := scanSupplierProviderMonitorTarget(rows)
		if err != nil {
			return service.SupplierProviderMonitorTargetListResult{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return service.SupplierProviderMonitorTargetListResult{}, fmt.Errorf("iterate supplier provider monitor targets: %w", err)
	}
	return service.SupplierProviderMonitorTargetListResult{Items: items, Total: total, Page: params.Page, PageSize: params.PageSize}, nil
}

func (r *supplierProviderDataRepository) BindMonitorTarget(ctx context.Context, monitorTargetID, localAccountID int64) error {
	if monitorTargetID <= 0 || localAccountID <= 0 {
		return service.ErrSupplierProviderMonitorBindingInvalid
	}
	result, err := r.db.ExecContext(ctx, `
WITH valid_binding AS (
  SELECT target.provider_id, target.id AS monitor_target_id, local_account.id AS local_account_id
  FROM supplier_provider_monitor_targets target
  JOIN accounts local_account ON local_account.id = $2 AND local_account.deleted_at IS NULL
  WHERE target.id = $1
)
INSERT INTO supplier_provider_monitor_bindings (
  provider_id, monitor_target_id, local_account_id, match_source, match_status, created_at, updated_at
)
SELECT provider_id, monitor_target_id, local_account_id, 'manual', 'active', NOW(), NOW()
FROM valid_binding
ON CONFLICT (provider_id, monitor_target_id) DO UPDATE SET
  local_account_id = EXCLUDED.local_account_id,
  match_source = 'manual',
  match_status = 'active',
  updated_at = NOW()`, monitorTargetID, localAccountID)
	if err != nil {
		return fmt.Errorf("bind supplier provider monitor target: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read supplier provider monitor binding result: %w", err)
	}
	if affected == 0 {
		return service.ErrSupplierProviderMonitorTargetNotFound
	}
	return nil
}

func (r *supplierProviderDataRepository) UnbindMonitorTarget(ctx context.Context, monitorTargetID int64) error {
	if monitorTargetID <= 0 {
		return service.ErrSupplierProviderMonitorBindingInvalid
	}
	if _, err := r.db.ExecContext(ctx, `
UPDATE supplier_provider_monitor_bindings
SET match_status = 'inactive', updated_at = NOW()
WHERE monitor_target_id = $1
  AND match_status = 'active'`, monitorTargetID); err != nil {
		return fmt.Errorf("unbind supplier provider monitor target: %w", err)
	}
	return nil
}

func (r *supplierProviderDataRepository) ListAccounts(ctx context.Context, params service.SupplierProviderDataListParams) (service.SupplierProviderAccountListResult, error) {
	params = normalizeSupplierProviderDataListParams(params)
	where, args := supplierProviderAccountWhere(params)

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM supplier_provider_accounts a JOIN supplier_providers p ON p.id = a.provider_id WHERE "+where, args...).Scan(&total); err != nil {
		return service.SupplierProviderAccountListResult{}, fmt.Errorf("count supplier provider accounts: %w", err)
	}

	queryArgs := append(append([]any{}, args...), params.PageSize, (params.Page-1)*params.PageSize)
	rows, err := r.db.QueryContext(ctx, `
SELECT a.id, a.provider_id, p.name AS provider_name, a.upstream_account_key, a.name, a.status,
       a.group_key, a.group_name,
       COALESCE((
         SELECT local_group.platform
         FROM supplier_provider_groups mapped_group
         JOIN groups local_group ON local_group.id = mapped_group.local_group_id AND local_group.deleted_at IS NULL
         WHERE mapped_group.provider_id = a.provider_id
           AND mapped_group.upstream_group_key = a.group_key
         LIMIT 1
       ), '') AS platform,
       CASE
         WHEN NULLIF(TRIM(a.group_key), '') IS NULL THEN ''
         WHEN a.active = FALSE AND LOWER(a.status) = 'deleted' THEN ''
         WHEN EXISTS (
           SELECT 1 FROM supplier_provider_groups g
           WHERE g.provider_id = a.provider_id
             AND g.upstream_group_key = a.group_key
             AND g.active = TRUE
         ) THEN 'active'
         WHEN EXISTS (
           SELECT 1 FROM supplier_provider_groups g
           WHERE g.provider_id = a.provider_id
             AND g.upstream_group_key = a.group_key
         ) THEN 'inactive'
         ELSE 'missing'
       END AS group_status,
       a.rate_multiplier, a.raw_status, a.active,
       a.last_seen_at, a.inactive_at,
       CASE
         WHEN local_match.match_count = 0 THEN 'unmatched'
         WHEN local_match.match_count = 1 THEN 'matched'
         ELSE 'conflict'
       END AS local_account_match_status,
       local_match.match_count AS local_account_match_count,
       matched_account.id AS local_account_id,
       COALESCE(matched_account.name, '') AS local_account_name,
       COALESCE(matched_account.platform, '') AS local_account_platform,
       COALESCE(matched_account.type, '') AS local_account_type,
       COALESCE(platform_override.platform, '') AS platform_override,
       COALESCE(NULLIF(platform_override.platform, ''), NULLIF(matched_account.platform, ''), COALESCE((
         SELECT local_group.platform
         FROM supplier_provider_groups mapped_group
         JOIN groups local_group ON local_group.id = mapped_group.local_group_id AND local_group.deleted_at IS NULL
         WHERE mapped_group.provider_id = a.provider_id
           AND mapped_group.upstream_group_key = a.group_key
         LIMIT 1
       ), '')) AS effective_platform,
       matched_account.priority AS local_account_priority,
       COALESCE(matched_account.status, '') AS local_account_status,
       matched_account.schedulable AS local_account_schedulable,
       COALESCE(matched_account.extra->>'last_test_status', '') AS local_account_last_test_status,
       COALESCE(matched_account.extra->>'last_tested_at', '') AS local_account_last_tested_at,
       COALESCE(matched_account.extra->>'last_test_error', '') AS local_account_last_test_error,
       COALESCE((
         SELECT jsonb_agg(
           jsonb_build_object(
             'id', local_group.id,
             'name', local_group.name,
             'platform', local_group.platform,
             'rate_multiplier', local_group.rate_multiplier,
             'subscription_type', local_group.subscription_type
           )
           ORDER BY LOWER(local_group.name), local_group.id
         )
         FROM account_groups account_group
         JOIN groups local_group
           ON local_group.id = account_group.group_id
          AND local_group.deleted_at IS NULL
         WHERE account_group.account_id = matched_account.id
       ), '[]'::jsonb) AS binding_groups,
       COALESCE(runtime.current_balance, 0) AS supplier_current_balance,
       COALESCE(runtime.today_cost, 0) AS supplier_today_cost,
       inactive_group_record.id AS group_record_id,
       COALESCE(
         inactive_group_record.id IS NOT NULL
         AND inactive_group_record.rate_guard_selected = FALSE,
         FALSE
       ) AS group_record_delete_eligible
FROM supplier_provider_accounts a
JOIN supplier_providers p ON p.id = a.provider_id
LEFT JOIN supplier_provider_runtime_stats runtime ON runtime.provider_id = p.id
LEFT JOIN LATERAL (
  SELECT g.id, g.provider_id, g.upstream_group_key, g.local_group_id, g.rate_guard_selected
  FROM supplier_provider_groups g
  WHERE g.provider_id = a.provider_id
    AND g.upstream_group_key = a.group_key
    AND g.active = FALSE
  ORDER BY g.id DESC
  LIMIT 1
) inactive_group_record ON TRUE
LEFT JOIN LATERAL (
  SELECT COUNT(*) AS match_count,
         MIN(local_account.id) AS local_account_id
  FROM accounts local_account
  WHERE local_account.deleted_at IS NULL
    AND `+supplierProviderLocalAccountMatchCondition("local_account.name", "a.name")+`
) local_match ON TRUE
LEFT JOIN accounts matched_account
  ON matched_account.id = local_match.local_account_id
 AND local_match.match_count = 1
LEFT JOIN supplier_local_account_platform_overrides platform_override
  ON platform_override.local_account_id = matched_account.id
WHERE `+where+fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", supplierProviderAccountOrderBy(params), len(args)+1, len(args)+2), queryArgs...)
	if err != nil {
		return service.SupplierProviderAccountListResult{}, fmt.Errorf("query supplier provider accounts: %w", err)
	}
	defer rows.Close()

	items := make([]service.SupplierProviderAccount, 0)
	for rows.Next() {
		item, err := scanSupplierProviderAccount(rows)
		if err != nil {
			return service.SupplierProviderAccountListResult{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return service.SupplierProviderAccountListResult{}, fmt.Errorf("iterate supplier provider accounts: %w", err)
	}
	return service.SupplierProviderAccountListResult{Items: items, Total: total, Page: params.Page, PageSize: params.PageSize}, nil
}

func (r *supplierProviderDataRepository) ListGroups(ctx context.Context, params service.SupplierProviderDataListParams) (service.SupplierProviderGroupListResult, error) {
	params = normalizeSupplierProviderDataListParams(params)
	summaryParams := params
	summaryParams.Active = nil
	summaryWhere, summaryArgs := supplierProviderGroupBaseWhere(summaryParams)
	where, args := supplierProviderGroupListWhere(params)

	summaryScope := "TRUE"
	if params.Active != nil {
		if *params.Active {
			summaryScope = "active = TRUE"
		} else {
			summaryScope = "active = FALSE"
		}
	}

	var summary service.SupplierProviderGroupSummary
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FILTER (WHERE `+summaryScope+`) AS group_count,
       COALESCE(SUM(account_count) FILTER (WHERE `+summaryScope+`), 0) AS account_count,
       COUNT(*) FILTER (WHERE `+summaryScope+` AND local_group_id IS NOT NULL) AS linked_group_count,
       COUNT(*) FILTER (WHERE `+summaryScope+` AND local_group_id IS NULL) AS unlinked_group_count,
       COUNT(*) FILTER (
         WHERE `+summaryScope+`
           AND local_group_id IS NOT NULL
           AND local_group_status = 'active'
           AND local_rate_multiplier < upstream_rate_multiplier
       ) AS rate_risk_count,
       COUNT(*) FILTER (WHERE active = TRUE) AS active_group_count,
       COUNT(*) FILTER (WHERE active = FALSE) AS removed_group_count,
       COUNT(*) FILTER (WHERE active = TRUE AND account_count > 0) AS created_key_group_count,
       COUNT(*) FILTER (WHERE active = TRUE AND attention_required) AS attention_group_count
FROM (
  SELECT g.id, g.active, g.local_group_id,
         g.rate_multiplier AS upstream_rate_multiplier,
         lg.rate_multiplier AS local_rate_multiplier,
         COALESCE(lg.status, '') AS local_group_status,
         COUNT(a.id) FILTER (WHERE a.active = TRUE) AS account_count,
         (
           g.name_change_pending
           OR g.auto_match_status = 'ambiguous'
           OR (
             g.local_group_id IS NOT NULL
             AND COALESCE(lg.status, '') <> 'inactive'
             AND g.rate_multiplier > 0
             AND lg.rate_multiplier > 0
             AND lg.rate_multiplier < g.rate_multiplier - 0.000000001
           )
         ) AS attention_required
  FROM supplier_provider_groups g
  JOIN supplier_providers p ON p.id = g.provider_id
  LEFT JOIN groups lg ON lg.id = g.local_group_id
  LEFT JOIN monitor_group_platform_overrides group_platform_override ON group_platform_override.group_id = lg.id
  LEFT JOIN LATERAL (
    SELECT r.status
    FROM supplier_provider_sync_runs r
    WHERE r.provider_id = g.provider_id
      AND r.sync_scope = 'accounts'
    ORDER BY r.started_at DESC, r.id DESC
    LIMIT 1
  ) account_sync ON TRUE
  LEFT JOIN supplier_provider_accounts a ON a.provider_id = g.provider_id AND a.group_key = g.upstream_group_key
  WHERE `+summaryWhere+`
  GROUP BY g.id, lg.id, group_platform_override.actual_platform, account_sync.status
) matched_groups`, summaryArgs...).Scan(
		&summary.GroupCount,
		&summary.AccountCount,
		&summary.LinkedGroupCount,
		&summary.UnlinkedGroupCount,
		&summary.RateRiskCount,
		&summary.ActiveGroupCount,
		&summary.RemovedGroupCount,
		&summary.CreatedKeyGroupCount,
		&summary.AttentionGroupCount,
	); err != nil {
		return service.SupplierProviderGroupListResult{}, fmt.Errorf("summarize supplier provider groups: %w", err)
	}

	total := summary.GroupCount
	if supplierProviderGroupHasListFilters(params) {
		if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM supplier_provider_groups g
JOIN supplier_providers p ON p.id = g.provider_id
LEFT JOIN groups lg ON lg.id = g.local_group_id
LEFT JOIN monitor_group_platform_overrides group_platform_override ON group_platform_override.group_id = lg.id
WHERE `+where, args...).Scan(&total); err != nil {
			return service.SupplierProviderGroupListResult{}, fmt.Errorf("count filtered supplier provider groups: %w", err)
		}
	}

	queryArgs := append(append([]any{}, args...), params.PageSize, (params.Page-1)*params.PageSize)
	rows, err := r.db.QueryContext(ctx, `
SELECT g.id, g.provider_id, p.name AS provider_name, g.upstream_group_key, g.name,
       g.rate_multiplier, g.raw_status, g.active,
       lg.id AS local_group_id, COALESCE(lg.name, '') AS local_group_name,
       COALESCE(lg.platform, '') AS local_group_platform,
       COALESCE(group_platform_override.actual_platform, '') AS platform_override,
       COALESCE(NULLIF(group_platform_override.actual_platform, ''), COALESCE(lg.platform, '')) AS effective_platform,
       lg.rate_multiplier AS local_rate_multiplier,
       COALESCE(lg.status, '') AS local_group_status,
       g.auto_match_ignored, g.auto_match_status,
       COALESCE(g.matched_upstream_name, '') AS matched_upstream_name,
       g.name_change_pending,
	   g.rate_guard_selected, g.rate_guard_enabled, g.rate_guard_selection_mode,
	   g.rate_guard_last_snapshot_at, g.rate_guard_last_checked_at,
	   COALESCE(s.group_sync_status, 'never') AS group_sync_status,
	   s.last_group_sync_at,
	   COALESCE(guard_state.active_mapping_count, 0) AS local_group_active_mapping_count,
	   guard_state.rate_guard_group_id AS local_group_rate_guard_group_id,
	   COALESCE(guardian_group.name, '') AS local_group_rate_guard_group_name,
	   COALESCE(guardian_provider.name, '') AS local_group_rate_guard_provider_name,
       COALESCE(COUNT(a.id) FILTER (WHERE a.active = TRUE), 0) AS account_count,
       g.last_seen_at, g.inactive_at,
       COALESCE(account_sync.status, 'never') AS key_sync_status
FROM supplier_provider_groups g
JOIN supplier_providers p ON p.id = g.provider_id
LEFT JOIN groups lg ON lg.id = g.local_group_id
LEFT JOIN monitor_group_platform_overrides group_platform_override ON group_platform_override.group_id = lg.id
LEFT JOIN supplier_provider_runtime_stats s ON s.provider_id = g.provider_id
LEFT JOIN LATERAL (
  SELECT r.status
  FROM supplier_provider_sync_runs r
  WHERE r.provider_id = g.provider_id
    AND r.sync_scope = 'accounts'
  ORDER BY r.started_at DESC, r.id DESC
  LIMIT 1
) account_sync ON TRUE
LEFT JOIN (
  SELECT local_group_id,
         COUNT(*) FILTER (WHERE active = TRUE) AS active_mapping_count,
         MAX(id) FILTER (WHERE rate_guard_selected = TRUE) AS rate_guard_group_id
  FROM supplier_provider_groups
  WHERE local_group_id IS NOT NULL
  GROUP BY local_group_id
) guard_state ON guard_state.local_group_id = g.local_group_id
LEFT JOIN supplier_provider_groups guardian_group ON guardian_group.id = guard_state.rate_guard_group_id
LEFT JOIN supplier_providers guardian_provider ON guardian_provider.id = guardian_group.provider_id
LEFT JOIN supplier_provider_accounts a ON a.provider_id = g.provider_id AND a.group_key = g.upstream_group_key
WHERE `+where+fmt.Sprintf(" GROUP BY g.id, p.name, lg.id, group_platform_override.actual_platform, s.group_sync_status, s.last_group_sync_at, account_sync.status, guard_state.active_mapping_count, guard_state.rate_guard_group_id, guardian_group.id, guardian_provider.id ORDER BY %s LIMIT $%d OFFSET $%d", supplierProviderGroupOrderBy(params), len(args)+1, len(args)+2), queryArgs...)
	if err != nil {
		return service.SupplierProviderGroupListResult{}, fmt.Errorf("query supplier provider groups: %w", err)
	}
	defer rows.Close()

	items := make([]service.SupplierProviderGroup, 0)
	for rows.Next() {
		item, err := scanSupplierProviderGroup(rows)
		if err != nil {
			return service.SupplierProviderGroupListResult{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return service.SupplierProviderGroupListResult{}, fmt.Errorf("iterate supplier provider groups: %w", err)
	}
	return service.SupplierProviderGroupListResult{
		Items:    items,
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
		Summary:  summary,
	}, nil
}

func (r *supplierProviderDataRepository) ListGroupHealthTrends(ctx context.Context, params service.SupplierProviderGroupHealthTrendParams) ([]service.SupplierProviderGroupHealthTrend, error) {
	return r.listHealthTrends(ctx, params, false)
}

func (r *supplierProviderDataRepository) ListLocalGroupHealthTrends(ctx context.Context, params service.SupplierProviderGroupHealthTrendParams) ([]service.SupplierProviderGroupHealthTrend, error) {
	return r.listHealthTrends(ctx, params, true)
}

func (r *supplierProviderDataRepository) listHealthTrends(ctx context.Context, params service.SupplierProviderGroupHealthTrendParams, byLocalGroup bool) ([]service.SupplierProviderGroupHealthTrend, error) {
	if params.Period <= 0 {
		params.Period = 90 * time.Minute
	}
	if params.BucketCount <= 0 {
		params.BucketCount = 18
	}
	if params.Now.IsZero() {
		params.Now = time.Now().UTC()
	} else {
		params.Now = params.Now.UTC()
	}

	groupIDs := make([]int64, 0, len(params.GroupIDs))
	seen := make(map[int64]struct{}, len(params.GroupIDs))
	for _, groupID := range params.GroupIDs {
		if groupID <= 0 {
			continue
		}
		if _, exists := seen[groupID]; exists {
			continue
		}
		seen[groupID] = struct{}{}
		groupIDs = append(groupIDs, groupID)
	}
	if len(groupIDs) == 0 {
		return []service.SupplierProviderGroupHealthTrend{}, nil
	}

	if byLocalGroup {
		args := []any{
			service.SupplierAutomationTaskAccountHealthGuard,
			service.SupplierAutomationTaskMonitorSync,
			params.Now,
			pq.Array(groupIDs),
		}
		healthGuardWhereSQL := `
WHERE run.task_code = $1
  AND run.finished_at IS NOT NULL
  AND run.finished_at <= $3
  AND account_group.group_id IS NOT NULL
  AND account_group.group_id = ANY($4)
`
		latestMonitorRunWhereSQL := `
WHERE run.task_code = $2
  AND run.finished_at IS NOT NULL
  AND run.finished_at <= $3
  AND run.status IN ('success', 'partial')
  AND jsonb_array_length(
    CASE
      WHEN jsonb_typeof(run.result_detail->'supplier_monitor'->'items') = 'array'
      THEN run.result_detail->'supplier_monitor'->'items'
      ELSE '[]'::jsonb
    END
  ) > 0
`
		monitorWhereSQL := `
WHERE COALESCE(NULLIF(item->>'checked_at', '')::timestamptz, run.finished_at) <= $3
  AND account_group.group_id IS NOT NULL
  AND account_group.group_id = ANY($4)
`
		structuredMonitorWhereSQL := `
WHERE sample.checked_at <= $3
  AND account_group.group_id IS NOT NULL
  AND account_group.group_id = ANY($4)
`
		if !params.AllHistory {
			args = []any{
				service.SupplierAutomationTaskAccountHealthGuard,
				service.SupplierAutomationTaskMonitorSync,
				params.Now.Add(-params.Period),
				params.Now,
				pq.Array(groupIDs),
			}
			healthGuardWhereSQL = `
WHERE run.task_code = $1
  AND run.finished_at IS NOT NULL
  AND run.finished_at >= $3
  AND run.finished_at <= $4
  AND account_group.group_id IS NOT NULL
  AND account_group.group_id = ANY($5)
`
			latestMonitorRunWhereSQL = `
WHERE run.task_code = $2
  AND run.finished_at IS NOT NULL
  AND run.finished_at <= $4
  AND run.status IN ('success', 'partial')
  AND jsonb_array_length(
    CASE
      WHEN jsonb_typeof(run.result_detail->'supplier_monitor'->'items') = 'array'
      THEN run.result_detail->'supplier_monitor'->'items'
      ELSE '[]'::jsonb
    END
  ) > 0
`
			monitorWhereSQL = `
WHERE COALESCE(NULLIF(item->>'checked_at', '')::timestamptz, run.finished_at) <= $4
  AND account_group.group_id IS NOT NULL
  AND account_group.group_id = ANY($5)
`
			structuredMonitorWhereSQL = `
WHERE sample.checked_at >= $3
  AND sample.checked_at <= $4
  AND account_group.group_id IS NOT NULL
  AND account_group.group_id = ANY($5)
`
		}
		query := fmt.Sprintf(`
WITH latest_monitor_run AS (
  SELECT run.result_detail, run.finished_at
  FROM supplier_automation_runs run
  %s
  ORDER BY run.finished_at DESC
  LIMIT 1
), trend_samples AS (
  SELECT account_group.group_id AS group_id,
         NULLIF(item->>'local_account_id', '')::bigint AS account_id,
         COALESCE(item->>'status', '') AS status,
         COALESCE((NULLIF(item->>'latency_ms', ''))::bigint, 0) AS latency_ms,
         run.finished_at,
         'supplier_account_health_guard' AS source
  FROM supplier_automation_runs run
  CROSS JOIN LATERAL jsonb_array_elements(
    CASE
      WHEN jsonb_typeof(run.result_detail->'account_health_guard'->'items') = 'array'
      THEN run.result_detail->'account_health_guard'->'items'
      ELSE '[]'::jsonb
    END
  ) AS item
  CROSS JOIN LATERAL jsonb_array_elements(
    CASE
      WHEN jsonb_typeof(item->'sources') = 'array'
      THEN item->'sources'
      ELSE '[]'::jsonb
    END
  ) AS source
  JOIN supplier_provider_accounts account
    ON account.id = (source->>'supplier_provider_account_id')::bigint
   AND account.active = TRUE
  JOIN account_groups account_group
    ON account_group.account_id = NULLIF(item->>'local_account_id', '')::bigint
  %s

  UNION ALL

  SELECT account_group.group_id AS group_id,
         NULLIF(item->>'local_account_id', '')::bigint AS account_id,
         COALESCE(item->>'status', '') AS status,
         COALESCE((NULLIF(item->>'latency_ms', ''))::bigint, 0) AS latency_ms,
         COALESCE(NULLIF(item->>'checked_at', '')::timestamptz, run.finished_at) AS finished_at,
         'supplier_monitor' AS source
  FROM latest_monitor_run run
  CROSS JOIN LATERAL jsonb_array_elements(
    CASE
      WHEN jsonb_typeof(run.result_detail->'supplier_monitor'->'items') = 'array'
      THEN run.result_detail->'supplier_monitor'->'items'
      ELSE '[]'::jsonb
    END
  ) AS item
  JOIN account_groups account_group
    ON account_group.account_id = NULLIF(item->>'local_account_id', '')::bigint
  %s

  UNION ALL

  SELECT account_group.group_id AS group_id,
         binding.local_account_id AS account_id,
         COALESCE(sample.status, '') AS status,
         COALESCE(sample.latency_ms, 0) AS latency_ms,
         sample.checked_at AS finished_at,
         'supplier_monitor' AS source
  FROM supplier_provider_monitor_samples sample
  JOIN supplier_provider_monitor_targets target
    ON target.id = sample.monitor_target_id
   AND target.active = TRUE
  JOIN supplier_provider_monitor_bindings binding
    ON binding.provider_id = target.provider_id
   AND binding.monitor_target_id = target.id
   AND binding.match_status = 'active'
  JOIN accounts local_account
    ON local_account.id = binding.local_account_id
   AND local_account.deleted_at IS NULL
  JOIN account_groups account_group
    ON account_group.account_id = binding.local_account_id
  %s
)
SELECT group_id, account_id, status, latency_ms, finished_at, source
FROM trend_samples`, latestMonitorRunWhereSQL, healthGuardWhereSQL, monitorWhereSQL, structuredMonitorWhereSQL)
		rows, err := r.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("query supplier provider group health trends: %w", err)
		}
		defer rows.Close()

		samples := make([]service.SupplierProviderGroupHealthSample, 0)
		for rows.Next() {
			var sample service.SupplierProviderGroupHealthSample
			if err := rows.Scan(&sample.GroupID, &sample.AccountID, &sample.Status, &sample.Latency, &sample.FinishedAt, &sample.Source); err != nil {
				return nil, fmt.Errorf("scan supplier provider group health trend: %w", err)
			}
			samples = append(samples, sample)
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate supplier provider group health trends: %w", err)
		}

		trendIndex := service.BuildSupplierProviderGroupHealthTrends(samples, params)
		trends := make([]service.SupplierProviderGroupHealthTrend, 0, len(trendIndex))
		for _, groupID := range groupIDs {
			if trend, ok := trendIndex[groupID]; ok {
				trends = append(trends, trend)
			}
		}
		return trends, nil
	}

	groupColumn := "g.id"
	accountIDColumn := "(source->>'supplier_provider_account_id')::bigint"
	groupJoinSQL := `
JOIN supplier_provider_groups g
  ON g.provider_id = account.provider_id
 AND g.upstream_group_key = account.group_key
`
	whereSQL := fmt.Sprintf(`
WHERE run.task_code = $1
  AND run.finished_at IS NOT NULL
  AND run.finished_at <= $2
  AND %s IS NOT NULL
  AND %s = ANY($3)
`, groupColumn, groupColumn)
	args := []any{
		service.SupplierAutomationTaskAccountHealthGuard,
		params.Now,
		pq.Array(groupIDs),
	}
	if !params.AllHistory {
		whereSQL = fmt.Sprintf(`
WHERE run.task_code = $1
  AND run.finished_at IS NOT NULL
  AND run.finished_at >= $2
  AND run.finished_at <= $3
  AND %s IS NOT NULL
  AND %s = ANY($4)
`, groupColumn, groupColumn)
		args = []any{
			service.SupplierAutomationTaskAccountHealthGuard,
			params.Now.Add(-params.Period),
			params.Now,
			pq.Array(groupIDs),
		}
	}
	query := fmt.Sprintf(`
SELECT %s AS group_id,
       %s AS account_id,
       COALESCE(item->>'status', '') AS status,
       COALESCE((item->>'latency_ms')::bigint, 0) AS latency_ms,
       run.finished_at,
       'supplier_account_health_guard' AS source
FROM supplier_automation_runs run
CROSS JOIN LATERAL jsonb_array_elements(
  CASE
    WHEN jsonb_typeof(run.result_detail->'account_health_guard'->'items') = 'array'
    THEN run.result_detail->'account_health_guard'->'items'
    ELSE '[]'::jsonb
  END
) AS item
CROSS JOIN LATERAL jsonb_array_elements(
  CASE
    WHEN jsonb_typeof(item->'sources') = 'array'
    THEN item->'sources'
    ELSE '[]'::jsonb
  END
) AS source
JOIN supplier_provider_accounts account
  ON account.id = (source->>'supplier_provider_account_id')::bigint
 AND account.active = TRUE
%s

%s`, groupColumn, accountIDColumn, groupJoinSQL, whereSQL)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query supplier provider group health trends: %w", err)
	}
	defer rows.Close()

	samples := make([]service.SupplierProviderGroupHealthSample, 0)
	for rows.Next() {
		var sample service.SupplierProviderGroupHealthSample
		if err := rows.Scan(&sample.GroupID, &sample.AccountID, &sample.Status, &sample.Latency, &sample.FinishedAt, &sample.Source); err != nil {
			return nil, fmt.Errorf("scan supplier provider group health trend: %w", err)
		}
		samples = append(samples, sample)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier provider group health trends: %w", err)
	}

	trendIndex := service.BuildSupplierProviderGroupHealthTrends(samples, params)
	trends := make([]service.SupplierProviderGroupHealthTrend, 0, len(trendIndex))
	for _, groupID := range groupIDs {
		if trend, ok := trendIndex[groupID]; ok {
			trends = append(trends, trend)
		}
	}
	return trends, nil
}
func (r *supplierProviderDataRepository) UpdateGroupMapping(ctx context.Context, groupID int64, localGroupID *int64) error {
	if localGroupID != nil {
		var exists bool
		if err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM groups WHERE id = $1 AND status = 'active')", *localGroupID).Scan(&exists); err != nil {
			return fmt.Errorf("check supplier local group: %w", err)
		}
		if !exists {
			return service.ErrSupplierLocalGroupNotFound
		}
	}

	result, err := r.db.ExecContext(ctx, "UPDATE supplier_provider_groups SET local_group_id = $2::bigint, auto_match_status = CASE WHEN $2::bigint IS NULL THEN 'unmatched' ELSE 'manual' END, auto_match_ignored = CASE WHEN $2::bigint IS NULL THEN TRUE ELSE auto_match_ignored END, matched_upstream_name = CASE WHEN $2::bigint IS NULL THEN NULL ELSE name END, name_change_pending = FALSE, rate_guard_selected = CASE WHEN local_group_id IS DISTINCT FROM $2::bigint THEN FALSE ELSE rate_guard_selected END, rate_guard_selection_mode = CASE WHEN local_group_id IS DISTINCT FROM $2::bigint THEN '' ELSE rate_guard_selection_mode END, updated_at = NOW() WHERE id = $1", groupID, localGroupID)
	if err != nil {
		return fmt.Errorf("update supplier provider group mapping: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read supplier provider group mapping result: %w", err)
	}
	if affected == 0 {
		return service.ErrSupplierProviderGroupNotFound
	}
	return nil
}

func (r *supplierProviderDataRepository) DeleteGroup(ctx context.Context, groupID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin supplier provider group delete: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var providerID int64
	var upstreamGroupKey string
	err = tx.QueryRowContext(ctx, `
DELETE FROM supplier_provider_groups g
WHERE g.id = $1
RETURNING g.provider_id, g.upstream_group_key`, groupID).Scan(&providerID, &upstreamGroupKey)
	if err == sql.ErrNoRows {
		return service.ErrSupplierProviderGroupDeleteConflict
	}
	if err != nil {
		return fmt.Errorf("delete supplier provider group record: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE supplier_provider_accounts AS a
SET group_key = '', group_name = '', updated_at = NOW()
WHERE a.provider_id = $1
  AND a.group_key = $2`, providerID, upstreamGroupKey); err != nil {
		return fmt.Errorf("clear supplier account group reference: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit supplier provider group delete: %w", err)
	}
	return nil
}

func (r *supplierProviderDataRepository) DeleteAccount(ctx context.Context, accountID int64) error {
	result, err := r.db.ExecContext(ctx, `
DELETE FROM supplier_provider_accounts a
WHERE a.id = $1`, accountID)
	if err != nil {
		return fmt.Errorf("delete supplier provider account record: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read supplier provider account delete result: %w", err)
	}
	if affected == 0 {
		return service.ErrSupplierProviderAccountDeleteConflict
	}
	return nil
}

func (r *supplierProviderDataRepository) ListGroupsForAutoMatch(ctx context.Context, providerID int64) ([]service.SupplierProviderGroup, error) {
	query := `
SELECT g.id, g.provider_id, p.name AS provider_name, g.upstream_group_key, g.name,
       g.rate_multiplier, g.raw_status, g.active, g.local_group_id,
       g.auto_match_ignored, g.auto_match_status,
       COALESCE(g.matched_upstream_name, '') AS matched_upstream_name,
       g.name_change_pending, g.last_seen_at, g.inactive_at
FROM supplier_provider_groups g
JOIN supplier_providers p ON p.id = g.provider_id
WHERE (g.active = TRUE OR g.rate_guard_selected = TRUE)`
	args := make([]any, 0, 1)
	if providerID > 0 {
		query += " AND g.provider_id = $1"
		args = append(args, providerID)
	}
	query += " ORDER BY g.provider_id ASC, g.id ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query supplier groups for auto match: %w", err)
	}
	defer rows.Close()

	groups := make([]service.SupplierProviderGroup, 0)
	for rows.Next() {
		group, err := scanSupplierProviderGroupAutoMatch(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier groups for auto match: %w", err)
	}
	return groups, nil
}

func (r *supplierProviderDataRepository) GetGroupForAutoMatch(ctx context.Context, groupID int64) (service.SupplierProviderGroup, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT g.id, g.provider_id, p.name AS provider_name, g.upstream_group_key, g.name,
       g.rate_multiplier, g.raw_status, g.active, g.local_group_id,
       g.auto_match_ignored, g.auto_match_status,
       COALESCE(g.matched_upstream_name, '') AS matched_upstream_name,
       g.name_change_pending, g.last_seen_at, g.inactive_at
FROM supplier_provider_groups g
JOIN supplier_providers p ON p.id = g.provider_id
WHERE g.id = $1`, groupID)
	group, err := scanSupplierProviderGroupAutoMatch(row)
	if err == sql.ErrNoRows {
		return service.SupplierProviderGroup{}, service.ErrSupplierProviderGroupNotFound
	}
	if err != nil {
		return service.SupplierProviderGroup{}, err
	}
	return group, nil
}

func (r *supplierProviderDataRepository) ApplyAutoMatch(ctx context.Context, groupID, localGroupID int64, matchedUpstreamName string) (bool, error) {
	result, err := r.db.ExecContext(ctx, "UPDATE supplier_provider_groups SET local_group_id = $2, auto_match_status = 'auto_matched', matched_upstream_name = $3, name_change_pending = FALSE, updated_at = NOW() WHERE id = $1 AND active = TRUE AND local_group_id IS NULL AND auto_match_ignored = FALSE", groupID, localGroupID, matchedUpstreamName)
	if err != nil {
		return false, fmt.Errorf("apply supplier group auto match: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read supplier group auto match result: %w", err)
	}
	return affected > 0, nil
}

func (r *supplierProviderDataRepository) UpdateAutoMatchState(ctx context.Context, groupID int64, status string, nameChangePending bool) error {
	result, err := r.db.ExecContext(ctx, "UPDATE supplier_provider_groups SET auto_match_status = $2, name_change_pending = $3, updated_at = NOW() WHERE id = $1", groupID, status, nameChangePending)
	if err != nil {
		return fmt.Errorf("update supplier group auto match state: %w", err)
	}
	return supplierGroupUpdateFound(result)
}

func (r *supplierProviderDataRepository) UpdateAutoMatchIgnored(ctx context.Context, groupID int64, ignored bool) error {
	result, err := r.db.ExecContext(ctx, "UPDATE supplier_provider_groups SET auto_match_ignored = $2, updated_at = NOW() WHERE id = $1", groupID, ignored)
	if err != nil {
		return fmt.Errorf("update supplier group auto match policy: %w", err)
	}
	return supplierGroupUpdateFound(result)
}

func (r *supplierProviderDataRepository) AcknowledgeNameChange(ctx context.Context, groupID int64, matchedUpstreamName string) error {
	result, err := r.db.ExecContext(ctx, "UPDATE supplier_provider_groups SET matched_upstream_name = $2, name_change_pending = FALSE, updated_at = NOW() WHERE id = $1 AND local_group_id IS NOT NULL", groupID, matchedUpstreamName)
	if err != nil {
		return fmt.Errorf("acknowledge supplier group name change: %w", err)
	}
	return supplierGroupUpdateFound(result)
}

func (r *supplierProviderDataRepository) ListMappingsByLocalGroup(ctx context.Context, localGroupIDs []int64) ([]service.SupplierProviderGroup, error) {
	if len(localGroupIDs) == 0 {
		return []service.SupplierProviderGroup{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT g.id, g.provider_id, p.name AS provider_name, g.upstream_group_key, g.name,
       g.rate_multiplier, g.raw_status, g.active, g.local_group_id,
       g.auto_match_ignored, g.auto_match_status,
       COALESCE(g.matched_upstream_name, ''), g.name_change_pending,
       g.rate_guard_selected, g.rate_guard_enabled, g.rate_guard_selection_mode,
       g.last_seen_at, g.inactive_at
FROM supplier_provider_groups g
JOIN supplier_providers p ON p.id = g.provider_id
WHERE g.local_group_id = ANY($1)
  AND (g.active = TRUE OR g.rate_guard_selected = TRUE)
ORDER BY g.local_group_id, g.id`, pq.Array(localGroupIDs))
	if err != nil {
		return nil, fmt.Errorf("query supplier group guard mappings: %w", err)
	}
	defer rows.Close()

	result := make([]service.SupplierProviderGroup, 0)
	for rows.Next() {
		group, err := scanSupplierProviderGroupGuard(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier group guard mappings: %w", err)
	}
	return result, nil
}

func (r *supplierProviderDataRepository) GetGroupForRateGuard(ctx context.Context, groupID int64) (service.SupplierProviderGroup, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT g.id, g.provider_id, p.name AS provider_name, g.upstream_group_key, g.name,
       g.rate_multiplier, g.raw_status, g.active, g.local_group_id,
       g.auto_match_ignored, g.auto_match_status,
       COALESCE(g.matched_upstream_name, ''), g.name_change_pending,
       g.rate_guard_selected, g.rate_guard_enabled, g.rate_guard_selection_mode,
       g.last_seen_at, g.inactive_at
FROM supplier_provider_groups g
JOIN supplier_providers p ON p.id = g.provider_id
WHERE g.id = $1`, groupID)
	group, err := scanSupplierProviderGroupGuard(row)
	if err == sql.ErrNoRows {
		return service.SupplierProviderGroup{}, service.ErrSupplierProviderGroupNotFound
	}
	if err != nil {
		return service.SupplierProviderGroup{}, fmt.Errorf("get supplier group for rate guard: %w", err)
	}
	return group, nil
}

func (r *supplierProviderDataRepository) SelectRateGuard(ctx context.Context, groupID int64, mode string) error {
	if mode != service.RateGuardSelectionModeAuto && mode != service.RateGuardSelectionModeManual {
		return service.ErrSupplierRateGuardSelectionInvalid
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin supplier rate guard selection: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var localGroupID sql.NullInt64
	var active bool
	if err := tx.QueryRowContext(ctx, "SELECT local_group_id, active FROM supplier_provider_groups WHERE id=$1 FOR UPDATE", groupID).Scan(&localGroupID, &active); err != nil {
		if err == sql.ErrNoRows {
			return service.ErrSupplierProviderGroupNotFound
		}
		return fmt.Errorf("lock supplier rate guard mapping: %w", err)
	}
	if !localGroupID.Valid || !active {
		return service.ErrSupplierRateGuardSelectionInvalid
	}
	var lockValue any
	if err := tx.QueryRowContext(ctx, "SELECT pg_advisory_xact_lock($1)", localGroupID.Int64).Scan(&lockValue); err != nil {
		return fmt.Errorf("lock supplier rate guard local group: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE supplier_provider_groups SET rate_guard_selected=FALSE, rate_guard_selection_mode='', updated_at=NOW() WHERE local_group_id=$1 AND rate_guard_selected=TRUE", localGroupID.Int64); err != nil {
		return fmt.Errorf("clear existing supplier rate guard: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE supplier_provider_groups SET rate_guard_selected=TRUE, rate_guard_selection_mode=$2, updated_at=NOW() WHERE id=$1", groupID, mode); err != nil {
		return fmt.Errorf("select supplier rate guard: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit supplier rate guard selection: %w", err)
	}
	return nil
}

func (r *supplierProviderDataRepository) ClearRateGuard(ctx context.Context, groupID int64, mode string) error {
	query := "UPDATE supplier_provider_groups SET rate_guard_selected=FALSE, rate_guard_selection_mode='', updated_at=NOW() WHERE id=$1"
	args := []any{groupID}
	if mode != "" {
		query += " AND rate_guard_selection_mode=$2"
		args = append(args, mode)
	}
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("clear supplier rate guard: %w", err)
	}
	return nil
}

func (r *supplierProviderDataRepository) SetRateGuardEnabled(ctx context.Context, groupID int64, enabled bool) error {
	result, err := r.db.ExecContext(ctx, "UPDATE supplier_provider_groups SET rate_guard_enabled=$2, updated_at=NOW() WHERE id=$1", groupID, enabled)
	if err != nil {
		return fmt.Errorf("update supplier rate guard enabled policy: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read supplier rate guard enabled policy result: %w", err)
	}
	if affected == 0 {
		return service.ErrSupplierProviderGroupNotFound
	}
	return nil
}

func (r *supplierProviderDataRepository) ListRateGuardCandidates(ctx context.Context) ([]service.SupplierRateGuardCandidate, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT g.id AS mapping_id, g.provider_id, p.name AS provider_name, p.enabled AS provider_enabled,
       g.upstream_group_key, g.name AS upstream_group_name, g.rate_multiplier AS upstream_rate_multiplier,
       g.active AS guardian_active,
       COALESCE(lg.id, 0) AS local_group_id, COALESCE(lg.name, '') AS local_group_name,
       COALESCE(lg.status, '') AS local_group_status, COALESCE(lg.rate_multiplier, 0) AS local_rate_multiplier,
       g.last_seen_at AS snapshot_at, g.rate_guard_last_snapshot_at AS last_snapshot_at,
       COALESCE(s.group_sync_status, 'never') AS group_sync_status, s.last_group_sync_at
FROM supplier_provider_groups g
JOIN supplier_providers p ON p.id = g.provider_id
LEFT JOIN groups lg ON lg.id = g.local_group_id AND lg.deleted_at IS NULL
LEFT JOIN supplier_provider_runtime_stats s ON s.provider_id = g.provider_id
WHERE g.rate_guard_selected = TRUE
  AND g.rate_guard_enabled = TRUE
ORDER BY g.provider_id, g.id`)
	if err != nil {
		return nil, fmt.Errorf("query supplier rate guard candidates: %w", err)
	}
	defer rows.Close()

	result := make([]service.SupplierRateGuardCandidate, 0)
	for rows.Next() {
		var candidate service.SupplierRateGuardCandidate
		var lastSnapshotAt, lastGroupSyncAt sql.NullTime
		if err := rows.Scan(
			&candidate.MappingID, &candidate.ProviderID, &candidate.ProviderName, &candidate.ProviderEnabled,
			&candidate.UpstreamGroupKey, &candidate.UpstreamGroupName, &candidate.UpstreamRateMultiplier,
			&candidate.GuardianActive, &candidate.LocalGroupID, &candidate.LocalGroupName,
			&candidate.LocalGroupStatus, &candidate.LocalRateMultiplier, &candidate.SnapshotAt,
			&lastSnapshotAt, &candidate.GroupSyncStatus, &lastGroupSyncAt,
		); err != nil {
			return nil, fmt.Errorf("scan supplier rate guard candidate: %w", err)
		}
		if lastSnapshotAt.Valid {
			candidate.LastSnapshotAt = &lastSnapshotAt.Time
		}
		if lastGroupSyncAt.Valid {
			candidate.LastGroupSyncAt = &lastGroupSyncAt.Time
		}
		result = append(result, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier rate guard candidates: %w", err)
	}
	return result, nil
}

func (r *supplierProviderDataRepository) ApplyRateGuard(ctx context.Context, input service.SupplierRateGuardApplyInput) (service.SupplierRateGuardApplyResult, error) {
	result := service.SupplierRateGuardApplyResult{TargetRate: input.TargetRate}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("begin supplier rate guard update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var selected, guardianActive, providerEnabled bool
	var snapshotAt time.Time
	var lastSnapshotAt, lastGroupSyncAt sql.NullTime
	var localGroupID int64
	var localRate float64
	var localStatus, groupSyncStatus string
	err = tx.QueryRowContext(ctx, `
SELECT g.rate_guard_selected, g.active AS guardian_active, g.last_seen_at AS snapshot_at,
       g.rate_guard_last_snapshot_at AS last_snapshot_at, p.enabled AS provider_enabled,
       lg.id AS local_group_id, lg.rate_multiplier AS local_rate_multiplier, lg.status AS local_group_status,
       COALESCE(s.group_sync_status, 'never') AS group_sync_status, s.last_group_sync_at
FROM supplier_provider_groups g
JOIN supplier_providers p ON p.id = g.provider_id
JOIN groups lg ON lg.id = g.local_group_id AND lg.deleted_at IS NULL
LEFT JOIN supplier_provider_runtime_stats s ON s.provider_id = g.provider_id
WHERE g.id = $1
FOR UPDATE OF g, lg`, input.MappingID).Scan(
		&selected, &guardianActive, &snapshotAt, &lastSnapshotAt, &providerEnabled,
		&localGroupID, &localRate, &localStatus, &groupSyncStatus, &lastGroupSyncAt,
	)
	if err == sql.ErrNoRows {
		return service.SupplierRateGuardApplyResult{TargetRate: input.TargetRate, Action: service.SupplierRateGuardActionInvalid, Reason: service.SupplierRateGuardReasonSelectionChanged}, nil
	}
	if err != nil {
		return result, fmt.Errorf("lock supplier rate guard candidate: %w", err)
	}
	result.OldRate = localRate

	switch {
	case !selected:
		result.Action, result.Reason = service.SupplierRateGuardActionInvalid, service.SupplierRateGuardReasonSelectionChanged
	case !snapshotAt.Equal(input.ExpectedSnapshotAt):
		result.Action, result.Reason = service.SupplierRateGuardActionInvalid, service.SupplierRateGuardReasonSnapshotChanged
	case !providerEnabled:
		result.Action, result.Reason = service.SupplierRateGuardActionInvalid, service.SupplierRateGuardReasonProviderInactive
	case !guardianActive:
		result.Action, result.Reason = service.SupplierRateGuardActionInvalid, service.SupplierRateGuardReasonGuardianInactive
	case localStatus != service.StatusActive:
		result.Action, result.Reason = service.SupplierRateGuardActionInvalid, service.SupplierRateGuardReasonLocalGroupInactive
	case groupSyncStatus != service.SupplierSyncStatusSuccess:
		result.Action, result.Reason = service.SupplierRateGuardActionInvalid, service.SupplierRateGuardReasonGroupSyncFailed
	case snapshotAt.IsZero() || input.CheckedAt.Sub(snapshotAt) > input.MaxSnapshotAge:
		result.Action, result.Reason = service.SupplierRateGuardActionStale, service.SupplierRateGuardReasonSnapshotStale
	case lastSnapshotAt.Valid && !snapshotAt.After(lastSnapshotAt.Time):
		result.Action, result.Reason = service.SupplierRateGuardActionDuplicate, service.SupplierRateGuardReasonSnapshotDuplicate
	case localRate <= 0 || input.TargetRate <= 0:
		result.Action, result.Reason = service.SupplierRateGuardActionInvalid, service.SupplierRateGuardReasonRateInvalid
	case localRate >= input.TargetRate:
		result.Action = service.SupplierRateGuardActionUnchanged
	default:
		updateResult, err := tx.ExecContext(ctx, "UPDATE groups SET rate_multiplier=$2, updated_at=NOW() WHERE id=$1 AND rate_multiplier < $2", localGroupID, input.TargetRate)
		if err != nil {
			return result, fmt.Errorf("raise local group rate: %w", err)
		}
		affected, err := updateResult.RowsAffected()
		if err != nil {
			return result, fmt.Errorf("read local group rate update result: %w", err)
		}
		if affected > 0 {
			result.Action = service.SupplierRateGuardActionRaised
			if err := enqueueSchedulerOutbox(ctx, tx, service.SchedulerOutboxEventGroupChanged, nil, &localGroupID, nil); err != nil {
				return result, fmt.Errorf("enqueue guarded group rate change: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `
INSERT INTO supplier_rate_guard_change_logs (
    mapping_id, local_group_id, local_group_name, upstream_group_key, upstream_group_name,
    old_rate, new_rate, status, changed_at
)
SELECT g.id, lg.id, lg.name, g.upstream_group_key, g.name, $2, $3, $4, $5
FROM supplier_provider_groups g
JOIN groups lg ON lg.id = g.local_group_id AND lg.deleted_at IS NULL
WHERE g.id = $1`, input.MappingID, localRate, input.TargetRate, service.SupplierRateGuardChangeLogStatusPending, input.CheckedAt); err != nil {
				return result, fmt.Errorf("create supplier rate guard change log: %w", err)
			}
		} else {
			result.Action = service.SupplierRateGuardActionUnchanged
		}
	}

	if result.Action == service.SupplierRateGuardActionRaised || result.Action == service.SupplierRateGuardActionUnchanged {
		if _, err := tx.ExecContext(ctx, "UPDATE supplier_provider_groups SET rate_guard_last_snapshot_at=$2, rate_guard_last_checked_at=$3, updated_at=NOW() WHERE id=$1", input.MappingID, snapshotAt, input.CheckedAt); err != nil {
			return result, fmt.Errorf("mark supplier rate guard snapshot: %w", err)
		}
	} else if _, err := tx.ExecContext(ctx, "UPDATE supplier_provider_groups SET rate_guard_last_checked_at=$2, updated_at=NOW() WHERE id=$1", input.MappingID, input.CheckedAt); err != nil {
		return result, fmt.Errorf("mark supplier rate guard checked: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit supplier rate guard update: %w", err)
	}
	return result, nil
}

func (r *supplierProviderDataRepository) ListRateGuardChangeLogs(ctx context.Context, params service.SupplierRateGuardChangeLogListParams) (service.SupplierRateGuardChangeLogListResult, error) {
	params = normalizeSupplierRateGuardChangeLogListParams(params)
	var total, pendingCount int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM supplier_rate_guard_change_logs").Scan(&total); err != nil {
		return service.SupplierRateGuardChangeLogListResult{}, fmt.Errorf("count supplier rate guard change logs: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM supplier_rate_guard_change_logs WHERE status = $1", service.SupplierRateGuardChangeLogStatusPending).Scan(&pendingCount); err != nil {
		return service.SupplierRateGuardChangeLogListResult{}, fmt.Errorf("count pending supplier rate guard change logs: %w", err)
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, mapping_id, local_group_id, local_group_name, upstream_group_key, upstream_group_name,
       old_rate, new_rate, status, changed_at, handled_at, created_at
FROM supplier_rate_guard_change_logs
ORDER BY changed_at DESC, id DESC
LIMIT $1 OFFSET $2`, params.PageSize, (params.Page-1)*params.PageSize)
	if err != nil {
		return service.SupplierRateGuardChangeLogListResult{}, fmt.Errorf("query supplier rate guard change logs: %w", err)
	}
	defer rows.Close()
	items := make([]service.SupplierRateGuardChangeLog, 0)
	for rows.Next() {
		item, err := scanSupplierRateGuardChangeLog(rows)
		if err != nil {
			return service.SupplierRateGuardChangeLogListResult{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return service.SupplierRateGuardChangeLogListResult{}, err
	}
	return service.SupplierRateGuardChangeLogListResult{
		Items: items, Total: total, PendingCount: pendingCount, Page: params.Page, PageSize: params.PageSize,
	}, nil
}

func (r *supplierProviderDataRepository) MarkRateGuardChangeLogHandled(ctx context.Context, id int64) (service.SupplierRateGuardChangeLog, error) {
	item, err := scanSupplierRateGuardChangeLog(r.db.QueryRowContext(ctx, `
UPDATE supplier_rate_guard_change_logs
SET status = $2, handled_at = COALESCE(handled_at, NOW())
WHERE id = $1
RETURNING id, mapping_id, local_group_id, local_group_name, upstream_group_key, upstream_group_name,
          old_rate, new_rate, status, changed_at, handled_at, created_at`, id, service.SupplierRateGuardChangeLogStatusHandled))
	if err == sql.ErrNoRows {
		return service.SupplierRateGuardChangeLog{}, service.ErrSupplierProviderInvalid
	}
	if err != nil {
		return service.SupplierRateGuardChangeLog{}, fmt.Errorf("mark supplier rate guard change log handled: %w", err)
	}
	return item, nil
}

func normalizeSupplierRateGuardChangeLogListParams(params service.SupplierRateGuardChangeLogListParams) service.SupplierRateGuardChangeLogListParams {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 200 {
		params.PageSize = 20
	}
	return params
}

type supplierRateGuardChangeLogScanner interface{ Scan(dest ...any) error }

func scanSupplierRateGuardChangeLog(scanner supplierRateGuardChangeLogScanner) (service.SupplierRateGuardChangeLog, error) {
	var item service.SupplierRateGuardChangeLog
	var handledAt sql.NullTime
	if err := scanner.Scan(
		&item.ID, &item.MappingID, &item.LocalGroupID, &item.LocalGroupName, &item.UpstreamGroupKey, &item.UpstreamGroupName,
		&item.OldRate, &item.NewRate, &item.Status, &item.ChangedAt, &handledAt, &item.CreatedAt,
	); err != nil {
		return service.SupplierRateGuardChangeLog{}, err
	}
	if handledAt.Valid {
		item.HandledAt = &handledAt.Time
	}
	return item, nil
}

func (r *supplierProviderDataRepository) MarkRateGuardChecked(ctx context.Context, mappingID int64, checkedAt time.Time) error {
	if _, err := r.db.ExecContext(ctx, "UPDATE supplier_provider_groups SET rate_guard_last_checked_at=$2, updated_at=NOW() WHERE id=$1 AND rate_guard_selected=TRUE", mappingID, checkedAt); err != nil {
		return fmt.Errorf("mark supplier rate guard checked: %w", err)
	}
	return nil
}

func supplierGroupUpdateFound(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read supplier group update result: %w", err)
	}
	if affected == 0 {
		return service.ErrSupplierProviderGroupNotFound
	}
	return nil
}

func (r *supplierProviderDataRepository) ReplaceAccounts(ctx context.Context, providerID int64, items []service.SupplierProviderRemoteAccount, seenAt time.Time) (service.SupplierSyncCounts, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return service.SupplierSyncCounts{}, fmt.Errorf("begin supplier account replace: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	counts := service.SupplierSyncCounts{CheckedCount: len(items)}
	keys := make([]string, 0, len(items))
	validCount := 0
	schedulableCount := 0
	for _, item := range items {
		key := strings.TrimSpace(item.Key)
		name := strings.TrimSpace(item.Name)
		if key == "" && name != "" {
			key = strings.ToLower(strings.Join(strings.Fields(name), " "))
		}
		if key == "" && name == "" {
			counts.SkippedCount++
			continue
		}
		keys = append(keys, key)
		if strings.EqualFold(strings.TrimSpace(item.Status), "active") {
			validCount++
			schedulableCount++
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO supplier_provider_accounts (
  provider_id, upstream_account_key, name, status, group_key, group_name,
  rate_multiplier, raw_status, active, first_seen_at, last_seen_at, inactive_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,TRUE,$9,$9,NULL,$9)
ON CONFLICT (provider_id, upstream_account_key) DO UPDATE SET
  name=EXCLUDED.name, status=EXCLUDED.status, group_key=EXCLUDED.group_key,
  group_name=EXCLUDED.group_name, rate_multiplier=EXCLUDED.rate_multiplier,
  raw_status=EXCLUDED.raw_status, active=TRUE, last_seen_at=EXCLUDED.last_seen_at,
  inactive_at=NULL, updated_at=EXCLUDED.updated_at`, providerID, key, name, strings.TrimSpace(item.Status),
			strings.TrimSpace(item.GroupKey), strings.TrimSpace(item.GroupName), item.RateMultiplier,
			strings.TrimSpace(item.RawStatus), seenAt); err != nil {
			return service.SupplierSyncCounts{}, fmt.Errorf("upsert supplier account: %w", err)
		}
		counts.UpdatedCount++
	}

	result, err := tx.ExecContext(ctx, `
UPDATE supplier_provider_accounts
SET active = FALSE, status = 'deleted', raw_status = 'deleted', inactive_at = $3, updated_at = $3
WHERE provider_id = $1 AND active = TRUE AND NOT (upstream_account_key = ANY($2))`, providerID, pq.Array(keys), seenAt)
	if err != nil {
		return service.SupplierSyncCounts{}, fmt.Errorf("deactivate missing supplier accounts: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected > 0 {
		counts.SkippedCount += int(affected)
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE supplier_provider_runtime_stats
SET valid_account_count=$2, schedulable_account_count=$3, updated_at=NOW()
WHERE provider_id=$1`, providerID, validCount, schedulableCount); err != nil {
		return service.SupplierSyncCounts{}, fmt.Errorf("update supplier account runtime stats: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return service.SupplierSyncCounts{}, fmt.Errorf("commit supplier account replace: %w", err)
	}
	return counts, nil
}

func (r *supplierProviderDataRepository) ReplaceGroups(ctx context.Context, providerID int64, items []service.SupplierProviderRemoteGroup, seenAt time.Time) (service.SupplierSyncCounts, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return service.SupplierSyncCounts{}, fmt.Errorf("begin supplier group replace: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	counts := service.SupplierSyncCounts{CheckedCount: len(items)}
	keys := make([]string, 0, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.Key)
		name := strings.TrimSpace(item.Name)
		if key == "" && name != "" {
			key = strings.ToLower(strings.Join(strings.Fields(name), " "))
		}
		if key == "" && name == "" {
			counts.SkippedCount++
			continue
		}
		keys = append(keys, key)
		if _, err := tx.ExecContext(ctx, `
INSERT INTO supplier_provider_groups (
  provider_id, upstream_group_key, name, rate_multiplier, raw_status,
  active, first_seen_at, last_seen_at, inactive_at, updated_at
) VALUES ($1,$2,$3,$4,$5,TRUE,$6,$6,NULL,$6)
ON CONFLICT (provider_id, upstream_group_key) DO UPDATE SET
  name=EXCLUDED.name, rate_multiplier=EXCLUDED.rate_multiplier, raw_status=EXCLUDED.raw_status,
  active=TRUE, last_seen_at=EXCLUDED.last_seen_at, inactive_at=NULL, updated_at=EXCLUDED.updated_at`, providerID, key, name, item.RateMultiplier, strings.TrimSpace(item.RawStatus), seenAt); err != nil {
			return service.SupplierSyncCounts{}, fmt.Errorf("upsert supplier group: %w", err)
		}
		counts.UpdatedCount++
	}
	result, err := tx.ExecContext(ctx, `
UPDATE supplier_provider_groups SET active = FALSE, inactive_at = $3, updated_at = $3
WHERE provider_id = $1 AND active = TRUE AND NOT (upstream_group_key = ANY($2))`, providerID, pq.Array(keys), seenAt)
	if err != nil {
		return service.SupplierSyncCounts{}, fmt.Errorf("deactivate missing supplier groups: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected > 0 {
		counts.SkippedCount += int(affected)
	}
	if err := tx.Commit(); err != nil {
		return service.SupplierSyncCounts{}, fmt.Errorf("commit supplier group replace: %w", err)
	}
	return counts, nil
}

func (r *supplierProviderDataRepository) UpdateBalance(ctx context.Context, providerID int64, balance float64, seenAt time.Time) error {
	return r.updateMetric(ctx, providerID, &balance, nil, nil, nil, seenAt)
}

func (r *supplierProviderDataRepository) UpdateCost(ctx context.Context, providerID int64, cost float64, seenAt time.Time) error {
	return r.updateMetric(ctx, providerID, nil, &cost, nil, nil, seenAt)
}

// UpdateCostDetailed 写入成本生效值（today_cost），同时记录上游原始值 rawUpstream 与
// 偏差覆盖提示 costWarning；rawUpstream/warning 为 nil 时表示不记录。
func (r *supplierProviderDataRepository) UpdateCostDetailed(ctx context.Context, providerID int64, cost float64, rawUpstream *float64, warning *string, statDay time.Time) error {
	return r.updateMetric(ctx, providerID, nil, &cost, rawUpstream, warning, statDay)
}

func (r *supplierProviderDataRepository) UpdateCostDetailedWithReview(ctx context.Context, providerID int64, cost float64, rawUpstream *float64, warning *string, statDay time.Time, reviewInput service.SupplierProviderCostReviewSyncInput) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始写入供应商成本核对事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	review, err := syncCostReviewTx(ctx, tx, reviewInput, false)
	if err != nil {
		return err
	}
	if err := r.updateMetricTx(ctx, tx, providerID, nil, &review.EffectiveCost, rawUpstream, warning, statDay); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交供应商成本核对事务失败: %w", err)
	}
	// review.EffectiveCost 是已审批记录的保留值，或待审批记录的最新计算成本。
	// cost 参数保留用于兼容既有调用方，实际业务生效值以核对结果为准。
	_ = cost
	return nil
}

// GetCostFallbackBalances 返回成本保底估算所需的余额基线：
// CurrentBalance 为当前余额（runtime_stats.current_balance），
// DayStartBalance 为统计日 statDay 当天开始时的余额（前一统计日的 daily_stats 最终余额）。
// 当前余额记录缺失或为负时返回 ok=false，表示无法进行保底估算。
func (r *supplierProviderDataRepository) GetCostFallbackBalances(ctx context.Context, providerID int64, statDay time.Time) (service.SupplierProviderCostFallbackBalance, bool, error) {
	var bal service.SupplierProviderCostFallbackBalance
	err := r.db.QueryRowContext(ctx, `
SELECT s.current_balance,
       COALESCE((SELECT d.current_balance FROM supplier_provider_daily_stats d
                 WHERE d.provider_id = $1 AND d.stat_date < $2::date
                 ORDER BY d.stat_date DESC LIMIT 1), 0)
FROM supplier_provider_runtime_stats s
WHERE s.provider_id = $1`, providerID, supplierStatDate(statDay)).Scan(&bal.CurrentBalance, &bal.DayStartBalance)
	if err == sql.ErrNoRows {
		return service.SupplierProviderCostFallbackBalance{}, false, nil
	}
	if err != nil {
		return service.SupplierProviderCostFallbackBalance{}, false, fmt.Errorf("query supplier cost fallback balances: %w", err)
	}
	if bal.CurrentBalance < 0 {
		return service.SupplierProviderCostFallbackBalance{}, false, nil
	}
	return bal, true, nil
}

// GetBalanceDeltaForDay 获取指定供应商指定统计日的余额差值（当天第一条余额 - 当天最后一条余额）。
// 从 supplier_provider_metric_snapshots 表查询当天的余额快照，计算余额减少量作为成本估算。
// ok=false 表示当天没有足够的余额快照数据，无法计算差值。
func (r *supplierProviderDataRepository) GetBalanceDeltaForDay(ctx context.Context, providerID int64, day time.Time) (float64, bool, error) {
	start := supplierStatDate(day)
	end := start.AddDate(0, 0, 1)
	var firstBalance, lastBalance sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `
SELECT
  (SELECT current_balance FROM supplier_provider_metric_snapshots
   WHERE provider_id = $1 AND captured_at >= $2 AND captured_at < $3 AND current_balance > 0
   ORDER BY captured_at ASC LIMIT 1) AS first_balance,
  (SELECT current_balance FROM supplier_provider_metric_snapshots
   WHERE provider_id = $1 AND captured_at >= $2 AND captured_at < $3 AND current_balance > 0
   ORDER BY captured_at DESC LIMIT 1) AS last_balance
`, providerID, start, end).Scan(&firstBalance, &lastBalance)
	if err != nil {
		return 0, false, fmt.Errorf("query supplier balance delta for day: %w", err)
	}
	if !firstBalance.Valid || !lastBalance.Valid {
		return 0, false, nil
	}
	delta := firstBalance.Float64 - lastBalance.Float64
	if delta <= 0 {
		return 0, false, nil
	}
	return delta, true, nil
}

// GetLocalCostForDay 返回指定供应商在指定统计日的本地成本
// （唯一匹配本地账号 usage_logs.actual_cost 之和，口径与 ListCostBreakdowns 一致）。
// ok=false 表示当天没有唯一匹配且产生用量的本地账号，无法用本地口径校验。
func (r *supplierProviderDataRepository) GetLocalCostForDay(ctx context.Context, providerID int64, day time.Time) (float64, bool, error) {
	start := supplierStatDate(day)
	end := start.AddDate(0, 0, 1)
	var localCost float64
	var matchedCount int
	err := r.db.QueryRowContext(ctx, `
WITH provider_account_matches AS (
  SELECT p.id AS provider_id, local_account.id AS local_account_id
  FROM supplier_providers p
  JOIN supplier_provider_accounts spa
    ON spa.provider_id = p.id
   AND spa.active = TRUE
  JOIN accounts local_account
    ON local_account.deleted_at IS NULL
   AND regexp_replace(lower(local_account.name), '[^[:alnum:]]', '', 'g')
     = regexp_replace(lower(p.account_name_prefix || spa.name), '[^[:alnum:]]', '', 'g')
  WHERE p.id = $1
  GROUP BY p.id, local_account.id
),
unique_account_matches AS (
  SELECT MIN(provider_id) AS provider_id, local_account_id
  FROM provider_account_matches
  GROUP BY local_account_id
  HAVING COUNT(*) = 1
)
SELECT COALESCE(SUM(ul.actual_cost), 0) AS local_cost,
       COUNT(DISTINCT u.local_account_id) AS matched_count
FROM usage_logs ul
JOIN unique_account_matches u ON u.local_account_id = ul.account_id
WHERE ul.created_at >= $2
  AND ul.created_at < $3`, providerID, start, end).Scan(&localCost, &matchedCount)
	if err != nil {
		return 0, false, fmt.Errorf("query supplier local cost for day: %w", err)
	}
	return localCost, matchedCount > 0, nil
}

func (r *supplierProviderDataRepository) updateMetric(ctx context.Context, providerID int64, balance *float64, cost *float64, rawUpstream *float64, warning *string, seenAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin supplier metric update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := r.updateMetricTx(ctx, tx, providerID, balance, cost, rawUpstream, warning, seenAt); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit supplier metric update: %w", err)
	}
	return nil
}

func (r *supplierProviderDataRepository) updateMetricTx(ctx context.Context, tx *sql.Tx, providerID int64, balance *float64, cost *float64, rawUpstream *float64, warning *string, seenAt time.Time) error {
	if balance != nil {
		if _, err := tx.ExecContext(ctx, "UPDATE supplier_provider_runtime_stats SET current_balance=$2, updated_at=$3 WHERE provider_id=$1", providerID, *balance, seenAt); err != nil {
			return fmt.Errorf("update supplier balance: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO supplier_provider_metric_snapshots (provider_id, current_balance, today_cost, captured_at) VALUES ($1,$2,$3,$4)", providerID, *balance, 0.0, seenAt); err != nil {
			return fmt.Errorf("insert supplier balance snapshot: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO supplier_provider_daily_stats (provider_id, stat_date, current_balance)
VALUES ($1,$2,$3)
ON CONFLICT (provider_id, stat_date) DO UPDATE SET current_balance=EXCLUDED.current_balance, updated_at=NOW()`, providerID, supplierStatDate(seenAt), *balance); err != nil {
			return fmt.Errorf("upsert supplier daily balance: %w", err)
		}
	}
	if cost != nil {
		// seenAt 表示成本归属统计日；runtime.today_cost 仅在统计日为今天时更新。
		statDay := supplierStatDate(seenAt)
		capturedAt := time.Now()
		if supplierStatDate(capturedAt).Equal(statDay) {
			if _, err := tx.ExecContext(ctx, "UPDATE supplier_provider_runtime_stats SET today_cost=$2, updated_at=$3 WHERE provider_id=$1", providerID, *cost, capturedAt); err != nil {
				return fmt.Errorf("update supplier cost: %w", err)
			}
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO supplier_provider_metric_snapshots (provider_id, current_balance, today_cost, captured_at) VALUES ($1,$2,$3,$4)", providerID, 0.0, *cost, capturedAt); err != nil {
			return fmt.Errorf("insert supplier cost snapshot: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO supplier_provider_daily_stats (provider_id, stat_date, today_cost, raw_upstream_cost, cost_warning)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (provider_id, stat_date) DO UPDATE SET
  today_cost=EXCLUDED.today_cost,
  raw_upstream_cost=EXCLUDED.raw_upstream_cost,
  cost_warning=EXCLUDED.cost_warning,
  updated_at=NOW()`, providerID, statDay, *cost, rawUpstream, warning); err != nil {
			return fmt.Errorf("upsert supplier daily cost: %w", err)
		}
	}
	return nil
}

func (r *supplierProviderDataRepository) CreateSyncRun(ctx context.Context, run *service.SupplierProviderSyncRun) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO supplier_provider_sync_runs (
  provider_id, sync_scope, trigger_source, status, checked_count,
  created_count, updated_count, skipped_count, error_message, started_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
RETURNING id`, run.ProviderID, run.SyncScope, run.TriggerSource, run.Status, run.Counts.CheckedCount,
		run.Counts.CreatedCount, run.Counts.UpdatedCount, run.Counts.SkippedCount, run.ErrorMessage, run.StartedAt).Scan(&run.ID)
}

func (r *supplierProviderDataRepository) FinishSyncRun(ctx context.Context, run *service.SupplierProviderSyncRun) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE supplier_provider_sync_runs
SET status=$2, checked_count=$3, created_count=$4, updated_count=$5,
    skipped_count=$6, error_message=$7, finished_at=$8
WHERE id=$1`, run.ID, run.Status, run.Counts.CheckedCount, run.Counts.CreatedCount,
		run.Counts.UpdatedCount, run.Counts.SkippedCount, run.ErrorMessage, run.FinishedAt)
	return err
}

func (r *supplierProviderDataRepository) UpdateSyncStatus(ctx context.Context, providerID int64, status, message string, syncedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE supplier_provider_runtime_stats
SET sync_status=$2, sync_message=$3, last_sync_at=$4, updated_at=NOW()
WHERE provider_id=$1`, providerID, status, message, syncedAt)
	return err
}

func (r *supplierProviderDataRepository) UpdateGroupSyncStatus(ctx context.Context, providerID int64, status, message string, syncedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, "UPDATE supplier_provider_runtime_stats SET group_sync_status=$2, group_sync_message=$3, last_group_sync_at=$4, updated_at=$4 WHERE provider_id=$1", providerID, status, message, syncedAt)
	if err != nil {
		return fmt.Errorf("update supplier provider group sync status: %w", err)
	}
	return nil
}

func (r *supplierProviderDataRepository) Cleanup(ctx context.Context, policy service.SupplierCleanupPolicy, now time.Time, batchSize int) (service.SupplierCleanupCounts, error) {
	if batchSize <= 0 {
		batchSize = 1000
	}
	var counts service.SupplierCleanupCounts
	cleanupSpecs := []struct {
		table     string
		where     string
		cutoff    time.Time
		recipient *int
	}{
		{"supplier_automation_runs", "created_at < $1", now.AddDate(0, 0, -policy.AutomationRunRetentionDays), &counts.AutomationRuns},
		{"supplier_provider_sync_runs", "started_at < $1", now.AddDate(0, 0, -policy.SyncRunRetentionDays), &counts.SyncRuns},
		{"supplier_provider_metric_snapshots", "captured_at < $1", now.AddDate(0, 0, -policy.MetricRetentionDays), &counts.MetricSnapshots},
		{"supplier_provider_daily_stats", "stat_date < $1", now.AddDate(0, 0, -policy.DailyStatRetentionDays), &counts.DailyStats},
		{"supplier_provider_accounts", "active = FALSE AND inactive_at < $1", now.AddDate(0, 0, -policy.InactiveAccountDays), &counts.Accounts},
		{"supplier_provider_groups", "active = FALSE AND inactive_at < $1", now.AddDate(0, 0, -policy.InactiveGroupDays), &counts.Groups},
	}
	for _, spec := range cleanupSpecs {
		for {
			if err := ctx.Err(); err != nil {
				return counts, err
			}
			query := fmt.Sprintf(`
WITH target AS (
  SELECT id FROM %s WHERE %s ORDER BY id ASC LIMIT $2
)
DELETE FROM %s WHERE id IN (SELECT id FROM target)`, spec.table, spec.where, spec.table)
			result, err := r.db.ExecContext(ctx, query, spec.cutoff, batchSize)
			if err != nil {
				return counts, fmt.Errorf("cleanup %s: %w", spec.table, err)
			}
			affected, _ := result.RowsAffected()
			*spec.recipient += int(affected)
			if affected < int64(batchSize) {
				break
			}
		}
	}
	return counts, nil
}

func supplierProviderDataWhere(alias string, params service.SupplierProviderDataListParams) (string, []any) {
	conditions := []string{"p.deleted_at IS NULL"}
	args := make([]any, 0, 3)
	if params.ProviderID > 0 {
		args = append(args, params.ProviderID)
		conditions = append(conditions, fmt.Sprintf("%s.provider_id = $%d", alias, len(args)))
	}
	if params.Active != nil {
		args = append(args, *params.Active)
		conditions = append(conditions, fmt.Sprintf("%s.active = $%d", alias, len(args)))
	}
	if search := strings.TrimSpace(params.Search); search != "" {
		args = append(args, "%"+search+"%")
		conditions = append(conditions, fmt.Sprintf("(%s.name ILIKE $%d OR %s.upstream_%s_key ILIKE $%d)", alias, len(args), alias, supplierProviderDataKeyName(alias), len(args)))
	}
	return strings.Join(conditions, " AND "), args
}

// normalizeSupplierAccountStatusFilter 仅允许统一后的上游密钥状态，非法值视为不筛选。
func normalizeSupplierAccountStatusFilter(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active", "disabled", "expired", "quota_exhausted", "unknown":
		return strings.ToLower(strings.TrimSpace(status))
	default:
		return ""
	}
}

func supplierProviderAccountWhere(params service.SupplierProviderDataListParams) (string, []any) {
	where, args := supplierProviderDataWhere("a", params)
	conditions := []string{where}
	if status := normalizeSupplierAccountStatusFilter(params.Status); status != "" {
		args = append(args, status)
		conditions = append(conditions, fmt.Sprintf("LOWER(a.status) = $%d", len(args)))
	}
	if params.GroupID > 0 {
		args = append(args, params.GroupID)
		conditions = append(conditions, fmt.Sprintf(`EXISTS (
SELECT 1
FROM accounts local_account
JOIN account_groups account_group
  ON account_group.account_id = local_account.id
 AND account_group.group_id = $%d
JOIN groups local_group
  ON local_group.id = account_group.group_id
 AND local_group.deleted_at IS NULL
WHERE local_account.deleted_at IS NULL
  AND `+supplierProviderLocalAccountMatchCondition("local_account.name", "a.name")+`
  AND (
    SELECT COUNT(*)
    FROM accounts candidate
    WHERE candidate.deleted_at IS NULL
      AND `+supplierProviderLocalAccountMatchCondition("candidate.name", "a.name")+`
  ) = 1
)`, len(args)))
	}
	if platform := strings.TrimSpace(params.Platform); platform != "" {
		args = append(args, platform)
		conditions = append(conditions, fmt.Sprintf(`COALESCE(
(
  SELECT CASE
    WHEN COUNT(*) = 1 THEN MAX(COALESCE(NULLIF(platform_override.platform, ''), local_account.platform))
  END
  FROM accounts local_account
  LEFT JOIN supplier_local_account_platform_overrides platform_override
    ON platform_override.local_account_id = local_account.id
  WHERE local_account.deleted_at IS NULL
    AND `+supplierProviderLocalAccountMatchCondition("local_account.name", "a.name")+`
), (
  SELECT local_group.platform
  FROM supplier_provider_groups mapped_group
  JOIN groups local_group ON local_group.id = mapped_group.local_group_id AND local_group.deleted_at IS NULL
  WHERE mapped_group.provider_id = a.provider_id
    AND mapped_group.upstream_group_key = a.group_key
  LIMIT 1
), '') = $%d`, len(args)))
	}
	return strings.Join(conditions, " AND "), args
}
func supplierProviderGroupBaseWhere(params service.SupplierProviderDataListParams) (string, []any) {
	where, args := supplierProviderDataWhere("g", params)
	conditions := []string{where}
	if platform := strings.TrimSpace(params.Platform); platform != "" {
		args = append(args, platform)
		conditions = append(conditions, fmt.Sprintf("COALESCE(NULLIF(group_platform_override.actual_platform, ''), lg.platform) = $%d", len(args)))
	}
	return strings.Join(conditions, " AND "), args
}

func supplierProviderGroupListWhere(params service.SupplierProviderDataListParams) (string, []any) {
	where, args := supplierProviderGroupBaseWhere(params)
	conditions := []string{where}

	switch strings.TrimSpace(params.MatchStatus) {
	case "linked":
		conditions = append(conditions, "g.local_group_id IS NOT NULL")
	case "unlinked":
		conditions = append(conditions, "g.local_group_id IS NULL")
	case service.AutoMatchStatusAutoMatched:
		conditions = append(conditions, "g.local_group_id IS NOT NULL AND g.auto_match_status = 'auto_matched'")
	case service.AutoMatchStatusManual:
		conditions = append(conditions, "g.local_group_id IS NOT NULL AND g.auto_match_status = 'manual'")
	case service.AutoMatchStatusAmbiguous:
		conditions = append(conditions, "g.local_group_id IS NULL AND g.auto_match_status = 'ambiguous'")
	case "ignored":
		conditions = append(conditions, "g.auto_match_ignored = TRUE")
	case "name_changed":
		conditions = append(conditions, "g.name_change_pending = TRUE")
	}

	if keyStatusCondition := supplierProviderGroupKeyStatusCondition(params.KeyStatus); keyStatusCondition != "" {
		conditions = append(conditions, keyStatusCondition)
	}

	validRateCondition := "g.local_group_id IS NOT NULL AND COALESCE(lg.status, '') <> 'inactive' AND g.rate_multiplier > 0 AND lg.rate_multiplier > 0"
	switch strings.TrimSpace(params.RateStatus) {
	case "normal":
		conditions = append(conditions, validRateCondition+" AND lg.rate_multiplier >= g.rate_multiplier * 1.1")
	case "low":
		conditions = append(conditions, validRateCondition+" AND lg.rate_multiplier > g.rate_multiplier + 0.000000001 AND lg.rate_multiplier < g.rate_multiplier * 1.1")
	case "equal":
		conditions = append(conditions, validRateCondition+" AND ABS(lg.rate_multiplier - g.rate_multiplier) <= 0.000000001")
	case "inverted":
		conditions = append(conditions, validRateCondition+" AND lg.rate_multiplier < g.rate_multiplier - 0.000000001")
	case "inactive":
		conditions = append(conditions, "g.local_group_id IS NOT NULL AND COALESCE(lg.status, '') = 'inactive'")
	case "invalid":
		conditions = append(conditions, "g.local_group_id IS NOT NULL AND COALESCE(lg.status, '') <> 'inactive' AND (g.rate_multiplier <= 0 OR lg.rate_multiplier IS NULL OR lg.rate_multiplier <= 0)")
	}

	return strings.Join(conditions, " AND "), args
}

func supplierProviderGroupKeyStatusCondition(status string) string {
	activeKeyExists := `EXISTS (
  SELECT 1
  FROM supplier_provider_accounts key_account
  WHERE key_account.provider_id = g.provider_id
    AND key_account.group_key = g.upstream_group_key
    AND key_account.active = TRUE
)`
	lastAccountSyncStatus := `COALESCE((
  SELECT account_sync.status
  FROM supplier_provider_sync_runs account_sync
  WHERE account_sync.provider_id = g.provider_id
    AND account_sync.sync_scope = 'accounts'
  ORDER BY account_sync.started_at DESC, account_sync.id DESC
  LIMIT 1
), 'never')`

	switch strings.ToLower(strings.TrimSpace(status)) {
	case "created":
		return activeKeyExists
	case "not_created":
		return "NOT " + activeKeyExists + " AND " + lastAccountSyncStatus + " IN ('success', 'partial')"
	case "unknown":
		return "NOT " + activeKeyExists + " AND " + lastAccountSyncStatus + " NOT IN ('success', 'partial')"
	default:
		return ""
	}
}

func supplierProviderGroupHasListFilters(params service.SupplierProviderDataListParams) bool {
	return strings.TrimSpace(params.MatchStatus) != "" || strings.TrimSpace(params.RateStatus) != "" || supplierProviderGroupKeyStatusCondition(params.KeyStatus) != ""
}

func supplierProviderAccountOrderBy(params service.SupplierProviderDataListParams) string {
	sortBy := strings.TrimSpace(params.SortBy)
	direction := "ASC"
	if strings.EqualFold(strings.TrimSpace(params.SortOrder), "desc") {
		direction = "DESC"
	}

	var expression string
	var nullsLast bool
	switch sortBy {
	case "provider_name":
		expression = "LOWER(p.name)"
	case "upstream_account_key":
		return fmt.Sprintf("LOWER(a.name) %s, LOWER(a.upstream_account_key) %s, a.id ASC", direction, direction)
	case "upstream_account_status":
		expression = "LOWER(a.status)"
	case "local_account_name":
		expression = "LOWER(matched_account.name)"
		nullsLast = true
	case "local_account_priority":
		expression = "matched_account.priority"
		nullsLast = true
	case "rate_multiplier":
		expression = "a.rate_multiplier"
	case "local_account_status":
		expression = "LOWER(matched_account.status)"
		nullsLast = true
	case "local_account_schedulable":
		expression = "matched_account.schedulable"
		nullsLast = true
	case "local_account_last_test_status":
		expression = "LOWER(NULLIF(matched_account.extra->>'last_test_status', ''))"
		nullsLast = true
	case "local_account_last_tested_at":
		expression = "NULLIF(matched_account.extra->>'last_tested_at', '')"
		nullsLast = true
	case "supplier_current_balance":
		expression = "COALESCE(runtime.current_balance, 0)"
	case "supplier_today_cost":
		expression = "COALESCE(runtime.today_cost, 0)"
	default:
		return "a.active DESC, a.last_seen_at DESC, a.id ASC"
	}

	nulls := ""
	if nullsLast {
		nulls = " NULLS LAST"
	}
	return fmt.Sprintf("%s %s%s, a.id ASC", expression, direction, nulls)
}

func supplierProviderGroupOrderBy(params service.SupplierProviderDataListParams) string {
	sortBy := strings.TrimSpace(params.SortBy)
	direction := "ASC"
	if strings.EqualFold(strings.TrimSpace(params.SortOrder), "desc") {
		direction = "DESC"
	}

	var expression string
	switch sortBy {
	case "provider_name":
		expression = "LOWER(p.name)"
	case "name":
		expression = "LOWER(g.name)"
	case "rate_multiplier":
		expression = "g.rate_multiplier"
	case "local_group_name":
		expression = "LOWER(lg.name)"
	case "local_rate_multiplier":
		expression = "lg.rate_multiplier"
	case "account_count":
		expression = "COALESCE(COUNT(a.id) FILTER (WHERE a.active = TRUE), 0)"
	default:
		return "g.active DESC, g.last_seen_at DESC, g.id ASC"
	}

	if sortBy == "local_group_name" || sortBy == "local_rate_multiplier" {
		return fmt.Sprintf("%s %s NULLS LAST, g.id ASC", expression, direction)
	}
	return fmt.Sprintf("%s %s, g.id ASC", expression, direction)
}

func supplierProviderDataKeyName(alias string) string {
	if alias == "g" {
		return "group"
	}
	return "account"
}

func normalizeSupplierProviderDataListParams(params service.SupplierProviderDataListParams) service.SupplierProviderDataListParams {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 200 {
		params.PageSize = 50
	}
	return params
}

func normalizeSupplierProviderMonitorTargetListParams(params service.SupplierProviderMonitorTargetListParams) service.SupplierProviderMonitorTargetListParams {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 200 {
		params.PageSize = 50
	}
	params.Search = strings.TrimSpace(params.Search)
	return params
}

func supplierStatDate(seenAt time.Time) time.Time {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	local := seenAt.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

func normalizeSupplierProviderMonitorStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "operational", service.SupplierAccountHealthGuardStatusHealthy:
		return service.SupplierAccountHealthGuardStatusHealthy
	case "degraded", service.SupplierAccountHealthGuardStatusSlow:
		return service.SupplierAccountHealthGuardStatusSlow
	case "error", service.SupplierAccountHealthGuardStatusFailed:
		return service.SupplierAccountHealthGuardStatusFailed
	default:
		return service.SupplierAccountHealthGuardStatusUnavailable
	}
}

type supplierProviderAccountScanner interface{ Scan(dest ...any) error }

type supplierProviderMonitorTargetScanner interface{ Scan(dest ...any) error }

func scanSupplierProviderMonitorTarget(scanner supplierProviderMonitorTargetScanner) (service.SupplierProviderMonitorTarget, error) {
	var item service.SupplierProviderMonitorTarget
	var bindingGroupsJSON []byte
	if err := scanner.Scan(&item.ID, &item.ProviderID, &item.ProviderName, &item.MonitorKey, &item.MonitorName, &item.MonitorProvider,
		&item.PrimaryModel, &item.Availability7D, &item.Active, &item.LastSeenAt,
		&item.LocalAccountID, &item.LocalAccountName, &bindingGroupsJSON); err != nil {
		return service.SupplierProviderMonitorTarget{}, err
	}
	if err := json.Unmarshal(bindingGroupsJSON, &item.BindingGroups); err != nil {
		return service.SupplierProviderMonitorTarget{}, fmt.Errorf("decode supplier provider monitor target binding groups: %w", err)
	}
	return item, nil
}

func scanSupplierProviderAccount(scanner supplierProviderAccountScanner) (service.SupplierProviderAccount, error) {
	var item service.SupplierProviderAccount
	var inactiveAt sql.NullTime
	var localAccountID sql.NullInt64
	var localAccountPriority sql.NullInt64
	var localAccountSchedulable sql.NullBool
	var bindingGroupsJSON []byte
	var groupRecordID sql.NullInt64
	err := scanner.Scan(&item.ID, &item.ProviderID, &item.ProviderName, &item.UpstreamKey,
		&item.Name, &item.Status, &item.GroupKey, &item.GroupName, &item.Platform, &item.GroupStatus, &item.RateMultiplier,
		&item.RawStatus, &item.Active, &item.LastSeenAt, &inactiveAt,
		&item.LocalAccountMatchStatus, &item.LocalAccountMatchCount,
		&localAccountID, &item.LocalAccountName, &item.LocalAccountPlatform, &item.LocalAccountType, &item.PlatformOverride, &item.EffectivePlatform, &localAccountPriority,
		&item.LocalAccountStatus, &localAccountSchedulable,
		&item.LocalAccountLastTestStatus, &item.LocalAccountLastTestedAt, &item.LocalAccountLastTestError,
		&bindingGroupsJSON,
		&item.SupplierCurrentBalance, &item.SupplierTodayCost,
		&groupRecordID, &item.GroupRecordDeleteEligible)
	if err != nil {
		return service.SupplierProviderAccount{}, err
	}
	if inactiveAt.Valid {
		item.InactiveAt = &inactiveAt.Time
	}
	if localAccountID.Valid {
		value := localAccountID.Int64
		item.LocalAccountID = &value
	}
	if localAccountPriority.Valid {
		value := int(localAccountPriority.Int64)
		item.LocalAccountPriority = &value
	}
	if localAccountSchedulable.Valid {
		value := localAccountSchedulable.Bool
		item.LocalAccountSchedulable = &value
	}
	if groupRecordID.Valid {
		value := groupRecordID.Int64
		item.GroupRecordID = &value
	}
	if err := json.Unmarshal(bindingGroupsJSON, &item.BindingGroups); err != nil {
		return service.SupplierProviderAccount{}, fmt.Errorf("decode supplier provider account binding groups: %w", err)
	}
	item.AccountRecordDeleteEligible = true
	return item, nil
}

type supplierProviderGroupScanner interface{ Scan(dest ...any) error }

func supplierProviderGroupKeyStatus(keySyncStatus string, activeKeyCount int) string {
	if activeKeyCount > 0 {
		return "created"
	}
	switch strings.ToLower(strings.TrimSpace(keySyncStatus)) {
	case service.SupplierSyncStatusSuccess, service.SupplierSyncStatusPartial:
		return "not_created"
	default:
		return "unknown"
	}
}

func scanSupplierProviderGroup(scanner supplierProviderGroupScanner) (service.SupplierProviderGroup, error) {
	var item service.SupplierProviderGroup
	var localGroupRateGuardGroupID sql.NullInt64
	var rateGuardLastSnapshotAt, rateGuardLastCheckedAt, lastGroupSyncAt, inactiveAt sql.NullTime
	err := scanner.Scan(&item.ID, &item.ProviderID, &item.ProviderName, &item.UpstreamKey,
		&item.Name, &item.RateMultiplier, &item.RawStatus, &item.Active,
		&item.LocalGroupID, &item.LocalGroupName, &item.LocalGroupPlatform,
		&item.PlatformOverride, &item.EffectivePlatform,
		&item.LocalRateMultiplier, &item.LocalGroupStatus,
		&item.AutoMatchIgnored, &item.AutoMatchStatus, &item.MatchedUpstreamName,
		&item.NameChangePending,
		&item.RateGuardSelected, &item.RateGuardEnabled, &item.RateGuardSelectionMode,
		&rateGuardLastSnapshotAt, &rateGuardLastCheckedAt,
		&item.GroupSyncStatus, &lastGroupSyncAt,
		&item.LocalGroupActiveMappingCount, &localGroupRateGuardGroupID,
		&item.LocalGroupRateGuardGroupName, &item.LocalGroupRateGuardProviderName,
		&item.AccountCount, &item.LastSeenAt, &inactiveAt,
		&item.KeySyncStatus)
	if err != nil {
		return service.SupplierProviderGroup{}, err
	}
	if inactiveAt.Valid {
		item.InactiveAt = &inactiveAt.Time
	}
	if rateGuardLastSnapshotAt.Valid {
		item.RateGuardLastSnapshotAt = &rateGuardLastSnapshotAt.Time
	}
	if rateGuardLastCheckedAt.Valid {
		item.RateGuardLastCheckedAt = &rateGuardLastCheckedAt.Time
	}
	if lastGroupSyncAt.Valid {
		item.LastGroupSyncAt = &lastGroupSyncAt.Time
	}
	if localGroupRateGuardGroupID.Valid {
		item.LocalGroupRateGuardGroupID = &localGroupRateGuardGroupID.Int64
	}
	item.KeyStatus = supplierProviderGroupKeyStatus(item.KeySyncStatus, item.AccountCount)
	return item, nil
}

func scanSupplierProviderGroupAutoMatch(scanner supplierProviderGroupScanner) (service.SupplierProviderGroup, error) {
	var item service.SupplierProviderGroup
	var inactiveAt sql.NullTime
	err := scanner.Scan(
		&item.ID, &item.ProviderID, &item.ProviderName, &item.UpstreamKey, &item.Name,
		&item.RateMultiplier, &item.RawStatus, &item.Active, &item.LocalGroupID,
		&item.AutoMatchIgnored, &item.AutoMatchStatus, &item.MatchedUpstreamName,
		&item.NameChangePending, &item.LastSeenAt, &inactiveAt,
	)
	if err != nil {
		return service.SupplierProviderGroup{}, err
	}
	if inactiveAt.Valid {
		item.InactiveAt = &inactiveAt.Time
	}
	return item, nil
}

func scanSupplierProviderGroupGuard(scanner supplierProviderGroupScanner) (service.SupplierProviderGroup, error) {
	var item service.SupplierProviderGroup
	var inactiveAt sql.NullTime
	err := scanner.Scan(
		&item.ID, &item.ProviderID, &item.ProviderName, &item.UpstreamKey, &item.Name,
		&item.RateMultiplier, &item.RawStatus, &item.Active, &item.LocalGroupID,
		&item.AutoMatchIgnored, &item.AutoMatchStatus, &item.MatchedUpstreamName,
		&item.NameChangePending, &item.RateGuardSelected, &item.RateGuardEnabled, &item.RateGuardSelectionMode,
		&item.LastSeenAt, &inactiveAt,
	)
	if err != nil {
		return service.SupplierProviderGroup{}, err
	}
	if inactiveAt.Valid {
		item.InactiveAt = &inactiveAt.Time
	}
	return item, nil
}

func (r *supplierProviderDataRepository) IsUniqueMatchedLocalAccount(ctx context.Context, localAccountID int64) (bool, error) {
	if localAccountID <= 0 {
		return false, fmt.Errorf("local account id must be positive")
	}
	var matched bool
	query := `
SELECT EXISTS (
  SELECT 1
  FROM supplier_provider_accounts a
  JOIN supplier_providers p ON p.id = a.provider_id
  WHERE p.deleted_at IS NULL
    AND (
      SELECT COUNT(*)
      FROM accounts local_account
      WHERE local_account.deleted_at IS NULL
        AND %s
    ) = 1
    AND (
      SELECT MIN(local_account.id)
      FROM accounts local_account
      WHERE local_account.deleted_at IS NULL
        AND %s
    ) = $1
)`
	query = fmt.Sprintf(query, supplierProviderLocalAccountMatchCondition("local_account.name", "a.name"), supplierProviderLocalAccountMatchCondition("local_account.name", "a.name"))
	err := r.db.QueryRowContext(ctx, query, localAccountID).Scan(&matched)
	if err != nil {
		return false, fmt.Errorf("check supplier local account match: %w", err)
	}
	return matched, nil
}

func supplierProviderLocalAccountMatchCondition(localAccountNameSQL, upstreamAccountNameSQL string) string {
	normalize := supplierProviderLocalAccountNormalizeSQL
	reorderedUpstreamName := supplierProviderReorderedAccountNameAliasSQL(upstreamAccountNameSQL)
	prefix := normalize("COALESCE(p.account_name_prefix, '')")
	providerName := normalize("COALESCE(p.name, '')")
	return fmt.Sprintf(`%s IN (
  %s,
  %s || %s,
  %s,
  %s,
  %s,
  %s || %s
)`, normalize(localAccountNameSQL),
		normalize("COALESCE(p.account_name_prefix, '') || "+upstreamAccountNameSQL),
		prefix, reorderedUpstreamName,
		normalize(upstreamAccountNameSQL),
		reorderedUpstreamName,
		normalize("COALESCE(p.name, '') || '-' || "+upstreamAccountNameSQL),
		providerName, reorderedUpstreamName)
}

func supplierProviderLocalAccountNormalizeSQL(value string) string {
	return fmt.Sprintf("regexp_replace(lower(%s), '[^[:alnum:]]', '', 'g')", value)
}

func supplierProviderReorderedAccountNameAliasSQL(value string) string {
	normalized := supplierProviderLocalAccountNormalizeSQL(value)
	return fmt.Sprintf(`CASE
  WHEN %s ~ '^[^a-z0-9]+[a-z]+[0-9]*$'
    THEN regexp_replace(%s, '^([^a-z0-9]+)([a-z]+)([0-9]*)$', '\2\1\3')
  WHEN %s ~ '^[a-z]+[^a-z0-9]+[0-9]*$'
    THEN regexp_replace(%s, '^([a-z]+)([^a-z0-9]+)([0-9]*)$', '\2\1\3')
  ELSE %s
END`, normalized, normalized, normalized, normalized, normalized)
}
func (r *supplierProviderDataRepository) GetLocalAccountEffectivePlatform(ctx context.Context, localAccountID int64) (string, error) {
	if localAccountID <= 0 {
		return "", fmt.Errorf("local account id must be positive")
	}
	var platform string
	err := r.db.QueryRowContext(ctx, `
SELECT COALESCE(
  NULLIF(platform_override.platform, ''),
  NULLIF(local_account.platform, ''),
  ''
)
FROM accounts local_account
LEFT JOIN supplier_local_account_platform_overrides platform_override
  ON platform_override.local_account_id = local_account.id
WHERE local_account.id = $1
  AND local_account.deleted_at IS NULL`, localAccountID).Scan(&platform)
	if err != nil {
		return "", fmt.Errorf("get supplier local account effective platform: %w", err)
	}
	return strings.ToLower(strings.TrimSpace(platform)), nil
}

func (r *supplierProviderDataRepository) GetLocalAccountPlatformOverride(ctx context.Context, localAccountID int64) (string, error) {
	if localAccountID <= 0 {
		return "", fmt.Errorf("local account id must be positive")
	}
	var platform string
	err := r.db.QueryRowContext(ctx, `SELECT platform FROM supplier_local_account_platform_overrides WHERE local_account_id = $1`, localAccountID).Scan(&platform)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get supplier local account platform override: %w", err)
	}
	return strings.ToLower(strings.TrimSpace(platform)), nil
}

func (r *supplierProviderDataRepository) SetLocalAccountPlatformOverride(ctx context.Context, localAccountID int64, platform string) error {
	if localAccountID <= 0 {
		return fmt.Errorf("local account id must be positive")
	}
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform == "" {
		return fmt.Errorf("platform must not be empty")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO supplier_local_account_platform_overrides (local_account_id, platform)
VALUES ($1, $2)
ON CONFLICT (local_account_id) DO UPDATE
SET platform = EXCLUDED.platform, updated_at = NOW()`, localAccountID, platform)
	if err != nil {
		return fmt.Errorf("set supplier local account platform override: %w", err)
	}
	return nil
}

func (r *supplierProviderDataRepository) ClearLocalAccountPlatformOverride(ctx context.Context, localAccountID int64) error {
	if localAccountID <= 0 {
		return fmt.Errorf("local account id must be positive")
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM supplier_local_account_platform_overrides WHERE local_account_id = $1`, localAccountID)
	if err != nil {
		return fmt.Errorf("clear supplier local account platform override: %w", err)
	}
	return nil
}

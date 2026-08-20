package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type supplierProviderRepository struct {
	db *sql.DB
}

type supplierProviderTypeRepository struct {
	db *sql.DB
}

func NewSupplierProviderRepository(db *sql.DB) service.SupplierProviderRepository {
	return &supplierProviderRepository{db: db}
}

func NewSupplierProviderTypeRepository(db *sql.DB) service.SupplierProviderTypeRepository {
	return &supplierProviderTypeRepository{db: db}
}

const supplierProviderSelect = `
SELECT p.id, p.code, p.name, p.provider_type, p.newapi_auth_mode, p.base_url, p.login_url,
       p.api_keys_url, p.groups_url, p.available_groups_url, p.balance_url,
       p.usage_cost_url, p.recharge_url, p.monitor_url, p.account_name_prefix, p.temp_disable_minutes,
       p.account_rate_multiplier_scale, p.sort_order, p.enabled, p.turnstile_enabled, p.is_default,
       p.created_at, p.updated_at,
       COALESCE(c.email, ''), COALESCE(c.username, ''), COALESCE(c.password_encrypted, ''),
       COALESCE(s.status, 'unknown'), COALESCE(s.risk_level, 'normal'),
       COALESCE(s.valid_account_count, 0), COALESCE(s.schedulable_account_count, 0),
       COALESCE(s.request_count, 0), COALESCE(s.success_rate, 0),
       COALESCE(s.period_cost, 0), COALESCE(s.current_balance, 0),
       COALESCE(s.today_cost, 0), s.estimated_days, COALESCE(s.rate_risk_count, 0),
       COALESCE(s.sync_status, 'never'), COALESCE(s.sync_message, ''), s.last_sync_at,
       COALESCE(s.auth_login_count, 0), COALESCE(s.auth_login_success_count, 0),
       COALESCE(s.auth_login_failure_count, 0), COALESCE(s.auth_cache_hit_count, 0),
       COALESCE(s.auth_cache_miss_count, 0), s.auth_last_login_at,
       COALESCE(s.auth_last_login_status, ''), COALESCE(s.auth_last_login_error, ''),
       s.auth_last_cache_hit_at, COALESCE(s.auth_last_cache_error, ''),
       s.auth_last_token_expires_at, COALESCE(s.auth_last_token_fingerprint, '')
FROM supplier_providers p
LEFT JOIN supplier_provider_credentials c ON c.provider_id = p.id
LEFT JOIN supplier_provider_runtime_stats s ON s.provider_id = p.id`

func (r *supplierProviderRepository) List(ctx context.Context, params service.SupplierProviderListParams) ([]*service.SupplierProvider, int64, error) {
	where, args := supplierProviderWhere(params)
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM supplier_providers p WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count supplier providers: %w", err)
	}
	page, pageSize := params.Page, params.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 100
	}
	queryArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	query := supplierProviderSelect + " WHERE " + where + fmt.Sprintf(" ORDER BY p.sort_order ASC, p.id ASC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query supplier providers: %w", err)
	}
	defer rows.Close()
	items := make([]*service.SupplierProvider, 0)
	for rows.Next() {
		provider, scanErr := scanSupplierProvider(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, provider)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate supplier providers: %w", err)
	}
	return items, total, nil
}

func (r *supplierProviderRepository) Summary(ctx context.Context, params service.SupplierProviderListParams) (service.SupplierProviderSummary, error) {
	where, args := supplierProviderWhere(params)
	var summary service.SupplierProviderSummary
	err := r.db.QueryRowContext(ctx, `
SELECT
  COUNT(*) AS total_count,
  COUNT(*) FILTER (WHERE p.enabled = TRUE) AS enabled_count,
  COUNT(*) FILTER (WHERE COALESCE(s.risk_level, 'normal') IN ('high', 'critical')) AS high_risk_count,
  COUNT(*) FILTER (WHERE s.estimated_days IS NOT NULL AND s.estimated_days < 3) AS low_balance_count,
  COUNT(*) FILTER (WHERE COALESCE(s.sync_status, 'never') = 'failed') AS sync_failure_count,
  COALESCE(SUM(COALESCE(s.rate_risk_count, 0)), 0) AS rate_risk_count
FROM supplier_providers p
LEFT JOIN supplier_provider_runtime_stats s ON s.provider_id = p.id
WHERE `+where, args...).Scan(
		&summary.TotalCount,
		&summary.EnabledCount,
		&summary.HighRiskCount,
		&summary.LowBalanceCount,
		&summary.SyncFailureCount,
		&summary.RateRiskCount,
	)
	if err != nil {
		return service.SupplierProviderSummary{}, fmt.Errorf("summarize supplier providers: %w", err)
	}
	return summary, nil
}

func (r *supplierProviderRepository) ListCostTrends(ctx context.Context, start, end time.Time, providerID int64) ([]service.SupplierProviderCostTrendPoint, error) {
	byDate := make(map[string]*service.SupplierProviderCostTrendPoint)

	upstreamQuery := `
SELECT TO_CHAR(d.stat_date, 'YYYY-MM-DD') AS date,
       COALESCE(SUM(d.today_cost), 0) AS upstream_cost,
       SUM(d.raw_upstream_cost) AS raw_upstream_cost,
       COALESCE(string_agg(DISTINCT d.cost_warning, '；') FILTER (WHERE d.cost_warning IS NOT NULL), '') AS cost_warning
FROM supplier_provider_daily_stats d
JOIN supplier_providers p ON p.id = d.provider_id AND p.deleted_at IS NULL
WHERE d.stat_date >= $1::date
  AND d.stat_date < $2::date`
	upstreamArgs := []any{start, end}
	if providerID > 0 {
		upstreamQuery += `
  AND d.provider_id = $3`
		upstreamArgs = append(upstreamArgs, providerID)
	}
	upstreamQuery += `
GROUP BY d.stat_date
ORDER BY d.stat_date`

	upstreamRows, err := r.db.QueryContext(ctx, upstreamQuery, upstreamArgs...)
	if err != nil {
		return nil, fmt.Errorf("query supplier upstream cost trends: %w", err)
	}
	defer upstreamRows.Close()

	for upstreamRows.Next() {
		var date string
		var upstreamCost float64
		var rawUpstream sql.NullFloat64
		var costWarning sql.NullString
		if scanErr := upstreamRows.Scan(&date, &upstreamCost, &rawUpstream, &costWarning); scanErr != nil {
			return nil, fmt.Errorf("scan supplier upstream cost trend: %w", scanErr)
		}
		point := byDate[date]
		if point == nil {
			point = &service.SupplierProviderCostTrendPoint{Date: date}
			byDate[date] = point
		}
		point.UpstreamCost = upstreamCost
		if rawUpstream.Valid {
			raw := rawUpstream.Float64
			point.RawUpstreamCost = &raw
		}
		point.Warning = costWarning.String
	}
	if err := upstreamRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier upstream cost trends: %w", err)
	}

	tzName := timezone.Name()
	if strings.TrimSpace(tzName) == "" {
		tzName = "Asia/Shanghai"
	}

	localQuery := `
WITH matched_accounts AS (
  SELECT local_account.id AS local_account_id
  FROM supplier_providers p
  JOIN supplier_provider_accounts spa
    ON spa.provider_id = p.id
   AND spa.active = TRUE
  JOIN accounts local_account
    ON local_account.deleted_at IS NULL
   AND regexp_replace(lower(local_account.name), '[^[:alnum:]]', '', 'g')
     = regexp_replace(lower(p.account_name_prefix || spa.name), '[^[:alnum:]]', '', 'g')
  WHERE p.deleted_at IS NULL`
	localArgs := []any{start, end, tzName}
	if providerID > 0 {
		localQuery += `
    AND p.id = $4`
		localArgs = append(localArgs, providerID)
	}
	localQuery += `
  GROUP BY local_account.id
  HAVING COUNT(*) = 1
)
SELECT TO_CHAR(ul.created_at AT TIME ZONE $3, 'YYYY-MM-DD') AS date,
       COALESCE(SUM(ul.actual_cost), 0) AS local_cost
FROM usage_logs ul
JOIN matched_accounts matched ON matched.local_account_id = ul.account_id
WHERE ul.created_at >= $1
  AND ul.created_at < $2
GROUP BY 1
ORDER BY 1`

	localRows, err := r.db.QueryContext(ctx, localQuery, localArgs...)
	if err != nil {
		return nil, fmt.Errorf("query supplier local cost trends: %w", err)
	}
	defer localRows.Close()

	for localRows.Next() {
		var date string
		var localCost float64
		if scanErr := localRows.Scan(&date, &localCost); scanErr != nil {
			return nil, fmt.Errorf("scan supplier local cost trend: %w", scanErr)
		}
		point := byDate[date]
		if point == nil {
			point = &service.SupplierProviderCostTrendPoint{Date: date}
			byDate[date] = point
		}
		point.LocalCost = localCost
	}
	if err := localRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier local cost trends: %w", err)
	}

	points := make([]service.SupplierProviderCostTrendPoint, 0, len(byDate))
	for _, point := range byDate {
		points = append(points, *point)
	}
	sort.Slice(points, func(i, j int) bool {
		return points[i].Date < points[j].Date
	})
	return points, nil
}

func (r *supplierProviderRepository) ListCostBreakdowns(ctx context.Context, start, end time.Time, providerID int64) ([]service.SupplierProviderCostBreakdown, error) {
	query := `
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
  WHERE p.deleted_at IS NULL
  GROUP BY p.id, local_account.id
),
unique_account_matches AS (
  SELECT MIN(provider_id) AS provider_id, local_account_id
  FROM provider_account_matches
  GROUP BY local_account_id
  HAVING COUNT(*) = 1
),
upstream_costs AS (
  SELECT d.provider_id,
         COALESCE(SUM(d.today_cost), 0) AS upstream_cost,
         SUM(d.raw_upstream_cost) AS raw_upstream_cost,
         COALESCE(string_agg(DISTINCT d.cost_warning, '；') FILTER (WHERE d.cost_warning IS NOT NULL), '') AS cost_warning
  FROM supplier_provider_daily_stats d
  WHERE d.stat_date >= $1::date
    AND d.stat_date < $2::date
  GROUP BY d.provider_id
),
local_costs AS (
  SELECT matches.provider_id, COALESCE(SUM(ul.actual_cost), 0) AS local_cost
  FROM unique_account_matches matches
  JOIN usage_logs ul ON ul.account_id = matches.local_account_id
  WHERE ul.created_at >= $1
    AND ul.created_at < $2
  GROUP BY matches.provider_id
)
SELECT p.id, p.name, p.provider_type,
       COALESCE(upstream.upstream_cost, 0) AS upstream_cost,
       COALESCE(local_agg.local_cost, 0) AS local_cost,
       COALESCE(upstream.raw_upstream_cost, 0) AS raw_upstream_cost,
       COALESCE(upstream.cost_warning, '') AS cost_warning
FROM supplier_providers p
LEFT JOIN upstream_costs upstream ON upstream.provider_id = p.id
LEFT JOIN local_costs local_agg ON local_agg.provider_id = p.id
WHERE p.deleted_at IS NULL`
	args := []any{start, end}
	if providerID > 0 {
		query += `
  AND p.id = $3`
		args = append(args, providerID)
	}
	query += `
ORDER BY p.sort_order ASC, p.id ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query supplier provider cost breakdowns: %w", err)
	}
	defer rows.Close()

	breakdowns := make([]service.SupplierProviderCostBreakdown, 0)
	for rows.Next() {
		var breakdown service.SupplierProviderCostBreakdown
		if scanErr := rows.Scan(
			&breakdown.ProviderID,
			&breakdown.ProviderName,
			&breakdown.ProviderType,
			&breakdown.UpstreamCost,
			&breakdown.LocalCost,
			&breakdown.RawUpstreamCost,
			&breakdown.CostWarning,
		); scanErr != nil {
			return nil, fmt.Errorf("scan supplier provider cost breakdown: %w", scanErr)
		}
		breakdowns = append(breakdowns, breakdown)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier provider cost breakdowns: %w", err)
	}
	return breakdowns, nil
}

// ListBalanceSummaryDays 按统计日汇总全部供应商的余额与成本快照。
func (r *supplierProviderRepository) ListBalanceSummaryDays(ctx context.Context) ([]service.SupplierProviderBalanceSummaryDay, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT TO_CHAR(d.stat_date, 'YYYY-MM-DD') AS date,
       COALESCE(SUM(d.current_balance), 0) AS balance,
       COALESCE(SUM(d.today_cost), 0) AS cost
FROM supplier_provider_daily_stats d
JOIN supplier_providers p ON p.id = d.provider_id AND p.deleted_at IS NULL
GROUP BY d.stat_date
ORDER BY d.stat_date`)
	if err != nil {
		return nil, fmt.Errorf("query supplier provider balance summary days: %w", err)
	}
	defer rows.Close()

	days := make([]service.SupplierProviderBalanceSummaryDay, 0)
	for rows.Next() {
		var day service.SupplierProviderBalanceSummaryDay
		if scanErr := rows.Scan(&day.Date, &day.Balance, &day.Cost); scanErr != nil {
			return nil, fmt.Errorf("scan supplier provider balance summary day: %w", scanErr)
		}
		days = append(days, day)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier provider balance summary days: %w", err)
	}
	return days, nil
}

// ListBalanceCosts returns balance difference costs per day per provider for the date range.
// Cost is computed as (first balance snapshot of day) - (last balance snapshot of day).
// Only positive differences are returned (consumption detected).
func (r *supplierProviderRepository) ListBalanceCosts(ctx context.Context, start, end time.Time, providerID int64) ([]service.SupplierProviderBalanceCostDay, error) {
	tzName := timezone.Name()
	if strings.TrimSpace(tzName) == "" {
		tzName = "Asia/Shanghai"
	}

	query := `
WITH daily_snapshots AS (
  SELECT (captured_at AT TIME ZONE $3)::date AS stat_date,
         provider_id,
         array_agg(current_balance ORDER BY captured_at ASC) AS balances
  FROM supplier_provider_metric_snapshots
  WHERE current_balance > 0
    AND captured_at >= $1
    AND captured_at < $2`
	args := []any{start, end, tzName}
	if providerID > 0 {
		query += ` AND provider_id = $4`
		args = append(args, providerID)
	}
	query += `
  GROUP BY stat_date, provider_id
),
balance_costs AS (
  SELECT stat_date,
         provider_id,
         GREATEST(balances[1]::NUMERIC - COALESCE(balances[array_length(balances, 1)], balances[1]), 0) AS cost
  FROM daily_snapshots
  WHERE array_length(balances, 1) >= 1
)
SELECT TO_CHAR(stat_date, 'YYYY-MM-DD'), provider_id, cost::FLOAT
FROM balance_costs
WHERE cost > 0
ORDER BY stat_date ASC, provider_id ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query supplier balance costs: %w", err)
	}
	defer rows.Close()

	costs := make([]service.SupplierProviderBalanceCostDay, 0)
	for rows.Next() {
		var cd service.SupplierProviderBalanceCostDay
		var pid int64
		var cost float64
		if scanErr := rows.Scan(&cd.Date, &pid, &cost); scanErr != nil {
			return nil, fmt.Errorf("scan supplier balance cost: %w", scanErr)
		}
		cd.ProviderID = pid
		cd.BalanceCost = cost
		costs = append(costs, cd)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier balance costs: %w", err)
	}
	return costs, nil
}

func supplierProviderWhere(params service.SupplierProviderListParams) (string, []any) {
	conditions := []string{"p.deleted_at IS NULL"}
	args := make([]any, 0, 2)
	if search := strings.TrimSpace(params.Search); search != "" {
		args = append(args, "%"+search+"%")
		conditions = append(conditions, fmt.Sprintf("(p.name ILIKE $%d OR p.code ILIKE $%d OR p.base_url ILIKE $%d)", len(args), len(args), len(args)))
	}
	if params.Enabled != nil {
		args = append(args, *params.Enabled)
		conditions = append(conditions, fmt.Sprintf("p.enabled = $%d", len(args)))
	}
	return strings.Join(conditions, " AND "), args
}

func (r *supplierProviderRepository) GetByID(ctx context.Context, id int64) (*service.SupplierProvider, error) {
	provider, err := scanSupplierProvider(r.db.QueryRowContext(ctx, supplierProviderSelect+" WHERE p.id = $1 AND p.deleted_at IS NULL", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSupplierProviderNotFound
	}
	return provider, err
}

func (r *supplierProviderRepository) Create(ctx context.Context, provider *service.SupplierProvider) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin supplier provider create: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var hasActive bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM supplier_providers WHERE deleted_at IS NULL)").Scan(&hasActive); err != nil {
		return err
	}
	if provider.IsDefault || !hasActive {
		if _, err := tx.ExecContext(ctx, "UPDATE supplier_providers SET is_default = FALSE, updated_at = NOW() WHERE deleted_at IS NULL AND is_default = TRUE"); err != nil {
			return err
		}
		provider.IsDefault = true
	}
	err = tx.QueryRowContext(ctx, `
INSERT INTO supplier_providers (
  code, name, provider_type, newapi_auth_mode, base_url, login_url, api_keys_url, groups_url,
  available_groups_url, balance_url, usage_cost_url, recharge_url, monitor_url, account_name_prefix,
  temp_disable_minutes, account_rate_multiplier_scale, sort_order, enabled, turnstile_enabled, is_default
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
RETURNING id, created_at, updated_at`, provider.Code, provider.Name, provider.ProviderType,
		provider.NewAPIAuthMode, provider.BaseURL, provider.LoginURL, provider.APIKeysURL, provider.GroupsURL,
		provider.AvailableGroupsURL, provider.BalanceURL, provider.UsageCostURL, provider.RechargeURL,
		provider.MonitorURL, provider.AccountNamePrefix, provider.TempDisableMinutes, provider.AccountRateMultiplierScale,
		provider.SortOrder, provider.Enabled, provider.TurnstileEnabled, provider.IsDefault).Scan(&provider.ID, &provider.CreatedAt, &provider.UpdatedAt)
	if err != nil {
		return mapSupplierProviderError(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO supplier_provider_credentials (provider_id, email, username, password_encrypted) VALUES ($1,$2,$3,$4)`, provider.ID, provider.Email, provider.Username, provider.PasswordEncrypted); err != nil {
		return fmt.Errorf("insert supplier credentials: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO supplier_provider_runtime_stats (provider_id) VALUES ($1)`, provider.ID); err != nil {
		return fmt.Errorf("insert supplier runtime stats: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit supplier provider create: %w", err)
	}
	return nil
}

func (r *supplierProviderRepository) Update(ctx context.Context, provider *service.SupplierProvider) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin supplier provider update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if provider.IsDefault {
		if _, err := tx.ExecContext(ctx, "UPDATE supplier_providers SET is_default = FALSE, updated_at = NOW() WHERE id <> $1 AND deleted_at IS NULL AND is_default = TRUE", provider.ID); err != nil {
			return err
		}
	}
	result, err := tx.ExecContext(ctx, `
UPDATE supplier_providers SET
  code=$2, name=$3, provider_type=$4, newapi_auth_mode=$5, base_url=$6, login_url=$7,
  api_keys_url=$8, groups_url=$9, available_groups_url=$10, balance_url=$11,
  usage_cost_url=$12, recharge_url=$13, monitor_url=$14, account_name_prefix=$15, temp_disable_minutes=$15,
  account_rate_multiplier_scale=$16, sort_order=$17, enabled=$18,
  turnstile_enabled=$19, is_default=$20, updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`, provider.ID, provider.Code, provider.Name,
		provider.ProviderType, provider.NewAPIAuthMode, provider.BaseURL, provider.LoginURL, provider.APIKeysURL,
		provider.GroupsURL, provider.AvailableGroupsURL, provider.BalanceURL,
		provider.UsageCostURL, provider.RechargeURL, provider.MonitorURL, provider.AccountNamePrefix, provider.TempDisableMinutes,
		provider.AccountRateMultiplierScale, provider.SortOrder, provider.Enabled,
		provider.TurnstileEnabled, provider.IsDefault)
	if err != nil {
		return mapSupplierProviderError(err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrSupplierProviderNotFound
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO supplier_provider_credentials (provider_id, email, username, password_encrypted, updated_at)
VALUES ($1,$2,$3,$4,NOW())
ON CONFLICT (provider_id) DO UPDATE SET email=EXCLUDED.email, username=EXCLUDED.username,
password_encrypted=EXCLUDED.password_encrypted, updated_at=NOW()`, provider.ID, provider.Email, provider.Username, provider.PasswordEncrypted)
	if err != nil {
		return fmt.Errorf("update supplier credentials: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit supplier provider update: %w", err)
	}
	return nil
}

func (r *supplierProviderRepository) DisableAfterAuthFailure(ctx context.Context, providerID int64, message string, syncedAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin supplier provider auth failure update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx, `
UPDATE supplier_providers
SET enabled=FALSE, updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`, providerID)
	if err != nil {
		return fmt.Errorf("disable supplier provider after auth failure: %w", err)
	}
	if affected, affectedErr := result.RowsAffected(); affectedErr != nil {
		return fmt.Errorf("check supplier provider auth failure update: %w", affectedErr)
	} else if affected == 0 {
		return service.ErrSupplierProviderNotFound
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO supplier_provider_runtime_stats (
  provider_id, sync_status, sync_message, last_sync_at, updated_at
) VALUES ($1,$2,$3,$4,$4)
ON CONFLICT (provider_id) DO UPDATE SET
  sync_status=EXCLUDED.sync_status,
  sync_message=EXCLUDED.sync_message,
  last_sync_at=EXCLUDED.last_sync_at,
  updated_at=EXCLUDED.updated_at`, providerID, service.SupplierSyncStatusFailed, message, syncedAt); err != nil {
		return fmt.Errorf("record supplier provider auth failure: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit supplier provider auth failure update: %w", err)
	}
	return nil
}

func (r *supplierProviderRepository) Delete(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var wasDefault bool
	if err := tx.QueryRowContext(ctx, "SELECT is_default FROM supplier_providers WHERE id=$1 AND deleted_at IS NULL FOR UPDATE", id).Scan(&wasDefault); errors.Is(err, sql.ErrNoRows) {
		return service.ErrSupplierProviderNotFound
	} else if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM supplier_provider_credentials WHERE provider_id=$1", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE supplier_providers SET enabled=FALSE, is_default=FALSE, deleted_at=NOW(), updated_at=NOW() WHERE id=$1", id); err != nil {
		return err
	}
	if wasDefault {
		if _, err := tx.ExecContext(ctx, `UPDATE supplier_providers SET is_default=TRUE, updated_at=NOW() WHERE id=(SELECT id FROM supplier_providers WHERE deleted_at IS NULL ORDER BY enabled DESC, sort_order ASC, id ASC LIMIT 1)`); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *supplierProviderRepository) SetDefault(ctx context.Context, id int64) (*service.SupplierProvider, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var exists bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM supplier_providers WHERE id=$1 AND deleted_at IS NULL)", id).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, service.ErrSupplierProviderNotFound
	}
	if _, err := tx.ExecContext(ctx, "UPDATE supplier_providers SET is_default=(id=$1), updated_at=NOW() WHERE deleted_at IS NULL", id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

type supplierProviderScanner interface{ Scan(dest ...any) error }

func scanSupplierProvider(scanner supplierProviderScanner) (*service.SupplierProvider, error) {
	provider := &service.SupplierProvider{}
	var estimatedDays sql.NullFloat64
	var lastSyncAt sql.NullTime
	var authLastLoginAt sql.NullTime
	var authLastCacheHitAt sql.NullTime
	var authLastTokenExpiresAt sql.NullTime
	err := scanner.Scan(&provider.ID, &provider.Code, &provider.Name, &provider.ProviderType, &provider.NewAPIAuthMode,
		&provider.BaseURL, &provider.LoginURL, &provider.APIKeysURL, &provider.GroupsURL,
		&provider.AvailableGroupsURL, &provider.BalanceURL, &provider.UsageCostURL, &provider.RechargeURL, &provider.MonitorURL,
		&provider.AccountNamePrefix, &provider.TempDisableMinutes,
		&provider.AccountRateMultiplierScale, &provider.SortOrder, &provider.Enabled,
		&provider.TurnstileEnabled, &provider.IsDefault, &provider.CreatedAt, &provider.UpdatedAt, &provider.Email,
		&provider.Username, &provider.PasswordEncrypted, &provider.Status, &provider.RiskLevel,
		&provider.ValidAccountCount, &provider.SchedulableAccountCount, &provider.RequestCount,
		&provider.SuccessRate, &provider.PeriodCost, &provider.CurrentBalance, &provider.TodayCost,
		&estimatedDays, &provider.RateRiskCount, &provider.SyncStatus, &provider.SyncMessage, &lastSyncAt,
		&provider.AuthSummary.LoginCount, &provider.AuthSummary.LoginSuccessCount,
		&provider.AuthSummary.LoginFailureCount, &provider.AuthSummary.CacheHitCount,
		&provider.AuthSummary.CacheMissCount, &authLastLoginAt,
		&provider.AuthSummary.LastLoginStatus, &provider.AuthSummary.LastLoginError,
		&authLastCacheHitAt, &provider.AuthSummary.LastCacheError,
		&authLastTokenExpiresAt, &provider.AuthSummary.LastTokenFingerprint)
	if err != nil {
		return nil, err
	}
	if estimatedDays.Valid {
		provider.EstimatedDays = &estimatedDays.Float64
	}
	if lastSyncAt.Valid {
		provider.LastSyncAt = &lastSyncAt.Time
	}
	if authLastLoginAt.Valid {
		provider.AuthSummary.LastLoginAt = &authLastLoginAt.Time
	}
	if authLastCacheHitAt.Valid {
		provider.AuthSummary.LastCacheHitAt = &authLastCacheHitAt.Time
	}
	if authLastTokenExpiresAt.Valid {
		provider.AuthSummary.LastTokenExpiresAt = &authLastTokenExpiresAt.Time
	}
	provider.CredentialConfigured = provider.PasswordEncrypted != ""
	return provider, nil
}

func mapSupplierProviderError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return service.ErrSupplierProviderExists
	}
	return err
}

const supplierProviderTypeSelect = `
SELECT id, code, name, login_url, api_keys_url, groups_url, available_groups_url,
       balance_url, usage_cost_url, recharge_url, monitor_url, enabled, sort_order, created_at, updated_at
FROM supplier_provider_types`

func (r *supplierProviderTypeRepository) List(ctx context.Context, enabledOnly bool) ([]*service.SupplierProviderType, error) {
	where := "deleted_at IS NULL"
	if enabledOnly {
		where += " AND enabled = TRUE"
	}
	rows, err := r.db.QueryContext(ctx, supplierProviderTypeSelect+" WHERE "+where+" ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, fmt.Errorf("query supplier provider types: %w", err)
	}
	defer rows.Close()
	items := make([]*service.SupplierProviderType, 0)
	for rows.Next() {
		item, scanErr := scanSupplierProviderType(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier provider types: %w", err)
	}
	return items, nil
}

func (r *supplierProviderTypeRepository) GetByID(ctx context.Context, id int64) (*service.SupplierProviderType, error) {
	item, err := scanSupplierProviderType(r.db.QueryRowContext(ctx, supplierProviderTypeSelect+" WHERE id=$1 AND deleted_at IS NULL", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSupplierProviderTypeNotFound
	}
	return item, err
}

func (r *supplierProviderTypeRepository) GetByCode(ctx context.Context, code string) (*service.SupplierProviderType, error) {
	item, err := scanSupplierProviderType(r.db.QueryRowContext(ctx, supplierProviderTypeSelect+" WHERE code=$1 AND deleted_at IS NULL", code))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSupplierProviderTypeNotFound
	}
	return item, err
}

func (r *supplierProviderTypeRepository) Create(ctx context.Context, item *service.SupplierProviderType) error {
	err := r.db.QueryRowContext(ctx, `
INSERT INTO supplier_provider_types (
  code, name, login_url, api_keys_url, groups_url, available_groups_url,
  balance_url, usage_cost_url, recharge_url, monitor_url, enabled, sort_order
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
RETURNING id, created_at, updated_at`, item.Code, item.Name, item.LoginURL, item.APIKeysURL,
		item.GroupsURL, item.AvailableGroupsURL, item.BalanceURL, item.UsageCostURL,
		item.MonitorURL, item.Enabled, item.SortOrder).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return mapSupplierProviderTypeError(err)
	}
	return nil
}

func (r *supplierProviderTypeRepository) Update(ctx context.Context, item *service.SupplierProviderType) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE supplier_provider_types SET
  code=$2, name=$3, login_url=$4, api_keys_url=$5, groups_url=$6,
  available_groups_url=$7, balance_url=$8, usage_cost_url=$9,
  monitor_url=$10, enabled=$11, sort_order=$12, updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`, item.ID, item.Code, item.Name, item.LoginURL,
		item.APIKeysURL, item.GroupsURL, item.AvailableGroupsURL, item.BalanceURL,
		item.UsageCostURL, item.RechargeURL, item.MonitorURL, item.Enabled, item.SortOrder)
	if err != nil {
		return mapSupplierProviderTypeError(err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrSupplierProviderTypeNotFound
	}
	return nil
}

func (r *supplierProviderTypeRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "UPDATE supplier_provider_types SET enabled=FALSE, deleted_at=NOW(), updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL", id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrSupplierProviderTypeNotFound
	}
	return nil
}

type supplierProviderTypeScanner interface{ Scan(dest ...any) error }

func scanSupplierProviderType(scanner supplierProviderTypeScanner) (*service.SupplierProviderType, error) {
	item := &service.SupplierProviderType{}
	err := scanner.Scan(&item.ID, &item.Code, &item.Name, &item.LoginURL,
		&item.APIKeysURL, &item.GroupsURL, &item.AvailableGroupsURL,
		&item.BalanceURL, &item.UsageCostURL, &item.RechargeURL, &item.MonitorURL, &item.Enabled, &item.SortOrder,
		&item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func mapSupplierProviderTypeError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return service.ErrSupplierProviderTypeExists
	}
	return err
}

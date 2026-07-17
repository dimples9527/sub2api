package repository

import (
	"context"
	"database/sql"
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

func (r *supplierProviderDataRepository) ListAccounts(ctx context.Context, params service.SupplierProviderDataListParams) (service.SupplierProviderAccountListResult, error) {
	params = normalizeSupplierProviderDataListParams(params)
	where, args := supplierProviderDataWhere("a", params)

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM supplier_provider_accounts a JOIN supplier_providers p ON p.id = a.provider_id WHERE "+where, args...).Scan(&total); err != nil {
		return service.SupplierProviderAccountListResult{}, fmt.Errorf("count supplier provider accounts: %w", err)
	}

	queryArgs := append(append([]any{}, args...), params.PageSize, (params.Page-1)*params.PageSize)
	rows, err := r.db.QueryContext(ctx, `
SELECT a.id, a.provider_id, p.name AS provider_name, a.upstream_account_key, a.name, a.status,
       a.group_key, a.group_name, a.rate_multiplier, a.raw_status, a.active,
       a.last_seen_at, a.inactive_at
FROM supplier_provider_accounts a
JOIN supplier_providers p ON p.id = a.provider_id
WHERE `+where+fmt.Sprintf(" ORDER BY a.active DESC, a.last_seen_at DESC, a.id ASC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2), queryArgs...)
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
	where, args := supplierProviderDataWhere("g", params)

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM supplier_provider_groups g JOIN supplier_providers p ON p.id = g.provider_id WHERE "+where, args...).Scan(&total); err != nil {
		return service.SupplierProviderGroupListResult{}, fmt.Errorf("count supplier provider groups: %w", err)
	}

	queryArgs := append(append([]any{}, args...), params.PageSize, (params.Page-1)*params.PageSize)
	rows, err := r.db.QueryContext(ctx, `
SELECT g.id, g.provider_id, p.name AS provider_name, g.upstream_group_key, g.name,
       g.rate_multiplier, g.raw_status, g.active,
       COALESCE(COUNT(a.id) FILTER (WHERE a.active = TRUE), 0) AS account_count,
       g.last_seen_at, g.inactive_at
FROM supplier_provider_groups g
JOIN supplier_providers p ON p.id = g.provider_id
LEFT JOIN supplier_provider_accounts a ON a.provider_id = g.provider_id AND a.group_key = g.upstream_group_key
WHERE `+where+fmt.Sprintf(" GROUP BY g.id, p.name ORDER BY g.active DESC, g.last_seen_at DESC, g.id ASC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2), queryArgs...)
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
	return service.SupplierProviderGroupListResult{Items: items, Total: total, Page: params.Page, PageSize: params.PageSize}, nil
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
UPDATE supplier_provider_accounts SET active = FALSE, inactive_at = $3, updated_at = $3
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
	return r.updateMetric(ctx, providerID, &balance, nil, seenAt)
}

func (r *supplierProviderDataRepository) UpdateCost(ctx context.Context, providerID int64, cost float64, seenAt time.Time) error {
	return r.updateMetric(ctx, providerID, nil, &cost, seenAt)
}

func (r *supplierProviderDataRepository) updateMetric(ctx context.Context, providerID int64, balance *float64, cost *float64, seenAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin supplier metric update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

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
		if _, err := tx.ExecContext(ctx, "UPDATE supplier_provider_runtime_stats SET today_cost=$2, updated_at=$3 WHERE provider_id=$1", providerID, *cost, seenAt); err != nil {
			return fmt.Errorf("update supplier cost: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO supplier_provider_metric_snapshots (provider_id, current_balance, today_cost, captured_at) VALUES ($1,$2,$3,$4)", providerID, 0.0, *cost, seenAt); err != nil {
			return fmt.Errorf("insert supplier cost snapshot: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO supplier_provider_daily_stats (provider_id, stat_date, today_cost)
VALUES ($1,$2,$3)
ON CONFLICT (provider_id, stat_date) DO UPDATE SET today_cost=EXCLUDED.today_cost, updated_at=NOW()`, providerID, supplierStatDate(seenAt), *cost); err != nil {
			return fmt.Errorf("upsert supplier daily cost: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit supplier metric update: %w", err)
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

func supplierStatDate(seenAt time.Time) time.Time {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	local := seenAt.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

type supplierProviderAccountScanner interface{ Scan(dest ...any) error }

func scanSupplierProviderAccount(scanner supplierProviderAccountScanner) (service.SupplierProviderAccount, error) {
	var item service.SupplierProviderAccount
	var inactiveAt sql.NullTime
	err := scanner.Scan(&item.ID, &item.ProviderID, &item.ProviderName, &item.UpstreamKey,
		&item.Name, &item.Status, &item.GroupKey, &item.GroupName, &item.RateMultiplier,
		&item.RawStatus, &item.Active, &item.LastSeenAt, &inactiveAt)
	if err != nil {
		return service.SupplierProviderAccount{}, err
	}
	if inactiveAt.Valid {
		item.InactiveAt = &inactiveAt.Time
	}
	return item, nil
}

type supplierProviderGroupScanner interface{ Scan(dest ...any) error }

func scanSupplierProviderGroup(scanner supplierProviderGroupScanner) (service.SupplierProviderGroup, error) {
	var item service.SupplierProviderGroup
	var inactiveAt sql.NullTime
	err := scanner.Scan(&item.ID, &item.ProviderID, &item.ProviderName, &item.UpstreamKey,
		&item.Name, &item.RateMultiplier, &item.RawStatus, &item.Active,
		&item.AccountCount, &item.LastSeenAt, &inactiveAt)
	if err != nil {
		return service.SupplierProviderGroup{}, err
	}
	if inactiveAt.Valid {
		item.InactiveAt = &inactiveAt.Time
	}
	return item, nil
}

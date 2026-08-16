package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type supplierDashboardQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type supplierDashboardRepository struct {
	db supplierDashboardQueryer
}

func NewSupplierDashboardRepository(db *sql.DB) *supplierDashboardRepository {
	return newSupplierDashboardRepository(db)
}

func newSupplierDashboardRepository(db supplierDashboardQueryer) *supplierDashboardRepository {
	return &supplierDashboardRepository{db: db}
}

const supplierDashboardOpsCTE = `
WITH dashboard_target_providers AS MATERIALIZED (
  SELECT target_provider.id AS provider_id
  FROM supplier_providers target_provider
  WHERE target_provider.deleted_at IS NULL
    AND ($3 = '' OR target_provider.code = $3)
),
dashboard_target_supplier_accounts AS MATERIALIZED (
  SELECT spa.id AS supplier_account_id,
         spa.provider_id,
         spa.group_key,
         spa.active,
         spa.supplier_dashboard_normalized_effective_name AS normalized_name
  FROM supplier_provider_accounts spa
  JOIN dashboard_target_providers target_provider ON target_provider.provider_id = spa.provider_id
  WHERE ($4 = '' OR spa.group_key = $4)
),
dashboard_target_active_supplier_accounts AS MATERIALIZED (
  SELECT supplier_account_id, provider_id, group_key, normalized_name
  FROM dashboard_target_supplier_accounts
  WHERE active = TRUE
),
dashboard_target_normalized_names AS MATERIALIZED (
  SELECT DISTINCT normalized_name
  FROM dashboard_target_active_supplier_accounts
),
dashboard_target_provider_groups AS MATERIALIZED (
  SELECT DISTINCT provider_id, group_key
  FROM dashboard_target_active_supplier_accounts
),
dashboard_local_account_candidates AS MATERIALIZED (
  SELECT local_account.id AS local_account_id,
         regexp_replace(lower(local_account.name), '[^[:alnum:]]', '', 'g') AS normalized_name
  FROM accounts local_account
  JOIN dashboard_target_normalized_names target_name
    ON target_name.normalized_name = regexp_replace(lower(local_account.name), '[^[:alnum:]]', '', 'g')
  WHERE local_account.deleted_at IS NULL
),
dashboard_supplier_account_candidates AS MATERIALIZED (
  SELECT spa.id AS supplier_account_id,
         spa.provider_id,
         spa.supplier_dashboard_normalized_effective_name AS normalized_name
  FROM supplier_provider_accounts spa
  JOIN supplier_providers sp ON sp.id = spa.provider_id AND sp.deleted_at IS NULL
  JOIN dashboard_target_normalized_names target_name
    ON target_name.normalized_name = spa.supplier_dashboard_normalized_effective_name
  WHERE spa.active = TRUE
),
dashboard_forward_match_counts AS (
  SELECT target.supplier_account_id,
         target.provider_id,
         MIN(local.local_account_id) AS local_account_id,
         COUNT(local.local_account_id) AS forward_match_count
  FROM dashboard_target_active_supplier_accounts target
  LEFT JOIN dashboard_local_account_candidates local ON local.normalized_name = target.normalized_name
  GROUP BY target.supplier_account_id, target.provider_id
),
dashboard_reverse_match_counts AS (
  SELECT local.local_account_id,
         COUNT(supplier.supplier_account_id) AS reverse_match_count
  FROM dashboard_local_account_candidates local
  LEFT JOIN dashboard_supplier_account_candidates supplier ON supplier.normalized_name = local.normalized_name
  GROUP BY local.local_account_id
),
dashboard_unique_account_matches AS (
  SELECT forward.supplier_account_id,
         forward.provider_id,
         forward.forward_match_count,
         COALESCE(reverse_match.reverse_match_count, 0) AS reverse_match_count,
         CASE
           WHEN forward.forward_match_count = 1 AND reverse_match.reverse_match_count = 1
             THEN forward.local_account_id
           ELSE NULL
         END AS local_account_id
  FROM dashboard_forward_match_counts forward
  LEFT JOIN dashboard_reverse_match_counts reverse_match ON reverse_match.local_account_id = forward.local_account_id
),
dashboard_account_matches AS (
  SELECT * FROM dashboard_unique_account_matches
),
dashboard_unique_local_account_ids AS MATERIALIZED (
  SELECT DISTINCT local_account_id
  FROM dashboard_account_matches
  WHERE local_account_id IS NOT NULL
),
dashboard_latest_task_facts AS MATERIALIZED (
  SELECT DISTINCT ON (item.account_id)
         item.account_id AS local_account_id,
         item.status AS task_status,
         COALESCE(NULLIF(item.reason, ''), NULLIF(item.error_message, ''), '') AS task_reason,
         item.finished_at AS task_finished_at
  FROM upstream_account_health_guard_run_items item
  JOIN dashboard_unique_local_account_ids unique_accounts ON unique_accounts.local_account_id = item.account_id
  WHERE item.finished_at >= $1
    AND item.finished_at < $2
  ORDER BY item.account_id, item.finished_at DESC, item.id DESC
),
dashboard_usage AS (
  SELECT usage_logs.account_id,
         COUNT(*) AS success_count,
         COALESCE(SUM(usage_logs.actual_cost), 0) AS period_cost
  FROM usage_logs
  JOIN dashboard_unique_local_account_ids unique_accounts ON unique_accounts.local_account_id = usage_logs.account_id
  WHERE usage_logs.created_at >= $1
    AND usage_logs.created_at < $2
  GROUP BY usage_logs.account_id
),
dashboard_errors AS (
  SELECT oel.account_id,
         COUNT(*) AS error_count
  FROM ops_error_logs oel
  JOIN dashboard_unique_local_account_ids unique_accounts ON unique_accounts.local_account_id = oel.account_id
  WHERE oel.is_count_tokens = FALSE
    AND COALESCE(oel.status_code, 0) >= 400
    AND oel.created_at >= $1
    AND oel.created_at < $2
  GROUP BY oel.account_id
),
dashboard_account_ops AS (
  SELECT matches.supplier_account_id,
         matches.provider_id,
         matches.local_account_id,
         matches.forward_match_count,
         matches.reverse_match_count,
         COALESCE(usage.success_count, 0) AS success_count,
         COALESCE(errors.error_count, 0) AS error_count,
         COALESCE(usage.period_cost, 0) AS period_cost,
         task.task_status,
         task.task_reason,
         task.task_finished_at
  FROM dashboard_account_matches matches
  LEFT JOIN dashboard_usage usage ON usage.account_id = matches.local_account_id
  LEFT JOIN dashboard_errors errors ON errors.account_id = matches.local_account_id
  LEFT JOIN dashboard_latest_task_facts task ON task.local_account_id = matches.local_account_id
)`

const supplierDashboardRateHistoryCTE = `,
dashboard_rate_change_events AS MATERIALIZED (
  SELECT mapping.provider_id,
         COALESCE(NULLIF(history.upstream_group_key, ''), mapping.upstream_group_key) AS group_key,
         history.old_rate,
         history.new_rate,
         history.changed_at,
         COUNT(*) OVER (
           PARTITION BY mapping.provider_id, COALESCE(NULLIF(history.upstream_group_key, ''), mapping.upstream_group_key)
         ) AS change_count,
         ROW_NUMBER() OVER (
           PARTITION BY mapping.provider_id, COALESCE(NULLIF(history.upstream_group_key, ''), mapping.upstream_group_key)
           ORDER BY history.changed_at DESC, history.id DESC
         ) AS change_rank
  FROM dashboard_target_provider_groups target_group
  JOIN supplier_provider_groups mapping
    ON mapping.provider_id = target_group.provider_id
  JOIN supplier_rate_guard_change_logs history
    ON history.mapping_id = mapping.id
   AND history.changed_at >= $1
   AND history.changed_at < $2
  WHERE target_group.group_key = COALESCE(NULLIF(history.upstream_group_key, ''), mapping.upstream_group_key)
),
dashboard_rate_changes AS (
  SELECT provider_id, group_key, old_rate, new_rate, changed_at, change_count
  FROM dashboard_rate_change_events
  WHERE change_rank = 1
)`
const supplierDashboardCollectionCTE = `,
dashboard_ranked_sync_runs AS MATERIALIZED (
  SELECT run.provider_id, run.sync_scope, run.status, run.finished_at,
         ROW_NUMBER() OVER (
           PARTITION BY run.provider_id, run.sync_scope
           ORDER BY run.finished_at DESC, run.id DESC
         ) AS sync_rank
  FROM supplier_provider_sync_runs run
  JOIN dashboard_target_providers target_provider ON target_provider.provider_id = run.provider_id
  WHERE run.finished_at IS NOT NULL
    AND run.sync_scope IN ('accounts', 'groups', 'balance', 'all')
),
dashboard_latest_sync_runs AS (
  SELECT provider_id, sync_scope, status, finished_at
  FROM dashboard_ranked_sync_runs
  WHERE sync_rank = 1
),
dashboard_latest_account_runs AS (
  SELECT provider_id, status, finished_at
  FROM dashboard_latest_sync_runs
  WHERE sync_scope = 'accounts'
),
dashboard_latest_group_runs AS (
  SELECT provider_id, status, finished_at
  FROM dashboard_latest_sync_runs
  WHERE sync_scope = 'groups'
),
dashboard_latest_balance_runs AS (
  SELECT provider_id, status, finished_at
  FROM dashboard_latest_sync_runs
  WHERE sync_scope = 'balance'
),
dashboard_latest_all_runs AS (
  SELECT provider_id, status, finished_at
  FROM dashboard_latest_sync_runs
  WHERE sync_scope = 'all'
),dashboard_provider_collection_status AS (
  SELECT sp.id AS provider_id,
         CASE
           WHEN account_run.finished_at IS NOT NULL AND (all_run.finished_at IS NULL OR account_run.finished_at >= all_run.finished_at)
             THEN CASE WHEN account_run.status IN ('success', 'failed') THEN account_run.status ELSE 'unknown' END
           WHEN all_run.finished_at IS NOT NULL
             THEN CASE WHEN all_run.status IN ('success', 'failed') THEN all_run.status ELSE 'unknown' END
           ELSE 'never'
         END AS account_sync_status,
         CASE
           WHEN group_run.finished_at IS NOT NULL AND (all_run.finished_at IS NULL OR group_run.finished_at >= all_run.finished_at)
             THEN CASE WHEN group_run.status IN ('success', 'failed') THEN group_run.status ELSE 'unknown' END
           WHEN all_run.finished_at IS NOT NULL
             THEN CASE WHEN all_run.status IN ('success', 'failed') THEN all_run.status ELSE 'unknown' END
           ELSE 'never'
         END AS group_sync_status,
         CASE
           WHEN balance_run.finished_at IS NOT NULL AND (all_run.finished_at IS NULL OR balance_run.finished_at >= all_run.finished_at)
             THEN CASE WHEN balance_run.status IN ('success', 'failed') THEN balance_run.status ELSE 'unknown' END
           WHEN all_run.finished_at IS NOT NULL
             THEN CASE WHEN all_run.status IN ('success', 'failed') THEN all_run.status ELSE 'unknown' END
           ELSE 'never'
         END AS balance_sync_status,
         CASE
           WHEN account_run.finished_at IS NOT NULL AND (all_run.finished_at IS NULL OR account_run.finished_at >= all_run.finished_at)
             THEN account_run.finished_at
           ELSE all_run.finished_at
         END AS account_finished_at,
         CASE
           WHEN group_run.finished_at IS NOT NULL AND (all_run.finished_at IS NULL OR group_run.finished_at >= all_run.finished_at)
             THEN group_run.finished_at
           ELSE all_run.finished_at
         END AS group_finished_at,
         CASE
           WHEN balance_run.finished_at IS NOT NULL AND (all_run.finished_at IS NULL OR balance_run.finished_at >= all_run.finished_at)
             THEN balance_run.finished_at
           ELSE all_run.finished_at
         END AS balance_finished_at,
         all_run.finished_at AS all_finished_at
  FROM supplier_providers sp
  JOIN dashboard_target_providers target_provider ON target_provider.provider_id = sp.id
  LEFT JOIN dashboard_latest_account_runs account_run ON account_run.provider_id = sp.id
  LEFT JOIN dashboard_latest_group_runs group_run ON group_run.provider_id = sp.id
  LEFT JOIN dashboard_latest_balance_runs balance_run ON balance_run.provider_id = sp.id
  LEFT JOIN dashboard_latest_all_runs all_run ON all_run.provider_id = sp.id
  WHERE sp.deleted_at IS NULL
),
dashboard_provider_collection_evidence AS (
  SELECT status.*,
         status.account_sync_status = 'success' AS account_data_complete,
         status.group_sync_status = 'success' AS group_data_complete,
         status.balance_sync_status = 'success' AS balance_data_complete
  FROM dashboard_provider_collection_status status
)`

const supplierDashboardTrafficQuery = supplierDashboardOpsCTE + `
SELECT
  TO_CHAR(date_trunc('hour', usage_logs.created_at), 'YYYY-MM-DD"T"HH24:00:00') AS time,
  usage_logs.account_id,
  spa.name AS account_name,
  sp.code AS provider_slug,
  sp.name AS provider_name,
  spa.group_key,
  spa.group_name,
  COUNT(*) AS requests,
  COALESCE(SUM(usage_logs.input_tokens + usage_logs.output_tokens + usage_logs.cache_creation_tokens + usage_logs.cache_read_tokens), 0) AS tokens
FROM usage_logs
JOIN dashboard_unique_local_account_ids unique_accounts ON unique_accounts.local_account_id = usage_logs.account_id
JOIN dashboard_account_matches matches ON matches.local_account_id = usage_logs.account_id
JOIN supplier_provider_accounts spa ON spa.id = matches.supplier_account_id
JOIN supplier_providers sp ON sp.id = spa.provider_id AND sp.deleted_at IS NULL
WHERE usage_logs.created_at >= $1
  AND usage_logs.created_at < $2
GROUP BY time, usage_logs.account_id, spa.name, sp.code, sp.name, spa.group_key, spa.group_name
ORDER BY time ASC, usage_logs.account_id ASC`

const supplierDashboardProfitQuery = supplierDashboardOpsCTE + `
SELECT
  usage_logs.account_id,
  spa.name AS account_name,
  sp.code AS provider_slug,
  sp.name AS provider_name,
  spa.group_key,
  spa.group_name,
  COUNT(*) AS requests,
  COALESCE(SUM(usage_logs.input_tokens + usage_logs.output_tokens + usage_logs.cache_creation_tokens + usage_logs.cache_read_tokens), 0) AS tokens,
  COALESCE(SUM(COALESCE(usage_logs.account_stats_cost, usage_logs.total_cost) * COALESCE(usage_logs.account_rate_multiplier, 1)), 0) AS actual_cost,
  COALESCE(SUM(usage_logs.actual_cost), 0) AS user_cost
FROM usage_logs
JOIN dashboard_unique_local_account_ids unique_accounts ON unique_accounts.local_account_id = usage_logs.account_id
JOIN dashboard_account_matches matches ON matches.local_account_id = usage_logs.account_id
JOIN supplier_provider_accounts spa ON spa.id = matches.supplier_account_id
JOIN supplier_providers sp ON sp.id = spa.provider_id AND sp.deleted_at IS NULL
WHERE usage_logs.created_at >= $1
  AND usage_logs.created_at < $2
GROUP BY usage_logs.account_id, spa.name, sp.code, sp.name, spa.group_key, spa.group_name
ORDER BY COALESCE(SUM(usage_logs.actual_cost), 0) - COALESCE(SUM(COALESCE(usage_logs.account_stats_cost, usage_logs.total_cost) * COALESCE(usage_logs.account_rate_multiplier, 1)), 0) DESC, usage_logs.account_id ASC
LIMIT $5`

const supplierDashboardHealthQuery = supplierDashboardOpsCTE + `,
dashboard_health_items AS MATERIALIZED (
  SELECT DISTINCT ON (item.account_id, hour_bucket)
    item.account_id,
    date_trunc('hour', item.finished_at) AS hour_bucket,
    item.status,
    item.finished_at
  FROM upstream_account_health_guard_run_items item
  JOIN dashboard_unique_local_account_ids unique_accounts ON unique_accounts.local_account_id = item.account_id
  WHERE item.finished_at >= $1
    AND item.finished_at < $2
  ORDER BY item.account_id, hour_bucket, item.finished_at DESC, item.id DESC
)
SELECT
  item.account_id,
  accounts.name AS account_name,
  sp.code AS provider_slug,
  sp.name AS provider_name,
  spa.group_key,
  spa.group_name,
  TO_CHAR(item.hour_bucket, 'YYYY-MM-DD"T"HH24:00:00') AS time,
  item.status
FROM dashboard_health_items item
JOIN accounts ON accounts.id = item.account_id AND accounts.deleted_at IS NULL
JOIN dashboard_account_matches matches ON matches.local_account_id = item.account_id
JOIN supplier_provider_accounts spa ON spa.id = matches.supplier_account_id
JOIN supplier_providers sp ON sp.id = spa.provider_id AND sp.deleted_at IS NULL
ORDER BY item.account_id ASC, time ASC`

const supplierDashboardAccountsQuery = supplierDashboardOpsCTE + supplierDashboardRateHistoryCTE + supplierDashboardCollectionCTE + `
SELECT spa.id AS account_id,
       spa.name AS account_name,
       sp.code AS provider_slug,
       sp.name AS provider_name,
       sp.enabled AS provider_enabled,
       spa.active AS account_enabled,
       spa.group_key,
       spa.group_name,
       COALESCE(runtime.risk_level, 'normal') AS provider_risk_level,
       runtime.risk_updated_at AS provider_risk_updated_at,
       spa.status AS account_status,
       spa.rate_sync_status,
       evidence.balance_sync_status,
       evidence.balance_finished_at AS balance_synced_at,
       ops.task_status,
       ops.task_reason,
       ops.task_finished_at,
       spa.rate_multiplier * sp.account_rate_multiplier_scale AS current_rate,
       rate_change.old_rate AS previous_rate,
       COALESCE(rate_change.change_count, 0) + 1 AS snapshot_count,
       rate_change.old_rate AS rate_change_old,
       rate_change.new_rate AS rate_change_new,
       COALESCE(rate_change.change_count, 0) AS rate_change_count,
       rate_change.changed_at AS rate_changed_at,
       CASE WHEN evidence.balance_data_complete THEN runtime.current_balance ELSE NULL END AS balance,
       CASE WHEN evidence.balance_data_complete THEN runtime.estimated_days ELSE NULL END AS estimated_days,
       CASE WHEN ops.local_account_id IS NULL THEN NULL ELSE ops.success_count END AS success_count,
       CASE WHEN ops.local_account_id IS NULL THEN NULL ELSE ops.error_count END AS error_count,
       CASE WHEN ops.local_account_id IS NULL THEN NULL ELSE ops.period_cost END AS period_cost,
       spa.last_rate_sync_at,
       GREATEST(
         spa.updated_at,
         COALESCE(rate_change.changed_at, spa.updated_at),
         COALESCE(ops.task_finished_at, spa.updated_at)
       ) AS observed_at
FROM supplier_provider_accounts spa
JOIN dashboard_target_supplier_accounts target_account ON target_account.supplier_account_id = spa.id
JOIN supplier_providers sp ON sp.id = spa.provider_id AND sp.deleted_at IS NULL
LEFT JOIN supplier_provider_runtime_stats runtime ON runtime.provider_id = sp.id
JOIN dashboard_provider_collection_evidence evidence ON evidence.provider_id = sp.id
LEFT JOIN dashboard_rate_changes rate_change ON rate_change.provider_id = spa.provider_id AND rate_change.group_key = spa.group_key
LEFT JOIN dashboard_account_ops ops ON ops.supplier_account_id = spa.id
ORDER BY spa.id`

const supplierDashboardRatesQuery = supplierDashboardOpsCTE + supplierDashboardRateHistoryCTE + `
SELECT spa.id AS account_id,
       spa.name AS account_name,
       sp.code AS provider_slug,
       sp.name AS provider_name,
       sp.enabled AS provider_enabled,
       spa.active AS account_enabled,
       spa.group_key,
       spa.group_name,
       spa.rate_multiplier * sp.account_rate_multiplier_scale AS current_rate,
       rate_change.old_rate AS previous_rate,
       COALESCE(rate_change.change_count, 0) + 1 AS snapshot_count,
       rate_change.old_rate AS rate_change_old,
       rate_change.new_rate AS rate_change_new,
       COALESCE(rate_change.change_count, 0) AS rate_change_count,
       rate_change.changed_at AS rate_changed_at,
       CASE WHEN ops.local_account_id IS NULL THEN NULL ELSE ops.success_count END AS success_count,
       CASE WHEN ops.local_account_id IS NULL THEN NULL ELSE ops.error_count END AS error_count,
       CASE WHEN ops.local_account_id IS NULL THEN NULL ELSE ops.period_cost END AS period_cost,
       spa.last_rate_sync_at,
       GREATEST(spa.updated_at, COALESCE(rate_change.changed_at, spa.updated_at)) AS observed_at
FROM supplier_provider_accounts spa
JOIN dashboard_target_supplier_accounts target_account ON target_account.supplier_account_id = spa.id
JOIN supplier_providers sp ON sp.id = spa.provider_id AND sp.deleted_at IS NULL
LEFT JOIN dashboard_rate_changes rate_change ON rate_change.provider_id = spa.provider_id AND rate_change.group_key = spa.group_key
LEFT JOIN dashboard_account_ops ops ON ops.supplier_account_id = spa.id
ORDER BY spa.id`

const supplierDashboardProvidersQuery = supplierDashboardOpsCTE + supplierDashboardCollectionCTE + `,
dashboard_provider_ops AS (
  SELECT spa.provider_id,
         COUNT(*) FILTER (WHERE spa.active = TRUE) AS enabled_account_count,
         COUNT(*) FILTER (WHERE spa.active = TRUE AND ops.local_account_id IS NOT NULL) AS matched_account_count,
         COUNT(*) FILTER (
           WHERE spa.active = TRUE
             AND ops.local_account_id IS NOT NULL
             AND local_account.status = 'active'
             AND local_account.schedulable = TRUE
         ) AS schedulable_account_count,
         COALESCE(SUM(ops.success_count) FILTER (WHERE spa.active = TRUE AND ops.local_account_id IS NOT NULL), 0) AS success_count,
         COALESCE(SUM(ops.error_count) FILTER (WHERE spa.active = TRUE AND ops.local_account_id IS NOT NULL), 0) AS error_count,
         COALESCE(SUM(ops.period_cost) FILTER (WHERE spa.active = TRUE AND ops.local_account_id IS NOT NULL), 0) AS period_cost
  FROM supplier_provider_accounts spa
  LEFT JOIN dashboard_account_ops ops ON ops.supplier_account_id = spa.id
  LEFT JOIN accounts local_account ON local_account.id = ops.local_account_id AND local_account.deleted_at IS NULL
  GROUP BY spa.provider_id
)
SELECT sp.code AS provider_slug,
       sp.name AS provider_name,
       sp.enabled,
       evidence.account_data_complete
         AND evidence.group_data_complete
         AND evidence.balance_data_complete AS data_complete,
       COALESCE(runtime.risk_level, 'normal') AS provider_risk_level,
       evidence.account_sync_status AS sync_status,
       evidence.group_sync_status AS group_sync_status,
       evidence.balance_sync_status AS balance_sync_status,
       COALESCE(runtime.rate_risk_count, 0) AS rate_risk_count,
       COALESCE(provider_ops.enabled_account_count, 0) AS enabled_account_count,
       COALESCE(provider_ops.schedulable_account_count, 0) AS schedulable_account_count,
       CASE
         WHEN COALESCE(provider_ops.enabled_account_count, 0) = 0 THEN 0
         WHEN provider_ops.matched_account_count <> provider_ops.enabled_account_count THEN NULL
         ELSE provider_ops.success_count
       END AS success_count,
       CASE
         WHEN COALESCE(provider_ops.enabled_account_count, 0) = 0 THEN 0
         WHEN provider_ops.matched_account_count <> provider_ops.enabled_account_count THEN NULL
         ELSE provider_ops.error_count
       END AS error_count,
       CASE WHEN evidence.balance_data_complete THEN runtime.current_balance ELSE NULL END AS balance,
       CASE WHEN evidence.balance_data_complete THEN runtime.estimated_days ELSE NULL END AS estimated_days,
       CASE
         WHEN COALESCE(provider_ops.enabled_account_count, 0) = 0 THEN 0
         WHEN provider_ops.matched_account_count <> provider_ops.enabled_account_count THEN NULL
         ELSE provider_ops.period_cost
       END AS period_cost,
       GREATEST(
         evidence.account_finished_at,
         evidence.group_finished_at,
         evidence.balance_finished_at,
         evidence.all_finished_at
       ) AS last_synced_at
FROM supplier_providers sp
LEFT JOIN supplier_provider_runtime_stats runtime ON runtime.provider_id = sp.id
LEFT JOIN dashboard_provider_ops provider_ops ON provider_ops.provider_id = sp.id
JOIN dashboard_provider_collection_evidence evidence ON evidence.provider_id = sp.id
WHERE sp.deleted_at IS NULL
ORDER BY sp.sort_order, sp.id`

func (r *supplierDashboardRepository) ListDashboardAccounts(ctx context.Context, start, end time.Time, providerSlug, groupKey string) ([]service.SupplierDashboardAccountSnapshot, error) {
	rows, err := r.db.QueryContext(ctx, supplierDashboardAccountsQuery, start, end, providerSlug, groupKey)
	if err != nil {
		return nil, fmt.Errorf("query supplier dashboard accounts: %w", err)
	}
	defer rows.Close()

	items := make([]service.SupplierDashboardAccountSnapshot, 0)
	for rows.Next() {
		var item service.SupplierDashboardAccountSnapshot
		var taskStatus, taskReason sql.NullString
		var providerRiskUpdatedAt, balanceSyncedAt, taskFinishedAt, rateChangedAt, lastRateSyncedAt sql.NullTime
		var currentRate, previousRate, rateChangeOld, rateChangeNew, balance, estimatedDays, periodCost sql.NullFloat64
		var successCount, errorCount sql.NullInt64
		if err := rows.Scan(
			&item.AccountID, &item.AccountName, &item.ProviderSlug, &item.ProviderName,
			&item.ProviderEnabled, &item.AccountEnabled, &item.GroupKey, &item.GroupName,
			&item.ProviderRiskLevel, &providerRiskUpdatedAt, &item.AccountStatus, &item.RateSyncStatus, &item.BalanceSyncStatus, &balanceSyncedAt,
			&taskStatus, &taskReason, &taskFinishedAt,
			&currentRate, &previousRate, &item.SnapshotCount, &rateChangeOld, &rateChangeNew, &item.RateChangeCount, &rateChangedAt, &balance, &estimatedDays,
			&successCount, &errorCount, &periodCost, &lastRateSyncedAt, &item.ObservedAt,
		); err != nil {
			return nil, fmt.Errorf("scan supplier dashboard account: %w", err)
		}
		item.TaskStatus = dashboardNullString(taskStatus)
		item.TaskReason = dashboardNullString(taskReason)
		item.ProviderRiskUpdatedAt = dashboardNullTime(providerRiskUpdatedAt)
		item.BalanceSyncedAt = dashboardNullTime(balanceSyncedAt)
		item.TaskFinishedAt = dashboardNullTime(taskFinishedAt)
		item.CurrentRate = dashboardNullFloat(currentRate)
		item.PreviousRate = dashboardNullFloat(previousRate)
		item.RateChangeOld = dashboardNullFloat(rateChangeOld)
		item.RateChangeNew = dashboardNullFloat(rateChangeNew)
		item.RateChangedAt = dashboardNullTime(rateChangedAt)
		item.Balance = dashboardNullFloat(balance)
		item.EstimatedDays = dashboardNullFloat(estimatedDays)
		item.SuccessCount = dashboardNullInt(successCount)
		item.ErrorCount = dashboardNullInt(errorCount)
		item.PeriodCost = dashboardNullFloat(periodCost)
		item.LastRateSyncedAt = dashboardNullTime(lastRateSyncedAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier dashboard accounts: %w", err)
	}
	return items, nil
}

func (r *supplierDashboardRepository) ListDashboardRates(ctx context.Context, start, end time.Time, providerSlug, groupKey string) ([]service.SupplierDashboardRateSnapshot, error) {
	rows, err := r.db.QueryContext(ctx, supplierDashboardRatesQuery, start, end, providerSlug, groupKey)
	if err != nil {
		return nil, fmt.Errorf("query supplier dashboard rates: %w", err)
	}
	defer rows.Close()

	items := make([]service.SupplierDashboardRateSnapshot, 0)
	for rows.Next() {
		var item service.SupplierDashboardRateSnapshot
		var currentRate, previousRate, rateChangeOld, rateChangeNew, periodCost sql.NullFloat64
		var successCount, errorCount sql.NullInt64
		var rateChangedAt, lastRateSyncedAt sql.NullTime
		if err := rows.Scan(
			&item.AccountID, &item.AccountName, &item.ProviderSlug, &item.ProviderName,
			&item.ProviderEnabled, &item.AccountEnabled, &item.GroupKey, &item.GroupName,
			&currentRate, &previousRate, &item.SnapshotCount, &rateChangeOld, &rateChangeNew, &item.RateChangeCount, &rateChangedAt, &successCount, &errorCount, &periodCost,
			&lastRateSyncedAt, &item.ObservedAt,
		); err != nil {
			return nil, fmt.Errorf("scan supplier dashboard rate: %w", err)
		}
		item.CurrentRate = dashboardNullFloat(currentRate)
		item.PreviousRate = dashboardNullFloat(previousRate)
		item.RateChangeOld = dashboardNullFloat(rateChangeOld)
		item.RateChangeNew = dashboardNullFloat(rateChangeNew)
		item.RateChangedAt = dashboardNullTime(rateChangedAt)
		item.SuccessCount = dashboardNullInt(successCount)
		item.ErrorCount = dashboardNullInt(errorCount)
		item.PeriodCost = dashboardNullFloat(periodCost)
		item.LastRateSyncedAt = dashboardNullTime(lastRateSyncedAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier dashboard rates: %w", err)
	}
	return items, nil
}

func (r *supplierDashboardRepository) ListDashboardProviders(ctx context.Context, start, end time.Time) ([]service.SupplierDashboardProviderSnapshot, error) {
	rows, err := r.db.QueryContext(ctx, supplierDashboardProvidersQuery, start, end, "", "")
	if err != nil {
		return nil, fmt.Errorf("query supplier dashboard providers: %w", err)
	}
	defer rows.Close()

	items := make([]service.SupplierDashboardProviderSnapshot, 0)
	for rows.Next() {
		var item service.SupplierDashboardProviderSnapshot
		var successCount, errorCount sql.NullInt64
		var balance, estimatedDays, periodCost sql.NullFloat64
		var lastSyncedAt sql.NullTime
		if err := rows.Scan(
			&item.ProviderSlug, &item.ProviderName, &item.Enabled, &item.DataComplete,
			&item.ProviderRiskLevel, &item.SyncStatus, &item.GroupSyncStatus, &item.BalanceSyncStatus,
			&item.RateRiskCount, &item.EnabledAccountCount, &item.SchedulableAccountCount,
			&successCount, &errorCount, &balance, &estimatedDays, &periodCost, &lastSyncedAt,
		); err != nil {
			return nil, fmt.Errorf("scan supplier dashboard provider: %w", err)
		}
		item.SuccessCount = dashboardNullInt(successCount)
		item.ErrorCount = dashboardNullInt(errorCount)
		item.Balance = dashboardNullFloat(balance)
		item.EstimatedDays = dashboardNullFloat(estimatedDays)
		item.PeriodCost = dashboardNullFloat(periodCost)
		item.LastSyncedAt = dashboardNullTime(lastSyncedAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier dashboard providers: %w", err)
	}
	return items, nil
}

func (r *supplierDashboardRepository) ListDashboardAccountTraffic(ctx context.Context, start, end time.Time, providerSlug, groupKey string) ([]service.SupplierDashboardTrafficSnapshot, error) {
	rows, err := r.db.QueryContext(ctx, supplierDashboardTrafficQuery, start, end, providerSlug, groupKey)
	if err != nil {
		return nil, fmt.Errorf("query supplier dashboard account traffic: %w", err)
	}
	defer rows.Close()

	items := make([]service.SupplierDashboardTrafficSnapshot, 0)
	for rows.Next() {
		var item service.SupplierDashboardTrafficSnapshot
		if err := rows.Scan(
			&item.Time, &item.AccountID, &item.AccountName, &item.ProviderSlug, &item.ProviderName,
			&item.GroupKey, &item.GroupName, &item.Requests, &item.Tokens,
		); err != nil {
			return nil, fmt.Errorf("scan supplier dashboard account traffic: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier dashboard account traffic: %w", err)
	}
	return items, nil
}

func (r *supplierDashboardRepository) ListDashboardAccountProfit(ctx context.Context, start, end time.Time, providerSlug, groupKey string, limit int) ([]service.SupplierDashboardProfitSnapshot, error) {
	rows, err := r.db.QueryContext(ctx, supplierDashboardProfitQuery, start, end, providerSlug, groupKey, limit)
	if err != nil {
		return nil, fmt.Errorf("query supplier dashboard account profit: %w", err)
	}
	defer rows.Close()

	items := make([]service.SupplierDashboardProfitSnapshot, 0)
	for rows.Next() {
		var item service.SupplierDashboardProfitSnapshot
		if err := rows.Scan(
			&item.AccountID, &item.AccountName, &item.ProviderSlug, &item.ProviderName,
			&item.GroupKey, &item.GroupName, &item.Requests, &item.Tokens,
			&item.ActualCost, &item.UserCost,
		); err != nil {
			return nil, fmt.Errorf("scan supplier dashboard account profit: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier dashboard account profit: %w", err)
	}
	return items, nil
}

func (r *supplierDashboardRepository) ListDashboardAccountHealth(ctx context.Context, start, end time.Time, providerSlug, groupKey string) ([]service.SupplierDashboardHealthSnapshot, error) {
	rows, err := r.db.QueryContext(ctx, supplierDashboardHealthQuery, start, end, providerSlug, groupKey)
	if err != nil {
		return nil, fmt.Errorf("query supplier dashboard account health: %w", err)
	}
	defer rows.Close()

	items := make([]service.SupplierDashboardHealthSnapshot, 0)
	for rows.Next() {
		var item service.SupplierDashboardHealthSnapshot
		if err := rows.Scan(
			&item.AccountID, &item.AccountName, &item.ProviderSlug, &item.ProviderName,
			&item.GroupKey, &item.GroupName, &item.Time, &item.Status,
		); err != nil {
			return nil, fmt.Errorf("scan supplier dashboard account health: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier dashboard account health: %w", err)
	}
	return items, nil
}

func dashboardNullFloat(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	result := value.Float64
	return &result
}

func dashboardNullInt(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	result := value.Int64
	return &result
}

func dashboardNullString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func dashboardNullTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

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

type supplierAccountHealthHistoryRepository struct {
	db *sql.DB
}

func NewSupplierAccountHealthHistoryRepository(db *sql.DB) *supplierAccountHealthHistoryRepository {
	return &supplierAccountHealthHistoryRepository{db: db}
}

func (r *supplierAccountHealthHistoryRepository) Save(ctx context.Context, record service.SupplierAccountHealthHistoryRecord) error {
	if record.LocalAccountID <= 0 {
		return fmt.Errorf("本地账号 ID 必须为正数")
	}
	if record.CheckedAt.IsZero() {
		return fmt.Errorf("健康检测时间不能为空")
	}
	if record.StartedAt.IsZero() {
		record.StartedAt = record.CheckedAt
	}
	if record.FinishedAt.IsZero() {
		record.FinishedAt = record.CheckedAt
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO supplier_account_health_history (
    local_account_id, local_account_name, provider_id, provider_name, platform,
    checked_at, started_at, finished_at, status, latency_ms, latency_limit_ms, model_id,
    schedulable_before, schedulable_after, action, consecutive_failed, consecutive_slow,
    consecutive_healthy, reason, error_message
) VALUES ($1, $2, NULLIF($3, 0), $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)`,
		record.LocalAccountID, record.LocalAccountName, record.ProviderID, record.ProviderName, record.Platform,
		record.CheckedAt, record.StartedAt, record.FinishedAt, record.Status, record.LatencyMs, record.LatencyLimitMs,
		record.ModelID, record.SchedulableBefore, record.SchedulableAfter, record.Action, record.ConsecutiveFailed,
		record.ConsecutiveSlow, record.ConsecutiveHealthy, record.Reason, record.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("保存账号健康历史失败: %w", err)
	}
	return nil
}

func (r *supplierAccountHealthHistoryRepository) ListAccounts(ctx context.Context, params service.SupplierAccountHealthAccountListParams) (service.SupplierAccountHealthAccountListResult, error) {
	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	args, conditions := supplierAccountHealthAccountFilters(params)
	where := strings.Join(conditions, " AND ")

	guardArg := len(args) + 1
	args = append(args, service.SupplierAutomationTaskAccountHealthGuard)
	offset := (page - 1) * pageSize
	listArgs := append(append([]any(nil), args...), pageSize, offset)
	listQuery := fmt.Sprintf(`
WITH account_sources AS (
%s
)
SELECT src.local_account_id,
       local_account.name,
       src.provider_id,
       src.provider_name,
       src.platform,
       COALESCE(local_account.schedulable, FALSE),
       latest.status,
       latest.checked_at,
       latest.latency_ms,
       COALESCE(latest.latency_limit_ms, 0),
       COALESCE(latest.consecutive_failed, 0),
       COALESCE(local_account.rate_multiplier, 0),
       COUNT(*) OVER() AS total_count,
       EXISTS (
           SELECT 1 FROM supplier_automation_tasks task
           WHERE task.task_code = $%d AND task.enabled = TRUE
       ) AS guard_enabled
FROM account_sources src
JOIN accounts local_account ON local_account.id = src.local_account_id AND local_account.deleted_at IS NULL
LEFT JOIN LATERAL (
    SELECT h.status, h.checked_at, h.latency_ms, h.latency_limit_ms, h.consecutive_failed
    FROM supplier_account_health_history h
    WHERE h.local_account_id = src.local_account_id
    ORDER BY h.checked_at DESC, h.id DESC
    LIMIT 1
) latest ON TRUE
WHERE %s
ORDER BY local_account.name ASC, src.local_account_id ASC
LIMIT $%d OFFSET $%d`, supplierAccountHealthAccountSourcesSQL(), guardArg, where, len(listArgs)-1, len(listArgs))
	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return service.SupplierAccountHealthAccountListResult{}, fmt.Errorf("查询账号健康历史失败: %w", err)
	}
	defer rows.Close()

	items := make([]service.SupplierAccountHealthAccount, 0)
	var total int64
	for rows.Next() {
		item, rowTotal, err := scanSupplierAccountHealthAccount(rows)
		if err != nil {
			return service.SupplierAccountHealthAccountListResult{}, fmt.Errorf("扫描账号健康历史失败: %w", err)
		}
		if rowTotal > 0 {
			total = rowTotal
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return service.SupplierAccountHealthAccountListResult{}, fmt.Errorf("遍历账号健康历史失败: %w", err)
	}
	return service.SupplierAccountHealthAccountListResult{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func supplierAccountHealthAccountSourcesSQL() string {
	return `
SELECT local_account.id AS local_account_id,
       MIN(a.provider_id) AS provider_id,
       ARRAY_AGG(DISTINCT a.provider_id) AS provider_ids,
       STRING_AGG(DISTINCT p.name, ', ' ORDER BY p.name) AS provider_name,
       COALESCE(
           MAX(NULLIF(platform_override.platform, '')),
           MAX(NULLIF(local_account.platform, '')),
           ''
       ) AS platform
FROM accounts local_account
JOIN supplier_provider_accounts a ON a.active = TRUE
JOIN supplier_providers p ON p.id = a.provider_id AND p.enabled = TRUE
LEFT JOIN supplier_local_account_platform_overrides platform_override
  ON platform_override.local_account_id = local_account.id
WHERE local_account.deleted_at IS NULL
  AND ` + supplierProviderLocalAccountMatchCondition("local_account.name", "a.name") + `
GROUP BY local_account.id`
}

func supplierAccountHealthAccountFilters(params service.SupplierAccountHealthAccountListParams) ([]any, []string) {
	args := make([]any, 0, 5)
	conditions := []string{"TRUE"}
	if params.ProviderID > 0 {
		args = append(args, params.ProviderID)
		conditions = append(conditions, fmt.Sprintf("src.provider_ids @> ARRAY[$%d]::BIGINT[]", len(args)))
	}
	if platform := strings.TrimSpace(params.Platform); platform != "" {
		args = append(args, platform)
		conditions = append(conditions, fmt.Sprintf("src.platform = $%d", len(args)))
	}
	if search := strings.TrimSpace(params.Search); search != "" {
		args = append(args, "%"+search+"%")
		conditions = append(conditions, fmt.Sprintf("(local_account.name ILIKE $%d OR local_account.id::text ILIKE $%d)", len(args), len(args)))
	}
	if status := strings.ToLower(strings.TrimSpace(params.HealthStatus)); status != "" {
		if status == service.SupplierAccountHealthStatusUnchecked {
			conditions = append(conditions, "latest.status IS NULL")
		} else {
			args = append(args, status)
			conditions = append(conditions, fmt.Sprintf("latest.status = $%d", len(args)))
		}
	}
	return args, conditions
}

// GetSummary 按最近一次检测状态聚合账号数量。状态筛选是概览卡自身的切换维度，
// 因此这里忽略 HealthStatus，保证点选某个状态后其它卡片数字不会归零。
func (r *supplierAccountHealthHistoryRepository) GetSummary(ctx context.Context, params service.SupplierAccountHealthAccountListParams) (service.SupplierAccountHealthSummary, error) {
	params.HealthStatus = ""
	args, conditions := supplierAccountHealthAccountFilters(params)
	query := fmt.Sprintf(`
WITH account_sources AS (
%s
)
SELECT COUNT(*) AS total,
       COUNT(*) FILTER (WHERE latest.status = 'healthy') AS healthy,
       COUNT(*) FILTER (WHERE latest.status = 'slow') AS slow,
       COUNT(*) FILTER (WHERE latest.status = 'failed') AS failed,
       COUNT(*) FILTER (WHERE latest.status IS NULL) AS unchecked
FROM account_sources src
JOIN accounts local_account ON local_account.id = src.local_account_id AND local_account.deleted_at IS NULL
LEFT JOIN LATERAL (
    SELECT h.status
    FROM supplier_account_health_history h
    WHERE h.local_account_id = src.local_account_id
    ORDER BY h.checked_at DESC, h.id DESC
    LIMIT 1
) latest ON TRUE
WHERE %s`, supplierAccountHealthAccountSourcesSQL(), strings.Join(conditions, " AND "))

	var summary service.SupplierAccountHealthSummary
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&summary.Total, &summary.Healthy, &summary.Slow, &summary.Failed, &summary.Unchecked,
	); err != nil {
		return service.SupplierAccountHealthSummary{}, fmt.Errorf("查询账号健康概览失败: %w", err)
	}
	return summary, nil
}

func scanSupplierAccountHealthAccount(scanner interface{ Scan(dest ...any) error }) (service.SupplierAccountHealthAccount, int64, error) {
	var item service.SupplierAccountHealthAccount
	var status sql.NullString
	var rateMultiplier float64
	var providerName, platform string
	var checkedAt sql.NullTime
	var latency sql.NullInt64
	var total sql.NullInt64
	if err := scanner.Scan(
		&item.LocalAccountID, &item.LocalAccountName, &item.ProviderID, &providerName, &platform, &item.Schedulable,
		&status, &checkedAt, &latency, &item.LatencyLimitMs, &item.ConsecutiveFailures, &rateMultiplier,
		&total, &item.GuardEnabled,
	); err != nil {
		return service.SupplierAccountHealthAccount{}, 0, err
	}
	item.ProviderName = providerName
	item.Platform = platform
	item.RateMultiplier = rateMultiplier
	if status.Valid {
		item.Status = status.String
	}
	if checkedAt.Valid {
		value := checkedAt.Time
		item.CheckedAt = &value
	}
	if latency.Valid {
		value := latency.Int64
		item.LatencyMs = &value
	}
	return item, total.Int64, nil
}

func (r *supplierAccountHealthHistoryRepository) ValidateAccount(ctx context.Context, accountID int64) error {
	if accountID <= 0 {
		return service.ErrAccountNotFound
	}
	var exists bool
	err := r.db.QueryRowContext(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM accounts local_account
    JOIN supplier_provider_accounts a ON a.active = TRUE
    JOIN supplier_providers p ON p.id = a.provider_id AND p.enabled = TRUE
    WHERE local_account.id = $1
      AND local_account.deleted_at IS NULL
      AND `+supplierProviderLocalAccountMatchCondition("local_account.name", "a.name")+`
)`, accountID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("校验账号健康范围失败: %w", err)
	}
	if !exists {
		return service.ErrAccountNotFound
	}
	return nil
}

func (r *supplierAccountHealthHistoryRepository) GetTrend(ctx context.Context, accountID int64, since time.Time) (service.SupplierAccountHealthTrendResult, error) {
	if accountID <= 0 {
		return service.SupplierAccountHealthTrendResult{}, fmt.Errorf("本地账号 ID 必须为正数")
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT checked_at, status, latency_ms, latency_limit_ms, reason, action, error_message
FROM supplier_account_health_history
WHERE local_account_id = $1
  AND checked_at >= $2
ORDER BY checked_at ASC, id ASC`, accountID, since)
	if err != nil {
		return service.SupplierAccountHealthTrendResult{}, fmt.Errorf("查询账号健康趋势失败: %w", err)
	}
	defer rows.Close()
	result := service.SupplierAccountHealthTrendResult{AccountID: accountID, Points: make([]service.SupplierAccountHealthPoint, 0)}
	for rows.Next() {
		var point service.SupplierAccountHealthPoint
		var latency sql.NullInt64
		if err := rows.Scan(&point.CheckedAt, &point.Status, &latency, &point.LatencyLimitMs, &point.Reason, &point.Action, &point.ErrorMessage); err != nil {
			return service.SupplierAccountHealthTrendResult{}, fmt.Errorf("扫描账号健康趋势失败: %w", err)
		}
		if latency.Valid {
			value := latency.Int64
			point.LatencyMs = &value
		}
		result.Points = append(result.Points, point)
	}
	if err := rows.Err(); err != nil {
		return service.SupplierAccountHealthTrendResult{}, fmt.Errorf("遍历账号健康趋势失败: %w", err)
	}
	return result, nil
}

// GetTrends 批量查询多个账号的趋势。先一次校验账号仍在供应商可见范围内，
// 再用窗口函数限制每个账号返回的最新记录数，避免响应过大。
func (r *supplierAccountHealthHistoryRepository) GetTrends(ctx context.Context, accountIDs []int64, since time.Time, pointLimit int) (map[int64]service.SupplierAccountHealthTrendResult, error) {
	result := make(map[int64]service.SupplierAccountHealthTrendResult, len(accountIDs))
	if len(accountIDs) == 0 {
		return result, nil
	}
	if pointLimit <= 0 {
		pointLimit = 1
	}
	validRows, err := r.db.QueryContext(ctx, `
SELECT DISTINCT local_account.id
FROM accounts local_account
JOIN supplier_provider_accounts a ON a.active = TRUE
JOIN supplier_providers p ON p.id = a.provider_id AND p.enabled = TRUE
WHERE local_account.id = ANY($1)
  AND local_account.deleted_at IS NULL
  AND `+supplierProviderLocalAccountMatchCondition("local_account.name", "a.name"), pq.Array(accountIDs))
	if err != nil {
		return nil, fmt.Errorf("校验账号健康趋势范围失败: %w", err)
	}
	var validIDs []int64
	for validRows.Next() {
		var accountID int64
		if err := validRows.Scan(&accountID); err != nil {
			validRows.Close()
			return nil, fmt.Errorf("扫描账号健康趋势范围失败: %w", err)
		}
		validIDs = append(validIDs, accountID)
	}
	if err := validRows.Err(); err != nil {
		validRows.Close()
		return nil, fmt.Errorf("遍历账号健康趋势范围失败: %w", err)
	}
	validRows.Close()
	if len(validIDs) == 0 {
		return result, nil
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT local_account_id, checked_at, status, latency_ms, latency_limit_ms
FROM (
    SELECT h.id,
           h.local_account_id,
           h.checked_at,
           h.status,
           h.latency_ms,
           h.latency_limit_ms,
           ROW_NUMBER() OVER (
               PARTITION BY h.local_account_id
               ORDER BY h.checked_at DESC, h.id DESC
           ) AS row_number
    FROM supplier_account_health_history h
    WHERE h.local_account_id = ANY($1)
      AND h.checked_at >= $2
) trend
WHERE trend.row_number <= $3
ORDER BY trend.local_account_id ASC, trend.checked_at ASC, trend.id ASC`, pq.Array(validIDs), since, pointLimit)
	if err != nil {
		return nil, fmt.Errorf("批量查询账号健康趋势失败: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var accountID int64
		var point service.SupplierAccountHealthPoint
		var latency sql.NullInt64
		if err := rows.Scan(&accountID, &point.CheckedAt, &point.Status, &latency, &point.LatencyLimitMs); err != nil {
			return nil, fmt.Errorf("扫描账号健康趋势失败: %w", err)
		}
		if latency.Valid {
			value := latency.Int64
			point.LatencyMs = &value
		}
		trend := result[accountID]
		trend.AccountID = accountID
		trend.Points = append(trend.Points, point)
		result[accountID] = trend
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历账号健康趋势失败: %w", err)
	}
	return result, nil
}

func (r *supplierAccountHealthHistoryRepository) DeleteBefore(ctx context.Context, before time.Time, batchSize int) (int, error) {
	if batchSize <= 0 {
		batchSize = 1000
	}
	deleted := 0
	for {
		if err := ctx.Err(); err != nil {
			return deleted, err
		}
		result, err := r.db.ExecContext(ctx, `
WITH target AS (
    SELECT id FROM supplier_account_health_history
    WHERE checked_at < $1
    ORDER BY id ASC
    LIMIT $2
)
DELETE FROM supplier_account_health_history
WHERE id IN (SELECT id FROM target)`, before, batchSize)
		if err != nil {
			return deleted, fmt.Errorf("清理账号健康历史失败: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return deleted, fmt.Errorf("读取账号健康历史清理数量失败: %w", err)
		}
		deleted += int(affected)
		if affected < int64(batchSize) {
			return deleted, nil
		}
	}
}

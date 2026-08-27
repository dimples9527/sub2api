package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
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
	countArgs := append([]any(nil), args...)
	where := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf(`
WITH account_sources AS (
%s
)
SELECT COUNT(*)
FROM account_sources src
JOIN accounts local_account ON local_account.id = src.local_account_id AND local_account.deleted_at IS NULL
LEFT JOIN LATERAL (
    SELECT h.status, h.checked_at, h.latency_ms, h.latency_limit_ms, h.consecutive_failed
    FROM supplier_account_health_history h
    WHERE h.local_account_id = src.local_account_id
    ORDER BY h.checked_at DESC, h.id DESC
    LIMIT 1
) latest ON TRUE
WHERE %s`, supplierAccountHealthAccountSourcesSQL(), where)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return service.SupplierAccountHealthAccountListResult{}, fmt.Errorf("统计账号健康历史失败: %w", err)
	}

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
	for rows.Next() {
		item, err := scanSupplierAccountHealthAccount(rows)
		if err != nil {
			return service.SupplierAccountHealthAccountListResult{}, fmt.Errorf("扫描账号健康历史失败: %w", err)
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
		args = append(args, status)
		conditions = append(conditions, fmt.Sprintf("latest.status = $%d", len(args)))
	}
	return args, conditions
}

func scanSupplierAccountHealthAccount(scanner interface{ Scan(dest ...any) error }) (service.SupplierAccountHealthAccount, error) {
	var item service.SupplierAccountHealthAccount
	var status sql.NullString
	var providerName, platform string
	var checkedAt sql.NullTime
	var latency sql.NullInt64
	if err := scanner.Scan(
		&item.LocalAccountID, &item.LocalAccountName, &item.ProviderID, &providerName, &platform, &item.Schedulable,
		&status, &checkedAt, &latency, &item.LatencyLimitMs, &item.ConsecutiveFailures, &item.GuardEnabled,
	); err != nil {
		return service.SupplierAccountHealthAccount{}, err
	}
	item.ProviderName = providerName
	item.Platform = platform
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
	return item, nil
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

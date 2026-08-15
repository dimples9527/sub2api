package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
)

type supplierBalanceAlertRepository struct {
	db *sql.DB
}

func NewSupplierBalanceAlertRepository(db *sql.DB) service.SupplierBalanceAlertRepository {
	return &supplierBalanceAlertRepository{db: db}
}

const supplierBalanceAlertConfigSelect = `
SELECT COALESCE(c.id, 0), p.id, p.code, p.name, p.provider_type, p.enabled,
       COALESCE(c.enabled, FALSE), COALESCE(c.threshold, 0::numeric),
       COALESCE(c.cooldown_seconds, 3600), c.last_scan_at, c.last_balance,
       COALESCE(c.last_scan_status, 'never'), COALESCE(c.last_scan_error, ''),
       COALESCE(c.created_at, p.created_at), COALESCE(c.updated_at, p.updated_at)
FROM supplier_providers p
LEFT JOIN supplier_balance_alert_configs c ON c.provider_id = p.id
WHERE p.deleted_at IS NULL`

func (r *supplierBalanceAlertRepository) ListConfigs(ctx context.Context, providerID int64) ([]service.SupplierBalanceAlertConfig, error) {
	query := supplierBalanceAlertConfigSelect
	args := make([]any, 0, 1)
	if providerID > 0 {
		query += " AND p.id = $1"
		args = append(args, providerID)
	}
	query += " ORDER BY p.sort_order ASC, p.id ASC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询供应商余额预警配置失败: %w", err)
	}
	defer rows.Close()
	items := make([]service.SupplierBalanceAlertConfig, 0)
	for rows.Next() {
		item, scanErr := scanSupplierBalanceAlertConfig(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历供应商余额预警配置失败: %w", err)
	}
	return items, nil
}

func (r *supplierBalanceAlertRepository) GetConfig(ctx context.Context, providerID int64) (*service.SupplierBalanceAlertConfig, error) {
	item, err := scanSupplierBalanceAlertConfig(r.db.QueryRowContext(ctx, supplierBalanceAlertConfigSelect+" AND p.id = $1", providerID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSupplierBalanceAlertConfigNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询供应商余额预警配置失败: %w", err)
	}
	return &item, nil
}

func (r *supplierBalanceAlertRepository) UpsertConfig(ctx context.Context, providerID int64, enabled bool, threshold decimal.Decimal, cooldownSeconds int) (*service.SupplierBalanceAlertConfig, error) {
	if cooldownSeconds <= 0 {
		cooldownSeconds = int(service.SupplierBalanceAlertDefaultCooldown / time.Second)
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO supplier_balance_alert_configs (provider_id, enabled, threshold, cooldown_seconds)
VALUES ($1, $2, $3, $4)
ON CONFLICT (provider_id) DO UPDATE SET
  enabled = EXCLUDED.enabled,
  threshold = EXCLUDED.threshold,
  cooldown_seconds = EXCLUDED.cooldown_seconds,
  updated_at = NOW()`, providerID, enabled, threshold.String(), cooldownSeconds)
	if err != nil {
		return nil, fmt.Errorf("保存供应商余额预警配置失败: %w", err)
	}
	return r.GetConfig(ctx, providerID)
}

func (r *supplierBalanceAlertRepository) UpdateScanState(ctx context.Context, providerID int64, now time.Time, balance *decimal.Decimal, status, message string) error {
	var balanceValue any
	if balance != nil {
		balanceValue = balance.String()
	}
	result, err := r.db.ExecContext(ctx, `
UPDATE supplier_balance_alert_configs
SET last_scan_at = $2,
    last_balance = $3,
    last_scan_status = $4,
    last_scan_error = $5,
    updated_at = NOW()
WHERE provider_id = $1`, providerID, now, balanceValue, status, truncateSupplierNotificationText(message, 2000))
	if err != nil {
		return fmt.Errorf("更新供应商余额预警扫描状态失败: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrSupplierBalanceAlertConfigNotFound
	}
	return nil
}

const supplierBalanceAlertEventSelect = `
SELECT e.id, e.provider_id, p.code, p.name, p.provider_type, e.event_type, e.status,
       e.balance, e.threshold, e.observed_at, e.resolved_at, e.last_seen_at,
       e.created_at, e.updated_at
FROM supplier_balance_alert_events e
JOIN supplier_providers p ON p.id = e.provider_id`

func (r *supplierBalanceAlertRepository) GetActiveLowEvent(ctx context.Context, providerID int64) (*service.SupplierBalanceAlertEvent, error) {
	item, err := scanSupplierBalanceAlertEvent(r.db.QueryRowContext(ctx, supplierBalanceAlertEventSelect+`
WHERE e.provider_id = $1 AND e.event_type = 'balance_low' AND e.status = 'active'`, providerID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询供应商活动余额预警事件失败: %w", err)
	}
	return &item, nil
}

func (r *supplierBalanceAlertRepository) CreateEvent(ctx context.Context, event *service.SupplierBalanceAlertEvent) error {
	if event == nil {
		return service.ErrSupplierBalanceAlertInvalid
	}
	if event.ObservedAt.IsZero() {
		event.ObservedAt = time.Now()
	}
	if event.LastSeenAt.IsZero() {
		event.LastSeenAt = event.ObservedAt
	}
	if event.Status == "" {
		event.Status = service.SupplierBalanceAlertEventActive
	}
	err := r.db.QueryRowContext(ctx, `
INSERT INTO supplier_balance_alert_events (
  provider_id, event_type, status, balance, threshold, observed_at, last_seen_at
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, created_at, updated_at`, event.ProviderID, event.EventType, event.Status,
		event.Balance.String(), event.Threshold.String(), event.ObservedAt, event.LastSeenAt).
		Scan(&event.ID, &event.CreatedAt, &event.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建供应商余额预警事件失败: %w", err)
	}
	return nil
}

func (r *supplierBalanceAlertRepository) TouchActiveLowEvent(ctx context.Context, eventID int64, balance decimal.Decimal, now time.Time) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE supplier_balance_alert_events
SET balance = $2, last_seen_at = $3, updated_at = NOW()
WHERE id = $1 AND event_type = 'balance_low' AND status = 'active'`, eventID, balance.String(), now)
	if err != nil {
		return fmt.Errorf("更新供应商余额预警事件失败: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrSupplierBalanceAlertEventNotFound
	}
	return nil
}

func (r *supplierBalanceAlertRepository) ResolveActiveLowEvent(ctx context.Context, eventID int64, now time.Time, balance decimal.Decimal) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE supplier_balance_alert_events
SET status = 'resolved', balance = $2, resolved_at = $3, last_seen_at = $3, updated_at = NOW()
WHERE id = $1 AND event_type = 'balance_low' AND status = 'active'`, eventID, balance.String(), now)
	if err != nil {
		return fmt.Errorf("恢复供应商余额预警事件失败: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrSupplierBalanceAlertEventNotFound
	}
	return nil
}

func (r *supplierBalanceAlertRepository) ListEvents(ctx context.Context, params service.SupplierBalanceAlertEventListParams) (service.SupplierBalanceAlertEventListResult, error) {
	page, pageSize := normalizeSupplierNotificationPage(params.Page, params.PageSize)
	where := []string{"1 = 1"}
	args := make([]any, 0, 3)
	if params.ProviderID > 0 {
		args = append(args, params.ProviderID)
		where = append(where, fmt.Sprintf("e.provider_id = $%d", len(args)))
	}
	if strings.TrimSpace(params.EventType) != "" {
		args = append(args, params.EventType)
		where = append(where, fmt.Sprintf("e.event_type = $%d", len(args)))
	}
	if strings.TrimSpace(params.Status) != "" {
		args = append(args, params.Status)
		where = append(where, fmt.Sprintf("e.status = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM supplier_balance_alert_events e WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return service.SupplierBalanceAlertEventListResult{}, fmt.Errorf("统计供应商余额预警事件失败: %w", err)
	}
	queryArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	query := supplierBalanceAlertEventSelect + " WHERE " + whereSQL + fmt.Sprintf(" ORDER BY e.observed_at DESC, e.id DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return service.SupplierBalanceAlertEventListResult{}, fmt.Errorf("查询供应商余额预警事件失败: %w", err)
	}
	defer rows.Close()
	items := make([]service.SupplierBalanceAlertEvent, 0)
	for rows.Next() {
		item, scanErr := scanSupplierBalanceAlertEvent(rows)
		if scanErr != nil {
			return service.SupplierBalanceAlertEventListResult{}, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return service.SupplierBalanceAlertEventListResult{}, fmt.Errorf("遍历供应商余额预警事件失败: %w", err)
	}
	return service.SupplierBalanceAlertEventListResult{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (r *supplierBalanceAlertRepository) DeleteEvent(ctx context.Context, eventID int64) error {
	if eventID <= 0 {
		return service.ErrSupplierBalanceAlertInvalid
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启删除供应商余额预警事件事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	err = tx.QueryRowContext(ctx, `SELECT status FROM supplier_balance_alert_events WHERE id = $1 FOR UPDATE`, eventID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrSupplierBalanceAlertEventNotFound
	}
	if err != nil {
		return fmt.Errorf("查询供应商余额预警事件状态失败: %w", err)
	}
	if status == service.SupplierBalanceAlertEventActive {
		return service.ErrSupplierBalanceAlertEventActive
	}
	if status != service.SupplierBalanceAlertEventResolved {
		return service.ErrSupplierBalanceAlertEventNotFound
	}

	result, err := tx.ExecContext(ctx, `DELETE FROM supplier_balance_alert_events WHERE id = $1 AND status = 'resolved'`, eventID)
	if err != nil {
		return fmt.Errorf("删除供应商余额预警事件失败: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("确认删除供应商余额预警事件结果失败: %w", err)
	}
	if affected == 0 {
		return service.ErrSupplierBalanceAlertEventNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交删除供应商余额预警事件事务失败: %w", err)
	}
	return nil
}

type supplierBalanceAlertScanner interface {
	Scan(dest ...any) error
}

func scanSupplierBalanceAlertConfig(scanner supplierBalanceAlertScanner) (service.SupplierBalanceAlertConfig, error) {
	var item service.SupplierBalanceAlertConfig
	var thresholdRaw, balanceRaw sql.NullString
	var lastScanAt sql.NullTime
	var providerEnabled bool
	if err := scanner.Scan(
		&item.ID, &item.ProviderID, &item.ProviderCode, &item.ProviderName, &item.ProviderType,
		&providerEnabled, &item.Enabled, &thresholdRaw, &item.CooldownSeconds, &lastScanAt,
		&balanceRaw, &item.LastScanStatus, &item.LastScanError, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return item, err
	}
	item.ProviderEnabled = providerEnabled
	item.Threshold = parseSupplierNotificationDecimal(thresholdRaw.String)
	if item.CooldownSeconds <= 0 {
		item.CooldownSeconds = int(service.SupplierBalanceAlertDefaultCooldown / time.Second)
	}
	item.Cooldown = time.Duration(item.CooldownSeconds) * time.Second
	if lastScanAt.Valid {
		item.LastScanAt = &lastScanAt.Time
	}
	if balanceRaw.Valid {
		value := parseSupplierNotificationDecimal(balanceRaw.String)
		item.LastBalance = &value
	}
	return item, nil
}

func scanSupplierBalanceAlertEvent(scanner supplierBalanceAlertScanner) (service.SupplierBalanceAlertEvent, error) {
	var item service.SupplierBalanceAlertEvent
	var balanceRaw, thresholdRaw sql.NullString
	if err := scanner.Scan(
		&item.ID, &item.ProviderID, &item.ProviderCode, &item.ProviderName, &item.ProviderType, &item.EventType,
		&item.Status, &balanceRaw, &thresholdRaw, &item.ObservedAt, &item.ResolvedAt,
		&item.LastSeenAt, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return item, err
	}
	item.Balance = parseSupplierNotificationDecimal(balanceRaw.String)
	item.Threshold = parseSupplierNotificationDecimal(thresholdRaw.String)
	return item, nil
}

func parseSupplierNotificationDecimal(raw string) decimal.Decimal {
	value, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil {
		return decimal.Zero
	}
	return value
}

func normalizeSupplierNotificationPage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func truncateSupplierNotificationText(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max]
}

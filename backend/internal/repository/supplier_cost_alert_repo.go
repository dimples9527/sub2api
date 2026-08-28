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

type supplierCostAlertRepository struct {
	db *sql.DB
}

func NewSupplierCostAlertRepository(db *sql.DB) service.SupplierCostAlertRepository {
	return &supplierCostAlertRepository{db: db}
}

func (r *supplierCostAlertRepository) GetSettings(ctx context.Context) (*service.SupplierCostAlertSettings, error) {
	var amount decimal.Decimal
	err := r.db.QueryRowContext(ctx, "SELECT alert_amount FROM supplier_cost_deviation_settings WHERE id = 1").Scan(&amount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSupplierCostAlertConfigNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询供应商成本超额预警全局配置失败: %w", err)
	}
	return &service.SupplierCostAlertSettings{Amount: amount}, nil
}

func (r *supplierCostAlertRepository) UpdateSettings(ctx context.Context, amount decimal.Decimal) (*service.SupplierCostAlertSettings, error) {
	if _, err := r.db.ExecContext(ctx, `
UPDATE supplier_cost_deviation_settings
SET alert_amount = $1, updated_at = NOW()
WHERE id = 1`, amount); err != nil {
		return nil, fmt.Errorf("保存供应商成本超额预警全局配置失败: %w", err)
	}
	return &service.SupplierCostAlertSettings{Amount: amount}, nil
}

const supplierCostAlertOverrideColumns = `id, provider_id, enabled, threshold_amount, created_at, updated_at`

func scanSupplierCostAlertOverride(scan func(dest ...any) error) (*service.SupplierCostAlertOverride, error) {
	var item service.SupplierCostAlertOverride
	if err := scan(&item.ID, &item.ProviderID, &item.Enabled, &item.Amount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *supplierCostAlertRepository) GetOverrideByProvider(ctx context.Context, providerID int64) (*service.SupplierCostAlertOverride, error) {
	item, err := scanSupplierCostAlertOverride(r.db.QueryRowContext(ctx,
		"SELECT "+supplierCostAlertOverrideColumns+" FROM supplier_cost_alert_configs WHERE provider_id = $1", providerID).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSupplierCostAlertConfigNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询供应商成本超额预警覆盖配置失败: %w", err)
	}
	return item, nil
}

func (r *supplierCostAlertRepository) ListOverrides(ctx context.Context) ([]service.SupplierCostAlertOverride, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+supplierCostAlertOverrideColumns+" FROM supplier_cost_alert_configs ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("查询供应商成本超额预警覆盖配置列表失败: %w", err)
	}
	defer rows.Close()
	result := make([]service.SupplierCostAlertOverride, 0)
	for rows.Next() {
		item, scanErr := scanSupplierCostAlertOverride(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("读取供应商成本超额预警覆盖配置失败: %w", scanErr)
		}
		result = append(result, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历供应商成本超额预警覆盖配置失败: %w", err)
	}
	return result, nil
}

func (r *supplierCostAlertRepository) UpsertOverride(ctx context.Context, item *service.SupplierCostAlertOverride) (*service.SupplierCostAlertOverride, error) {
	row := r.db.QueryRowContext(ctx, `
INSERT INTO supplier_cost_alert_configs (provider_id, enabled, threshold_amount)
VALUES ($1, $2, $3)
ON CONFLICT (provider_id) DO UPDATE SET
  enabled = EXCLUDED.enabled,
  threshold_amount = EXCLUDED.threshold_amount,
  updated_at = NOW()
RETURNING `+supplierCostAlertOverrideColumns, item.ProviderID, item.Enabled, item.Amount)
	saved, err := scanSupplierCostAlertOverride(row.Scan)
	if err != nil {
		return nil, fmt.Errorf("保存供应商成本超额预警覆盖配置失败: %w", err)
	}
	return saved, nil
}

func (r *supplierCostAlertRepository) DeleteOverride(ctx context.Context, id int64) error {
	if _, err := r.db.ExecContext(ctx, "DELETE FROM supplier_cost_alert_configs WHERE id = $1", id); err != nil {
		return fmt.Errorf("删除供应商成本超额预警覆盖配置失败: %w", err)
	}
	return nil
}

const supplierCostAlertEventColumns = `id, provider_id, event_type, status, stat_date, upstream_cost, local_cost, overrun_amount, threshold_amount, observed_at, resolved_at, last_seen_at, created_at, updated_at`

func scanSupplierCostAlertEvent(scan func(dest ...any) error) (*service.SupplierCostAlertEvent, error) {
	var item service.SupplierCostAlertEvent
	if err := scan(&item.ID, &item.ProviderID, &item.EventType, &item.Status, &item.StatDate,
		&item.UpstreamCost, &item.LocalCost, &item.OverrunAmount, &item.Threshold,
		&item.ObservedAt, &item.ResolvedAt, &item.LastSeenAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *supplierCostAlertRepository) GetActiveOverrunEvent(ctx context.Context, providerID int64) (*service.SupplierCostAlertEvent, error) {
	item, err := scanSupplierCostAlertEvent(r.db.QueryRowContext(ctx,
		"SELECT "+supplierCostAlertEventColumns+" FROM supplier_cost_alert_events WHERE provider_id = $1 AND event_type = $2 AND status = $3",
		providerID, service.SupplierCostAlertEventOverrun, service.SupplierCostAlertEventActive).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSupplierCostAlertEventNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询活动成本超额事件失败: %w", err)
	}
	return item, nil
}

func (r *supplierCostAlertRepository) CreateEvent(ctx context.Context, event *service.SupplierCostAlertEvent) error {
	now := time.Now()
	if event.LastSeenAt.IsZero() {
		event.LastSeenAt = now
	}
	err := r.db.QueryRowContext(ctx, `
INSERT INTO supplier_cost_alert_events (
  provider_id, event_type, status, stat_date, upstream_cost, local_cost, overrun_amount, threshold_amount,
  observed_at, resolved_at, last_seen_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, created_at, updated_at`,
		event.ProviderID, event.EventType, event.Status, event.StatDate, event.UpstreamCost, event.LocalCost,
		event.OverrunAmount, event.Threshold, event.ObservedAt, event.ResolvedAt, event.LastSeenAt,
	).Scan(&event.ID, &event.CreatedAt, &event.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建供应商成本超额预警事件失败: %w", err)
	}
	return nil
}

func (r *supplierCostAlertRepository) TouchActiveOverrunEvent(ctx context.Context, eventID int64, event service.SupplierCostAlertEvent) error {
	if _, err := r.db.ExecContext(ctx, `
UPDATE supplier_cost_alert_events
SET stat_date = $2, upstream_cost = $3, local_cost = $4, overrun_amount = $5, threshold_amount = $6,
    observed_at = $7, last_seen_at = $8, updated_at = NOW()
WHERE id = $1`, eventID, event.StatDate, event.UpstreamCost, event.LocalCost,
		event.OverrunAmount, event.Threshold, event.ObservedAt, event.LastSeenAt); err != nil {
		return fmt.Errorf("更新供应商成本超额预警事件失败: %w", err)
	}
	return nil
}

func (r *supplierCostAlertRepository) ResolveActiveOverrunEvent(ctx context.Context, eventID int64, now time.Time) error {
	if _, err := r.db.ExecContext(ctx, `
UPDATE supplier_cost_alert_events
SET status = $2, resolved_at = $3, last_seen_at = $3, updated_at = NOW()
WHERE id = $1 AND status = $4`,
		eventID, service.SupplierCostAlertEventResolved, now, service.SupplierCostAlertEventActive); err != nil {
		return fmt.Errorf("恢复供应商成本超额预警事件失败: %w", err)
	}
	return nil
}

func (r *supplierCostAlertRepository) ListEvents(ctx context.Context, params service.SupplierCostAlertEventListParams) (service.SupplierCostAlertEventListResult, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if params.ProviderID > 0 {
		args = append(args, params.ProviderID)
		where = append(where, fmt.Sprintf("provider_id = $%d", len(args)))
	}
	if params.EventType != "" {
		args = append(args, params.EventType)
		where = append(where, fmt.Sprintf("event_type = $%d", len(args)))
	}
	if params.Status != "" {
		args = append(args, params.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	totalArgs := make([]any, len(args))
	copy(totalArgs, args)
	countWhere := strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM supplier_cost_alert_events WHERE "+countWhere, totalArgs...).Scan(&total); err != nil {
		return service.SupplierCostAlertEventListResult{}, fmt.Errorf("统计供应商成本超额预警事件失败: %w", err)
	}
	args = append(args, pageSize, (page-1)*pageSize)
	query := "SELECT " + supplierCostAlertEventColumns + " FROM supplier_cost_alert_events WHERE " +
		strings.Join(where, " AND ") +
		fmt.Sprintf(" ORDER BY observed_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return service.SupplierCostAlertEventListResult{}, fmt.Errorf("查询供应商成本超额预警事件失败: %w", err)
	}
	defer rows.Close()
	items := make([]service.SupplierCostAlertEvent, 0)
	for rows.Next() {
		item, scanErr := scanSupplierCostAlertEvent(rows.Scan)
		if scanErr != nil {
			return service.SupplierCostAlertEventListResult{}, fmt.Errorf("读取供应商成本超额预警事件失败: %w", scanErr)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return service.SupplierCostAlertEventListResult{}, fmt.Errorf("遍历供应商成本超额预警事件失败: %w", err)
	}
	return service.SupplierCostAlertEventListResult{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

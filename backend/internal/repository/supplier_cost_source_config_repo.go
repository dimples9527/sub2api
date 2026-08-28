package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type supplierCostSourceConfigRepository struct {
	db *sql.DB
}

// NewSupplierCostSourceConfigRepository 创建供应商成本来源配置仓储。
func NewSupplierCostSourceConfigRepository(db *sql.DB) service.SupplierCostSourceRepository {
	return &supplierCostSourceConfigRepository{db: db}
}

func (r *supplierCostSourceConfigRepository) GetGlobalCostSource(ctx context.Context) (string, error) {
	var source string
	err := r.db.QueryRowContext(ctx, `SELECT cost_source FROM supplier_cost_deviation_settings WHERE id = 1`).Scan(&source)
	if errors.Is(err, sql.ErrNoRows) {
		return service.SupplierCostSourceAuto, nil
	}
	if err != nil {
		return "", fmt.Errorf("查询供应商成本来源失败: %w", err)
	}
	return source, nil
}

func (r *supplierCostSourceConfigRepository) SetGlobalCostSource(ctx context.Context, source string) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE supplier_cost_deviation_settings
SET cost_source = $1, updated_at = NOW()
WHERE id = 1`, source)
	if err != nil {
		return fmt.Errorf("保存供应商成本来源失败: %w", err)
	}
	return nil
}

const supplierCostSourceConfigSelect = `
SELECT c.id, c.provider_id, COALESCE(p.name, ''), c.cost_source, c.threshold, c.created_at, c.updated_at
FROM supplier_cost_source_configs c
LEFT JOIN supplier_providers p ON p.id = c.provider_id`

func (r *supplierCostSourceConfigRepository) ListConfigs(ctx context.Context) ([]service.SupplierCostSourceConfig, error) {
	rows, err := r.db.QueryContext(ctx, supplierCostSourceConfigSelect+` ORDER BY c.provider_id`)
	if err != nil {
		return nil, fmt.Errorf("查询供应商成本来源覆盖配置失败: %w", err)
	}
	defer rows.Close()
	items := make([]service.SupplierCostSourceConfig, 0)
	for rows.Next() {
		item, scanErr := scanSupplierCostSourceConfig(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历供应商成本来源覆盖配置失败: %w", err)
	}
	return items, nil
}

func (r *supplierCostSourceConfigRepository) GetConfigByProviderID(ctx context.Context, providerID int64) (*service.SupplierCostSourceConfig, error) {
	row := r.db.QueryRowContext(ctx, supplierCostSourceConfigSelect+` WHERE c.provider_id = $1`, providerID)
	item, err := scanSupplierCostSourceConfig(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *supplierCostSourceConfigRepository) UpsertConfig(ctx context.Context, input service.SupplierCostSourceOverrideInput) (*service.SupplierCostSourceConfig, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
INSERT INTO supplier_cost_source_configs (provider_id, cost_source, threshold)
VALUES ($1, $2, $3)
ON CONFLICT (provider_id) DO UPDATE SET
  cost_source = EXCLUDED.cost_source,
  threshold = EXCLUDED.threshold,
  updated_at = NOW()
RETURNING id`, input.ProviderID, input.CostSource, input.Threshold).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("保存供应商成本来源覆盖配置失败: %w", err)
	}
	return r.getByID(ctx, id)
}

func (r *supplierCostSourceConfigRepository) DeleteConfig(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM supplier_cost_source_configs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("删除供应商成本来源覆盖配置失败: %w", err)
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return service.ErrSupplierCostSourceConfigNotFound
	}
	return nil
}

func (r *supplierCostSourceConfigRepository) getByID(ctx context.Context, id int64) (*service.SupplierCostSourceConfig, error) {
	row := r.db.QueryRowContext(ctx, supplierCostSourceConfigSelect+` WHERE c.id = $1`, id)
	item, err := scanSupplierCostSourceConfig(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// Resolve 返回供应商最终生效的成本来源与阈值：
// 存在覆盖配置时以覆盖为准（threshold 为 NULL 时回退全局阈值），否则跟随全局配置。
func (r *supplierCostSourceConfigRepository) Resolve(ctx context.Context, providerID int64) (service.SupplierCostSourceResolution, error) {
	var resolution service.SupplierCostSourceResolution
	var overridden bool
	err := r.db.QueryRowContext(ctx, `
SELECT
  COALESCE(c.cost_source, g.cost_source, 'auto') AS source,
  CASE WHEN c.provider_id IS NOT NULL AND c.threshold IS NOT NULL THEN c.threshold ELSE g.threshold END AS threshold,
  (c.provider_id IS NOT NULL) AS overridden
FROM supplier_cost_deviation_settings g
LEFT JOIN supplier_cost_source_configs c ON c.provider_id = $1
WHERE g.id = 1`, providerID).Scan(&resolution.Source, &resolution.Threshold, &overridden)
	resolution.Overridden = overridden
	if errors.Is(err, sql.ErrNoRows) {
		return service.DefaultSupplierCostSourceResolution(), nil
	}
	if err != nil {
		return service.SupplierCostSourceResolution{}, fmt.Errorf("解析供应商成本来源失败: %w", err)
	}
	return resolution, nil
}

type supplierCostSourceConfigScanner interface {
	Scan(dest ...any) error
}

func scanSupplierCostSourceConfig(scanner supplierCostSourceConfigScanner) (service.SupplierCostSourceConfig, error) {
	var item service.SupplierCostSourceConfig
	var threshold sql.NullFloat64
	err := scanner.Scan(&item.ID, &item.ProviderID, &item.ProviderName, &item.CostSource, &threshold, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	if threshold.Valid {
		item.Threshold = &threshold.Float64
	}
	return item, nil
}

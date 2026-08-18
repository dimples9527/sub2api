package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type supplierCostDeviationSettingRepository struct {
	db *sql.DB
}

func NewSupplierCostDeviationSettingRepository(db *sql.DB) service.SupplierCostDeviationSettingRepository {
	return &supplierCostDeviationSettingRepository{db: db}
}

func (r *supplierCostDeviationSettingRepository) GetThreshold(ctx context.Context) (float64, error) {
	var threshold float64
	err := r.db.QueryRowContext(ctx, `SELECT threshold FROM supplier_cost_deviation_settings WHERE id = 1`).Scan(&threshold)
	if errors.Is(err, sql.ErrNoRows) {
		return service.DefaultSupplierCostDeviationThreshold, nil
	}
	if err != nil {
		return 0, fmt.Errorf("查询供应商成本偏差阈值失败: %w", err)
	}
	return threshold, nil
}

func (r *supplierCostDeviationSettingRepository) SetThreshold(ctx context.Context, threshold float64) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO supplier_cost_deviation_settings (id, threshold)
VALUES (1, $1)
ON CONFLICT (id) DO UPDATE SET
  threshold = EXCLUDED.threshold,
  updated_at = NOW()`, threshold)
	if err != nil {
		return fmt.Errorf("保存供应商成本偏差阈值失败: %w", err)
	}
	return nil
}

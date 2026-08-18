package service

import (
	"context"
	"fmt"
)

// 供应商成本偏差覆盖阈值默认值与上下限。
const (
	DefaultSupplierCostDeviationThreshold = 0.5
	supplierCostDeviationThresholdMin     = 0.05
	supplierCostDeviationThresholdMax     = 0.95
)

// SupplierCostDeviationSettings 供应商成本偏差覆盖阈值全局配置。
type SupplierCostDeviationSettings struct {
	Threshold float64 `json:"threshold"`
}

// SupplierCostDeviationThresholdProvider 提供当前成本偏差覆盖阈值。
// 写入时与展示时的成本覆盖逻辑都依赖该阈值。
type SupplierCostDeviationThresholdProvider interface {
	SupplierCostDeviationThreshold(ctx context.Context) float64
}

// SupplierCostDeviationSettingRepository 供应商模块自己的配置仓储（独立于框架 settings 表）。
type SupplierCostDeviationSettingRepository interface {
	GetThreshold(ctx context.Context) (float64, error)
	SetThreshold(ctx context.Context, threshold float64) error
}

func clampSupplierCostDeviationThreshold(threshold float64) float64 {
	if threshold < supplierCostDeviationThresholdMin {
		return supplierCostDeviationThresholdMin
	}
	if threshold > supplierCostDeviationThresholdMax {
		return supplierCostDeviationThresholdMax
	}
	return threshold
}

// SupplierCostDeviationSettingsService 管理供应商成本偏差覆盖阈值配置。
// 它只依赖供应商模块自己的配置表，不写入框架通用 settings 表。
type SupplierCostDeviationSettingsService struct {
	repo SupplierCostDeviationSettingRepository
}

func NewSupplierCostDeviationSettingsService(repo SupplierCostDeviationSettingRepository) *SupplierCostDeviationSettingsService {
	return &SupplierCostDeviationSettingsService{repo: repo}
}

// SupplierCostDeviationThreshold 返回当前配置的成本偏差覆盖阈值；
// 未配置或读取失败时回退到默认值。
func (s *SupplierCostDeviationSettingsService) SupplierCostDeviationThreshold(ctx context.Context) float64 {
	if s == nil || s.repo == nil {
		return DefaultSupplierCostDeviationThreshold
	}
	threshold, err := s.repo.GetThreshold(ctx)
	if err != nil || threshold <= 0 {
		return DefaultSupplierCostDeviationThreshold
	}
	return clampSupplierCostDeviationThreshold(threshold)
}

// GetSupplierCostDeviationSettings 读取供应商成本偏差覆盖阈值配置。
func (s *SupplierCostDeviationSettingsService) GetSupplierCostDeviationSettings(ctx context.Context) (*SupplierCostDeviationSettings, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("supplier cost deviation settings service is not configured")
	}
	return &SupplierCostDeviationSettings{Threshold: s.SupplierCostDeviationThreshold(ctx)}, nil
}

// UpdateSupplierCostDeviationSettings 更新供应商成本偏差覆盖阈值配置。
func (s *SupplierCostDeviationSettingsService) UpdateSupplierCostDeviationSettings(ctx context.Context, threshold float64) (*SupplierCostDeviationSettings, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("supplier cost deviation settings service is not configured")
	}
	clamped := clampSupplierCostDeviationThreshold(threshold)
	if err := s.repo.SetThreshold(ctx, clamped); err != nil {
		return nil, fmt.Errorf("update supplier cost deviation settings: %w", err)
	}
	// 成本趋势缓存结果依赖覆盖阈值，阈值变化后必须失效缓存。
	invalidateSupplierCostTrendCache()
	return &SupplierCostDeviationSettings{Threshold: clamped}, nil
}

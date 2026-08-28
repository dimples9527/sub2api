package service

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 供应商成本来源模式：
//
//	auto       智能模式（默认，等同历史行为）：待审批核对记录默认采用计算成本，
//	           同步写入以上游接口成本为准，偏差超过阈值时改写为充值修正余额成本或本地成本。
//	upstream   始终以上游接口成本为准，不做偏差改写。
//	calculated 始终以本地计算成本为准，不做偏差改写。
const (
	SupplierCostSourceAuto       = "auto"
	SupplierCostSourceUpstream   = "upstream"
	SupplierCostSourceCalculated = "calculated"
)

// ErrSupplierCostSourceConfigNotFound 供应商成本来源覆盖配置不存在。
var ErrSupplierCostSourceConfigNotFound = infraerrors.NotFound("SUPPLIER_COST_SOURCE_CONFIG_NOT_FOUND", "供应商成本来源覆盖配置不存在")

// SupplierCostSourceSettings 全局默认成本来源配置。
type SupplierCostSourceSettings struct {
	CostSource string `json:"cost_source"`
}

// SupplierCostSourceConfig 供应商级成本来源覆盖配置。
// Threshold 仅在 cost_source 为 auto 时参与偏差判断；nil 表示跟随全局阈值。
type SupplierCostSourceConfig struct {
	ID           int64     `json:"id"`
	ProviderID   int64     `json:"provider_id"`
	ProviderName string    `json:"provider_name"`
	CostSource   string    `json:"cost_source"`
	Threshold    *float64  `json:"threshold,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SupplierCostSourceOverrideInput 保存供应商成本来源覆盖配置的输入。
type SupplierCostSourceOverrideInput struct {
	ProviderID int64    `json:"provider_id"`
	CostSource string   `json:"cost_source"`
	Threshold  *float64 `json:"threshold"`
}

// SupplierCostSourceResolution 单个供应商最终生效的成本来源与偏差阈值。
type SupplierCostSourceResolution struct {
	Source     string
	Threshold  float64
	Overridden bool
}

// DefaultSupplierCostSourceResolution 返回未配置时的默认解析结果，等同历史行为。
func DefaultSupplierCostSourceResolution() SupplierCostSourceResolution {
	return SupplierCostSourceResolution{
		Source:    SupplierCostSourceAuto,
		Threshold: DefaultSupplierCostDeviationThreshold,
	}
}

// ValidateSupplierCostSource 校验成本来源模式取值。
func ValidateSupplierCostSource(source string) error {
	switch source {
	case SupplierCostSourceAuto, SupplierCostSourceUpstream, SupplierCostSourceCalculated:
		return nil
	default:
		return fmt.Errorf("成本来源无效")
	}
}

// validateSupplierCostSourceThreshold 校验覆盖阈值：nil 表示跟随全局阈值，否则必须为有限非负数字。
func validateSupplierCostSourceThreshold(threshold *float64) error {
	if threshold == nil {
		return nil
	}
	if math.IsNaN(*threshold) || math.IsInf(*threshold, 0) || *threshold < 0 {
		return fmt.Errorf("成本来源覆盖阈值必须是非负数字")
	}
	return nil
}

// SupplierCostSourceRepository 供应商成本来源配置仓储。
type SupplierCostSourceRepository interface {
	GetGlobalCostSource(ctx context.Context) (string, error)
	SetGlobalCostSource(ctx context.Context, source string) error
	ListConfigs(ctx context.Context) ([]SupplierCostSourceConfig, error)
	GetConfigByProviderID(ctx context.Context, providerID int64) (*SupplierCostSourceConfig, error)
	UpsertConfig(ctx context.Context, input SupplierCostSourceOverrideInput) (*SupplierCostSourceConfig, error)
	DeleteConfig(ctx context.Context, id int64) error
	// Resolve 一次查询返回供应商最终生效的成本来源与阈值（覆盖优先，回退全局）。
	Resolve(ctx context.Context, providerID int64) (SupplierCostSourceResolution, error)
}

// SupplierCostSourceResolver 供成本同步与趋势展示链路解析供应商最终生效成本来源。
type SupplierCostSourceResolver interface {
	ResolveCostSource(ctx context.Context, providerID int64) (SupplierCostSourceResolution, error)
}

// defaultSupplierCostSourceResolveTTL 解析结果缓存时长，避免同步热路径每次查询配置表。
const defaultSupplierCostSourceResolveTTL = 30 * time.Second

type supplierCostSourceCacheEntry struct {
	resolution SupplierCostSourceResolution
	expiresAt  time.Time
}

// SupplierCostSourceConfigService 管理成本来源全局与供应商覆盖配置，并提供带短缓存的解析能力。
type SupplierCostSourceConfigService struct {
	repo SupplierCostSourceRepository

	mu         sync.Mutex
	resolveTTL time.Duration
	cache      map[int64]supplierCostSourceCacheEntry
}

// NewSupplierCostSourceConfigService 创建成本来源配置服务。
func NewSupplierCostSourceConfigService(repo SupplierCostSourceRepository) *SupplierCostSourceConfigService {
	return &SupplierCostSourceConfigService{repo: repo, resolveTTL: defaultSupplierCostSourceResolveTTL, cache: make(map[int64]supplierCostSourceCacheEntry)}
}

// GetSettings 读取全局默认成本来源；未配置或读取失败时回退 auto。
func (s *SupplierCostSourceConfigService) GetSettings(ctx context.Context) (*SupplierCostSourceSettings, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("成本来源配置服务未初始化")
	}
	source, err := s.repo.GetGlobalCostSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("读取供应商成本来源配置失败: %w", err)
	}
	if ValidateSupplierCostSource(source) != nil {
		source = SupplierCostSourceAuto
	}
	return &SupplierCostSourceSettings{CostSource: source}, nil
}

// UpdateGlobalCostSource 更新全局默认成本来源，并失效解析缓存与趋势缓存。
func (s *SupplierCostSourceConfigService) UpdateGlobalCostSource(ctx context.Context, source string) (*SupplierCostSourceSettings, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("成本来源配置服务未初始化")
	}
	if ValidateSupplierCostSource(source) != nil {
		return nil, fmt.Errorf("成本来源无效")
	}
	if err := s.repo.SetGlobalCostSource(ctx, source); err != nil {
		return nil, fmt.Errorf("保存供应商成本来源配置失败: %w", err)
	}
	s.invalidateCache()
	// 趋势展示的偏差兜底改写依赖成本来源，配置变化后必须失效趋势缓存。
	invalidateSupplierCostTrendCache()
	return &SupplierCostSourceSettings{CostSource: source}, nil
}

// ListOverrides 列出全部供应商覆盖配置。
func (s *SupplierCostSourceConfigService) ListOverrides(ctx context.Context) ([]SupplierCostSourceConfig, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("成本来源配置服务未初始化")
	}
	return s.repo.ListConfigs(ctx)
}

// UpsertOverride 保存供应商覆盖配置；auto 模式允许仅覆盖阈值。
func (s *SupplierCostSourceConfigService) UpsertOverride(ctx context.Context, input SupplierCostSourceOverrideInput) (*SupplierCostSourceConfig, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("成本来源配置服务未初始化")
	}
	if input.ProviderID <= 0 {
		return nil, fmt.Errorf("供应商 ID 无效")
	}
	if err := ValidateSupplierCostSource(input.CostSource); err != nil {
		return nil, err
	}
	if err := validateSupplierCostSourceThreshold(input.Threshold); err != nil {
		return nil, err
	}
	config, err := s.repo.UpsertConfig(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("保存供应商成本来源覆盖配置失败: %w", err)
	}
	s.invalidateCache()
	invalidateSupplierCostTrendCache()
	return config, nil
}

// DeleteOverride 删除供应商覆盖配置，恢复跟随全局配置。
func (s *SupplierCostSourceConfigService) DeleteOverride(ctx context.Context, id int64) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("成本来源配置服务未初始化")
	}
	if id <= 0 {
		return fmt.Errorf("覆盖配置 ID 无效")
	}
	if err := s.repo.DeleteConfig(ctx, id); err != nil {
		return fmt.Errorf("删除供应商成本来源覆盖配置失败: %w", err)
	}
	s.invalidateCache()
	invalidateSupplierCostTrendCache()
	return nil
}

// ResolveCostSource 解析供应商最终生效的成本来源与阈值，带短缓存。
// 配置读取失败时回退默认值，保证同步与展示主链路不被配置读取故障阻断。
func (s *SupplierCostSourceConfigService) ResolveCostSource(ctx context.Context, providerID int64) (SupplierCostSourceResolution, error) {
	if s == nil || s.repo == nil || providerID <= 0 {
		return DefaultSupplierCostSourceResolution(), nil
	}
	if cached, ok := s.cachedResolution(providerID); ok {
		return cached, nil
	}
	resolution, err := s.repo.Resolve(ctx, providerID)
	if err != nil {
		return DefaultSupplierCostSourceResolution(), err
	}
	if ValidateSupplierCostSource(resolution.Source) != nil {
		resolution = DefaultSupplierCostSourceResolution()
	}
	s.storeCache(providerID, resolution)
	return resolution, nil
}

func (s *SupplierCostSourceConfigService) cachedResolution(providerID int64) (SupplierCostSourceResolution, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.cache[providerID]
	if !ok || time.Now().After(entry.expiresAt) {
		return SupplierCostSourceResolution{}, false
	}
	return entry.resolution, true
}

func (s *SupplierCostSourceConfigService) storeCache(providerID int64, resolution SupplierCostSourceResolution) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[providerID] = supplierCostSourceCacheEntry{resolution: resolution, expiresAt: time.Now().Add(s.resolveTTL)}
}

func (s *SupplierCostSourceConfigService) invalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[int64]supplierCostSourceCacheEntry)
}

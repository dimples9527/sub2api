package handler

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// 本文件实现模型广场用户只读聚合接口，供普通用户查看模型广场页面。
// 返回的数据结构需与前端 buildConfiguredModelSquareResult 聚合函数期望的字段保持一致。

// modelSquareConfigService 定义模型广场配置读取依赖。
type modelSquareConfigService interface {
	GetModelSquareConfig(ctx context.Context) (service.ModelSquareConfig, error)
}

// modelSquareChannelService 定义渠道列表读取依赖。
type modelSquareChannelService interface {
	List(ctx context.Context, params pagination.PaginationParams, status, search string) ([]service.Channel, *pagination.PaginationResult, error)
}

// modelSquareGroupService 定义分组读取依赖。
type modelSquareGroupService interface {
	GetAllGroupsIncludingInactive(ctx context.Context) ([]service.Group, error)
}

// modelSquareOverrideService 定义分组平台覆盖读取依赖。
type modelSquareOverrideService interface {
	ListByGroupIDs(ctx context.Context, groupIDs []int64) (map[int64]service.MonitorGroupPlatformOverride, error)
}

// modelSquareBillingService 定义模型参考价读取依赖。
type modelSquareBillingService interface {
	GetModelPricing(model string) (*service.ModelPricing, error)
}

// ModelSquareHandler 处理模型广场用户只读聚合接口。
//
// 聚合配置、分组、平台覆盖与参考价数据，输出结构与前端
// buildConfiguredModelSquareResult 聚合函数期望的字段保持一致。
// 仅暴露普通用户查看模型广场所需的白名单字段。
//
// 不再聚合渠道：模型可用性改由前端按模型绑定的分组判定，
// 渠道数据（列表、分组关联、模型定价）在该接口上已无消费方。
type ModelSquareHandler struct {
	configService   modelSquareConfigService
	groupService    modelSquareGroupService
	overrideService modelSquareOverrideService
	billingService  modelSquareBillingService
	// channelService 仍由 wire 注入：wire_gen.go 的 NewModelSquareHandler 调用依赖这个形参，
	// 删它必须重新运行 wire。本接口已不再读取它。
	channelService modelSquareChannelService
}

// NewModelSquareHandler 创建模型广场用户只读 handler。
func NewModelSquareHandler(
	upstreamService *service.UpstreamManagementService,
	channelService *service.ChannelService,
	adminService service.AdminService,
	overrideService service.MonitorGroupPlatformOverrideService,
	billingService *service.BillingService,
) *ModelSquareHandler {
	return &ModelSquareHandler{
		configService:   upstreamService,
		channelService:  channelService,
		groupService:    adminService,
		overrideService: overrideService,
		billingService:  billingService,
	}
}

// modelSquareUserModelConfig 定义模型广场配置中的单个模型（用户可见字段）。
type modelSquareUserModelConfig struct {
	ID                      string   `json:"id"`
	DisplayName             string   `json:"display_name,omitempty"`
	Source                  string   `json:"source,omitempty"`
	GroupIDs                []int64  `json:"group_ids,omitempty"`
	InputPrice              *float64 `json:"input_price,omitempty"`
	OutputPrice             *float64 `json:"output_price,omitempty"`
	CacheWritePrice         *float64 `json:"cache_write_price,omitempty"`
	CacheWrite1hPrice       *float64 `json:"cache_write_1h_price,omitempty"`
	CacheReadPrice          *float64 `json:"cache_read_price,omitempty"`
	InputPricePriority      *float64 `json:"input_price_priority,omitempty"`
	OutputPricePriority     *float64 `json:"output_price_priority,omitempty"`
	CacheWritePricePriority *float64 `json:"cache_write_price_priority,omitempty"`
	CacheReadPricePriority  *float64 `json:"cache_read_price_priority,omitempty"`
	ImageInputPrice         *float64 `json:"image_input_price,omitempty"`
	ImageOutputPrice        *float64 `json:"image_output_price,omitempty"`
	PerRequestPrice         *float64 `json:"per_request_price,omitempty"`
}

// modelSquareUserPlatformConfig 定义模型广场配置中的单个平台。
type modelSquareUserPlatformConfig struct {
	Platform string                        `json:"platform"`
	Name     string                        `json:"name,omitempty"`
	Models   []modelSquareUserModelConfig  `json:"models"`
}

// modelSquareUserConfig 定义模型广场用户可见配置。
type modelSquareUserConfig struct {
	Platforms []modelSquareUserPlatformConfig `json:"platforms"`
	UpdatedAt *time.Time                      `json:"updated_at,omitempty"`
}

// modelSquareUserGroup 定义用户可见分组。
type modelSquareUserGroup struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Platform       string  `json:"platform"`
	RateMultiplier float64 `json:"rate_multiplier"`
}

// modelSquareUserPlatformOverride 定义分组平台覆盖（分组 ID -> 生效平台）。
type modelSquareUserPlatformOverride struct {
	ID                int64  `json:"id"`
	EffectivePlatform string `json:"effective_platform"`
}

// modelSquareUserReferencePrice 定义模型参考价（按 token 计费，字段命名与管理员端一致）。
type modelSquareUserReferencePrice struct {
	Found                      bool     `json:"found"`
	InputPrice                 *float64 `json:"input_price,omitempty"`
	OutputPrice                *float64 `json:"output_price,omitempty"`
	CacheWritePrice            *float64 `json:"cache_write_price,omitempty"`
	CacheWrite1hPrice          *float64 `json:"cache_write_1h_price,omitempty"`
	CacheReadPrice             *float64 `json:"cache_read_price,omitempty"`
	InputPricePriority         *float64 `json:"input_price_priority,omitempty"`
	OutputPricePriority        *float64 `json:"output_price_priority,omitempty"`
	CacheWritePricePriority    *float64 `json:"cache_write_price_priority,omitempty"`
	CacheReadPricePriority     *float64 `json:"cache_read_price_priority,omitempty"`
	ImageInputPrice            *float64 `json:"image_input_price,omitempty"`
	ImageOutputPrice           *float64 `json:"image_output_price,omitempty"`
}

// modelSquareUserResponse 定义模型广场用户只读接口响应。
type modelSquareUserResponse struct {
	Config            modelSquareUserConfig                    `json:"config"`
	Groups            []modelSquareUserGroup                   `json:"groups"`
	PlatformOverrides []modelSquareUserPlatformOverride        `json:"platform_overrides"`
	ReferencePrices   map[string]modelSquareUserReferencePrice `json:"reference_prices"`
}

// Get 返回模型广场用户只读聚合数据。
// GET /api/v1/model-square
func (h *ModelSquareHandler) Get(c *gin.Context) {
	ctx := c.Request.Context()

	config, err := h.configService.GetModelSquareConfig(ctx)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	groups, err := h.groupService.GetAllGroupsIncludingInactive(ctx)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	groupIDs := make([]int64, 0, len(groups))
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}
	var overrides map[int64]service.MonitorGroupPlatformOverride
	if len(groupIDs) > 0 && h.overrideService != nil {
		overrides, err = h.overrideService.ListByGroupIDs(ctx, groupIDs)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}

	response.Success(c, modelSquareUserResponse{
		Config:            toModelSquareUserConfig(config),
		Groups:            toModelSquareUserGroups(groups),
		PlatformOverrides: toModelSquareUserPlatformOverrides(overrides),
		ReferencePrices:   h.loadReferencePrices(ctx, config),
	})
}

// loadReferencePrices 仅回查配置中缺失 token 价格的模型参考价。
// 配置已包含完整 token 价格的模型不重复查询计费服务，减少无效调用。
func (h *ModelSquareHandler) loadReferencePrices(ctx context.Context, config service.ModelSquareConfig) map[string]modelSquareUserReferencePrice {
	out := make(map[string]modelSquareUserReferencePrice)
	if h.billingService == nil {
		return out
	}
	for _, platform := range config.Platforms {
		for _, model := range platform.Models {
			id := strings.TrimSpace(model.ID)
			if id == "" || !hasMissingConfiguredTokenPrice(model) {
				continue
			}
			pricing, err := h.billingService.GetModelPricing(id)
			if err != nil || pricing == nil {
				continue
			}
			out[strings.ToLower(id)] = modelSquareUserReferencePriceFromPricing(pricing)
		}
	}
	return out
}

// toModelSquareUserConfig 将配置转换为用户 DTO（白名单字段）。
func toModelSquareUserConfig(config service.ModelSquareConfig) modelSquareUserConfig {
	out := modelSquareUserConfig{
		Platforms: make([]modelSquareUserPlatformConfig, 0, len(config.Platforms)),
		UpdatedAt: config.UpdatedAt,
	}
	for _, platform := range config.Platforms {
		models := make([]modelSquareUserModelConfig, 0, len(platform.Models))
		for _, model := range platform.Models {
			models = append(models, modelSquareUserModelConfig{
				ID:                      model.ID,
				DisplayName:             model.DisplayName,
				Source:                  model.Source,
				GroupIDs:                model.GroupIDs,
				InputPrice:              model.InputPrice,
				OutputPrice:             model.OutputPrice,
				CacheWritePrice:         model.CacheWritePrice,
				CacheWrite1hPrice:       model.CacheWrite1hPrice,
				CacheReadPrice:          model.CacheReadPrice,
				InputPricePriority:      model.InputPricePriority,
				OutputPricePriority:     model.OutputPricePriority,
				CacheWritePricePriority: model.CacheWritePricePriority,
				CacheReadPricePriority:  model.CacheReadPricePriority,
				ImageInputPrice:         model.ImageInputPrice,
				ImageOutputPrice:        model.ImageOutputPrice,
				PerRequestPrice:         model.PerRequestPrice,
			})
		}
		out.Platforms = append(out.Platforms, modelSquareUserPlatformConfig{
			Platform: platform.Platform,
			Name:     platform.Name,
			Models:   models,
		})
	}
	return out
}

// toModelSquareUserGroups 将分组转换为用户 DTO（白名单字段）。
func toModelSquareUserGroups(groups []service.Group) []modelSquareUserGroup {
	out := make([]modelSquareUserGroup, 0, len(groups))
	for _, g := range groups {
		out = append(out, modelSquareUserGroup{
			ID:             g.ID,
			Name:           g.Name,
			Platform:       g.Platform,
			RateMultiplier: g.RateMultiplier,
		})
	}
	return out
}

// toModelSquareUserPlatformOverrides 将平台覆盖转换为用户 DTO。
func toModelSquareUserPlatformOverrides(overrides map[int64]service.MonitorGroupPlatformOverride) []modelSquareUserPlatformOverride {
	if len(overrides) == 0 {
		return []modelSquareUserPlatformOverride{}
	}
	out := make([]modelSquareUserPlatformOverride, 0, len(overrides))
	for groupID, override := range overrides {
		out = append(out, modelSquareUserPlatformOverride{
			ID:                groupID,
			EffectivePlatform: override.ActualPlatform,
		})
	}
	return out
}

// modelSquareUserReferencePriceFromPricing 将计费服务价格转换为参考价 DTO。
func modelSquareUserReferencePriceFromPricing(pricing *service.ModelPricing) modelSquareUserReferencePrice {
	return modelSquareUserReferencePrice{
		Found:                      true,
		InputPrice:                 &pricing.InputPricePerToken,
		OutputPrice:                &pricing.OutputPricePerToken,
		CacheWritePrice:            &pricing.CacheCreationPricePerToken,
		CacheWrite1hPrice:          &pricing.CacheCreation1hPrice,
		CacheReadPrice:             &pricing.CacheReadPricePerToken,
		InputPricePriority:         &pricing.InputPricePerTokenPriority,
		OutputPricePriority:        &pricing.OutputPricePerTokenPriority,
		CacheWritePricePriority:    &pricing.CacheCreationPricePerTokenPriority,
		CacheReadPricePriority:     &pricing.CacheReadPricePerTokenPriority,
		ImageInputPrice:            &pricing.ImageInputPricePerToken,
		ImageOutputPrice:           &pricing.ImageOutputPricePerToken,
	}
}

// hasMissingConfiguredTokenPrice 判断配置中是否缺失任一 token 价格字段。
// 与 TOKEN_PRICE_FIELDS 保持一致，per_request_price 按次计费模型不属于 token 价格字段。
func hasMissingConfiguredTokenPrice(model service.ModelSquarePlatformModelConfig) bool {
	prices := []*float64{
		model.InputPrice,
		model.OutputPrice,
		model.CacheWritePrice,
		model.CacheWrite1hPrice,
		model.CacheReadPrice,
		model.InputPricePriority,
		model.OutputPricePriority,
		model.CacheWritePricePriority,
		model.CacheReadPricePriority,
		model.ImageInputPrice,
		model.ImageOutputPrice,
	}
	for _, price := range prices {
		if price == nil {
			return true
		}
	}
	return false
}

//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 以下 stub 通过模型广场 handler 的小接口注入依赖，避免依赖真实仓储。

type modelSquareStubConfigService struct {
	config service.ModelSquareConfig
	err    error
}

func (s *modelSquareStubConfigService) GetModelSquareConfig(ctx context.Context) (service.ModelSquareConfig, error) {
	return s.config, s.err
}

type modelSquareStubChannelService struct {
	channels []service.Channel
	result   *pagination.PaginationResult
	err      error
}

func (s *modelSquareStubChannelService) List(ctx context.Context, params pagination.PaginationParams, status, search string) ([]service.Channel, *pagination.PaginationResult, error) {
	return s.channels, s.result, s.err
}

type modelSquareStubGroupService struct {
	groups []service.Group
	err    error
}

func (s *modelSquareStubGroupService) GetAllGroupsIncludingInactive(ctx context.Context) ([]service.Group, error) {
	return s.groups, s.err
}

type modelSquareStubOverrideService struct {
	overrides map[int64]service.MonitorGroupPlatformOverride
	err       error
}

func (s *modelSquareStubOverrideService) ListByGroupIDs(ctx context.Context, groupIDs []int64) (map[int64]service.MonitorGroupPlatformOverride, error) {
	return s.overrides, s.err
}

type modelSquareStubBillingService struct {
	prices map[string]*service.ModelPricing
}

func (s *modelSquareStubBillingService) GetModelPricing(model string) (*service.ModelPricing, error) {
	if s.prices == nil {
		return nil, nil
	}
	return s.prices[model], nil
}

// modelSquareTestPrice 生成一组完整的 token 价格指针。
func modelSquareTestPrice(input, output, cacheWrite, cacheWrite1h, cacheRead, inputPriority, outputPriority, cacheWritePriority, cacheReadPriority, imageInput, imageOutput float64) *service.ModelPricing {
	return &service.ModelPricing{
		InputPricePerToken:                 input,
		OutputPricePerToken:                output,
		CacheCreationPricePerToken:         cacheWrite,
		CacheCreation1hPrice:               cacheWrite1h,
		CacheReadPricePerToken:             cacheRead,
		InputPricePerTokenPriority:         inputPriority,
		OutputPricePerTokenPriority:        outputPriority,
		CacheCreationPricePerTokenPriority: cacheWritePriority,
		CacheReadPricePerTokenPriority:     cacheReadPriority,
		ImageInputPricePerToken:            imageInput,
		ImageOutputPricePerToken:           imageOutput,
	}
}

// modelSquareTestConfig 构造一份模型广场配置：gpt-5.5 价格完整，ghost-model 缺全部 token 价格。
func modelSquareTestConfig() service.ModelSquareConfig {
	input := 3e-6
	output := 15e-6
	cacheWrite := 6e-6
	cacheWrite1h := 7e-6
	cacheRead := 1e-6
	inputPriority := 8e-6
	outputPriority := 40e-6
	cacheWritePriority := 9e-6
	cacheReadPriority := 2e-6
	imageInput := 10e-6
	imageOutput := 20e-6
	perRequest := 0.12
	return service.ModelSquareConfig{
		Platforms: []service.ModelSquarePlatformConfig{
			{
				Platform: "openai",
				Name:     "OpenAI Official",
				Models: []service.ModelSquarePlatformModelConfig{
					{
						ID:                      "gpt-5.5",
						DisplayName:             "GPT-5.5 Flagship",
						InputPrice:              &input,
						OutputPrice:             &output,
						CacheWritePrice:         &cacheWrite,
						CacheWrite1hPrice:       &cacheWrite1h,
						CacheReadPrice:          &cacheRead,
						InputPricePriority:      &inputPriority,
						OutputPricePriority:     &outputPriority,
						CacheWritePricePriority: &cacheWritePriority,
						CacheReadPricePriority:  &cacheReadPriority,
						ImageInputPrice:         &imageInput,
						ImageOutputPrice:        &imageOutput,
						PerRequestPrice:         &perRequest,
					},
					{
						ID:          "ghost-model",
						DisplayName: "Ghost Model",
					},
				},
			},
		},
	}
}

// modelSquareTestChannels 构造渠道：横跨 openai 与 glm 两个平台定价，并带有管理端敏感字段。
func modelSquareTestChannels() []service.Channel {
	return []service.Channel{
		{
			ID:                 1,
			Name:               "Local channel",
			Description:        "内部描述",
			Status:             "active",
			BillingModelSource: "channel_mapped",
			RestrictModels:     true,
			GroupIDs:           []int64{1, 2},
			ModelPricing: []service.ChannelModelPricing{
				{Platform: "openai", Models: []string{"gpt-5.5"}},
				{Platform: "glm", Models: []string{"glm-4.5"}},
			},
			ModelMapping: map[string]map[string]string{
				"openai": {"gpt-5.5": "upstream-model"},
			},
		},
	}
}

// modelSquareTestGroups 构造分组：GLM Group 原始平台是 openai，但分组平台配置会覆盖为 glm。
func modelSquareTestGroups() []service.Group {
	return []service.Group{
		{ID: 1, Name: "GLM Group", Platform: "openai", RateMultiplier: 0.3},
		{ID: 2, Name: "OpenAI Group", Platform: "openai", RateMultiplier: 0.8},
	}
}

func TestModelSquareHandler_Get_ReturnsAggregatedUserData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &ModelSquareHandler{
		configService: &modelSquareStubConfigService{config: modelSquareTestConfig()},
		channelService: &modelSquareStubChannelService{
			channels: modelSquareTestChannels(),
			result:   &pagination.PaginationResult{Total: 1, Page: 1, PageSize: 1000, Pages: 1},
		},
		groupService: &modelSquareStubGroupService{groups: modelSquareTestGroups()},
		overrideService: &modelSquareStubOverrideService{
			overrides: map[int64]service.MonitorGroupPlatformOverride{
				1: {ActualPlatform: "glm", ShowInMonitor: true},
			},
		},
		billingService: &modelSquareStubBillingService{
			prices: map[string]*service.ModelPricing{
				"ghost-model": modelSquareTestPrice(3e-6, 15e-6, 6e-6, 7e-6, 1e-6, 8e-6, 40e-6, 9e-6, 2e-6, 10e-6, 20e-6),
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/model-square", nil)

	handler.Get(c)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Code int                    `json:"code"`
		Data modelSquareUserResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)

	data := body.Data
	require.Len(t, data.Config.Platforms, 1)
	require.Equal(t, "openai", data.Config.Platforms[0].Platform)
	require.Len(t, data.Config.Platforms[0].Models, 2)
	require.Equal(t, "gpt-5.5", data.Config.Platforms[0].Models[0].ID)
	require.NotNil(t, data.Config.Platforms[0].Models[0].InputPrice)

	require.Len(t, data.Channels, 1)
	ch := data.Channels[0]
	require.Equal(t, int64(1), ch.ID)
	require.Equal(t, "active", ch.Status)
	require.Equal(t, []int64{1, 2}, ch.GroupIDs)
	require.Len(t, ch.ModelPricing, 2)
	require.Equal(t, "openai", ch.ModelPricing[0].Platform)
	require.Equal(t, []string{"gpt-5.5"}, ch.ModelPricing[0].Models)
	require.Equal(t, map[string]map[string]string{"openai": {"gpt-5.5": "upstream-model"}}, ch.ModelMapping)

	require.Len(t, data.Groups, 2)
	require.Equal(t, "GLM Group", data.Groups[0].Name)
	require.InDelta(t, 0.3, data.Groups[0].RateMultiplier, 1e-9)

	// 分组平台覆盖：GLM Group（id=1）应携带 effective_platform=glm，
	// 前端据此把 GLM 分组从 openai 目录移到 glm 目录下。
	require.Equal(t, []modelSquareUserPlatformOverride{{ID: 1, EffectivePlatform: "glm"}}, data.PlatformOverrides)

	// reference_prices 只回查缺 token 价格的 ghost-model，gpt-5.5 不在其中。
	require.Contains(t, data.ReferencePrices, "ghost-model")
	require.True(t, data.ReferencePrices["ghost-model"].Found)
	require.NotNil(t, data.ReferencePrices["ghost-model"].InputPrice)
	_, hasComplete := data.ReferencePrices["gpt-5.5"]
	require.False(t, hasComplete, "价格完整的模型不应进入 reference_prices")
}

func TestModelSquareUserResponse_FieldWhitelist(t *testing.T) {
	// 序列化整个响应结构体，确认管理端敏感字段不会泄漏到用户接口。
	resp := modelSquareUserResponse{
		Config: modelSquareUserConfig{
			Platforms: []modelSquareUserPlatformConfig{
				{
					Platform: "openai",
					Name:     "OpenAI",
					Models: []modelSquareUserModelConfig{
						{ID: "gpt-5.5", DisplayName: "GPT-5.5"},
					},
				},
			},
		},
		Channels: []modelSquareUserChannel{
			{
				ID:           1,
				Status:       "active",
				GroupIDs:     []int64{1},
				ModelPricing: []modelSquareUserChannelModelPricing{{Platform: "openai", Models: []string{"gpt-5.5"}}},
				ModelMapping: map[string]map[string]string{},
			},
		},
		Groups: []modelSquareUserGroup{
			{ID: 1, Name: "GLM Group", Platform: "glm", RateMultiplier: 0.3},
		},
		PlatformOverrides: []modelSquareUserPlatformOverride{{ID: 1, EffectivePlatform: "glm"}},
		ReferencePrices:   map[string]modelSquareUserReferencePrice{},
	}

	raw, err := json.Marshal(resp)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))

	for _, key := range []string{"config", "channels", "groups", "platform_overrides", "reference_prices"} {
		_, exists := decoded[key]
		require.Truef(t, exists, "response must expose %q", key)
	}

	platform := decoded["config"].(map[string]any)["platforms"].([]any)[0].(map[string]any)
	for _, key := range []string{"synced_from_account_id", "synced_from_account_name", "synced_at"} {
		_, exists := platform[key]
		require.Falsef(t, exists, "config platform must not expose %q", key)
	}

	channel := decoded["channels"].([]any)[0].(map[string]any)
	for _, key := range []string{"name", "description", "billing_model_source", "restrict_models", "features", "created_at", "updated_at", "apply_pricing_to_account_stats", "account_stats_pricing_rules"} {
		_, exists := channel[key]
		require.Falsef(t, exists, "channel must not expose %q", key)
	}
	for _, key := range []string{"id", "status", "group_ids", "model_pricing", "model_mapping"} {
		_, exists := channel[key]
		require.Truef(t, exists, "channel must expose %q", key)
	}

	group := decoded["groups"].([]any)[0].(map[string]any)
	for _, key := range []string{"description", "status", "subscription_type", "is_exclusive", "peak_rate_enabled", "created_at", "updated_at"} {
		_, exists := group[key]
		require.Falsef(t, exists, "group must not expose %q", key)
	}

	override := decoded["platform_overrides"].([]any)[0].(map[string]any)
	require.Equal(t, float64(1), override["id"])
	require.Equal(t, "glm", override["effective_platform"])
}

func TestModelSquareHandler_Get_ReferencePricesOnlyForMissingTokenPrices(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// 计费服务同时有 gpt-5.5 与 ghost-model 的价目，但 gpt-5.5 配置价格完整，不应回查。
	handler := &ModelSquareHandler{
		configService: &modelSquareStubConfigService{config: modelSquareTestConfig()},
		channelService: &modelSquareStubChannelService{
			channels: modelSquareTestChannels(),
			result:   &pagination.PaginationResult{Total: 1, Page: 1, PageSize: 1000, Pages: 1},
		},
		groupService: &modelSquareStubGroupService{groups: modelSquareTestGroups()},
		billingService: &modelSquareStubBillingService{
			prices: map[string]*service.ModelPricing{
				"gpt-5.5":     modelSquareTestPrice(3e-6, 15e-6, 6e-6, 7e-6, 1e-6, 8e-6, 40e-6, 9e-6, 2e-6, 10e-6, 20e-6),
				"ghost-model": modelSquareTestPrice(3e-6, 15e-6, 6e-6, 7e-6, 1e-6, 8e-6, 40e-6, 9e-6, 2e-6, 10e-6, 20e-6),
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/model-square", nil)

	handler.Get(c)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Code int                    `json:"code"`
		Data modelSquareUserResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data.ReferencePrices, 1)
	require.Contains(t, body.Data.ReferencePrices, "ghost-model")
	_, hasComplete := body.Data.ReferencePrices["gpt-5.5"]
	require.False(t, hasComplete)
}

func TestModelSquareHandler_LoadAllChannels_PaginatesUntilComplete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	page1 := []service.Channel{{ID: 1, Status: "active", GroupIDs: []int64{1}}}
	page2 := []service.Channel{{ID: 2, Status: "disabled", GroupIDs: []int64{2}}}
	handler := &ModelSquareHandler{
		configService: &modelSquareStubConfigService{config: service.ModelSquareConfig{Platforms: []service.ModelSquarePlatformConfig{}}},
		channelService: &modelSquarePagingChannelService{
			pages: [][]service.Channel{page1, page2},
			total: 2,
		},
		groupService:    &modelSquareStubGroupService{},
		overrideService: &modelSquareStubOverrideService{},
		billingService:  &modelSquareStubBillingService{},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/model-square", nil)

	handler.Get(c)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Code int                    `json:"code"`
		Data modelSquareUserResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data.Channels, 2)
	require.Equal(t, int64(1), body.Data.Channels[0].ID)
	require.Equal(t, int64(2), body.Data.Channels[1].ID)
}

// modelSquarePagingChannelService 按页码返回渠道分页数据，用于验证全量拉取逻辑。
type modelSquarePagingChannelService struct {
	pages [][]service.Channel
	total int64
}

func (s *modelSquarePagingChannelService) List(ctx context.Context, params pagination.PaginationParams, status, search string) ([]service.Channel, *pagination.PaginationResult, error) {
	index := params.Page - 1
	pages := len(s.pages)
	if index < 0 || index >= pages {
		return nil, &pagination.PaginationResult{Total: s.total, Page: params.Page, PageSize: params.PageSize, Pages: pages}, nil
	}
	return s.pages[index], &pagination.PaginationResult{Total: s.total, Page: params.Page, PageSize: params.PageSize, Pages: pages}, nil
}
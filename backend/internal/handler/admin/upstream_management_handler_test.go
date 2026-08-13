package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type upstreamManagementHandlerServiceStub struct {
	compareCalled                 bool
	applyCalled                   bool
	getConfigCalled               bool
	updateConfigCalled            bool
	saveMappingCalled             bool
	createLocalCalled             bool
	markHandledCalled             bool
	modelSquareCalled             bool
	getModelSquareConfigCalled    bool
	updateModelSquareConfigCalled bool
	mappingInput                  service.UpstreamGroupMappingInput
	configInput                   service.UpstreamGroupAutoRateFixConfig
	createLocalInput              service.UpstreamGroupLocalCreateInput
	handledRecordKey              string
	modelSquareConfigInput        service.ModelSquareConfig
	result                        service.UpstreamGroupCompareResult
	config                        service.UpstreamGroupAutoRateFixConfig
	modelSquareConfig             service.ModelSquareConfig
	modelSquare                   json.RawMessage
	defaultProvider               service.UpstreamProviderConfig
	err                           error
}

func (s *upstreamManagementHandlerServiceStub) CompareGroups(context.Context) (service.UpstreamGroupCompareResult, error) {
	s.compareCalled = true
	return s.result, s.err
}

func (s *upstreamManagementHandlerServiceStub) ApplyRateFixes(context.Context) (service.UpstreamGroupCompareResult, error) {
	s.applyCalled = true
	return s.result, s.err
}

func (s *upstreamManagementHandlerServiceStub) GetRateFixConfig(context.Context) (service.UpstreamGroupAutoRateFixConfig, error) {
	s.getConfigCalled = true
	return s.config, s.err
}

func (s *upstreamManagementHandlerServiceStub) UpdateRateFixConfig(_ context.Context, input service.UpstreamGroupAutoRateFixConfig) (service.UpstreamGroupAutoRateFixConfig, error) {
	s.updateConfigCalled = true
	s.configInput = input
	return input, s.err
}

func (s *upstreamManagementHandlerServiceStub) SaveGroupMapping(_ context.Context, input service.UpstreamGroupMappingInput) (service.UpstreamGroupCompareResult, error) {
	s.saveMappingCalled = true
	s.mappingInput = input
	return s.result, s.err
}

func (s *upstreamManagementHandlerServiceStub) CreateLocalGroupFromUpstream(_ context.Context, input service.UpstreamGroupLocalCreateInput) (service.UpstreamGroupCompareResult, error) {
	s.createLocalCalled = true
	s.createLocalInput = input
	return s.result, s.err
}

func (s *upstreamManagementHandlerServiceStub) MarkRateFixRecordHandled(_ context.Context, key string) ([]service.UpstreamGroupRateFixRecord, error) {
	s.markHandledCalled = true
	s.handledRecordKey = key
	return s.result.Records, s.err
}

func (s *upstreamManagementHandlerServiceStub) GetModelSquareConfig(context.Context) (service.ModelSquareConfig, error) {
	s.getModelSquareConfigCalled = true
	return s.modelSquareConfig, s.err
}

func (s *upstreamManagementHandlerServiceStub) UpdateModelSquareConfig(_ context.Context, input service.ModelSquareConfig) (service.ModelSquareConfig, error) {
	s.updateModelSquareConfigCalled = true
	s.modelSquareConfigInput = input
	return input, s.err
}

func (s *upstreamManagementHandlerServiceStub) FetchDefaultModelSquare(context.Context) (json.RawMessage, service.UpstreamProviderConfig, error) {
	s.modelSquareCalled = true
	return s.modelSquare, s.defaultProvider, s.err
}

func newUpstreamManagementHandlerTestRouter(svc upstreamManagementService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := newUpstreamManagementHandlerWithService(svc)
	router.GET("/admin/upstream-management/groups", handler.CompareGroups)
	router.POST("/admin/upstream-management/groups/rate-fixes", handler.ApplyRateFixes)
	router.POST("/admin/upstream-management/groups/rate-fix-records/:key/handled", handler.MarkRateFixRecordHandled)
	router.GET("/admin/upstream-management/groups/rate-fix-config", handler.GetRateFixConfig)
	router.PUT("/admin/upstream-management/groups/rate-fix-config", handler.UpdateRateFixConfig)
	router.PUT("/admin/upstream-management/groups/mappings", handler.SaveGroupMapping)
	router.POST("/admin/upstream-management/groups/local-groups", handler.CreateLocalGroupFromUpstream)
	router.GET("/admin/upstream-management/model-square/config", handler.GetModelSquareConfig)
	router.PUT("/admin/upstream-management/model-square/config", handler.UpdateModelSquareConfig)
	router.GET("/admin/upstream-management/model-square", handler.ModelSquare)
	router.GET("/model-square", handler.ModelSquare)
	return router
}

func newUpstreamManagementModelSquarePricingTestRouter(svc upstreamManagementService, billingService *service.BillingService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := newUpstreamManagementHandlerWithServices(svc, billingService)
	router.GET("/admin/upstream-management/model-square/model-pricing", handler.GetModelSquareModelPricing)
	return router
}

func TestUpstreamManagementHandlerCompareGroups(t *testing.T) {
	localRate := 1.0
	groupID := int64(9)
	svc := &upstreamManagementHandlerServiceStub{result: service.UpstreamGroupCompareResult{
		DefaultProvider: service.UpstreamProviderConfig{Slug: "default", Name: "Default"},
		Items: []service.UpstreamGroupComparison{{
			ProviderSlug:      "default",
			ProviderName:      "Default",
			UpstreamGroupName: "VIP",
			UpstreamRate:      2,
			UpstreamKeyCount:  3,
			LocalGroupID:      &groupID,
			LocalGroupName:    "vip",
			LocalRate:         &localRate,
			Matched:           true,
			NeedsRateIncrease: true,
		}},
		Records: []service.UpstreamGroupRateFixRecord{},
	}}
	router := newUpstreamManagementHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/groups", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.compareCalled)
	require.Contains(t, rec.Body.String(), `"upstream_group_name":"VIP"`)
}

func TestUpstreamManagementHandlerApplyRateFixes(t *testing.T) {
	svc := &upstreamManagementHandlerServiceStub{result: service.UpstreamGroupCompareResult{
		DefaultProvider: service.UpstreamProviderConfig{Slug: "default", Name: "Default"},
		Items:           []service.UpstreamGroupComparison{},
		Records: []service.UpstreamGroupRateFixRecord{{
			GroupID:           9,
			GroupName:         "vip",
			ProviderSlug:      "default",
			ProviderName:      "Default",
			UpstreamGroupName: "VIP",
			OldRate:           1,
			NewRate:           2,
		}},
	}}
	router := newUpstreamManagementHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/upstream-management/groups/rate-fixes", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.applyCalled)
	require.Contains(t, rec.Body.String(), `"records"`)
	require.Contains(t, rec.Body.String(), `"new_rate":2`)
}

func TestUpstreamManagementHandlerMarkRateFixRecordHandled(t *testing.T) {
	svc := &upstreamManagementHandlerServiceStub{result: service.UpstreamGroupCompareResult{
		Records: []service.UpstreamGroupRateFixRecord{{
			GroupID:           9,
			GroupName:         "vip",
			ProviderSlug:      "default",
			ProviderName:      "Default",
			UpstreamGroupName: "VIP",
			OldRate:           1,
			NewRate:           2,
			Handled:           true,
		}},
	}}
	router := newUpstreamManagementHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/upstream-management/groups/rate-fix-records/2026-06-20T00:00:00Z-9-default-VIP/handled", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.markHandledCalled)
	require.Equal(t, "2026-06-20T00:00:00Z-9-default-VIP", svc.handledRecordKey)
	require.Contains(t, rec.Body.String(), `"handled":true`)
}

func TestUpstreamManagementHandlerGetRateFixConfig(t *testing.T) {
	svc := &upstreamManagementHandlerServiceStub{config: service.UpstreamGroupAutoRateFixConfig{
		Enabled:         true,
		IntervalSeconds: 30,
		LastRunStatus:   "success",
	}}
	router := newUpstreamManagementHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/groups/rate-fix-config", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.getConfigCalled)
	require.Contains(t, rec.Body.String(), `"interval_seconds":30`)
}

func TestUpstreamManagementHandlerUpdateRateFixConfig(t *testing.T) {
	svc := &upstreamManagementHandlerServiceStub{}
	router := newUpstreamManagementHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/admin/upstream-management/groups/rate-fix-config",
		bytes.NewBufferString(`{"enabled":true,"interval_seconds":15}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.updateConfigCalled)
	require.True(t, svc.configInput.Enabled)
	require.Equal(t, 15, svc.configInput.IntervalSeconds)
	require.Contains(t, rec.Body.String(), `"interval_seconds":15`)
}

func TestUpstreamManagementHandlerSaveGroupMapping(t *testing.T) {
	groupID := int64(9)
	svc := &upstreamManagementHandlerServiceStub{result: service.UpstreamGroupCompareResult{
		DefaultProvider: service.UpstreamProviderConfig{Slug: "default", Name: "Default"},
		Items:           []service.UpstreamGroupComparison{},
		Records:         []service.UpstreamGroupRateFixRecord{},
	}}
	router := newUpstreamManagementHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/admin/upstream-management/groups/mappings",
		bytes.NewBufferString(`{"upstream_group_name":"VIP","local_group_id":9}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.saveMappingCalled)
	require.Equal(t, "VIP", svc.mappingInput.UpstreamGroupName)
	require.NotNil(t, svc.mappingInput.LocalGroupID)
	require.Equal(t, groupID, *svc.mappingInput.LocalGroupID)
}

func TestUpstreamManagementHandlerCreateLocalGroupFromUpstream(t *testing.T) {
	groupID := int64(42)
	svc := &upstreamManagementHandlerServiceStub{result: service.UpstreamGroupCompareResult{
		DefaultProvider: service.UpstreamProviderConfig{Slug: "default", Name: "Default"},
		Items: []service.UpstreamGroupComparison{{
			ProviderSlug:      "default",
			ProviderName:      "Default",
			UpstreamGroupName: "VIP",
			UpstreamRate:      2.5,
			LocalGroupID:      &groupID,
			LocalGroupName:    "VIP",
			Matched:           true,
		}},
		Records: []service.UpstreamGroupRateFixRecord{},
	}}
	router := newUpstreamManagementHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/admin/upstream-management/groups/local-groups",
		bytes.NewBufferString(`{"upstream_group_name":"VIP","platform":"gemini","rate_multiplier":2.5}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.createLocalCalled)
	require.Equal(t, "VIP", svc.createLocalInput.UpstreamGroupName)
	require.Equal(t, "gemini", svc.createLocalInput.Platform)
	require.Equal(t, 2.5, svc.createLocalInput.RateMultiplier)
	require.Contains(t, rec.Body.String(), `"local_group_id":42`)
}

func TestUpstreamManagementHandlerGetModelSquareConfig(t *testing.T) {
	svc := &upstreamManagementHandlerServiceStub{modelSquareConfig: service.ModelSquareConfig{
		Platforms: []service.ModelSquarePlatformConfig{{
			Platform: "openai",
			Name:     "OpenAI",
			Models:   []service.ModelSquarePlatformModelConfig{{ID: "gpt-5.5"}},
		}},
	}}
	router := newUpstreamManagementHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/model-square/config", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.getModelSquareConfigCalled)
	require.Contains(t, rec.Body.String(), `"platform":"openai"`)
	require.Contains(t, rec.Body.String(), `"id":"gpt-5.5"`)
}

func TestUpstreamManagementHandlerUpdateModelSquareConfig(t *testing.T) {
	svc := &upstreamManagementHandlerServiceStub{}
	router := newUpstreamManagementHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/admin/upstream-management/model-square/config",
		bytes.NewBufferString(`{"platforms":[{"platform":"openai","name":"OpenAI","models":[{"id":"gpt-5.2","input_price":0.1}]}]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.updateModelSquareConfigCalled)
	require.Len(t, svc.modelSquareConfigInput.Platforms, 1)
	require.Equal(t, "openai", svc.modelSquareConfigInput.Platforms[0].Platform)
	require.Len(t, svc.modelSquareConfigInput.Platforms[0].Models, 1)
	require.Equal(t, "gpt-5.2", svc.modelSquareConfigInput.Platforms[0].Models[0].ID)
	require.NotNil(t, svc.modelSquareConfigInput.Platforms[0].Models[0].InputPrice)
	require.Equal(t, 0.1, *svc.modelSquareConfigInput.Platforms[0].Models[0].InputPrice)
	require.Contains(t, rec.Body.String(), `"input_price":0.1`)
}

func TestUpstreamManagementHandlerModelSquare(t *testing.T) {
	svc := &upstreamManagementHandlerServiceStub{
		defaultProvider: service.UpstreamProviderConfig{Slug: "default", Name: "Default"},
		modelSquare:     json.RawMessage(`{"groups":[],"models":[{"id":"gpt-5.2"}]}`),
	}
	router := newUpstreamManagementHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/model-square", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.modelSquareCalled)
	require.Contains(t, rec.Body.String(), `"provider_slug":"default"`)
	require.Contains(t, rec.Body.String(), `"models":[{"id":"gpt-5.2"}]`)
}

func TestUpstreamManagementHandlerModelSquareUserPath(t *testing.T) {
	svc := &upstreamManagementHandlerServiceStub{
		defaultProvider: service.UpstreamProviderConfig{Slug: "default", Name: "Default"},
		modelSquare:     json.RawMessage(`{"groups":[],"models":[{"id":"gpt-5.2"}]}`),
	}
	router := newUpstreamManagementHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/model-square", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.modelSquareCalled)
	require.Contains(t, rec.Body.String(), `"provider_slug":"default"`)
	require.Contains(t, rec.Body.String(), `"models":[{"id":"gpt-5.2"}]`)
}

func TestUpstreamManagementHandlerGetModelSquareModelPricingIncludesOfficialReferenceFields(t *testing.T) {
	router := newUpstreamManagementModelSquarePricingTestRouter(
		&upstreamManagementHandlerServiceStub{},
		service.NewBillingService(&config.Config{}, nil),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/model-square/model-pricing?model=gpt-5.5", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, true, body.Data["found"])
	require.Equal(t, 2.5e-6, body.Data["input_price"])
	require.Equal(t, 15e-6, body.Data["output_price"])
	require.Equal(t, 5e-6, body.Data["input_price_priority"])
	require.Equal(t, 30e-6, body.Data["output_price_priority"])
	require.Contains(t, body.Data, "cache_write_1h_price")
	require.NotContains(t, body.Data, "per_request_price")
}

func TestUpstreamManagementHandlerGetModelSquareModelPricingMissingModel(t *testing.T) {
	router := newUpstreamManagementModelSquarePricingTestRouter(
		&upstreamManagementHandlerServiceStub{},
		service.NewBillingService(&config.Config{}, nil),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/model-square/model-pricing", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "model 参数不能为空")
}

func TestUpstreamManagementHandlerGetModelSquareModelPricingReturnsNotFound(t *testing.T) {
	router := newUpstreamManagementModelSquarePricingTestRouter(
		&upstreamManagementHandlerServiceStub{},
		service.NewBillingService(&config.Config{}, nil),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/model-square/model-pricing?model=unknown-model", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"found":false`)
}

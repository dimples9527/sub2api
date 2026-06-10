package admin

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type upstreamManagementHandlerServiceStub struct {
	compareCalled     bool
	applyCalled       bool
	saveMappingCalled bool
	mappingInput      service.UpstreamGroupMappingInput
	result            service.UpstreamGroupCompareResult
	err               error
}

func (s *upstreamManagementHandlerServiceStub) CompareGroups(context.Context) (service.UpstreamGroupCompareResult, error) {
	s.compareCalled = true
	return s.result, s.err
}

func (s *upstreamManagementHandlerServiceStub) ApplyRateFixes(context.Context) (service.UpstreamGroupCompareResult, error) {
	s.applyCalled = true
	return s.result, s.err
}

func (s *upstreamManagementHandlerServiceStub) SaveGroupMapping(_ context.Context, input service.UpstreamGroupMappingInput) (service.UpstreamGroupCompareResult, error) {
	s.saveMappingCalled = true
	s.mappingInput = input
	return s.result, s.err
}

func newUpstreamManagementHandlerTestRouter(svc upstreamManagementService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := newUpstreamManagementHandlerWithService(svc)
	router.GET("/admin/upstream-management/groups", handler.CompareGroups)
	router.POST("/admin/upstream-management/groups/rate-fixes", handler.ApplyRateFixes)
	router.PUT("/admin/upstream-management/groups/mappings", handler.SaveGroupMapping)
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

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

type monitorGroupPlatformOverrideHandlerStub struct {
	platforms         map[int64]service.MonitorGroupPlatformOverride
	setGroupID        int64
	setPlatform       string
	visibilityGroupID int64
	showInMonitor     bool
	clearGroupID      int64
}

func (s *monitorGroupPlatformOverrideHandlerStub) ListByGroupIDs(_ context.Context, _ []int64) (map[int64]service.MonitorGroupPlatformOverride, error) {
	return s.platforms, nil
}

func (s *monitorGroupPlatformOverrideHandlerStub) Set(_ context.Context, groupID int64, platform string) error {
	s.setGroupID = groupID
	s.setPlatform = platform
	return nil
}

func (s *monitorGroupPlatformOverrideHandlerStub) SetShowInMonitor(_ context.Context, groupID int64, show bool) error {
	s.visibilityGroupID = groupID
	s.showInMonitor = show
	return nil
}

func (s *monitorGroupPlatformOverrideHandlerStub) Clear(_ context.Context, groupID int64) error {
	s.clearGroupID = groupID
	return nil
}

type adminServiceRejectingMonitorOverrideStub struct {
	stubAdminService
}

func TestGroupHandlerGetLLMMonitorGroupsReturnsMinimalActiveGroupData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewGroupHandlerWithPlatformOverride(&stubAdminService{groups: []service.Group{
		{Name: "default", Platform: service.PlatformAnthropic, RateMultiplier: 1.5},
	}}, nil, nil, &monitorGroupPlatformOverrideHandlerStub{})
	router := gin.New()
	router.GET("/api/llm-monitor/groups", handler.GetLLMMonitorGroups)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/llm-monitor/groups", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"name":"default"`)
	require.Contains(t, rec.Body.String(), `"platform":"anthropic"`)
	require.Contains(t, rec.Body.String(), `"rate_multiplier":1.5`)
}

func TestGroupHandlerGetLLMMonitorGroupsUsesEffectivePlatformOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewGroupHandlerWithPlatformOverride(&stubAdminService{groups: []service.Group{
		{ID: 7, Name: "default", Platform: service.PlatformAnthropic, RateMultiplier: 1.5},
	}}, nil, nil, &monitorGroupPlatformOverrideHandlerStub{platforms: map[int64]service.MonitorGroupPlatformOverride{7: {ActualPlatform: service.PlatformOpenAI, ShowInMonitor: true}}})
	router := gin.New()
	router.GET("/api/llm-monitor/groups", handler.GetLLMMonitorGroups)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/llm-monitor/groups", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"effective_platform":"openai"`)
	require.Contains(t, rec.Body.String(), `"platform":"anthropic"`)
}

func TestGroupHandlerGetLLMMonitorGroupsFiltersHiddenGroups(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewGroupHandlerWithPlatformOverride(&stubAdminService{groups: []service.Group{
		{ID: 7, Name: "hidden", Platform: service.PlatformAnthropic, RateMultiplier: 1.5},
		{ID: 8, Name: "visible", Platform: service.PlatformOpenAI, RateMultiplier: 1},
	}}, nil, nil, &monitorGroupPlatformOverrideHandlerStub{platforms: map[int64]service.MonitorGroupPlatformOverride{
		7: {ActualPlatform: service.PlatformAnthropic, ShowInMonitor: false},
	}})
	router := gin.New()
	router.GET("/api/llm-monitor/groups", handler.GetLLMMonitorGroups)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/llm-monitor/groups", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotContains(t, rec.Body.String(), `"name":"hidden"`)
	require.Contains(t, rec.Body.String(), `"name":"visible"`)
}

func TestGroupHandlerUsesMonitorOverrideServiceForPlatformMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	overrideStub := &monitorGroupPlatformOverrideHandlerStub{}
	handler := NewGroupHandler(&adminServiceRejectingMonitorOverrideStub{}, nil, nil)
	handler.monitorGroupPlatformOverrideService = overrideStub
	router := gin.New()
	router.POST("/api/v1/admin/groups/:group_id/llm-monitor/platform", handler.SetLLMMonitorPlatformOverride)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/groups/7/llm-monitor/platform", bytes.NewBufferString(`{"actual_platform":"glm"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(7), overrideStub.setGroupID)
	require.Equal(t, "glm", overrideStub.setPlatform)
}

func TestGroupHandlerUsesMonitorOverrideServiceForVisibilityMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	overrideStub := &monitorGroupPlatformOverrideHandlerStub{}
	handler := NewGroupHandler(&adminServiceRejectingMonitorOverrideStub{}, nil, nil)
	handler.monitorGroupPlatformOverrideService = overrideStub
	router := gin.New()
	router.PUT("/api/v1/admin/model-monitor/visibility/:group_id", handler.SetLLMMonitorGroupVisibility)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/model-monitor/visibility/7", bytes.NewBufferString(`{"show_in_monitor":false}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(7), overrideStub.visibilityGroupID)
	require.False(t, overrideStub.showInMonitor)
}

func TestGroupHandlerUsesMonitorOverrideServiceForPlatformListing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	overrideStub := &monitorGroupPlatformOverrideHandlerStub{platforms: map[int64]service.MonitorGroupPlatformOverride{7: {ActualPlatform: "glm", ShowInMonitor: true}}}
	handler := NewGroupHandler(&adminServiceRejectingMonitorOverrideStub{stubAdminService: stubAdminService{groups: []service.Group{
		{ID: 7, Name: "default", Platform: service.PlatformAnthropic, RateMultiplier: 1.5},
	}}}, nil, nil)
	handler.monitorGroupPlatformOverrideService = overrideStub
	router := gin.New()
	router.GET("/api/llm-monitor/groups", handler.GetLLMMonitorGroups)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/llm-monitor/groups", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"effective_platform":"glm"`)
}

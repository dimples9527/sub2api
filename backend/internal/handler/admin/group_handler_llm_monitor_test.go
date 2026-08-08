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
	platforms    map[int64]string
	setGroupID   int64
	setPlatform  string
	clearGroupID int64
}

func (s *monitorGroupPlatformOverrideHandlerStub) ListByGroupIDs(_ context.Context, _ []int64) (map[int64]string, error) {
	return s.platforms, nil
}

func (s *monitorGroupPlatformOverrideHandlerStub) Set(_ context.Context, groupID int64, platform string) error {
	s.setGroupID = groupID
	s.setPlatform = platform
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
	}}, nil, nil, &monitorGroupPlatformOverrideHandlerStub{platforms: map[int64]string{7: service.PlatformOpenAI}})
	router := gin.New()
	router.GET("/api/llm-monitor/groups", handler.GetLLMMonitorGroups)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/llm-monitor/groups", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"effective_platform":"openai"`)
	require.Contains(t, rec.Body.String(), `"platform":"anthropic"`)
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

func TestGroupHandlerUsesMonitorOverrideServiceForPlatformListing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	overrideStub := &monitorGroupPlatformOverrideHandlerStub{platforms: map[int64]string{7: "glm"}}
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

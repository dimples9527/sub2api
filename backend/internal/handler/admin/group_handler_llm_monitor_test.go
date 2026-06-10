package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGroupHandlerGetLLMMonitorGroupsReturnsMinimalActiveGroupData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewGroupHandler(&stubAdminService{groups: []service.Group{
		{Name: "default", Platform: service.PlatformAnthropic, RateMultiplier: 1.5},
	}}, nil, nil)
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

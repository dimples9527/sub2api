package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type upstreamDashboardHandlerStub struct {
	rangeValue service.UpstreamDashboardRange
}

func (s *upstreamDashboardHandlerStub) Get(_ context.Context, rangeValue service.UpstreamDashboardRange) (service.UpstreamDashboardResponse, error) {
	s.rangeValue = rangeValue
	return service.UpstreamDashboardResponse{Range: rangeValue, GeneratedAt: time.Date(2026, 7, 13, 8, 0, 0, 0, time.UTC)}, nil
}

func TestUpstreamDashboardHandlerUsesDefaultRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &upstreamDashboardHandlerStub{}
	router := gin.New()
	router.GET("/dashboard", newUpstreamDashboardHandlerWithService(stub).Get)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/dashboard", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, service.UpstreamDashboardRange24Hours, stub.rangeValue)
	require.Contains(t, recorder.Body.String(), `"generated_at":"2026-07-13T08:00:00Z"`)
}

func TestUpstreamDashboardHandlerRejectsInvalidRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &upstreamDashboardHandlerStub{}
	router := gin.New()
	router.GET("/dashboard", newUpstreamDashboardHandlerWithService(stub).Get)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/dashboard?range=30d", nil))

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Empty(t, stub.rangeValue)
}

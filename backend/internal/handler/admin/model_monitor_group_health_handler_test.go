package admin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type modelMonitorGroupHealthHandlerStub struct {
	query  service.ModelMonitorGroupHealthQuery
	result []service.ModelMonitorGroupHealth
	err    error
}

func (s *modelMonitorGroupHealthHandlerStub) Get(_ context.Context, query service.ModelMonitorGroupHealthQuery) ([]service.ModelMonitorGroupHealth, error) {
	s.query = query
	return s.result, s.err
}

func TestModelMonitorGroupHealthHandlerParsesQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &modelMonitorGroupHealthHandlerStub{result: []service.ModelMonitorGroupHealth{{GroupID: 7, GroupName: "primary"}}}
	router := gin.New()
	router.GET("/group-health", newModelMonitorGroupHealthHandlerWithService(stub).Get)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/group-health?range=7D&group_ids=7,2,7&platform=OpenAI", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, service.ModelMonitorGroupHealthRange7D, stub.query.Range)
	require.Equal(t, []int64{7, 2}, stub.query.GroupIDs)
	require.Equal(t, "openai", stub.query.Platform)
	require.Contains(t, recorder.Body.String(), `"group_id":7`)
}

func TestModelMonitorGroupHealthHandlerRejectsInvalidGroupIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &modelMonitorGroupHealthHandlerStub{}
	router := gin.New()
	router.GET("/group-health", newModelMonitorGroupHealthHandlerWithService(stub).Get)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/group-health?group_ids=1,nope", nil)
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Empty(t, stub.query.GroupIDs)
}

func TestModelMonitorGroupHealthHandlerReturnsServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &modelMonitorGroupHealthHandlerStub{err: errors.New("database unavailable")}
	router := gin.New()
	router.GET("/group-health", newModelMonitorGroupHealthHandlerWithService(stub).Get)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/group-health", nil))

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}

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

type upstreamAccountSyncHandlerServiceStub struct {
	previewCalled bool
	syncCalled    bool
	recordsCalled bool
	runNowCalled  bool
	syncRequest   service.UpstreamAccountSyncRequest
	configInput   service.UpstreamAccountRateGuardConfig
	result        service.UpstreamAccountSyncResult
	records       []service.UpstreamAccountSyncRecord
	config        service.UpstreamAccountRateGuardConfig
	pollLogs      []service.UpstreamAccountRateGuardPollLog
	err           error
}

func (s *upstreamAccountSyncHandlerServiceStub) Preview(context.Context) (service.UpstreamAccountSyncResult, error) {
	s.previewCalled = true
	return s.result, s.err
}

func (s *upstreamAccountSyncHandlerServiceStub) Sync(_ context.Context, req service.UpstreamAccountSyncRequest) (service.UpstreamAccountSyncResult, error) {
	s.syncCalled = true
	s.syncRequest = req
	return s.result, s.err
}

func (s *upstreamAccountSyncHandlerServiceStub) ListRecords(context.Context) ([]service.UpstreamAccountSyncRecord, error) {
	s.recordsCalled = true
	return s.records, s.err
}

func (s *upstreamAccountSyncHandlerServiceStub) GetRateGuardConfig(context.Context) (service.UpstreamAccountRateGuardConfig, error) {
	return s.config, s.err
}

func (s *upstreamAccountSyncHandlerServiceStub) UpdateRateGuardConfig(_ context.Context, input service.UpstreamAccountRateGuardConfig) (service.UpstreamAccountRateGuardConfig, error) {
	s.configInput = input
	s.config = input
	return s.config, s.err
}

func (s *upstreamAccountSyncHandlerServiceStub) RunNow(context.Context) (service.UpstreamAccountRateGuardConfig, error) {
	s.runNowCalled = true
	return s.config, s.err
}

func (s *upstreamAccountSyncHandlerServiceStub) ListPollLogs() []service.UpstreamAccountRateGuardPollLog {
	return s.pollLogs
}

func newUpstreamAccountSyncHandlerTestRouter(svc upstreamAccountSyncService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	scheduler, _ := svc.(upstreamAccountRateGuardScheduler)
	handler := newUpstreamAccountSyncHandlerWithDeps(svc, scheduler)
	router.GET("/admin/upstream-management/accounts/sync-preview", handler.Preview)
	router.POST("/admin/upstream-management/accounts/sync", handler.Sync)
	router.GET("/admin/upstream-management/accounts/sync-records", handler.Records)
	router.GET("/admin/upstream-management/accounts/rate-guard-config", handler.GetRateGuardConfig)
	router.PUT("/admin/upstream-management/accounts/rate-guard-config", handler.UpdateRateGuardConfig)
	router.POST("/admin/upstream-management/accounts/rate-guard-runs", handler.RunRateGuardNow)
	router.GET("/admin/upstream-management/accounts/rate-guard-poll-logs", handler.RateGuardPollLogs)
	return router
}

func TestUpstreamAccountSyncHandlerPreview(t *testing.T) {
	svc := &upstreamAccountSyncHandlerServiceStub{result: service.UpstreamAccountSyncResult{
		DefaultProvider: service.UpstreamProviderConfig{Slug: "main", Name: "Main"},
		Summary:         service.UpstreamAccountSyncSummary{UpstreamKeyCount: 2, CreateCount: 1},
		Items:           []service.UpstreamAccountSyncItem{{Action: service.UpstreamAccountSyncActionCreate, UpstreamKeyName: "up-a"}},
		Records:         []service.UpstreamAccountSyncRecord{},
	}}
	router := newUpstreamAccountSyncHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/accounts/sync-preview", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.previewCalled)
	require.Contains(t, rec.Body.String(), `"upstream_key_count":2`)
	require.Contains(t, rec.Body.String(), `"upstream_key_name":"up-a"`)
}

func TestUpstreamAccountSyncHandlerSync(t *testing.T) {
	svc := &upstreamAccountSyncHandlerServiceStub{result: service.UpstreamAccountSyncResult{
		Summary: service.UpstreamAccountSyncSummary{CreateCount: 1, UpdateCount: 1},
		Items:   []service.UpstreamAccountSyncItem{},
		Records: []service.UpstreamAccountSyncRecord{},
	}}
	router := newUpstreamAccountSyncHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/admin/upstream-management/accounts/sync",
		bytes.NewBufferString(`{"create_missing":true,"update_existing":false,"apply_rate_guard":true}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.syncCalled)
	require.True(t, svc.syncRequest.CreateMissing)
	require.False(t, svc.syncRequest.UpdateExisting)
	require.True(t, svc.syncRequest.ApplyRateGuard)
	require.Contains(t, rec.Body.String(), `"create_count":1`)
}

func TestUpstreamAccountSyncHandlerSyncUsesDefaultsForEmptyBody(t *testing.T) {
	svc := &upstreamAccountSyncHandlerServiceStub{result: service.UpstreamAccountSyncResult{
		Summary: service.UpstreamAccountSyncSummary{},
		Items:   []service.UpstreamAccountSyncItem{},
		Records: []service.UpstreamAccountSyncRecord{},
	}}
	router := newUpstreamAccountSyncHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/upstream-management/accounts/sync", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.syncCalled)
	require.True(t, svc.syncRequest.CreateMissing)
	require.True(t, svc.syncRequest.UpdateExisting)
	require.True(t, svc.syncRequest.ApplyRateGuard)
}

func TestUpstreamAccountSyncHandlerRecords(t *testing.T) {
	svc := &upstreamAccountSyncHandlerServiceStub{records: []service.UpstreamAccountSyncRecord{{
		ProviderSlug: "main",
		CreatedCount: 1,
	}}}
	router := newUpstreamAccountSyncHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/accounts/sync-records", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.recordsCalled)
	require.Contains(t, rec.Body.String(), `"provider_slug":"main"`)
}

func TestUpstreamAccountSyncHandlerGetRateGuardConfig(t *testing.T) {
	svc := &upstreamAccountSyncHandlerServiceStub{config: service.UpstreamAccountRateGuardConfig{
		Enabled:         true,
		IntervalSeconds: 60,
		LastRunStatus:   "success",
	}}
	router := newUpstreamAccountSyncHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/accounts/rate-guard-config", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"enabled":true`)
	require.Contains(t, rec.Body.String(), `"interval_seconds":60`)
	require.Contains(t, rec.Body.String(), `"last_run_status":"success"`)
}

func TestUpstreamAccountSyncHandlerUpdateRateGuardConfig(t *testing.T) {
	svc := &upstreamAccountSyncHandlerServiceStub{}
	router := newUpstreamAccountSyncHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/admin/upstream-management/accounts/rate-guard-config",
		bytes.NewBufferString(`{"enabled":true,"interval_seconds":120}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.configInput.Enabled)
	require.Equal(t, 120, svc.configInput.IntervalSeconds)
	require.Contains(t, rec.Body.String(), `"interval_seconds":120`)
}

func TestUpstreamAccountSyncHandlerRunRateGuardNow(t *testing.T) {
	svc := &upstreamAccountSyncHandlerServiceStub{config: service.UpstreamAccountRateGuardConfig{
		Enabled:         true,
		IntervalSeconds: 60,
		LastRunStatus:   "success",
	}}
	router := newUpstreamAccountSyncHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/upstream-management/accounts/rate-guard-runs", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.runNowCalled)
	require.Contains(t, rec.Body.String(), `"last_run_status":"success"`)
}

func TestUpstreamAccountSyncHandlerRateGuardPollLogs(t *testing.T) {
	svc := &upstreamAccountSyncHandlerServiceStub{pollLogs: []service.UpstreamAccountRateGuardPollLog{{
		Trigger: "scheduled",
		Status:  "success",
		Message: "executed",
	}}}
	router := newUpstreamAccountSyncHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/accounts/rate-guard-poll-logs", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"trigger":"scheduled"`)
	require.Contains(t, rec.Body.String(), `"status":"success"`)
	require.Contains(t, rec.Body.String(), `"message":"executed"`)
}

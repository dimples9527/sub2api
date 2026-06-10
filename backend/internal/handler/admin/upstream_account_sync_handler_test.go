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
	syncRequest   service.UpstreamAccountSyncRequest
	result        service.UpstreamAccountSyncResult
	records       []service.UpstreamAccountSyncRecord
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

func newUpstreamAccountSyncHandlerTestRouter(svc upstreamAccountSyncService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := newUpstreamAccountSyncHandlerWithService(svc)
	router.GET("/admin/upstream-management/accounts/sync-preview", handler.Preview)
	router.POST("/admin/upstream-management/accounts/sync", handler.Sync)
	router.GET("/admin/upstream-management/accounts/sync-records", handler.Records)
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

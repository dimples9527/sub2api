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
	previewCalled       bool
	syncCalled          bool
	recordsCalled       bool
	markHandledCalled   bool
	runNowCalled        bool
	overviewCalled      bool
	configCalled        bool
	rechargeCalled      bool
	balanceRunNowCalled bool
	syncRequest         service.UpstreamAccountSyncRequest
	configInput         service.UpstreamAccountRateGuardConfig
	balanceConfigInput  service.UpstreamBalanceSamplerConfig
	rechargeInput       service.UpstreamBalanceRechargeInput
	handledRecordKey    string
	result              service.UpstreamAccountSyncResult
	records             []service.UpstreamAccountSyncRecord
	config              service.UpstreamAccountRateGuardConfig
	balanceOverview     service.UpstreamBalanceConsumptionOverview
	balanceConfig       service.UpstreamBalanceSamplerConfig
	recharge            service.UpstreamBalanceRecharge
	pollLogs            []service.UpstreamAccountRateGuardPollLog
	balancePollLogs     []service.UpstreamBalanceSamplerPollLog
	err                 error
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

func (s *upstreamAccountSyncHandlerServiceStub) MarkRecordHandled(_ context.Context, key string) ([]service.UpstreamAccountSyncRecord, error) {
	s.markHandledCalled = true
	s.handledRecordKey = key
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

func (s *upstreamAccountSyncHandlerServiceStub) GetOverview(context.Context, int) (service.UpstreamBalanceConsumptionOverview, error) {
	s.overviewCalled = true
	return s.balanceOverview, s.err
}

func (s *upstreamAccountSyncHandlerServiceStub) GetConfig(context.Context) (service.UpstreamBalanceSamplerConfig, error) {
	return s.balanceConfig, s.err
}

func (s *upstreamAccountSyncHandlerServiceStub) UpdateConfig(_ context.Context, input service.UpstreamBalanceSamplerConfig) (service.UpstreamBalanceSamplerConfig, error) {
	s.configCalled = true
	s.balanceConfigInput = input
	s.balanceConfig = input
	return s.balanceConfig, s.err
}

func (s *upstreamAccountSyncHandlerServiceStub) AddRecharge(_ context.Context, input service.UpstreamBalanceRechargeInput) (service.UpstreamBalanceRecharge, error) {
	s.rechargeCalled = true
	s.rechargeInput = input
	if s.recharge.ProviderSlug == "" {
		s.recharge = service.UpstreamBalanceRecharge{
			ID:           1,
			ProviderSlug: input.ProviderSlug,
			Amount:       input.Amount,
			Note:         input.Note,
			OccurredAt:   input.OccurredAt,
		}
	}
	return s.recharge, s.err
}

type upstreamBalanceSamplerSchedulerStub struct {
	called bool
	config service.UpstreamBalanceSamplerConfig
	logs   []service.UpstreamBalanceSamplerPollLog
	err    error
}

func (s *upstreamBalanceSamplerSchedulerStub) RunNow(context.Context) (service.UpstreamBalanceSamplerConfig, error) {
	s.called = true
	return s.config, s.err
}

func (s *upstreamBalanceSamplerSchedulerStub) ListPollLogs() []service.UpstreamBalanceSamplerPollLog {
	return s.logs
}

func newUpstreamAccountSyncHandlerTestRouter(svc upstreamAccountSyncService) *gin.Engine {
	return newUpstreamAccountSyncHandlerTestRouterWithBalanceScheduler(svc, nil)
}

func newUpstreamAccountSyncHandlerTestRouterWithBalanceScheduler(svc upstreamAccountSyncService, balanceScheduler upstreamBalanceSamplerScheduler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	scheduler, _ := svc.(upstreamAccountRateGuardScheduler)
	balance, _ := svc.(upstreamBalanceConsumptionService)
	handler := newUpstreamAccountSyncHandlerWithAllDeps(svc, scheduler, balance, balanceScheduler)
	router.GET("/admin/upstream-management/accounts/sync-preview", handler.Preview)
	router.POST("/admin/upstream-management/accounts/sync", handler.Sync)
	router.GET("/admin/upstream-management/accounts/sync-records", handler.Records)
	router.POST("/admin/upstream-management/accounts/sync-records/:key/handled", handler.MarkRecordHandled)
	router.GET("/admin/upstream-management/accounts/rate-guard-config", handler.GetRateGuardConfig)
	router.PUT("/admin/upstream-management/accounts/rate-guard-config", handler.UpdateRateGuardConfig)
	router.POST("/admin/upstream-management/accounts/rate-guard-runs", handler.RunRateGuardNow)
	router.GET("/admin/upstream-management/accounts/rate-guard-poll-logs", handler.RateGuardPollLogs)
	router.GET("/admin/upstream-management/accounts/balance-consumption", handler.BalanceConsumptionOverview)
	router.GET("/admin/upstream-management/accounts/balance-consumption/config", handler.GetBalanceSamplerConfig)
	router.PUT("/admin/upstream-management/accounts/balance-consumption/config", handler.UpdateBalanceSamplerConfig)
	router.POST("/admin/upstream-management/accounts/balance-consumption/recharges", handler.AddBalanceRecharge)
	router.POST("/admin/upstream-management/accounts/balance-consumption/samples", handler.RunBalanceSampleNow)
	router.GET("/admin/upstream-management/accounts/balance-consumption/poll-logs", handler.BalanceSamplerPollLogs)
	router.GET("/admin/upstream-management/providers/balance-consumption", handler.BalanceConsumptionOverview)
	router.GET("/admin/upstream-management/providers/balance-consumption/config", handler.GetBalanceSamplerConfig)
	router.PUT("/admin/upstream-management/providers/balance-consumption/config", handler.UpdateBalanceSamplerConfig)
	router.POST("/admin/upstream-management/providers/balance-consumption/recharges", handler.AddBalanceRecharge)
	router.POST("/admin/upstream-management/providers/balance-consumption/samples", handler.RunBalanceSampleNow)
	router.GET("/admin/upstream-management/providers/balance-consumption/poll-logs", handler.BalanceSamplerPollLogs)
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

func TestUpstreamAccountSyncHandlerMarkRecordHandled(t *testing.T) {
	svc := &upstreamAccountSyncHandlerServiceStub{records: []service.UpstreamAccountSyncRecord{{
		ProviderSlug: "main",
		UnbindDetails: []service.UpstreamAccountSyncUnbindDetail{{
			UpstreamKeyName: "key-a",
			Handled:         true,
		}},
	}}}
	router := newUpstreamAccountSyncHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/upstream-management/accounts/sync-records/2026-06-18T00:00:00Z-10-key-a-8/handled", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.markHandledCalled)
	require.Equal(t, "2026-06-18T00:00:00Z-10-key-a-8", svc.handledRecordKey)
	require.Contains(t, rec.Body.String(), `"handled":true`)
}

func TestUpstreamAccountSyncHandlerGetRateGuardConfig(t *testing.T) {
	svc := &upstreamAccountSyncHandlerServiceStub{config: service.UpstreamAccountRateGuardConfig{
		Enabled:           true,
		IntervalSeconds:   60,
		IgnoredAccountIDs: []int64{12},
		LastRunStatus:     "success",
	}}
	router := newUpstreamAccountSyncHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/accounts/rate-guard-config", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"enabled":true`)
	require.Contains(t, rec.Body.String(), `"interval_seconds":60`)
	require.Contains(t, rec.Body.String(), `"ignored_account_ids":[12]`)
	require.Contains(t, rec.Body.String(), `"last_run_status":"success"`)
}

func TestUpstreamAccountSyncHandlerUpdateRateGuardConfig(t *testing.T) {
	svc := &upstreamAccountSyncHandlerServiceStub{}
	router := newUpstreamAccountSyncHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/admin/upstream-management/accounts/rate-guard-config",
		bytes.NewBufferString(`{"enabled":true,"interval_seconds":120,"ignored_account_ids":[12,34]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.configInput.Enabled)
	require.Equal(t, 120, svc.configInput.IntervalSeconds)
	require.Equal(t, []int64{12, 34}, svc.configInput.IgnoredAccountIDs)
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

func TestUpstreamAccountSyncHandlerBalanceConsumptionOverview(t *testing.T) {
	svc := &upstreamAccountSyncHandlerServiceStub{balanceOverview: service.UpstreamBalanceConsumptionOverview{
		Config: service.UpstreamBalanceSamplerConfig{Enabled: true, IntervalSeconds: 600},
		Summaries: map[string]service.UpstreamBalanceProviderSummary{
			"backup": {
				ProviderSlug:     "backup",
				ProviderName:     "Backup",
				CurrentBalance:   80,
				TodayConsumption: 70,
				Complete:         true,
			},
		},
		Rows: []service.UpstreamBalanceDailyRow{{
			ProviderSlug:      "backup",
			Date:              "2026-06-15",
			OpeningBalance:    100,
			ClosingBalance:    80,
			RechargeAmount:    50,
			ConsumptionAmount: 70,
			Complete:          true,
		}},
		Snapshots: []service.UpstreamBalanceSnapshot{{
			ProviderSlug: "backup",
			Balance:      100,
			Status:       "success",
		}},
	}}
	router := newUpstreamAccountSyncHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/accounts/balance-consumption?days=7", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.overviewCalled)
	require.Contains(t, rec.Body.String(), `"provider_slug":"backup"`)
	require.Contains(t, rec.Body.String(), `"today_consumption":70`)
	require.Contains(t, rec.Body.String(), `"interval_seconds":600`)
	require.Contains(t, rec.Body.String(), `"snapshots"`)
}

func TestUpstreamAccountSyncHandlerProviderBalanceConsumptionOverviewAlias(t *testing.T) {
	svc := &upstreamAccountSyncHandlerServiceStub{balanceOverview: service.UpstreamBalanceConsumptionOverview{
		Config: service.UpstreamBalanceSamplerConfig{Enabled: true, IntervalSeconds: 600},
		Summaries: map[string]service.UpstreamBalanceProviderSummary{
			"backup": {
				ProviderSlug:     "backup",
				ProviderName:     "Backup",
				TodayConsumption: 70,
			},
		},
	}}
	router := newUpstreamAccountSyncHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/providers/balance-consumption?days=7", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.overviewCalled)
	require.Contains(t, rec.Body.String(), `"provider_slug":"backup"`)
	require.Contains(t, rec.Body.String(), `"today_consumption":70`)
}

func TestUpstreamAccountSyncHandlerUpdateBalanceSamplerConfig(t *testing.T) {
	svc := &upstreamAccountSyncHandlerServiceStub{}
	router := newUpstreamAccountSyncHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/admin/upstream-management/accounts/balance-consumption/config",
		bytes.NewBufferString(`{"enabled":true,"interval_seconds":900}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.configCalled)
	require.True(t, svc.balanceConfigInput.Enabled)
	require.Equal(t, 900, svc.balanceConfigInput.IntervalSeconds)
	require.Contains(t, rec.Body.String(), `"interval_seconds":900`)
}

func TestUpstreamAccountSyncHandlerAddBalanceRecharge(t *testing.T) {
	svc := &upstreamAccountSyncHandlerServiceStub{}
	router := newUpstreamAccountSyncHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/admin/upstream-management/accounts/balance-consumption/recharges",
		bytes.NewBufferString(`{"provider_slug":"backup","amount":50,"occurred_at":"2026-06-15T12:00:00Z","note":"manual top-up"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, svc.rechargeCalled)
	require.Equal(t, "backup", svc.rechargeInput.ProviderSlug)
	require.Equal(t, 50.0, svc.rechargeInput.Amount)
	require.Equal(t, "manual top-up", svc.rechargeInput.Note)
	require.Contains(t, rec.Body.String(), `"provider_slug":"backup"`)
	require.Contains(t, rec.Body.String(), `"amount":50`)
}

func TestUpstreamAccountSyncHandlerRunBalanceSampleNow(t *testing.T) {
	svc := &upstreamAccountSyncHandlerServiceStub{}
	balanceScheduler := &upstreamBalanceSamplerSchedulerStub{
		config: service.UpstreamBalanceSamplerConfig{
			Enabled:         true,
			IntervalSeconds: 600,
			LastRunStatus:   "success",
		},
	}
	router := newUpstreamAccountSyncHandlerTestRouterWithBalanceScheduler(svc, balanceScheduler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/upstream-management/accounts/balance-consumption/samples", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, balanceScheduler.called)
	require.Contains(t, rec.Body.String(), `"last_run_status":"success"`)
}

func TestUpstreamAccountSyncHandlerBalanceSamplerPollLogs(t *testing.T) {
	svc := &upstreamAccountSyncHandlerServiceStub{}
	balanceScheduler := &upstreamBalanceSamplerSchedulerStub{logs: []service.UpstreamBalanceSamplerPollLog{{
		Trigger: "manual",
		Status:  "success",
		Message: "executed",
	}}}
	router := newUpstreamAccountSyncHandlerTestRouterWithBalanceScheduler(svc, balanceScheduler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/accounts/balance-consumption/poll-logs", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"trigger":"manual"`)
	require.Contains(t, rec.Body.String(), `"status":"success"`)
}

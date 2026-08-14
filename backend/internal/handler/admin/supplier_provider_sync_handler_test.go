package admin

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type failingSupplierSyncProgressWriter struct {
	mu       sync.Mutex
	writes   int
	writeErr error
}

func (w *failingSupplierSyncProgressWriter) Write([]byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writes++
	return 0, w.writeErr
}

func (*failingSupplierSyncProgressWriter) Flush() {}

func (w *failingSupplierSyncProgressWriter) writeCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writes
}

func TestSupplierSyncProgressStreamWriterCancelsOnWriteFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writer := &failingSupplierSyncProgressWriter{writeErr: errors.New("client disconnected")}
	stream := newSupplierSyncProgressStream(writer, writer, ctx.Done(), cancel)

	stream.write(service.SupplierSyncProgressEvent{Stage: service.SupplierSyncProgressStagePrepare, Message: "prepare"})
	stream.write(service.SupplierSyncProgressEvent{Stage: service.SupplierSyncProgressStageDone, Message: "done"})

	require.ErrorIs(t, ctx.Err(), context.Canceled)
	require.Equal(t, 1, writer.writeCount())
}

type supplierProviderSyncHandlerSyncStub struct {
	calledScope          string
	refreshProviderID    int64
	refreshToken         service.SupplierProviderAuthToken
	testScope            string
	streamError          error
	streamFailureMessage string
	endpointError        error
	endpointResult       service.SupplierProviderEndpointTestResult
	emitStream           bool
}

func (s *supplierProviderSyncHandlerSyncStub) SyncAccounts(context.Context, int64, string) (service.SupplierProviderSyncResult, error) {
	s.calledScope = service.SupplierSyncScopeAccounts
	if s.streamError != nil {
		return service.SupplierProviderSyncResult{Scope: service.SupplierSyncScopeAccounts, Status: service.SupplierSyncStatusFailed, Message: s.streamFailureMessage}, s.streamError
	}
	return supplierProviderSyncHandlerResult(service.SupplierSyncScopeAccounts), nil
}
func (s *supplierProviderSyncHandlerSyncStub) SyncGroups(context.Context, int64, string) (service.SupplierProviderSyncResult, error) {
	s.calledScope = service.SupplierSyncScopeGroups
	return supplierProviderSyncHandlerResult(service.SupplierSyncScopeGroups), nil
}
func (s *supplierProviderSyncHandlerSyncStub) SyncBalance(context.Context, int64, string) (service.SupplierProviderSyncResult, error) {
	s.calledScope = service.SupplierSyncScopeBalance
	return supplierProviderSyncHandlerResult(service.SupplierSyncScopeBalance), nil
}
func (s *supplierProviderSyncHandlerSyncStub) SyncCost(context.Context, int64, time.Time, string) (service.SupplierProviderSyncResult, error) {
	s.calledScope = service.SupplierSyncScopeCost
	return supplierProviderSyncHandlerResult(service.SupplierSyncScopeCost), nil
}

func (s *supplierProviderSyncHandlerSyncStub) BackfillCosts(_ context.Context, startDate, endDate string, providerID int64, _ string) (service.SupplierProviderCostBackfillResult, error) {
	s.calledScope = service.SupplierSyncScopeCost
	return service.SupplierProviderCostBackfillResult{
		StartDate:     startDate,
		EndDate:       endDate,
		ProviderID:    providerID,
		ProviderCount: 1,
		DayCount:      1,
		SuccessCount:  1,
		Items: []service.SupplierProviderCostBackfillItem{{
			ProviderID: providerID,
			Date:       startDate,
			Status:     service.SupplierSyncStatusSuccess,
			Cost:       1.23,
		}},
	}, nil
}
func (s *supplierProviderSyncHandlerSyncStub) SyncAll(ctx context.Context, _ int64, _ string) (service.SupplierProviderSyncResult, error) {
	s.calledScope = service.SupplierSyncScopeAll
	if s.emitStream {
		service.SupplierSyncProgress(ctx, service.SupplierSyncProgressStagePrepare, "准备同步", nil)
	}
	if s.streamError != nil {
		return service.SupplierProviderSyncResult{}, s.streamError
	}
	return supplierProviderSyncHandlerResult(service.SupplierSyncScopeAll), nil
}
func (s *supplierProviderSyncHandlerSyncStub) RefreshToken(_ context.Context, providerID int64) (service.SupplierProviderAuthToken, error) {
	s.refreshProviderID = providerID
	if s.refreshToken.AccessToken != "" || s.refreshToken.CookieHeader != "" || !s.refreshToken.ExpiresAt.IsZero() {
		return s.refreshToken, nil
	}
	return service.SupplierProviderAuthToken{ExpiresAt: time.Date(2026, time.August, 8, 0, 0, 0, 0, time.UTC)}, nil
}
func (s *supplierProviderSyncHandlerSyncStub) TestEndpoint(_ context.Context, _ int64, scope string) (service.SupplierProviderEndpointTestResult, error) {
	s.testScope = scope
	if s.endpointError != nil {
		result := s.endpointResult
		if result.Scope == "" {
			result.Scope = scope
		}
		return result, s.endpointError
	}
	return service.SupplierProviderEndpointTestResult{Scope: scope, Endpoint: "/test/" + scope, HTTPStatus: 200, ResponseSummary: `{"code":0}`}, nil
}

func supplierProviderSyncHandlerResult(scope string) service.SupplierProviderSyncResult {
	now := time.Now()
	return service.SupplierProviderSyncResult{ProviderID: 42, Scope: scope, Status: service.SupplierSyncStatusSuccess, StartedAt: now, FinishedAt: now}
}

type supplierProviderSyncHandlerDataStub struct {
	mappedGroupID           int64
	mappedLocalGroupID      *int64
	accountListParams       service.SupplierProviderDataListParams
	groupListParams         service.SupplierProviderDataListParams
	monitorTargetListParams service.SupplierProviderMonitorTargetListParams
	monitorTargets          service.SupplierProviderMonitorTargetListResult
	boundMonitorTargetID    int64
	boundLocalAccountID     int64
	unboundMonitorTargetID  int64
	healthTrendParams       service.SupplierProviderGroupHealthTrendParams
	healthTrends            []service.SupplierProviderGroupHealthTrend
	mappingLocalGroupIDs    []int64
	mappings                []service.SupplierProviderGroup
	uniqueLocalAccount      bool
	effectivePlatform       string
	platformOverrideAccount int64
	platformOverride        string
	clearedOverrideAccount  int64
	deletedGroupID          int64
	deleteGroupErr          error
	deletedAccountID        int64
	deleteAccountErr        error
}

type supplierCustomPlatformResolverStub struct {
	platform   *service.CustomPlatform
	err        error
	calledCode string
}

func (s *supplierCustomPlatformResolverStub) ResolveEnabled(_ context.Context, code string) (*service.CustomPlatform, error) {
	s.calledCode = code
	return s.platform, s.err
}

type supplierGroupPlatformOverrideHandlerStub struct {
	setGroupID     int64
	setPlatform    string
	clearedGroupID int64
	listGroupIDs   []int64
	listResult     map[int64]service.MonitorGroupPlatformOverride
	err            error
}

func (s *supplierGroupPlatformOverrideHandlerStub) ListByGroupIDs(_ context.Context, groupIDs []int64) (map[int64]service.MonitorGroupPlatformOverride, error) {
	s.listGroupIDs = append([]int64(nil), groupIDs...)
	if s.listResult != nil {
		return s.listResult, s.err
	}
	return map[int64]service.MonitorGroupPlatformOverride{}, s.err
}

func (s *supplierGroupPlatformOverrideHandlerStub) Set(_ context.Context, groupID int64, platform string) error {
	s.setGroupID = groupID
	s.setPlatform = platform
	return s.err
}

func (s *supplierGroupPlatformOverrideHandlerStub) Clear(_ context.Context, groupID int64) error {
	s.clearedGroupID = groupID
	return s.err
}

func (s *supplierGroupPlatformOverrideHandlerStub) SetShowInMonitor(context.Context, int64, bool) error {
	return s.err
}

type supplierProviderGroupMatcherHandlerStub struct {
	autoProviderID  int64
	ignoredGroupID  int64
	ignored         bool
	resolvedGroupID int64
	resolvedAction  string
}

type supplierGroupGuardHandlerStub struct {
	groupID  int64
	selected bool
	ignored  bool
	err      error
}

func (s *supplierGroupGuardHandlerStub) SetManualGuard(_ context.Context, groupID int64, selected bool) error {
	s.groupID = groupID
	s.selected = selected
	return s.err
}

func (s *supplierGroupGuardHandlerStub) SetRateGuardIgnored(_ context.Context, groupID int64, ignored bool) error {
	s.groupID = groupID
	s.ignored = ignored
	return s.err
}

func (s *supplierProviderGroupMatcherHandlerStub) AutoMatch(_ context.Context, providerID int64) (service.SupplierGroupAutoMatchResult, error) {
	s.autoProviderID = providerID
	return service.SupplierGroupAutoMatchResult{ProviderID: providerID, Scanned: 3, AutoMatched: 1, Ambiguous: 1}, nil
}

func (*supplierProviderGroupMatcherHandlerStub) UpdateMapping(context.Context, int64, *int64) error {
	return nil
}

func (s *supplierProviderGroupMatcherHandlerStub) SetIgnored(_ context.Context, groupID int64, ignored bool) (service.SupplierGroupAutoMatchResult, error) {
	s.ignoredGroupID = groupID
	s.ignored = ignored
	return service.SupplierGroupAutoMatchResult{}, nil
}

func (s *supplierProviderGroupMatcherHandlerStub) ResolveNameChange(_ context.Context, groupID int64, action string) error {
	s.resolvedGroupID = groupID
	s.resolvedAction = action
	return nil
}

func (s *supplierProviderSyncHandlerDataStub) ListAccounts(_ context.Context, params service.SupplierProviderDataListParams) (service.SupplierProviderAccountListResult, error) {
	s.accountListParams = params
	return service.SupplierProviderAccountListResult{Items: []service.SupplierProviderAccount{{ID: 1, ProviderID: 42, Name: "Primary"}}, Total: 1, Page: 1, PageSize: 20}, nil
}
func (s *supplierProviderSyncHandlerDataStub) ListGroups(_ context.Context, params service.SupplierProviderDataListParams) (service.SupplierProviderGroupListResult, error) {
	s.groupListParams = params
	return service.SupplierProviderGroupListResult{Items: []service.SupplierProviderGroup{{ID: 1, ProviderID: 42, Name: "VIP"}}, Total: 1, Page: 1, PageSize: 20}, nil
}
func (s *supplierProviderSyncHandlerDataStub) ListMonitorTargets(_ context.Context, params service.SupplierProviderMonitorTargetListParams) (service.SupplierProviderMonitorTargetListResult, error) {
	s.monitorTargetListParams = params
	if s.monitorTargets.Page == 0 {
		return service.SupplierProviderMonitorTargetListResult{Items: []service.SupplierProviderMonitorTarget{{ID: 9, ProviderID: 42, MonitorKey: "2", MonitorName: "Plus-稳定", LocalAccountID: 7, LocalAccountName: "皓悦-福利-Codex高并发"}}, Total: 1, Page: 1, PageSize: 20}, nil
	}
	return s.monitorTargets, nil
}
func (s *supplierProviderSyncHandlerDataStub) BindMonitorTarget(_ context.Context, monitorTargetID, localAccountID int64) error {
	s.boundMonitorTargetID = monitorTargetID
	s.boundLocalAccountID = localAccountID
	return nil
}
func (s *supplierProviderSyncHandlerDataStub) UnbindMonitorTarget(_ context.Context, monitorTargetID int64) error {
	s.unboundMonitorTargetID = monitorTargetID
	return nil
}
func (s *supplierProviderSyncHandlerDataStub) ListGroupHealthTrends(_ context.Context, params service.SupplierProviderGroupHealthTrendParams) ([]service.SupplierProviderGroupHealthTrend, error) {
	s.healthTrendParams = params
	return s.healthTrends, nil
}
func (s *supplierProviderSyncHandlerDataStub) ListLocalGroupHealthTrends(_ context.Context, params service.SupplierProviderGroupHealthTrendParams) ([]service.SupplierProviderGroupHealthTrend, error) {
	s.healthTrendParams = params
	return s.healthTrends, nil
}
func (s *supplierProviderSyncHandlerDataStub) ListMappingsByLocalGroup(_ context.Context, localGroupIDs []int64) ([]service.SupplierProviderGroup, error) {
	s.mappingLocalGroupIDs = append([]int64(nil), localGroupIDs...)
	return s.mappings, nil
}
func (s *supplierProviderSyncHandlerDataStub) IsUniqueMatchedLocalAccount(context.Context, int64) (bool, error) {
	return s.uniqueLocalAccount, nil
}

func (s *supplierProviderSyncHandlerDataStub) GetLocalAccountEffectivePlatform(context.Context, int64) (string, error) {
	return s.effectivePlatform, nil
}

func (s *supplierProviderSyncHandlerDataStub) SetLocalAccountPlatformOverride(_ context.Context, localAccountID int64, platform string) error {
	s.platformOverrideAccount = localAccountID
	s.platformOverride = platform
	return nil
}

func (s *supplierProviderSyncHandlerDataStub) ClearLocalAccountPlatformOverride(_ context.Context, localAccountID int64) error {
	s.clearedOverrideAccount = localAccountID
	return nil
}

func (s *supplierProviderSyncHandlerDataStub) UpdateGroupMapping(_ context.Context, groupID int64, localGroupID *int64) error {
	s.mappedGroupID = groupID
	s.mappedLocalGroupID = localGroupID
	return nil
}

func (s *supplierProviderSyncHandlerDataStub) DeleteGroup(_ context.Context, groupID int64) error {
	s.deletedGroupID = groupID
	return s.deleteGroupErr
}

func (s *supplierProviderSyncHandlerDataStub) DeleteAccount(_ context.Context, accountID int64) error {
	s.deletedAccountID = accountID
	return s.deleteAccountErr
}

func TestSupplierProviderSyncHandlerRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	syncStub := &supplierProviderSyncHandlerSyncStub{}
	dataStub := &supplierProviderSyncHandlerDataStub{}
	handler := NewSupplierProviderSyncHandler(syncStub, dataStub)
	router := gin.New()
	router.POST("/providers/:id/sync/all", handler.SyncAll)
	router.POST("/providers/:id/test/:scope", handler.TestEndpoint)
	router.GET("/accounts", handler.ListAccounts)
	router.GET("/monitor-targets", handler.ListMonitorTargets)
	router.PUT("/monitor-targets/:id/binding", handler.BindMonitorTarget)
	router.DELETE("/monitor-targets/:id/binding", handler.UnbindMonitorTarget)
	router.PUT("/groups/:id/mapping", handler.UpdateGroupMapping)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/providers/42/sync/all", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.SupplierSyncScopeAll, syncStub.calledScope)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/providers/42/test/balance", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.SupplierSyncScopeBalance, syncStub.testScope)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/accounts?provider_id=42&active=true&platform=openai&sort_by=supplier_today_cost&sort_order=desc&page=1&page_size=20", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "openai", dataStub.accountListParams.Platform)
	require.Equal(t, "supplier_today_cost", dataStub.accountListParams.SortBy)
	require.Equal(t, "desc", dataStub.accountListParams.SortOrder)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/monitor-targets?provider_id=42&active=true&search=Plus&page=2&page_size=20", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), dataStub.monitorTargetListParams.ProviderID)
	require.NotNil(t, dataStub.monitorTargetListParams.Active)
	require.True(t, *dataStub.monitorTargetListParams.Active)
	require.Equal(t, "Plus", dataStub.monitorTargetListParams.Search)
	require.Equal(t, 2, dataStub.monitorTargetListParams.Page)
	require.Equal(t, 20, dataStub.monitorTargetListParams.PageSize)
	require.Contains(t, rec.Body.String(), `"monitor_name":"Plus-稳定"`)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/monitor-targets/9/binding", bytes.NewBufferString(`{"local_account_id":7}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(9), dataStub.boundMonitorTargetID)
	require.Equal(t, int64(7), dataStub.boundLocalAccountID)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/monitor-targets/9/binding", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(9), dataStub.unboundMonitorTargetID)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/groups?provider_id=42&active=true&key_status=created&page=1&page_size=20", nil)
	router.GET("/groups", handler.ListGroups)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), dataStub.groupListParams.ProviderID)
	require.NotNil(t, dataStub.groupListParams.Active)
	require.True(t, *dataStub.groupListParams.Active)
	require.Equal(t, "created", dataStub.groupListParams.KeyStatus)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/groups/7/mapping", bytes.NewBufferString(`{"local_group_id":12}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(7), dataStub.mappedGroupID)
	require.NotNil(t, dataStub.mappedLocalGroupID)
	require.Equal(t, int64(12), *dataStub.mappedLocalGroupID)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/groups/7/mapping", bytes.NewBufferString(`{"local_group_id":null}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(7), dataStub.mappedGroupID)
	require.Nil(t, dataStub.mappedLocalGroupID)
}

func TestSupplierProviderSyncHandlerSetsLocalGroupPlatformOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)
	localGroupID := int64(12)
	dataStub := &supplierProviderSyncHandlerDataStub{mappings: []service.SupplierProviderGroup{{ID: 7, LocalGroupID: &localGroupID}}}
	overrideStub := &supplierGroupPlatformOverrideHandlerStub{}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)
	handler.SetGroupPlatformOverrideService(overrideStub)
	router := gin.New()
	router.PUT("/local-groups/:id/platform-override", handler.SetLocalGroupPlatformOverride)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/local-groups/12/platform-override", bytes.NewBufferString(`{"platform":"Grok"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []int64{12}, dataStub.mappingLocalGroupIDs)
	require.Equal(t, int64(12), overrideStub.setGroupID)
	require.Equal(t, "grok", overrideStub.setPlatform)
	require.Contains(t, rec.Body.String(), `"local_group_id":12`)
	require.Contains(t, rec.Body.String(), `"platform_override":"grok"`)
}

func TestSupplierProviderSyncHandlerClearsLocalGroupPlatformOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)
	localGroupID := int64(12)
	dataStub := &supplierProviderSyncHandlerDataStub{mappings: []service.SupplierProviderGroup{{ID: 7, LocalGroupID: &localGroupID}}}
	overrideStub := &supplierGroupPlatformOverrideHandlerStub{}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)
	handler.SetGroupPlatformOverrideService(overrideStub)
	router := gin.New()
	router.DELETE("/local-groups/:id/platform-override", handler.ClearLocalGroupPlatformOverride)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/local-groups/12/platform-override", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []int64{12}, dataStub.mappingLocalGroupIDs)
	require.Equal(t, int64(12), overrideStub.clearedGroupID)
	require.Contains(t, rec.Body.String(), `"local_group_id":12`)
}

func TestSupplierProviderSyncHandlerRejectsUnmappedLocalGroupPlatformOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataStub := &supplierProviderSyncHandlerDataStub{}
	overrideStub := &supplierGroupPlatformOverrideHandlerStub{}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)
	handler.SetGroupPlatformOverrideService(overrideStub)
	router := gin.New()
	router.PUT("/local-groups/:id/platform-override", handler.SetLocalGroupPlatformOverride)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/local-groups/12/platform-override", bytes.NewBufferString(`{"platform":"openai"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, []int64{12}, dataStub.mappingLocalGroupIDs)
	require.Zero(t, overrideStub.setGroupID)
}

func TestSupplierProviderSyncHandlerStreamsProgress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	syncStub := &supplierProviderSyncHandlerSyncStub{emitStream: true}
	handler := NewSupplierProviderSyncHandler(syncStub, &supplierProviderSyncHandlerDataStub{})
	router := gin.New()
	router.POST("/providers/:id/sync/all/stream", handler.SyncAllStream)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/providers/42/sync/all/stream", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "text/event-stream")
	require.Contains(t, rec.Body.String(), `"stage":"prepare"`)
	require.Contains(t, rec.Body.String(), `"stage":"done"`)
}

func TestSupplierProviderSyncHandlerStreamsErrorEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	syncStub := &supplierProviderSyncHandlerSyncStub{streamError: errors.New("上游登录失败")}
	handler := NewSupplierProviderSyncHandler(syncStub, &supplierProviderSyncHandlerDataStub{})
	router := gin.New()
	router.POST("/providers/:id/sync/all/stream", handler.SyncAllStream)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/providers/42/sync/all/stream", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"stage":"error"`)
	require.Contains(t, rec.Body.String(), "上游登录失败")
}

func TestSupplierProviderSyncHandlerUsesFriendlyMessageForDisabledProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	syncStub := &supplierProviderSyncHandlerSyncStub{
		streamError: infraerrors.BadRequest("SUPPLIER_PROVIDER_DISABLED", "supplier provider is disabled"),
	}
	handler := NewSupplierProviderSyncHandler(syncStub, &supplierProviderSyncHandlerDataStub{})
	router := gin.New()
	router.POST("/providers/:id/sync/all/stream", handler.SyncAllStream)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/providers/42/sync/all/stream", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "供应商已停用，请先启用后再同步")
	require.NotContains(t, rec.Body.String(), "error: code=400 reason=\"SUPPLIER_PROVIDER_DISABLED\"")
}

func TestSupplierProviderSyncHandlerStreamsResultMessageWhenSyncReturnsError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	message := "供应商登录失败，已自动停用。请处理后手动重新启用。"
	syncStub := &supplierProviderSyncHandlerSyncStub{
		streamError:          errors.New("upstream login failed"),
		streamFailureMessage: message,
	}
	handler := NewSupplierProviderSyncHandler(syncStub, &supplierProviderSyncHandlerDataStub{})
	router := gin.New()
	router.POST("/providers/:id/sync/accounts/stream", handler.SyncAccountsStream)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/providers/42/sync/accounts/stream", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), message)
	require.NotContains(t, rec.Body.String(), "upstream login failed")
}

func TestSupplierProviderSyncHandlerReturnsResultMessageWhenSyncReturnsError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	message := "供应商登录失败，已自动停用。请处理后手动重新启用。"
	syncStub := &supplierProviderSyncHandlerSyncStub{
		streamError:          errors.New("upstream login failed"),
		streamFailureMessage: message,
	}
	handler := NewSupplierProviderSyncHandler(syncStub, &supplierProviderSyncHandlerDataStub{})
	router := gin.New()
	router.POST("/providers/:id/sync/accounts", handler.SyncAccounts)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/providers/42/sync/accounts", nil)
	router.ServeHTTP(rec, req)

	require.NotEqual(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), message)
	require.NotContains(t, rec.Body.String(), "upstream login failed")
}

func TestSupplierProviderSyncHandlerReturnsEndpointResultMessageWhenTestReturnsError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	message := "供应商登录失败，已自动停用。请处理后手动重新启用。"
	syncStub := &supplierProviderSyncHandlerSyncStub{
		endpointError: errors.New("upstream login failed"),
		endpointResult: service.SupplierProviderEndpointTestResult{
			Error: message,
		},
	}
	handler := NewSupplierProviderSyncHandler(syncStub, &supplierProviderSyncHandlerDataStub{})
	router := gin.New()
	router.POST("/providers/:id/test/:scope", handler.TestEndpoint)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/providers/42/test/balance", nil)
	router.ServeHTTP(rec, req)

	require.NotEqual(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), message)
	require.NotContains(t, rec.Body.String(), "upstream login failed")
}

func TestSupplierProviderSyncHandlerDeletesGroupRecord(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataStub := &supplierProviderSyncHandlerDataStub{}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)
	router := gin.New()
	router.DELETE("/groups/:id", handler.DeleteGroup)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/groups/7", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(7), dataStub.deletedGroupID)
	require.Contains(t, rec.Body.String(), `"group_id":7`)
}

func TestSupplierProviderSyncHandlerRejectsInvalidDeleteGroupID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataStub := &supplierProviderSyncHandlerDataStub{}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)
	router := gin.New()
	router.DELETE("/groups/:id", handler.DeleteGroup)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/groups/not-a-number", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Zero(t, dataStub.deletedGroupID)
}

func TestSupplierProviderSyncHandlerReturnsDeleteGroupConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataStub := &supplierProviderSyncHandlerDataStub{deleteGroupErr: service.ErrSupplierProviderGroupDeleteConflict}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)
	router := gin.New()
	router.DELETE("/groups/:id", handler.DeleteGroup)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/groups/7", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, int64(7), dataStub.deletedGroupID)
}

func TestSupplierProviderSyncHandlerDeletesSupplierAccountRecord(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataStub := &supplierProviderSyncHandlerDataStub{}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)
	router := gin.New()
	router.DELETE("/accounts/:id", handler.DeleteAccount)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/accounts/9", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(9), dataStub.deletedAccountID)
	require.Contains(t, rec.Body.String(), `"account_id":9`)
}

func TestSupplierProviderSyncHandlerRejectsInvalidDeleteSupplierAccountID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataStub := &supplierProviderSyncHandlerDataStub{}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)
	router := gin.New()
	router.DELETE("/accounts/:id", handler.DeleteAccount)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/accounts/not-a-number", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Zero(t, dataStub.deletedAccountID)
}

func TestSupplierProviderSyncHandlerReturnsDeleteSupplierAccountConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataStub := &supplierProviderSyncHandlerDataStub{deleteAccountErr: service.ErrSupplierProviderAccountDeleteConflict}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)
	router := gin.New()
	router.DELETE("/accounts/:id", handler.DeleteAccount)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/accounts/9", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, int64(9), dataStub.deletedAccountID)
}

func TestSupplierProviderSyncHandlerRejectsInvalidProviderID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, &supplierProviderSyncHandlerDataStub{})
	router := gin.New()
	router.POST("/providers/:id/sync/all", handler.SyncAll)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/providers/bad/sync/all", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSupplierProviderSyncHandlerRefreshesTokenWithoutReturningCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	syncStub := &supplierProviderSyncHandlerSyncStub{}
	handler := NewSupplierProviderSyncHandler(syncStub, &supplierProviderSyncHandlerDataStub{})
	router := gin.New()
	router.POST("/providers/:id/refresh-token", handler.RefreshToken)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/providers/42/refresh-token", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), syncStub.refreshProviderID)
	require.Contains(t, rec.Body.String(), `"provider_id":42`)
	require.Contains(t, rec.Body.String(), `"message":"Token 刷新成功"`)
	require.NotContains(t, rec.Body.String(), `"access_token"`)
	require.NotContains(t, rec.Body.String(), `"refresh_token"`)
}

func TestSupplierProviderSyncHandlerReportsCookieSessionReauthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	syncStub := &supplierProviderSyncHandlerSyncStub{refreshToken: service.SupplierProviderAuthToken{CookieHeader: "session=renewed"}}
	handler := NewSupplierProviderSyncHandler(syncStub, &supplierProviderSyncHandlerDataStub{})
	router := gin.New()
	router.POST("/providers/:id/refresh-token", handler.RefreshToken)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/providers/42/refresh-token", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"message":"登录会话已更新"`)
	require.NotContains(t, rec.Body.String(), `"cookie"`)
}

func TestSupplierProviderSyncHandlerRejectsInvalidRefreshTokenProviderID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, &supplierProviderSyncHandlerDataStub{})
	router := gin.New()
	router.POST("/providers/:id/refresh-token", handler.RefreshToken)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/providers/not-a-number/refresh-token", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSupplierProviderSyncHandlerListsGroupHealthTrends(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataStub := &supplierProviderSyncHandlerDataStub{
		healthTrends: []service.SupplierProviderGroupHealthTrend{{
			GroupID:      12,
			Source:       service.SupplierProviderGroupHealthTrendSource,
			Availability: 100,
		}},
	}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)
	router := gin.New()
	router.GET("/groups/health-trends", handler.ListGroupHealthTrends)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/groups/health-trends?group_ids=12,12,37&period=90m", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []int64{12, 37}, dataStub.healthTrendParams.GroupIDs)
	require.Equal(t, 90*time.Minute, dataStub.healthTrendParams.Period)
	require.Equal(t, 18, dataStub.healthTrendParams.BucketCount)
	require.False(t, dataStub.healthTrendParams.Now.IsZero())
	require.Contains(t, rec.Body.String(), `"group_id":12`)
	require.Contains(t, rec.Body.String(), `"source":"supplier_account_health_guard"`)
}

func TestSupplierProviderSyncHandlerListsMappingsByLocalGroup(t *testing.T) {
	dataStub := &supplierProviderSyncHandlerDataStub{
		mappings: []service.SupplierProviderGroup{{ID: 8}},
	}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)

	mappings, err := handler.ListMappingsByLocalGroup(context.Background(), []int64{12, 37})

	require.NoError(t, err)
	require.Equal(t, []int64{12, 37}, dataStub.mappingLocalGroupIDs)
	require.Equal(t, dataStub.mappings, mappings)
}

func TestSupplierProviderSyncHandlerRejectsInvalidGroupHealthTrendQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, &supplierProviderSyncHandlerDataStub{})
	router := gin.New()
	router.GET("/groups/health-trends", handler.ListGroupHealthTrends)

	for _, target := range []string{
		"/groups/health-trends",
		"/groups/health-trends?group_ids=1,broken",
		"/groups/health-trends?group_ids=1&period=1h",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, target, nil)
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code, target)
	}
}
func TestSupplierProviderSyncHandlerListGroupsPassesGroupFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataStub := &supplierProviderSyncHandlerDataStub{}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)
	router := gin.New()
	router.GET("/groups", handler.ListGroups)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/groups?provider_id=42&active=true&search=vip&platform=openai&match_status=manual&rate_status=low&sort_by=rate_multiplier&sort_order=desc&page=2&page_size=20",
		nil,
	)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), dataStub.groupListParams.ProviderID)
	require.Equal(t, "vip", dataStub.groupListParams.Search)
	require.Equal(t, "openai", dataStub.groupListParams.Platform)
	require.Equal(t, "manual", dataStub.groupListParams.MatchStatus)
	require.Equal(t, "low", dataStub.groupListParams.RateStatus)
	require.Equal(t, "rate_multiplier", dataStub.groupListParams.SortBy)
	require.Equal(t, "desc", dataStub.groupListParams.SortOrder)
	require.Equal(t, 2, dataStub.groupListParams.Page)
	require.Equal(t, 20, dataStub.groupListParams.PageSize)
}

func TestSupplierProviderSyncHandlerGroupAutoMatchRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	matcher := &supplierProviderGroupMatcherHandlerStub{}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, &supplierProviderSyncHandlerDataStub{})
	handler.SetGroupMatcher(matcher)
	router := gin.New()
	router.POST("/groups/auto-match", handler.AutoMatchGroups)
	router.PUT("/groups/:id/auto-match-policy", handler.UpdateAutoMatchPolicy)
	router.POST("/groups/:id/name-change/resolve", handler.ResolveGroupNameChange)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/groups/auto-match?provider_id=42", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), matcher.autoProviderID)
	require.Contains(t, rec.Body.String(), `"auto_matched":1`)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/groups/7/auto-match-policy", bytes.NewBufferString(`{"ignored":true}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(7), matcher.ignoredGroupID)
	require.True(t, matcher.ignored)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/groups/7/name-change/resolve", bytes.NewBufferString(`{"action":"keep_local"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(7), matcher.resolvedGroupID)
	require.Equal(t, service.NameChangeActionKeepLocal, matcher.resolvedAction)
}

func TestSupplierProviderSyncHandlerGroupRateGuardSelectsAndClears(t *testing.T) {
	gin.SetMode(gin.TestMode)
	guard := &supplierGroupGuardHandlerStub{}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, &supplierProviderSyncHandlerDataStub{})
	handler.SetGroupGuard(guard)
	router := gin.New()
	router.PUT("/groups/:id/rate-guard", handler.UpdateGroupRateGuard)

	for _, selected := range []bool{true, false} {
		rec := httptest.NewRecorder()
		body := `{"selected":false}`
		if selected {
			body = `{"selected":true}`
		}
		req := httptest.NewRequest(http.MethodPut, "/groups/7/rate-guard", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, int64(7), guard.groupID)
		require.Equal(t, selected, guard.selected)
		require.Contains(t, rec.Body.String(), `"selected":`)
	}
}

func TestSupplierProviderSyncHandlerGroupRateGuardRejectsInvalidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, &supplierProviderSyncHandlerDataStub{})
	handler.SetGroupGuard(&supplierGroupGuardHandlerStub{})
	router := gin.New()
	router.PUT("/groups/:id/rate-guard", handler.UpdateGroupRateGuard)

	for _, request := range []struct {
		path string
		body string
	}{
		{path: "/groups/bad/rate-guard", body: `{"selected":true}`},
		{path: "/groups/7/rate-guard", body: `{}`},
		{path: "/groups/7/rate-guard", body: `{"selected":"yes"}`},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, request.path, bytes.NewBufferString(request.body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	}
}

func TestSupplierProviderSyncHandlerGroupRateGuardReturnsServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, &supplierProviderSyncHandlerDataStub{})
	handler.SetGroupGuard(&supplierGroupGuardHandlerStub{err: errors.New("selection failed")})
	router := gin.New()
	router.PUT("/groups/:id/rate-guard", handler.UpdateGroupRateGuard)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/groups/7/rate-guard", bytes.NewBufferString(`{"selected":true}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestSupplierProviderSyncHandlerGroupRateGuardIgnoreUpdatesPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	guard := &supplierGroupGuardHandlerStub{}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, &supplierProviderSyncHandlerDataStub{})
	handler.SetGroupGuard(guard)
	router := gin.New()
	router.PUT("/groups/:id/rate-guard-ignore", handler.UpdateGroupRateGuardIgnored)

	for _, ignored := range []bool{true, false} {
		rec := httptest.NewRecorder()
		body := `{"ignored":false}`
		if ignored {
			body = `{"ignored":true}`
		}
		req := httptest.NewRequest(http.MethodPut, "/groups/7/rate-guard-ignore", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, int64(7), guard.groupID)
		require.Equal(t, ignored, guard.ignored)
		require.Contains(t, rec.Body.String(), `"ignored":`)
	}
}

func TestSupplierProviderSyncHandlerUpdatesLocalAccountBusinessPlatform(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataStub := &supplierProviderSyncHandlerDataStub{uniqueLocalAccount: true}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)
	router := gin.New()
	router.PUT("/accounts/:local_account_id/platform-override", handler.SetLocalAccountPlatformOverride)
	router.DELETE("/accounts/:local_account_id/platform-override", handler.ClearLocalAccountPlatformOverride)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/accounts/101/platform-override", bytes.NewBufferString(`{"platform":" GrOk "}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(101), dataStub.platformOverrideAccount)
	require.Equal(t, "grok", dataStub.platformOverride)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/accounts/101/platform-override", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(101), dataStub.clearedOverrideAccount)
}

func TestSupplierProviderSyncHandlerUpdatesLocalAccountWithEnabledCustomPlatform(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataStub := &supplierProviderSyncHandlerDataStub{uniqueLocalAccount: true}
	platformResolver := &supplierCustomPlatformResolverStub{
		platform: &service.CustomPlatform{Code: "glm", Name: "GLM", Enabled: true},
	}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)
	handler.SetCustomPlatformResolver(platformResolver)
	router := gin.New()
	router.PUT("/accounts/:local_account_id/platform-override", handler.SetLocalAccountPlatformOverride)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/accounts/101/platform-override", bytes.NewBufferString(`{"platform":" GLM "}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "glm", platformResolver.calledCode)
	require.Equal(t, int64(101), dataStub.platformOverrideAccount)
	require.Equal(t, "glm", dataStub.platformOverride)
}

func TestSupplierProviderSyncHandlerRejectsPlatformOverrideForNonSupplierAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataStub := &supplierProviderSyncHandlerDataStub{}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)
	router := gin.New()
	router.PUT("/accounts/:local_account_id/platform-override", handler.SetLocalAccountPlatformOverride)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/accounts/101/platform-override", bytes.NewBufferString(`{"platform":"grok"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Zero(t, dataStub.platformOverrideAccount)
}

func TestSupplierProviderSyncHandlerListsEmptyHealthGuardModelsForCustomPlatform(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataStub := &supplierProviderSyncHandlerDataStub{uniqueLocalAccount: true, effectivePlatform: "glm"}
	platformResolver := &supplierCustomPlatformResolverStub{
		platform: &service.CustomPlatform{Code: "glm", Name: "GLM", Enabled: true},
	}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)
	handler.SetCustomPlatformResolver(platformResolver)
	router := gin.New()
	router.GET("/accounts/:local_account_id/health-guard-models", handler.ListLocalAccountHealthGuardModels)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/accounts/101/health-guard-models", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `[]`)
}

func TestSupplierProviderSyncHandlerListsHealthGuardModelsByBusinessPlatform(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataStub := &supplierProviderSyncHandlerDataStub{uniqueLocalAccount: true, effectivePlatform: service.PlatformGrok}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)
	router := gin.New()
	router.GET("/accounts/:local_account_id/health-guard-models", handler.ListLocalAccountHealthGuardModels)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/accounts/101/health-guard-models", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"grok-4.5"`)
	require.NotContains(t, rec.Body.String(), `"gpt-5.6-sol"`)
}

func TestSupplierProviderSyncHandlerClearsPlatformOverrideWithSharedAccountIDRouteParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataStub := &supplierProviderSyncHandlerDataStub{uniqueLocalAccount: true}
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, dataStub)
	router := gin.New()
	router.DELETE("/accounts/:id/platform-override", handler.ClearLocalAccountPlatformOverride)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/accounts/101/platform-override", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(101), dataStub.clearedOverrideAccount)
}

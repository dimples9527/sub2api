package admin

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type supplierProviderSyncHandlerSyncStub struct {
	calledScope string
	testScope   string
}

func (s *supplierProviderSyncHandlerSyncStub) SyncAccounts(context.Context, int64, string) (service.SupplierProviderSyncResult, error) {
	s.calledScope = service.SupplierSyncScopeAccounts
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
func (s *supplierProviderSyncHandlerSyncStub) SyncAll(context.Context, int64, string) (service.SupplierProviderSyncResult, error) {
	s.calledScope = service.SupplierSyncScopeAll
	return supplierProviderSyncHandlerResult(service.SupplierSyncScopeAll), nil
}
func (s *supplierProviderSyncHandlerSyncStub) TestEndpoint(_ context.Context, _ int64, scope string) (service.SupplierProviderEndpointTestResult, error) {
	s.testScope = scope
	return service.SupplierProviderEndpointTestResult{Scope: scope, Endpoint: "/test/" + scope, HTTPStatus: 200, ResponseSummary: `{"code":0}`}, nil
}

func supplierProviderSyncHandlerResult(scope string) service.SupplierProviderSyncResult {
	now := time.Now()
	return service.SupplierProviderSyncResult{ProviderID: 42, Scope: scope, Status: service.SupplierSyncStatusSuccess, StartedAt: now, FinishedAt: now}
}

type supplierProviderSyncHandlerDataStub struct {
	mappedGroupID      int64
	mappedLocalGroupID *int64
	groupListParams    service.SupplierProviderDataListParams
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
	err      error
}

func (s *supplierGroupGuardHandlerStub) SetManualGuard(_ context.Context, groupID int64, selected bool) error {
	s.groupID = groupID
	s.selected = selected
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

func (*supplierProviderSyncHandlerDataStub) ListAccounts(context.Context, service.SupplierProviderDataListParams) (service.SupplierProviderAccountListResult, error) {
	return service.SupplierProviderAccountListResult{Items: []service.SupplierProviderAccount{{ID: 1, ProviderID: 42, Name: "Primary"}}, Total: 1, Page: 1, PageSize: 20}, nil
}
func (s *supplierProviderSyncHandlerDataStub) ListGroups(_ context.Context, params service.SupplierProviderDataListParams) (service.SupplierProviderGroupListResult, error) {
	s.groupListParams = params
	return service.SupplierProviderGroupListResult{Items: []service.SupplierProviderGroup{{ID: 1, ProviderID: 42, Name: "VIP"}}, Total: 1, Page: 1, PageSize: 20}, nil
}
func (s *supplierProviderSyncHandlerDataStub) UpdateGroupMapping(_ context.Context, groupID int64, localGroupID *int64) error {
	s.mappedGroupID = groupID
	s.mappedLocalGroupID = localGroupID
	return nil
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
	req = httptest.NewRequest(http.MethodGet, "/accounts?provider_id=42&active=true&page=1&page_size=20", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

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

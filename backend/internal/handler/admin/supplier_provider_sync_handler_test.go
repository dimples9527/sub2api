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

type supplierProviderSyncHandlerDataStub struct{}

func (supplierProviderSyncHandlerDataStub) ListAccounts(context.Context, service.SupplierProviderDataListParams) (service.SupplierProviderAccountListResult, error) {
	return service.SupplierProviderAccountListResult{Items: []service.SupplierProviderAccount{{ID: 1, ProviderID: 42, Name: "Primary"}}, Total: 1, Page: 1, PageSize: 20}, nil
}
func (supplierProviderSyncHandlerDataStub) ListGroups(context.Context, service.SupplierProviderDataListParams) (service.SupplierProviderGroupListResult, error) {
	return service.SupplierProviderGroupListResult{Items: []service.SupplierProviderGroup{{ID: 1, ProviderID: 42, Name: "VIP"}}, Total: 1, Page: 1, PageSize: 20}, nil
}

func TestSupplierProviderSyncHandlerRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	syncStub := &supplierProviderSyncHandlerSyncStub{}
	handler := NewSupplierProviderSyncHandler(syncStub, supplierProviderSyncHandlerDataStub{})
	router := gin.New()
	router.POST("/providers/:id/sync/all", handler.SyncAll)
	router.POST("/providers/:id/test/:scope", handler.TestEndpoint)
	router.GET("/accounts", handler.ListAccounts)

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
}

func TestSupplierProviderSyncHandlerRejectsInvalidProviderID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSupplierProviderSyncHandler(&supplierProviderSyncHandlerSyncStub{}, supplierProviderSyncHandlerDataStub{})
	router := gin.New()
	router.POST("/providers/:id/sync/all", handler.SyncAll)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/providers/bad/sync/all", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

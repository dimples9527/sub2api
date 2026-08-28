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

type supplierAccountHealthTrendHandlerStub struct {
	listParams  service.SupplierAccountHealthAccountListParams
	trendID     int64
	trendRange  string
	listResult  service.SupplierAccountHealthAccountListResult
	trendResult service.SupplierAccountHealthTrendResult
}

func (s *supplierAccountHealthTrendHandlerStub) ListAccounts(_ context.Context, params service.SupplierAccountHealthAccountListParams) (service.SupplierAccountHealthAccountListResult, error) {
	s.listParams = params
	return s.listResult, nil
}

func (s *supplierAccountHealthTrendHandlerStub) GetTrend(_ context.Context, accountID int64, rangeValue string) (service.SupplierAccountHealthTrendResult, error) {
	s.trendID = accountID
	s.trendRange = rangeValue
	return s.trendResult, nil
}

func TestSupplierAccountHealthHandlerListsAccountsWithFiltersAndPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &supplierAccountHealthTrendHandlerStub{
		listResult: service.SupplierAccountHealthAccountListResult{
			Items: []service.SupplierAccountHealthAccount{{LocalAccountID: 101, LocalAccountName: "账号 A", RateMultiplier: 1.25}},
			Total: 1, Page: 3, PageSize: 25,
		},
	}
	handler := NewSupplierAccountHealthHandler(stub)
	router := gin.New()
	router.GET("/accounts", handler.ListAccounts)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/accounts?provider_id=7&platform=%20grok%20&search=%20acct%20&health_status=slow&page=3&page_size=25", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.SupplierAccountHealthAccountListParams{
		ProviderID: 7, Platform: "grok", Search: "acct", HealthStatus: "slow", Page: 3, PageSize: 25,
	}, stub.listParams)
	require.Contains(t, rec.Body.String(), `"local_account_id":101`)
	require.Contains(t, rec.Body.String(), `"rate_multiplier":1.25`)
}

func TestSupplierAccountHealthHandlerGetsTrendWithDefaultRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &supplierAccountHealthTrendHandlerStub{trendResult: service.SupplierAccountHealthTrendResult{
		AccountID: 101,
		Points:    []service.SupplierAccountHealthPoint{{CheckedAt: time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC), Status: "healthy"}},
	}}
	handler := NewSupplierAccountHealthHandler(stub)
	router := gin.New()
	router.GET("/trend", handler.GetTrend)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trend?account_id=101", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(101), stub.trendID)
	require.Equal(t, service.SupplierAccountHealthRange24h, stub.trendRange)
	require.Contains(t, rec.Body.String(), `"account_id":101`)
}

func TestSupplierAccountHealthHandlerRejectsInvalidTrendAccountID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &supplierAccountHealthTrendHandlerStub{}
	handler := NewSupplierAccountHealthHandler(stub)
	router := gin.New()
	router.GET("/trend", handler.GetTrend)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trend?account_id=0&range=7d", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Zero(t, stub.trendID)
}

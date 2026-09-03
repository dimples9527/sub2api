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
	listParams    service.SupplierAccountHealthAccountListParams
	summaryParams service.SupplierAccountHealthAccountListParams
	trendID       int64
	trendRange    string
	batchIDs      []int64
	batchRange    string
	listResult    service.SupplierAccountHealthAccountListResult
	summaryResult service.SupplierAccountHealthSummary
	trendResult   service.SupplierAccountHealthTrendResult
	batchResults  []service.SupplierAccountHealthTrendResult
	recordParams  service.SupplierAccountHealthRecordListParams
	recordResult  service.SupplierAccountHealthRecordListResult
}

func (s *supplierAccountHealthTrendHandlerStub) ListAccounts(_ context.Context, params service.SupplierAccountHealthAccountListParams) (service.SupplierAccountHealthAccountListResult, error) {
	s.listParams = params
	return s.listResult, nil
}

func (s *supplierAccountHealthTrendHandlerStub) GetSummary(_ context.Context, params service.SupplierAccountHealthAccountListParams) (service.SupplierAccountHealthSummary, error) {
	s.summaryParams = params
	return s.summaryResult, nil
}

func (s *supplierAccountHealthTrendHandlerStub) GetTrend(_ context.Context, accountID int64, rangeValue string) (service.SupplierAccountHealthTrendResult, error) {
	s.trendID = accountID
	s.trendRange = rangeValue
	return s.trendResult, nil
}
func (s *supplierAccountHealthTrendHandlerStub) GetTrends(_ context.Context, accountIDs []int64, rangeValue string) ([]service.SupplierAccountHealthTrendResult, error) {
	s.batchIDs = accountIDs
	s.batchRange = rangeValue
	return s.batchResults, nil
}

func (s *supplierAccountHealthTrendHandlerStub) ListRecords(_ context.Context, params service.SupplierAccountHealthRecordListParams) (service.SupplierAccountHealthRecordListResult, error) {
	s.recordParams = params
	return s.recordResult, nil
}

func TestSupplierAccountHealthHandlerListsAccountsWithFiltersAndPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &supplierAccountHealthTrendHandlerStub{
		listResult: service.SupplierAccountHealthAccountListResult{
			Items: []service.SupplierAccountHealthAccount{{
				LocalAccountID: 101, LocalAccountName: "账号 A",
				UpstreamRateMultiplier: 1.25, EffectiveRateMultiplier: 2.5,
			}},
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
	require.Contains(t, rec.Body.String(), `"upstream_rate_multiplier":1.25`)
	require.Contains(t, rec.Body.String(), `"effective_rate_multiplier":2.5`)
}

func TestSupplierAccountHealthHandlerGetsSummaryIgnoringStatusFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &supplierAccountHealthTrendHandlerStub{
		summaryResult: service.SupplierAccountHealthSummary{Total: 9, Healthy: 5, Slow: 2, Failed: 1, Unchecked: 1},
	}
	handler := NewSupplierAccountHealthHandler(stub)
	router := gin.New()
	router.GET("/summary", handler.GetSummary)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/summary?provider_id=7&platform=%20grok%20&search=%20acct%20&health_status=failed", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.SupplierAccountHealthAccountListParams{ProviderID: 7, Platform: "grok", Search: "acct"}, stub.summaryParams)
	require.Contains(t, rec.Body.String(), `"total":9`)
	require.Contains(t, rec.Body.String(), `"unchecked":1`)
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

func TestSupplierAccountHealthHandlerListsRecordsWithStatusAndLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &supplierAccountHealthTrendHandlerStub{recordResult: service.SupplierAccountHealthRecordListResult{
		Items: []service.SupplierAccountHealthRecord{{
			CheckedAt: time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC),
			Status:    "failed", ConsecutiveFailed: 4, Action: "disable", ErrorMessage: "401 invalid api key",
		}},
		Limit: 20,
	}}
	handler := NewSupplierAccountHealthHandler(stub)
	router := gin.New()
	router.GET("/records", handler.ListRecords)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/records?account_id=101&status=%20failed%20&limit=20", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.SupplierAccountHealthRecordListParams{AccountID: 101, Status: "failed", Limit: 20}, stub.recordParams)
	require.Contains(t, rec.Body.String(), `"consecutive_failed":4`)
	require.Contains(t, rec.Body.String(), `"error_message":"401 invalid api key"`)
}

func TestSupplierAccountHealthHandlerRejectsInvalidRecordsAccountID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &supplierAccountHealthTrendHandlerStub{}
	handler := NewSupplierAccountHealthHandler(stub)
	router := gin.New()
	router.GET("/records", handler.ListRecords)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/records?account_id=abc", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Zero(t, stub.recordParams.AccountID)
}

// limit 非法时交给服务层落回默认值，不应因此拒绝请求。
func TestSupplierAccountHealthHandlerListsRecordsIgnoringInvalidLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &supplierAccountHealthTrendHandlerStub{}
	handler := NewSupplierAccountHealthHandler(stub)
	router := gin.New()
	router.GET("/records", handler.ListRecords)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/records?account_id=101&limit=abc", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.SupplierAccountHealthRecordListParams{AccountID: 101}, stub.recordParams)
}
func TestSupplierAccountHealthHandlerGetsTrendsWithoutDuplicateIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &supplierAccountHealthTrendHandlerStub{batchResults: []service.SupplierAccountHealthTrendResult{
		{AccountID: 12, Points: []service.SupplierAccountHealthPoint{}},
		{AccountID: 37, Points: []service.SupplierAccountHealthPoint{}},
	}}
	handler := NewSupplierAccountHealthHandler(stub)
	router := gin.New()
	router.GET("/trends", handler.GetTrends)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trends?ids=12,37,12&range=7d", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []int64{12, 37}, stub.batchIDs)
	require.Equal(t, service.SupplierAccountHealthRange7d, stub.batchRange)
	require.Contains(t, rec.Body.String(), `"items":[{"account_id":12`)
	require.Contains(t, rec.Body.String(), `"account_id":37`)
}

func TestSupplierAccountHealthHandlerRejectsInvalidTrendBatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &supplierAccountHealthTrendHandlerStub{}
	handler := NewSupplierAccountHealthHandler(stub)
	router := gin.New()
	router.GET("/trends", handler.GetTrends)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trends?ids=1,broken", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Nil(t, stub.batchIDs)
}

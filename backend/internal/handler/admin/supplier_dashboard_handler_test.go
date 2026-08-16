package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type supplierDashboardHandlerStub struct {
	accountsQuery  service.SupplierDashboardAccountsQuery
	ratesQuery     service.SupplierDashboardRatesQuery
	providersQuery service.SupplierDashboardProvidersQuery
	trafficQuery   service.SupplierDashboardTrafficQuery
	profitQuery    service.SupplierDashboardProfitQuery
	healthQuery    service.SupplierDashboardAccountHealthQuery
	accountsErr    error
	ratesErr       error
	providersErr   error
}

func (s *supplierDashboardHandlerStub) GetAccounts(_ context.Context, q service.SupplierDashboardAccountsQuery) (service.SupplierDashboardAccountsResponse, error) {
	s.accountsQuery = q
	if s.accountsErr != nil {
		return service.SupplierDashboardAccountsResponse{}, s.accountsErr
	}
	return service.SupplierDashboardAccountsResponse{
		Range:       q.Range,
		Items:       []service.SupplierDashboardAccountItem{},
		Total:       0,
		Page:        q.Page,
		PageSize:    q.PageSize,
		Warnings:    []service.SupplierDashboardWarning{},
		GeneratedAt: time.Date(2026, 7, 25, 8, 0, 0, 0, time.UTC),
	}, nil
}

func (s *supplierDashboardHandlerStub) GetRates(_ context.Context, q service.SupplierDashboardRatesQuery) (service.SupplierDashboardRatesResponse, error) {
	s.ratesQuery = q
	if s.ratesErr != nil {
		return service.SupplierDashboardRatesResponse{}, s.ratesErr
	}
	return service.SupplierDashboardRatesResponse{
		Range:       q.Range,
		Items:       []service.SupplierDashboardRateItem{},
		Total:       0,
		Page:        q.Page,
		PageSize:    q.PageSize,
		Warnings:    []service.SupplierDashboardWarning{},
		GeneratedAt: time.Date(2026, 7, 25, 8, 0, 0, 0, time.UTC),
	}, nil
}

func (s *supplierDashboardHandlerStub) GetProviders(_ context.Context, q service.SupplierDashboardProvidersQuery) (service.SupplierDashboardProvidersResponse, error) {
	s.providersQuery = q
	if s.providersErr != nil {
		return service.SupplierDashboardProvidersResponse{}, s.providersErr
	}
	return service.SupplierDashboardProvidersResponse{
		Range:       q.Range,
		Items:       []service.SupplierDashboardProviderItem{},
		Total:       0,
		Page:        q.Page,
		PageSize:    q.PageSize,
		Warnings:    []service.SupplierDashboardWarning{},
		GeneratedAt: time.Date(2026, 7, 25, 8, 0, 0, 0, time.UTC),
	}, nil
}

func (s *supplierDashboardHandlerStub) GetAccountTraffic(_ context.Context, q service.SupplierDashboardTrafficQuery) (service.SupplierDashboardTrafficResponse, error) {
	s.trafficQuery = q
	return service.SupplierDashboardTrafficResponse{
		Range:       q.Range,
		Series:      []service.SupplierDashboardTrafficPoint{},
		Accounts:    []service.SupplierDashboardTrafficAccount{},
		Warnings:    []service.SupplierDashboardWarning{},
		GeneratedAt: time.Date(2026, 7, 25, 8, 0, 0, 0, time.UTC),
	}, nil
}

func (s *supplierDashboardHandlerStub) GetAccountProfitRanking(_ context.Context, q service.SupplierDashboardProfitQuery) (service.SupplierDashboardProfitResponse, error) {
	s.profitQuery = q
	return service.SupplierDashboardProfitResponse{
		Items:       []service.SupplierDashboardProfitItem{},
		Warnings:    []service.SupplierDashboardWarning{},
		GeneratedAt: time.Date(2026, 7, 25, 8, 0, 0, 0, time.UTC),
	}, nil
}

func (s *supplierDashboardHandlerStub) GetAccountHealthTimeline(_ context.Context, q service.SupplierDashboardAccountHealthQuery) (service.SupplierDashboardAccountHealthResponse, error) {
	s.healthQuery = q
	return service.SupplierDashboardAccountHealthResponse{
		Range:       q.Range,
		Accounts:    []service.SupplierDashboardHealthAccount{},
		Hours:       []service.SupplierDashboardHealthHour{},
		Warnings:    []service.SupplierDashboardWarning{},
		GeneratedAt: time.Date(2026, 7, 25, 8, 0, 0, 0, time.UTC),
	}, nil
}

func newSupplierDashboardTestRouter(handler *SupplierDashboardHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/accounts", handler.GetAccounts)
	router.GET("/rates", handler.GetRates)
	router.GET("/providers", handler.GetProviders)
	router.GET("/traffic", handler.GetAccountTraffic)
	router.GET("/profit-ranking", handler.GetAccountProfitRanking)
	router.GET("/health-timeline", handler.GetAccountHealthTimeline)
	return router
}

func TestSupplierDashboardHandlerUsesSecureDefaults(t *testing.T) {
	stub := &supplierDashboardHandlerStub{}
	router := newSupplierDashboardTestRouter(newSupplierDashboardHandlerWithService(stub))

	for _, path := range []string{"/accounts", "/rates", "/providers", "/traffic", "/profit-ranking", "/health-timeline"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusOK, recorder.Code, path)
		require.Contains(t, recorder.Body.String(), `"generated_at":"2026-07-25T08:00:00Z"`, path)
	}

	require.Equal(t, service.SupplierDashboardRange24Hours, stub.accountsQuery.Range)
	require.Equal(t, service.SupplierDashboardRiskTypeAll, stub.accountsQuery.RiskType)
	require.Equal(t, 1, stub.accountsQuery.Page)
	require.Equal(t, 20, stub.accountsQuery.PageSize)

	require.Equal(t, service.SupplierDashboardRange24Hours, stub.ratesQuery.Range)
	require.Equal(t, service.SupplierDashboardRateViewRisk, stub.ratesQuery.View)
	require.Equal(t, 1, stub.ratesQuery.Page)
	require.Equal(t, 20, stub.ratesQuery.PageSize)

	require.Equal(t, service.SupplierDashboardRange24Hours, stub.providersQuery.Range)
	require.Equal(t, 1, stub.providersQuery.Page)
	require.Equal(t, 20, stub.providersQuery.PageSize)

	require.Equal(t, service.SupplierDashboardRange30Days, stub.trafficQuery.Range)
	require.Equal(t, service.SupplierDashboardRange30Days, stub.profitQuery.Range)
	require.Equal(t, 20, stub.profitQuery.Limit)
	require.Equal(t, service.SupplierDashboardRange30Days, stub.healthQuery.Range)
	require.Equal(t, 30, stub.healthQuery.Limit)
	require.Equal(t, 72, stub.healthQuery.Buckets)
	require.Equal(t, 1, stub.healthQuery.BucketHours)
}

func TestSupplierDashboardHandlerParsesFiltersAndPagination(t *testing.T) {
	stub := &supplierDashboardHandlerStub{}
	router := newSupplierDashboardTestRouter(newSupplierDashboardHandlerWithService(stub))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/accounts?range=7d&risk_type=balance&provider_slug=p1&group_key=g1&page=2&page_size=50", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, service.SupplierDashboardAccountsQuery{
		Range:        service.SupplierDashboardRange7Days,
		RiskType:     service.SupplierDashboardRiskTypeBalance,
		ProviderSlug: "p1",
		GroupKey:     "g1",
		Page:         2,
		PageSize:     50,
	}, stub.accountsQuery)

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/rates?range=24h&view=changed&comparison_status=not_lowest&provider_slug=p2&group_key=g2&page=3&page_size=100", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, service.SupplierDashboardRatesQuery{
		Range:            service.SupplierDashboardRange24Hours,
		View:             service.SupplierDashboardRateViewChanged,
		ComparisonStatus: service.SupplierDashboardComparisonStatusNotLowest,
		ProviderSlug:     "p2",
		GroupKey:         "g2",
		Page:             3,
		PageSize:         100,
	}, stub.ratesQuery)

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/providers?range=7d&status=high_risk&page=4&page_size=10", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, service.SupplierDashboardProvidersQuery{
		Range:    service.SupplierDashboardRange7Days,
		Status:   service.SupplierDashboardProviderStatusHighRisk,
		Page:     4,
		PageSize: 10,
	}, stub.providersQuery)
}

func TestSupplierDashboardHandlerRejectsInvalidQuery(t *testing.T) {
	stub := &supplierDashboardHandlerStub{}
	router := newSupplierDashboardTestRouter(newSupplierDashboardHandlerWithService(stub))

	cases := []struct {
		path string
	}{
		{path: "/accounts?range=90d"},
		{path: "/accounts?risk_type=unknown"},
		{path: "/accounts?page=0"},
		{path: "/accounts?page=-1"},
		{path: "/accounts?page=abc"},
		{path: "/accounts?page_size=0"},
		{path: "/accounts?page_size=101"},
		{path: "/accounts?page_size=abc"},
		{path: "/rates?view=latest"},
		{path: "/rates?comparison_status=weird"},
		{path: "/providers?status=broken"},
		{path: "/traffic?range=90d"},
		{path: "/profit-ranking?range=90d"},
		{path: "/profit-ranking?limit=0"},
		{path: "/profit-ranking?limit=101"},
		{path: "/profit-ranking?limit=abc"},
		{path: "/health-timeline?range=90d"},
		{path: "/health-timeline?limit=0"},
		{path: "/health-timeline?limit=101"},
		{path: "/health-timeline?buckets=0"},
		{path: "/health-timeline?buckets=721"},
		{path: "/health-timeline?buckets=abc"},
		{path: "/health-timeline?bucket_hours=0"},
		{path: "/health-timeline?bucket_hours=25"},
		{path: "/health-timeline?bucket_hours=abc"},
	}
	for _, tc := range cases {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tc.path, nil))
		require.Equal(t, http.StatusBadRequest, recorder.Code, tc.path)
	}
	require.Empty(t, stub.accountsQuery.Range)
	require.Empty(t, stub.ratesQuery.Range)
	require.Empty(t, stub.providersQuery.Range)
}

func TestSupplierDashboardHandlerParsesTrendQueries(t *testing.T) {
	stub := &supplierDashboardHandlerStub{}
	router := newSupplierDashboardTestRouter(newSupplierDashboardHandlerWithService(stub))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/traffic?range=30d&provider_slug=p1&group_key=g1", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, service.SupplierDashboardTrafficQuery{
		Range:        service.SupplierDashboardRange30Days,
		ProviderSlug: "p1",
		GroupKey:     "g1",
	}, stub.trafficQuery)

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/profit-ranking?range=7d&provider_slug=p2&group_key=g2&limit=50", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, service.SupplierDashboardProfitQuery{
		Range:        service.SupplierDashboardRange7Days,
		ProviderSlug: "p2",
		GroupKey:     "g2",
		Limit:        50,
	}, stub.profitQuery)

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health-timeline?range=24h&provider_slug=p3&group_key=g3&limit=10&buckets=48&bucket_hours=6", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, service.SupplierDashboardAccountHealthQuery{
		Range:        service.SupplierDashboardRange24Hours,
		ProviderSlug: "p3",
		GroupKey:     "g3",
		Limit:        10,
		Buckets:      48,
		BucketHours:  6,
	}, stub.healthQuery)
}

func TestSupplierDashboardHandlerPropagatesServiceErrors(t *testing.T) {
	stub := &supplierDashboardHandlerStub{accountsErr: errors.New("accounts boom")}
	router := newSupplierDashboardTestRouter(newSupplierDashboardHandlerWithService(stub))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/accounts", nil))
	require.Equal(t, http.StatusInternalServerError, recorder.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.NotEqual(t, float64(0), body["code"])
}

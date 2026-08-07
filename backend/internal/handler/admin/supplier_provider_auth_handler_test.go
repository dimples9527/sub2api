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

type supplierProviderAuthHandlerRepoStub struct {
	summary       service.SupplierProviderAuthSummary
	history       service.SupplierProviderAuthHistoryResult
	statusID      int64
	historyID     int64
	historyParams service.SupplierProviderAuthHistoryParams
	historyCalls  int
}

func (s *supplierProviderAuthHandlerRepoStub) Record(context.Context, service.SupplierProviderAuthEventRecord) error {
	return nil
}

func (s *supplierProviderAuthHandlerRepoStub) GetSummary(_ context.Context, providerID int64) (service.SupplierProviderAuthSummary, error) {
	s.statusID = providerID
	return s.summary, nil
}

func (s *supplierProviderAuthHandlerRepoStub) ListHistory(_ context.Context, providerID int64, params service.SupplierProviderAuthHistoryParams) (service.SupplierProviderAuthHistoryResult, error) {
	s.historyID = providerID
	s.historyParams = params
	s.historyCalls++
	return s.history, nil
}

type supplierProviderAuthHandlerTokenCacheStub struct {
	snapshot service.SupplierProviderTokenCacheSnapshot
}

func (s *supplierProviderAuthHandlerTokenCacheStub) Get(context.Context, int64) (service.SupplierProviderAuthToken, bool, error) {
	return service.SupplierProviderAuthToken{}, false, nil
}

func (s *supplierProviderAuthHandlerTokenCacheStub) Set(context.Context, int64, service.SupplierProviderAuthToken, time.Duration) error {
	return nil
}

func (s *supplierProviderAuthHandlerTokenCacheStub) Delete(context.Context, int64) error {
	return nil
}

func (s *supplierProviderAuthHandlerTokenCacheStub) TryAcquireLoginLock(context.Context, int64, string, time.Duration) (bool, error) {
	return false, nil
}

func (s *supplierProviderAuthHandlerTokenCacheStub) ReleaseLoginLock(context.Context, int64, string) error {
	return nil
}

func (s *supplierProviderAuthHandlerTokenCacheStub) Inspect(context.Context, int64) (service.SupplierProviderTokenCacheSnapshot, error) {
	return s.snapshot, nil
}

func newSupplierProviderAuthHandlerTestRouter(repo service.SupplierProviderAuthAuditRepository, tokenCache service.SupplierProviderTokenCache) *gin.Engine {
	gin.SetMode(gin.TestMode)
	authService := service.NewSupplierProviderAuthAuditService(repo, tokenCache)
	handler := NewSupplierProviderAuthHandler(authService)
	router := gin.New()
	router.GET("/providers/:id/auth-status", handler.GetStatus)
	router.GET("/providers/:id/auth-history", handler.ListHistory)
	return router
}

func TestSupplierProviderAuthHandlerReturnsMaskedCacheStatus(t *testing.T) {
	now := time.Now()
	repo := &supplierProviderAuthHandlerRepoStub{summary: service.SupplierProviderAuthSummary{
		LoginCount:          4,
		LoginSuccessCount:   3,
		LoginFailureCount:   1,
		RefreshCount:        6,
		RefreshSuccessCount: 5,
		RefreshFailureCount: 1,
	}}
	cache := &supplierProviderAuthHandlerTokenCacheStub{snapshot: service.SupplierProviderTokenCacheSnapshot{
		Found: true,
		Token: service.SupplierProviderAuthToken{
			AccessToken:  "0123456789abcdef",
			TokenType:    "Bearer",
			ExpiresAt:    now.Add(time.Hour),
			CookieHeader: "session=fake-cookie-value",
		},
		TTL:      45 * time.Minute,
		LockHeld: true,
		LockTTL:  2 * time.Minute,
	}}
	router := newSupplierProviderAuthHandlerTestRouter(repo, cache)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/providers/42/auth-status", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), repo.statusID)
	require.Contains(t, rec.Body.String(), `"login_count":4`)
	require.Contains(t, rec.Body.String(), `"refresh_count":6`)
	require.Contains(t, rec.Body.String(), `"refresh_success_count":5`)
	require.Contains(t, rec.Body.String(), `"refresh_failure_count":1`)
	require.Contains(t, rec.Body.String(), `"status":"cached"`)
	require.Contains(t, rec.Body.String(), `"token_summary":"0123…cdef"`)
	require.Contains(t, rec.Body.String(), `"cookie_present":true`)
	require.Contains(t, rec.Body.String(), `"held":true`)
	require.NotContains(t, rec.Body.String(), "0123456789abcdef")
	require.NotContains(t, rec.Body.String(), "fake-cookie-value")
}

func TestSupplierProviderAuthHandlerReturnsFilteredHistory(t *testing.T) {
	createdAt := time.Date(2026, time.August, 5, 7, 0, 0, 0, time.UTC)
	repo := &supplierProviderAuthHandlerRepoStub{history: service.SupplierProviderAuthHistoryResult{
		Items: []service.SupplierProviderAuthHistoryItem{{
			ID: 9, ProviderID: 42, EventType: service.SupplierProviderAuthEventLoginFailed,
			Source: service.SupplierProviderAuthSourceSync, Status: service.SupplierProviderAuthStatusFailed,
			ErrorMessage: "upstream unavailable", CreatedAt: createdAt,
		}},
		Total: 31, Page: 2, PageSize: 100,
	}}
	router := newSupplierProviderAuthHandlerTestRouter(repo, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/providers/42/auth-history?page=2&page_size=500&event_type=login_failed", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), repo.historyID)
	require.Equal(t, 2, repo.historyParams.Page)
	require.Equal(t, 100, repo.historyParams.PageSize)
	require.Equal(t, service.SupplierProviderAuthEventLoginFailed, repo.historyParams.EventType)
	require.Contains(t, rec.Body.String(), `"event_type":"login_failed"`)
	require.Contains(t, rec.Body.String(), `"total":31`)
}

func TestSupplierProviderAuthHandlerRejectsInvalidParameters(t *testing.T) {
	repo := &supplierProviderAuthHandlerRepoStub{}
	router := newSupplierProviderAuthHandlerTestRouter(repo, nil)

	for _, target := range []string{
		"/providers/bad/auth-status",
		"/providers/42/auth-history?event_type=password_dump",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, target, nil)
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code, target)
	}
	require.Zero(t, repo.historyCalls)
}

func TestSupplierProviderAuthHandlerAllowsRefreshHistoryFilter(t *testing.T) {
	repo := &supplierProviderAuthHandlerRepoStub{history: service.SupplierProviderAuthHistoryResult{
		Items: []service.SupplierProviderAuthHistoryItem{{
			ID: 9, ProviderID: 42, EventType: service.SupplierProviderAuthEventRefreshSuccess,
			Source: service.SupplierProviderAuthSourceSync, Status: service.SupplierProviderAuthStatusSuccess,
		}},
		Total: 1, Page: 1, PageSize: 20,
	}}
	router := newSupplierProviderAuthHandlerTestRouter(repo, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/providers/42/auth-history?event_type=refresh_success", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.SupplierProviderAuthEventRefreshSuccess, repo.historyParams.EventType)
	require.Contains(t, rec.Body.String(), `"event_type":"refresh_success"`)
}

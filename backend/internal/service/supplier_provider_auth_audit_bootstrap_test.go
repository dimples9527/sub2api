package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type supplierProviderAuthAuditRepoSpy struct {
	mu       sync.Mutex
	summary  SupplierProviderAuthSummary
	records  []SupplierProviderAuthEventRecord
	getCalls int
}

func (s *supplierProviderAuthAuditRepoSpy) Record(_ context.Context, event SupplierProviderAuthEventRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, event)
	switch event.EventType {
	case SupplierProviderAuthEventCacheHit:
		s.summary.CacheHitCount++
		s.summary.LastCacheHitAt = &event.FinishedAt
		s.summary.LastTokenFingerprint = event.TokenFingerprint
		if event.TokenExpiresAt != nil {
			s.summary.LastTokenExpiresAt = event.TokenExpiresAt
		}
	case SupplierProviderAuthEventLoginSuccess:
		s.summary.LoginCount++
		s.summary.LoginSuccessCount++
		s.summary.LastLoginAt = &event.FinishedAt
		s.summary.LastLoginStatus = string(SupplierProviderAuthStatusSuccess)
	case SupplierProviderAuthEventLoginFailed:
		s.summary.LoginCount++
		s.summary.LoginFailureCount++
		s.summary.LastLoginAt = &event.FinishedAt
		s.summary.LastLoginStatus = string(SupplierProviderAuthStatusFailed)
	case SupplierProviderAuthEventCacheMiss:
		s.summary.CacheMissCount++
	}
	return nil
}

func (s *supplierProviderAuthAuditRepoSpy) GetSummary(_ context.Context, _ int64) (SupplierProviderAuthSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getCalls++
	return s.summary, nil
}

func (s *supplierProviderAuthAuditRepoSpy) ListHistory(context.Context, int64, SupplierProviderAuthHistoryParams) (SupplierProviderAuthHistoryResult, error) {
	return SupplierProviderAuthHistoryResult{}, nil
}

type supplierProviderAuthTokenCacheSpy struct {
	snapshot SupplierProviderTokenCacheSnapshot
}

func (s *supplierProviderAuthTokenCacheSpy) Get(context.Context, int64) (SupplierProviderAuthToken, bool, error) {
	return SupplierProviderAuthToken{}, false, nil
}
func (s *supplierProviderAuthTokenCacheSpy) Set(context.Context, int64, SupplierProviderAuthToken, time.Duration) error {
	return nil
}
func (s *supplierProviderAuthTokenCacheSpy) Delete(context.Context, int64) error { return nil }
func (s *supplierProviderAuthTokenCacheSpy) TryAcquireLoginLock(context.Context, int64, string, time.Duration) (bool, error) {
	return false, nil
}
func (s *supplierProviderAuthTokenCacheSpy) ReleaseLoginLock(context.Context, int64, string) error {
	return nil
}
func (s *supplierProviderAuthTokenCacheSpy) Inspect(context.Context, int64) (SupplierProviderTokenCacheSnapshot, error) {
	return s.snapshot, nil
}

func TestGetStatusBootstrapsCacheHitWhenTokenCachedButAuditEmpty(t *testing.T) {
	now := time.Now()
	repo := &supplierProviderAuthAuditRepoSpy{}
	cache := &supplierProviderAuthTokenCacheSpy{snapshot: SupplierProviderTokenCacheSnapshot{
		Found: true,
		Token: SupplierProviderAuthToken{
			AccessToken: "cached-access-token-value",
			TokenType:   "Bearer",
			ExpiresAt:   now.Add(30 * time.Minute),
		},
		TTL: 30 * time.Minute,
	}}
	svc := NewSupplierProviderAuthAuditService(repo, cache)

	status, err := svc.GetStatus(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, SupplierProviderAuthCacheCached, status.Cache.Status)
	require.Equal(t, int64(1), status.Summary.CacheHitCount)
	require.Equal(t, int64(0), status.Summary.LoginCount)
	require.Equal(t, int64(0), status.Summary.LoginSuccessCount)
	require.NotNil(t, status.Summary.LastCacheHitAt)
	require.Len(t, repo.records, 1)
	require.Equal(t, SupplierProviderAuthEventCacheHit, repo.records[0].EventType)
	require.Equal(t, int64(42), repo.records[0].ProviderID)
	require.NotEmpty(t, repo.records[0].TokenFingerprint)
	require.GreaterOrEqual(t, repo.getCalls, 2)
}

func TestGetStatusDoesNotBootstrapWhenAuditAlreadyHasData(t *testing.T) {
	now := time.Now()
	lastHit := now.Add(-time.Minute)
	repo := &supplierProviderAuthAuditRepoSpy{summary: SupplierProviderAuthSummary{
		CacheHitCount:  3,
		LastCacheHitAt: &lastHit,
	}}
	cache := &supplierProviderAuthTokenCacheSpy{snapshot: SupplierProviderTokenCacheSnapshot{
		Found: true,
		Token: SupplierProviderAuthToken{
			AccessToken: "cached-access-token-value",
			ExpiresAt:   now.Add(time.Hour),
		},
		TTL: time.Hour,
	}}
	svc := NewSupplierProviderAuthAuditService(repo, cache)

	status, err := svc.GetStatus(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, int64(3), status.Summary.CacheHitCount)
	require.Empty(t, repo.records)
}

func TestGetStatusDoesNotBootstrapWhenCacheMissing(t *testing.T) {
	repo := &supplierProviderAuthAuditRepoSpy{}
	cache := &supplierProviderAuthTokenCacheSpy{snapshot: SupplierProviderTokenCacheSnapshot{Found: false}}
	svc := NewSupplierProviderAuthAuditService(repo, cache)

	status, err := svc.GetStatus(context.Background(), 9)
	require.NoError(t, err)
	require.Equal(t, SupplierProviderAuthCacheMissing, status.Cache.Status)
	require.Equal(t, int64(0), status.Summary.CacheHitCount)
	require.Empty(t, repo.records)
}

func TestSupplierProviderAuthSummaryEmptyHelper(t *testing.T) {
	require.True(t, supplierProviderAuthSummaryEmpty(SupplierProviderAuthSummary{}))
	require.False(t, supplierProviderAuthSummaryEmpty(SupplierProviderAuthSummary{CacheHitCount: 1}))
	require.False(t, supplierProviderAuthSummaryEmpty(SupplierProviderAuthSummary{LoginCount: 1}))
	last := time.Now()
	require.False(t, supplierProviderAuthSummaryEmpty(SupplierProviderAuthSummary{LastLoginAt: &last}))
}

package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newSupplierProviderTokenCacheTestClient(t *testing.T) (service.SupplierProviderTokenCache, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { require.NoError(t, rdb.Close()) })

	return NewSupplierProviderTokenCache(rdb), mr
}

func newSupplierProviderSyncLockTestClient(t *testing.T) (service.SupplierProviderSyncLock, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { require.NoError(t, rdb.Close()) })

	return NewSupplierProviderTokenCache(rdb), mr
}

func TestSupplierProviderTokenCacheStoresTokenWithTTL(t *testing.T) {
	cache, mr := newSupplierProviderTokenCacheTestClient(t)
	ctx := context.Background()
	token := service.SupplierProviderAuthToken{
		AccessToken: "access-token",
		TokenType:   "Bearer",
		ExpiresAt:   time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC),
	}

	require.NoError(t, cache.Set(ctx, 42, token, 5*time.Minute))

	cached, found, err := cache.Get(ctx, 42)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, token, cached)

	const key = "supplier:provider:auth:42"
	raw, err := mr.Get(key)
	require.NoError(t, err)
	var stored service.SupplierProviderAuthToken
	require.NoError(t, json.Unmarshal([]byte(raw), &stored))
	require.Equal(t, token, stored)
	require.Equal(t, 5*time.Minute, mr.TTL(key))
}

func TestSupplierProviderTokenCacheReturnsMissAfterExpiry(t *testing.T) {
	cache, mr := newSupplierProviderTokenCacheTestClient(t)
	ctx := context.Background()
	token := service.SupplierProviderAuthToken{AccessToken: "short-lived"}

	require.NoError(t, cache.Set(ctx, 7, token, time.Second))
	mr.FastForward(2 * time.Second)

	cached, found, err := cache.Get(ctx, 7)
	require.NoError(t, err)
	require.False(t, found)
	require.Equal(t, service.SupplierProviderAuthToken{}, cached)
}

func TestSupplierProviderTokenCacheSetUsesFallbackForInvalidTTL(t *testing.T) {
	cache, mr := newSupplierProviderTokenCacheTestClient(t)
	ctx := context.Background()

	require.NoError(t, cache.Set(ctx, 8, service.SupplierProviderAuthToken{AccessToken: "fallback-ttl"}, 0))

	require.Equal(t, 30*time.Minute, mr.TTL("supplier:provider:auth:8"))
}

func TestSupplierProviderTokenCacheDeletesToken(t *testing.T) {
	cache, _ := newSupplierProviderTokenCacheTestClient(t)
	ctx := context.Background()

	require.NoError(t, cache.Set(ctx, 9, service.SupplierProviderAuthToken{AccessToken: "delete-me"}, time.Minute))
	require.NoError(t, cache.Delete(ctx, 9))

	cached, found, err := cache.Get(ctx, 9)
	require.NoError(t, err)
	require.False(t, found)
	require.Equal(t, service.SupplierProviderAuthToken{}, cached)
}

func TestSupplierProviderTokenCacheReleasesOnlyOwnedLoginLock(t *testing.T) {
	cache, mr := newSupplierProviderTokenCacheTestClient(t)
	ctx := context.Background()

	acquired, err := cache.TryAcquireLoginLock(ctx, 12, "owner-a", time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)
	require.Equal(t, time.Minute, mr.TTL("supplier:provider:auth-lock:12"))

	acquired, err = cache.TryAcquireLoginLock(ctx, 12, "owner-b", time.Minute)
	require.NoError(t, err)
	require.False(t, acquired)

	require.NoError(t, cache.ReleaseLoginLock(ctx, 12, "owner-b"))
	lockOwner, err := mr.Get("supplier:provider:auth-lock:12")
	require.NoError(t, err)
	require.Equal(t, "owner-a", lockOwner)

	require.NoError(t, cache.ReleaseLoginLock(ctx, 12, "owner-a"))
	acquired, err = cache.TryAcquireLoginLock(ctx, 12, "owner-b", time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)
}

func TestSupplierProviderTokenCacheReleasesOnlyOwnedSyncLock(t *testing.T) {
	lock, mr := newSupplierProviderSyncLockTestClient(t)
	ctx := context.Background()

	acquired, err := lock.TryAcquireSyncLock(ctx, 15, "owner-a", 2*time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)
	require.Equal(t, 2*time.Minute, mr.TTL("supplier:provider:sync-lock:15"))

	acquired, err = lock.TryAcquireSyncLock(ctx, 15, "owner-b", 2*time.Minute)
	require.NoError(t, err)
	require.False(t, acquired)

	require.NoError(t, lock.ReleaseSyncLock(ctx, 15, "owner-b"))
	lockOwner, err := mr.Get("supplier:provider:sync-lock:15")
	require.NoError(t, err)
	require.Equal(t, "owner-a", lockOwner)

	require.NoError(t, lock.ReleaseSyncLock(ctx, 15, "owner-a"))
	acquired, err = lock.TryAcquireSyncLock(ctx, 15, "owner-b", 2*time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)
}

func TestSupplierProviderTokenCacheRejectsInvalidLockArguments(t *testing.T) {
	cache, _ := newSupplierProviderTokenCacheTestClient(t)
	ctx := context.Background()

	_, err := cache.TryAcquireLoginLock(ctx, 1, "", time.Minute)
	require.Error(t, err)

	_, err = cache.TryAcquireLoginLock(ctx, 1, "owner", 0)
	require.Error(t, err)
}

func TestSupplierProviderTokenTTLUsesSafetyWindow(t *testing.T) {
	tests := []struct {
		name      string
		expiresIn time.Duration
		want      time.Duration
	}{
		{name: "one hour subtracts sixty seconds", expiresIn: time.Hour, want: 59 * time.Minute},
		{name: "one hundred twenty seconds subtracts ten percent", expiresIn: 120 * time.Second, want: 108 * time.Second},
		{name: "one hundred twenty one seconds subtracts sixty seconds", expiresIn: 121 * time.Second, want: 61 * time.Second},
		{name: "one hundred seconds subtracts ten percent", expiresIn: 100 * time.Second, want: 90 * time.Second},
		{name: "zero uses fallback", expiresIn: 0, want: 30 * time.Minute},
		{name: "negative uses fallback", expiresIn: -time.Second, want: 30 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, service.SupplierProviderTokenTTL(tt.expiresIn))
		})
	}
}

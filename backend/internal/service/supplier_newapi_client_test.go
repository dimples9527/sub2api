package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSupplierNewAPIClientStopsWhenRedisIsUnavailable(t *testing.T) {
	tests := []struct {
		name               string
		configure          func(*supplierSub2APIFakeTokenCache)
		expectedLoginCalls int32
	}{
		{
			name: "get failure",
			configure: func(cache *supplierSub2APIFakeTokenCache) {
				cache.getErr = errors.New("redis get unavailable")
			},
		},
		{
			name: "lock failure",
			configure: func(cache *supplierSub2APIFakeTokenCache) {
				cache.lockErr = errors.New("redis lock unavailable")
			},
		},
		{
			name: "set failure",
			configure: func(cache *supplierSub2APIFakeTokenCache) {
				cache.setErr = errors.New("redis set unavailable")
			},
			expectedLoginCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newSupplierSub2APIFakeTokenCache()
			tt.configure(cache)
			var loginCalls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/api/user/login" {
					loginCalls.Add(1)
					_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"newapi-token","access_expires_at":4102444800,"user":{"id":42}}}`))
					return
				}
				_, _ = w.Write([]byte(`{"success":true,"data":{"quota":500000}}`))
			}))
			defer server.Close()

			client := NewSupplierNewAPIClient(server.Client(), cache, nil)
			_, err := client.FetchBalance(context.Background(), supplierNewAPICacheTestProvider(server.URL), "secret")
			require.Error(t, err)
			require.False(t, IsSupplierProviderAuthFailure(err))
			require.True(t, IsSupplierProviderSessionFailure(err))
			require.Equal(t, tt.expectedLoginCalls, loginCalls.Load())
		})
	}
}

func TestSupplierNewAPIClientRecordsCacheMissAndLoginSuccess(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	auditor := &supplierProviderAuthAuditorSpy{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"audited-token","access_expires_at":4102444800,"user":{"id":42}}}`))
		case "/api/user/self":
			_, _ = w.Write([]byte(`{"success":true,"data":{"quota":500000}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierNewAPIClient(server.Client(), cache, nil)
	client.SetAuthAuditor(auditor)
	_, err := client.FetchBalance(context.Background(), supplierNewAPICacheTestProvider(server.URL), "secret")
	require.NoError(t, err)
	require.Contains(t, auditor.eventTypes(), SupplierProviderAuthEventCacheMiss)
	require.Contains(t, auditor.eventTypes(), SupplierProviderAuthEventLoginSuccess)
}

type supplierTurnstileSolverStub struct {
	token string
}

func (s supplierTurnstileSolverStub) PrepareToken(context.Context, *SupplierProvider, string, func(context.Context) (string, error)) (string, error) {
	return s.token, nil
}

func supplierNewAPICacheTestProvider(baseURL string) *SupplierProvider {
	return &SupplierProvider{
		ID:           42,
		Code:         "supplier-newapi-cache",
		ProviderType: SupplierProviderTypeNewAPI,
		BaseURL:      baseURL,
		LoginURL:     "/api/user/login",
		BalanceURL:   "/api/user/self",
		Username:     "root",
	}
}

func TestSupplierNewAPIClientRecordsRefreshAuditWithoutRefreshToken(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{
		AccessToken:  "old-access-token",
		RefreshToken: "old-refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(time.Minute),
		UserID:       42,
	})
	auditor := &supplierProviderAuthAuditorSpy{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/auth/refresh":
			require.Equal(t, "new_api_refresh=old-refresh-token", r.Header.Get("Cookie"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"fresh-access-token","access_expires_at":4102444800,"user":{"id":42}}}`))
		case "/api/user/login":
			t.Fatalf("刷新成功时不应重新登录")
		case "/api/user/self":
			require.Equal(t, "Bearer fresh-access-token", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"quota":500000}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierNewAPIClient(server.Client(), cache, nil)
	client.SetAuthAuditor(auditor)
	_, err := client.FetchBalance(context.Background(), supplierNewAPICacheTestProvider(server.URL), "secret")

	require.NoError(t, err)
	auditor.mu.Lock()
	events := append([]SupplierProviderAuthEventInput(nil), auditor.events...)
	auditor.mu.Unlock()
	var refreshEvent *SupplierProviderAuthEventInput
	for i := range events {
		if events[i].EventType == SupplierProviderAuthEventRefreshSuccess {
			refreshEvent = &events[i]
			break
		}
	}
	require.NotNil(t, refreshEvent)
	require.NotNil(t, refreshEvent.Token)
	require.Equal(t, "fresh-access-token", refreshEvent.Token.AccessToken)
	require.Empty(t, refreshEvent.Token.RefreshToken)
}
func TestSupplierNewAPIClientRedactsRefreshTokenFromRefreshFailureErrorAndAudit(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	auditor := &supplierProviderAuthAuditorSpy{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/user/auth/refresh", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false,"message":"refresh_token=unredacted-refresh"}`))
	}))
	defer server.Close()

	client := NewSupplierNewAPIClient(server.Client(), cache, nil)
	client.SetAuthAuditor(auditor)
	provider := supplierNewAPICacheTestProvider(server.URL)
	_, err := client.refreshSessionWithAudit(context.Background(), provider, supplierNewAPISession{
		UserID:       42,
		AccessToken:  "old-access-token",
		RefreshToken: "old-refresh-token",
	})

	require.Error(t, err)
	require.NotContains(t, err.Error(), "unredacted-refresh")
	auditor.mu.Lock()
	events := append([]SupplierProviderAuthEventInput(nil), auditor.events...)
	auditor.mu.Unlock()
	require.Len(t, events, 1)
	require.Equal(t, SupplierProviderAuthEventRefreshFailed, events[0].EventType)
	require.Error(t, events[0].Error)
	require.NotContains(t, events[0].Error.Error(), "unredacted-refresh")
	require.NotNil(t, events[0].Token)
	require.Empty(t, events[0].Token.RefreshToken)
}

func TestSupplierNewAPIClientRedactsNewAPIRefreshCookieFromRefreshFailureErrorAndAudit(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	auditor := &supplierProviderAuthAuditorSpy{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/user/auth/refresh", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false,"message":"new_api_refresh=unredacted-new-api-refresh-cookie"}`))
	}))
	defer server.Close()

	client := NewSupplierNewAPIClient(server.Client(), cache, nil)
	client.SetAuthAuditor(auditor)
	provider := supplierNewAPICacheTestProvider(server.URL)
	_, err := client.refreshSessionWithAudit(context.Background(), provider, supplierNewAPISession{
		UserID:       42,
		AccessToken:  "old-access-token",
		RefreshToken: "old-refresh-token",
	})

	require.Error(t, err)
	require.NotContains(t, err.Error(), "unredacted-new-api-refresh-cookie")
	auditor.mu.Lock()
	events := append([]SupplierProviderAuthEventInput(nil), auditor.events...)
	auditor.mu.Unlock()
	require.Len(t, events, 1)
	require.Equal(t, SupplierProviderAuthEventRefreshFailed, events[0].EventType)
	require.Error(t, events[0].Error)
	require.NotContains(t, events[0].Error.Error(), "unredacted-new-api-refresh-cookie")
	require.NotNil(t, events[0].Token)
	require.Empty(t, events[0].Token.RefreshToken)
}
func TestSupplierNewAPIClientReusesSessionFromSharedTokenCache(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	var loginCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			loginCalls.Add(1)
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "cached-session"})
			http.SetCookie(w, &http.Cookie{Name: "new_api_refresh", Value: "cached-refresh"})
			_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"newapi-token","access_expires_at":4102444800,"user":{"id":42}}}`))
		case "/api/user/self":
			require.Equal(t, "42", r.Header.Get("New-Api-User"))
			require.Equal(t, "Bearer newapi-token", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"quota":500000}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	provider := supplierNewAPICacheTestProvider(server.URL)
	firstRegistry := NewSupplierProviderRemoteRegistry(server.Client(), cache, nil)
	secondRegistry := NewSupplierProviderRemoteRegistry(server.Client(), cache, nil)

	firstBalance, err := firstRegistry.FetchBalance(context.Background(), provider, "secret")
	require.NoError(t, err)
	require.Equal(t, float64(1), firstBalance)
	secondBalance, err := secondRegistry.FetchBalance(context.Background(), provider, "secret")
	require.NoError(t, err)
	require.Equal(t, float64(1), secondBalance)

	require.Equal(t, int32(1), loginCalls.Load())
	cache.mu.Lock()
	cachedToken := cache.tokens[provider.ID]
	setTTLs := append([]time.Duration(nil), cache.setTTLs...)
	cache.mu.Unlock()
	require.Equal(t, "newapi-token", cachedToken.AccessToken)
	require.Equal(t, "cached-refresh", cachedToken.RefreshToken)
	require.Equal(t, int64(42), cachedToken.UserID)
	require.Equal(t, "session=cached-session", cachedToken.CookieHeader)
	require.Equal(t, time.Unix(4102444800, 0), cachedToken.ExpiresAt)
	require.Equal(t, []time.Duration{0}, setTTLs)
}

func TestSupplierNewAPIClientReusesLegacySessionWithoutRefreshTokenUntilAPIRejects(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{
		AccessToken:  "expired-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(-time.Minute),
		UserID:       42,
		CookieHeader: "session=expired",
	})
	var loginCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			loginCalls.Add(1)
			_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"refreshed-token","access_expires_at":4102444800,"user":{"id":42}}}`))
		case "/api/user/self":
			// 时间已过期的缓存仍应直接复用，直到接口真正返回鉴权失败。
			require.Equal(t, "Bearer expired-token", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"quota":500000}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	balance, err := NewSupplierProviderRemoteRegistry(server.Client(), cache, nil).FetchBalance(
		context.Background(), supplierNewAPICacheTestProvider(server.URL), "secret",
	)

	require.NoError(t, err)
	require.Equal(t, float64(1), balance)
	require.Equal(t, int32(0), loginCalls.Load())
	cache.mu.Lock()
	setCalls := cache.setCalls
	cache.mu.Unlock()
	require.Equal(t, 0, setCalls)
}

func TestSupplierNewAPIClientRefreshesNearExpiredCachedSession(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	refreshExpiresAt := time.Now().Add(30 * time.Minute).Truncate(time.Second)
	cache.preload(42, SupplierProviderAuthToken{
		AccessToken:  "old-access-token",
		RefreshToken: "old-refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(time.Minute),
		UserID:       42,
		CookieHeader: "session=legacy-session",
	})
	var refreshCalls atomic.Int32
	var loginCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/auth/refresh":
			refreshCalls.Add(1)
			require.Equal(t, http.MethodPost, r.Method)
			require.Empty(t, r.Header.Get("Authorization"))
			require.Equal(t, "42", r.Header.Get("New-Api-User"))
			require.Contains(t, r.Header.Get("Cookie"), "new_api_refresh=old-refresh-token")
			http.SetCookie(w, &http.Cookie{Name: "new_api_refresh", Value: "rotated-refresh-token"})
			_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"new-access-token","access_expires_at":` + strconv.FormatInt(refreshExpiresAt.Unix(), 10) + `,"user":{"id":42}}}`))
		case "/api/user/login":
			loginCalls.Add(1)
			t.Fatalf("不应在刷新成功时重新登录")
		case "/api/user/self":
			require.Equal(t, "Bearer new-access-token", r.Header.Get("Authorization"))
			require.Empty(t, r.Header.Get("Cookie"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"quota":500000}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	balance, err := NewSupplierProviderRemoteRegistry(server.Client(), cache, nil).FetchBalance(
		context.Background(), supplierNewAPICacheTestProvider(server.URL), "secret",
	)

	require.NoError(t, err)
	require.Equal(t, float64(1), balance)
	require.Equal(t, int32(1), refreshCalls.Load())
	require.Equal(t, int32(0), loginCalls.Load())
	cache.mu.Lock()
	cachedToken := cache.tokens[42]
	setCalls := cache.setCalls
	cache.mu.Unlock()
	require.Equal(t, "new-access-token", cachedToken.AccessToken)
	require.Equal(t, "rotated-refresh-token", cachedToken.RefreshToken)
	require.Equal(t, refreshExpiresAt, cachedToken.ExpiresAt)
	require.Equal(t, 1, setCalls)
}

func TestSupplierNewAPIClientFallsBackToLoginWhenRefreshFails(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{
		AccessToken:  "old-access-token",
		RefreshToken: "expired-refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(time.Minute),
		UserID:       42,
	})
	var refreshCalls atomic.Int32
	var loginCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/auth/refresh":
			refreshCalls.Add(1)
			require.Contains(t, r.Header.Get("Cookie"), "new_api_refresh=expired-refresh-token")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"success":false,"message":"refresh token expired"}`))
		case "/api/user/login":
			loginCalls.Add(1)
			http.SetCookie(w, &http.Cookie{Name: "new_api_refresh", Value: "login-refresh-token"})
			_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"login-access-token","access_expires_at":4102444800,"user":{"id":42}}}`))
		case "/api/user/self":
			require.Equal(t, "Bearer login-access-token", r.Header.Get("Authorization"))
			require.Empty(t, r.Header.Get("Cookie"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"quota":500000}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	balance, err := NewSupplierProviderRemoteRegistry(server.Client(), cache, nil).FetchBalance(
		context.Background(), supplierNewAPICacheTestProvider(server.URL), "secret",
	)

	require.NoError(t, err)
	require.Equal(t, float64(1), balance)
	require.Equal(t, int32(1), refreshCalls.Load())
	require.Equal(t, int32(1), loginCalls.Load())
	cache.mu.Lock()
	cachedToken := cache.tokens[42]
	cache.mu.Unlock()
	require.Equal(t, "login-access-token", cachedToken.AccessToken)
	require.Equal(t, "login-refresh-token", cachedToken.RefreshToken)
}

func TestSupplierNewAPIClientRefreshesAfterAuthFailureBeforeLogin(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{
		AccessToken:  "revoked-access-token",
		RefreshToken: "valid-refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(30 * time.Minute),
		UserID:       42,
	})
	var refreshCalls atomic.Int32
	var loginCalls atomic.Int32
	var balanceCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/auth/refresh":
			refreshCalls.Add(1)
			require.Equal(t, http.MethodPost, r.Method)
			require.Empty(t, r.Header.Get("Authorization"))
			require.Equal(t, "42", r.Header.Get("New-Api-User"))
			require.Equal(t, "new_api_refresh=valid-refresh-token", r.Header.Get("Cookie"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"recovered-access-token","access_expires_at":4102444800,"user":{"id":42}}}`))
		case "/api/user/login":
			loginCalls.Add(1)
			http.SetCookie(w, &http.Cookie{Name: "new_api_refresh", Value: "login-refresh-token"})
			_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"login-access-token","access_expires_at":4102444800,"user":{"id":42}}}`))
		case "/api/user/self":
			if balanceCalls.Add(1) == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"success":false,"message":"unauthorized"}`))
				return
			}
			require.Equal(t, "Bearer recovered-access-token", r.Header.Get("Authorization"))
			require.Empty(t, r.Header.Get("Cookie"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"quota":500000}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	balance, err := NewSupplierProviderRemoteRegistry(server.Client(), cache, nil).FetchBalance(
		context.Background(), supplierNewAPICacheTestProvider(server.URL), "secret",
	)

	require.NoError(t, err)
	require.Equal(t, float64(1), balance)
	require.Equal(t, int32(1), refreshCalls.Load())
	require.Equal(t, int32(0), loginCalls.Load())
	require.Equal(t, int32(2), balanceCalls.Load())
	cache.mu.Lock()
	cachedToken := cache.tokens[42]
	cache.mu.Unlock()
	require.Equal(t, "recovered-access-token", cachedToken.AccessToken)
}

func TestSupplierNewAPIClientRefreshesGroupsAfterAuthFailureBeforeLogin(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{
		AccessToken:  "revoked-access-token",
		RefreshToken: "valid-refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(30 * time.Minute),
		UserID:       42,
	})
	var refreshCalls atomic.Int32
	var loginCalls atomic.Int32
	var groupCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/auth/refresh":
			refreshCalls.Add(1)
			require.Equal(t, "new_api_refresh=valid-refresh-token", r.Header.Get("Cookie"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"recovered-access-token","access_expires_at":4102444800,"user":{"id":42}}}`))
		case "/api/user/login":
			loginCalls.Add(1)
			t.Errorf("刷新会话成功时不应重新登录")
		case "/api/group/":
			if groupCalls.Add(1) == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"success":false,"message":"unauthorized"}`))
				return
			}
			require.Equal(t, "Bearer recovered-access-token", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"default":{"id":1,"ratio":1,"status":1}}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	provider := supplierNewAPICacheTestProvider(server.URL)
	provider.GroupsURL = "/api/group/"
	groups, err := NewSupplierProviderRemoteRegistry(server.Client(), cache, nil).FetchGroups(
		context.Background(), provider, "secret",
	)

	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, int32(1), refreshCalls.Load())
	require.Equal(t, int32(0), loginCalls.Load())
	require.Equal(t, int32(2), groupCalls.Load())
}
func TestSupplierNewAPIClientDeletesInvalidSessionAndRetriesLogin(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{
		AccessToken: "cached-token",
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(20 * time.Minute),
		UserID:      42,
	})
	var loginCalls atomic.Int32
	var balanceCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			loginCalls.Add(1)
			_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"refreshed-token","access_expires_at":4102444800,"user":{"id":42}}}`))
		case "/api/user/self":
			if balanceCalls.Add(1) == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"success":false,"message":"unauthorized"}`))
				return
			}
			require.Equal(t, "Bearer refreshed-token", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"quota":500000}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	balance, err := NewSupplierProviderRemoteRegistry(server.Client(), cache, nil).FetchBalance(
		context.Background(), supplierNewAPICacheTestProvider(server.URL), "secret",
	)

	require.NoError(t, err)
	require.Equal(t, float64(1), balance)
	require.Equal(t, int32(1), loginCalls.Load())
	require.Equal(t, int32(2), balanceCalls.Load())
	cache.mu.Lock()
	deleteCalls := cache.deleteCalls
	cache.mu.Unlock()
	require.Equal(t, 1, deleteCalls)
}

func TestSupplierNewAPIClientMarksFinalUnauthorizedAsAuthFailure(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{
		AccessToken: "cached-token",
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(20 * time.Minute),
		UserID:      42,
	})
	var loginCalls atomic.Int32
	var balanceCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			loginCalls.Add(1)
			_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"refreshed-token","access_expires_at":4102444800,"user":{"id":42}}}`))
		case "/api/user/self":
			balanceCalls.Add(1)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"success":false,"message":"unauthorized"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	_, err := NewSupplierProviderRemoteRegistry(server.Client(), cache, nil).FetchBalance(
		context.Background(), supplierNewAPICacheTestProvider(server.URL), "secret",
	)

	require.Error(t, err)
	require.True(t, IsSupplierProviderAuthFailure(err))
	require.Equal(t, int32(1), loginCalls.Load())
	require.Equal(t, int32(2), balanceCalls.Load())
}

func TestSupplierNewAPIClientClearsSessionAndMarksFinalUnauthorizedForAccountsAndGroups(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		fetch func(*SupplierNewAPIClient, context.Context, *SupplierProvider) error
	}{
		{
			name: "accounts",
			path: defaultSupplierNewAPIKeysPath,
			fetch: func(client *SupplierNewAPIClient, ctx context.Context, provider *SupplierProvider) error {
				_, err := client.FetchAccounts(ctx, provider, "secret")
				return err
			},
		},
		{
			name: "groups",
			path: defaultSupplierNewAPIGroupsPath,
			fetch: func(client *SupplierNewAPIClient, ctx context.Context, provider *SupplierProvider) error {
				_, err := client.FetchGroups(ctx, provider, "secret")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newSupplierSub2APIFakeTokenCache()
			cache.preload(42, SupplierProviderAuthToken{
				AccessToken:  "cached-token",
				RefreshToken: "old-refresh-token",
				TokenType:    "Bearer",
				ExpiresAt:    time.Now().Add(20 * time.Minute),
				UserID:       42,
			})
			var refreshCalls atomic.Int32
			var requestCalls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case defaultSupplierNewAPIRefreshPath:
					refreshCalls.Add(1)
					_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"fresh-token","access_expires_at":4102444800,"user":{"id":42}}}`))
				case tt.path:
					requestCalls.Add(1)
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"success":false,"message":"unauthorized"}`))
				default:
					t.Fatalf("unexpected path %s", r.URL.Path)
				}
			}))
			defer server.Close()

			provider := supplierNewAPICacheTestProvider(server.URL)
			if tt.name == "groups" {
				provider.GroupsURL = defaultSupplierNewAPIGroupsPath
			}
			err := tt.fetch(NewSupplierNewAPIClient(server.Client(), cache, nil), context.Background(), provider)

			require.Error(t, err)
			require.True(t, IsSupplierProviderAuthFailure(err))
			require.Equal(t, int32(1), refreshCalls.Load())
			require.Equal(t, int32(2), requestCalls.Load())
			cache.mu.Lock()
			deleteCalls := cache.deleteCalls
			cache.mu.Unlock()
			require.Equal(t, 1, deleteCalls)
		})
	}
}

func TestSupplierNewAPIClientMarksLoginCredentialFailureAsAuthFailure(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"success":false,"message":"invalid username or password"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	_, err := NewSupplierProviderRemoteRegistry(server.Client(), cache, nil).FetchBalance(
		context.Background(), supplierNewAPICacheTestProvider(server.URL), "secret",
	)

	require.Error(t, err)
	require.True(t, IsSupplierProviderAuthFailure(err))
}

func TestSupplierNewAPIClientUsesSharedLoginLockForRefresh(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{
		AccessToken:  "old-access-token",
		RefreshToken: "old-refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(time.Minute),
		UserID:       42,
	})
	refreshExpiresAt := time.Now().Add(30 * time.Minute).Truncate(time.Second)
	var refreshCalls atomic.Int32
	var loginCalls atomic.Int32
	refreshStarted := make(chan struct{}, 2)
	releaseRefresh := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/auth/refresh":
			refreshCalls.Add(1)
			require.Equal(t, "new_api_refresh=old-refresh-token", r.Header.Get("Cookie"))
			refreshStarted <- struct{}{}
			<-releaseRefresh
			_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"new-access-token","access_expires_at":` + strconv.FormatInt(refreshExpiresAt.Unix(), 10) + `,"user":{"id":42}}}`))
		case "/api/user/login":
			loginCalls.Add(1)
			t.Errorf("刷新会话成功时不应重新登录")
		case "/api/user/self":
			require.Equal(t, "Bearer new-access-token", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"quota":500000}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	provider := supplierNewAPICacheTestProvider(server.URL)
	firstRegistry := NewSupplierProviderRemoteRegistry(server.Client(), cache, nil)
	secondRegistry := NewSupplierProviderRemoteRegistry(server.Client(), cache, nil)
	results := make(chan error, 2)
	var waitGroup sync.WaitGroup
	waitGroup.Add(2)
	go func() {
		defer waitGroup.Done()
		_, err := firstRegistry.FetchBalance(context.Background(), provider, "secret")
		results <- err
	}()
	select {
	case <-refreshStarted:
	case <-time.After(time.Second):
		t.Fatal("第一次会话刷新没有开始")
	}
	go func() {
		defer waitGroup.Done()
		_, err := secondRegistry.FetchBalance(context.Background(), provider, "secret")
		results <- err
	}()

	secondRefreshStarted := false
	select {
	case <-refreshStarted:
		secondRefreshStarted = true
	case <-time.After(500 * time.Millisecond):
	}
	close(releaseRefresh)
	waitGroup.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	require.False(t, secondRefreshStarted)
	require.Equal(t, int32(1), refreshCalls.Load())
	require.Equal(t, int32(0), loginCalls.Load())
}
func TestSupplierNewAPIClientUsesSharedLoginLock(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	var loginCalls atomic.Int32
	loginStarted := make(chan struct{}, 2)
	releaseLogin := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			loginCalls.Add(1)
			loginStarted <- struct{}{}
			<-releaseLogin
			_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"lock-token","access_expires_at":4102444800,"user":{"id":42}}}`))
		case "/api/user/self":
			require.Equal(t, "Bearer lock-token", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"quota":500000}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	provider := supplierNewAPICacheTestProvider(server.URL)
	firstRegistry := NewSupplierProviderRemoteRegistry(server.Client(), cache, nil)
	secondRegistry := NewSupplierProviderRemoteRegistry(server.Client(), cache, nil)
	results := make(chan error, 2)
	var waitGroup sync.WaitGroup
	waitGroup.Add(2)
	go func() {
		defer waitGroup.Done()
		_, err := firstRegistry.FetchBalance(context.Background(), provider, "secret")
		results <- err
	}()
	select {
	case <-loginStarted:
	case <-time.After(time.Second):
		t.Fatal("first NewAPI login did not start")
	}
	go func() {
		defer waitGroup.Done()
		_, err := secondRegistry.FetchBalance(context.Background(), provider, "secret")
		results <- err
	}()

	secondLoginStarted := false
	select {
	case <-loginStarted:
		secondLoginStarted = true
	case <-time.After(500 * time.Millisecond):
	}
	close(releaseLogin)
	waitGroup.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	require.False(t, secondLoginStarted)
	require.Equal(t, int32(1), loginCalls.Load())
}

func TestSupplierNewAPIClientFetchesAndParsesProviderData(t *testing.T) {
	var loginCalls int
	var accountCalls int
	var groupCalls int
	var balanceCalls int
	var costCalls int
	day := time.Date(2026, 6, 17, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			loginCalls++
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":42}}`))
		case "/api/token/":
			accountCalls++
			require.Equal(t, "42", r.Header.Get("New-Api-User"))
			require.Contains(t, r.Header.Get("Cookie"), "session=abc")
			_, _ = w.Write([]byte(`{"success":true,"data":{"items":[
				{"name":"key-1","group":"VIP","status":1,"key":"sk-secret-must-not-return"},
				{"name":"key-2","group":"trial","status":2}
			]}}`))
		case "/api/group/":
			groupCalls++
			require.Equal(t, "42", r.Header.Get("New-Api-User"))
			require.Contains(t, r.Header.Get("Cookie"), "session=abc")
			_, _ = w.Write([]byte(`{"success":true,"data":{
				"VIP":{"id":7,"ratio":"3.25"},
				"Trial":{"id":8,"ratio":0.75},
				"Archive":{"id":9,"ratio":1.0,"status":"disabled"}
			}}`))
		case "/api/user/self":
			balanceCalls++
			require.Equal(t, "42", r.Header.Get("New-Api-User"))
			require.Contains(t, r.Header.Get("Cookie"), "session=abc")
			_, _ = w.Write([]byte(`{"success":true,"data":{"quota":9402397}}`))
		case "/api/log/self/stat":
			costCalls++
			require.Equal(t, "1781625600", r.URL.Query().Get("start_timestamp"))
			require.Equal(t, "1781711999", r.URL.Query().Get("end_timestamp"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"quota":1306899}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	provider := &SupplierProvider{
		ID:                42,
		Code:              "supplier-newapi",
		Name:              "NewAPI main",
		ProviderType:      "newapi",
		BaseURL:           server.URL,
		LoginURL:          "/api/user/login",
		APIKeysURL:        "/api/token/",
		GroupsURL:         "/api/group/",
		BalanceURL:        "/api/user/self",
		UsageCostURL:      "/api/log/self/stat?type=0&token_name=&model_name=&start_timestamp={start_timestamp}&end_timestamp={end_timestamp}&group=",
		Username:          "root",
		AccountNamePrefix: "ignored-prefix",
	}
	client := NewSupplierNewAPIClient(server.Client(), nil, nil)

	accounts, err := client.FetchAccounts(context.Background(), provider, "secret")
	require.NoError(t, err)
	require.Equal(t, []SupplierProviderRemoteAccount{{
		Key:            "key-1",
		Name:           "key-1",
		Status:         "active",
		GroupKey:       "7",
		GroupName:      "VIP",
		RateMultiplier: 3.25,
		RawStatus:      "1",
	}, {
		Key:            "key-2",
		Name:           "key-2",
		Status:         "disabled",
		GroupKey:       "8",
		GroupName:      "trial",
		RateMultiplier: 0.75,
		RawStatus:      "2",
	}}, accounts)
	for _, account := range accounts {
		require.NotContains(t, strings.ToLower(account.Key), "sk-secret")
	}

	groups, err := client.FetchGroups(context.Background(), provider, "secret")
	require.NoError(t, err)
	require.Equal(t, []SupplierProviderRemoteGroup{{
		Key:            "8",
		Name:           "Trial",
		RateMultiplier: 0.75,
		RawStatus:      "",
	}, {
		Key:            "7",
		Name:           "VIP",
		RateMultiplier: 3.25,
		RawStatus:      "",
	}}, groups)

	balance, err := client.FetchBalance(context.Background(), provider, "secret")
	require.NoError(t, err)
	require.Equal(t, 18.804794, balance)

	cost, err := client.FetchCost(context.Background(), provider, "secret", day)
	require.NoError(t, err)
	require.Equal(t, 2.613798, cost)
	require.Equal(t, 1, loginCalls)
	require.Equal(t, 1, accountCalls)
	require.Equal(t, 2, groupCalls)
	require.Equal(t, 1, balanceCalls)
	require.Equal(t, 1, costCalls)
}

func TestSupplierNewAPIClientTestEndpointRefreshesAfterAuthFailureBeforeLogin(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{
		AccessToken:  "revoked-access-token",
		RefreshToken: "valid-refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(30 * time.Minute),
		UserID:       42,
	})
	var refreshCalls atomic.Int32
	var loginCalls atomic.Int32
	var balanceCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/auth/refresh":
			refreshCalls.Add(1)
			require.Equal(t, "new_api_refresh=valid-refresh-token", r.Header.Get("Cookie"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"recovered-access-token","access_expires_at":4102444800,"user":{"id":42}}}`))
		case "/api/user/login":
			loginCalls.Add(1)
			t.Fatal("refresh succeeds, so endpoint test must not fall back to password login")
		case "/api/user/self":
			if balanceCalls.Add(1) == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"success":false,"message":"unauthorized"}`))
				return
			}
			require.Equal(t, "Bearer recovered-access-token", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"quota":500000}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	result, err := NewSupplierNewAPIClient(server.Client(), cache, nil).TestEndpoint(
		context.Background(), supplierNewAPICacheTestProvider(server.URL), "secret", SupplierSyncScopeBalance,
	)

	require.NoError(t, err)
	require.Empty(t, result.Error)
	require.Equal(t, http.StatusOK, result.HTTPStatus)
	require.Equal(t, int32(1), refreshCalls.Load())
	require.Equal(t, int32(0), loginCalls.Load())
	require.Equal(t, int32(2), balanceCalls.Load())
}
func TestSupplierNewAPIClientTestEndpointCountsAccountsWithoutGroupsPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":42}}`))
		case "/api/token/":
			_, _ = w.Write([]byte(`{"success":true,"data":{"items":[
				{"name":"key-1","group":"VIP"},
				{"name":"key-2","group":"Trial"}
			]}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	provider := &SupplierProvider{
		ID:           42,
		Code:         "supplier-newapi",
		ProviderType: "newapi",
		BaseURL:      server.URL,
		LoginURL:     "/api/user/login",
		APIKeysURL:   "/api/token/",
		Username:     "root",
	}
	client := NewSupplierNewAPIClient(server.Client(), nil, nil)

	result, err := client.TestEndpoint(context.Background(), provider, "secret", SupplierSyncScopeAccounts)

	require.NoError(t, err)
	require.Empty(t, result.ParseError)
	require.Equal(t, map[string]any{
		"count": 2,
		"items": []map[string]string{
			{"name": "key-1", "group": "VIP"},
			{"name": "key-2", "group": "Trial"},
		},
	}, result.ParsedData)
}

func TestSupplierNewAPIClientAcceptsNestedUserIDFromTurnstileLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			require.Equal(t, "turnstile-token", r.URL.Query().Get("turnstile"))
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			_, _ = w.Write([]byte(`{"success":true,"data":{"user":{"id":42}}}`))
		case "/api/user/self":
			require.Equal(t, "42", r.Header.Get("New-Api-User"))
			require.Contains(t, r.Header.Get("Cookie"), "session=abc")
			_, _ = w.Write([]byte(`{"success":true,"data":{"quota":500000}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	provider := &SupplierProvider{
		ID:               42,
		Code:             "supplier-newapi-turnstile",
		ProviderType:     "newapi",
		BaseURL:          server.URL,
		LoginURL:         "/api/user/login",
		BalanceURL:       "/api/user/self",
		Username:         "root",
		TurnstileEnabled: true,
	}
	client := NewSupplierNewAPIClient(server.Client(), nil, supplierTurnstileSolverStub{token: "turnstile-token"})

	balance, err := client.FetchBalance(context.Background(), provider, "secret")

	require.NoError(t, err)
	require.Equal(t, float64(1), balance)
}

func TestSupplierNewAPIClientUsesAccessTokenFromTurnstileLogin(t *testing.T) {
	var loginCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			loginCalls++
			require.Equal(t, "turnstile-token", r.URL.Query().Get("turnstile"))
			http.SetCookie(w, &http.Cookie{Name: "new_api_refresh", Value: "refresh-token"})
			_, _ = w.Write([]byte(`{"success":true,"data":{"access_token":"jwt-token","access_expires_at":1800000000,"session":{"sid":"sess-1"},"user":{"id":42}}}`))
		case "/api/user/self":
			require.Equal(t, "Bearer jwt-token", r.Header.Get("Authorization"))
			require.Empty(t, r.Header.Get("Cookie"))
			require.Equal(t, "42", r.Header.Get("New-Api-User"))
			_, _ = w.Write([]byte(`{"success":true,"data":{"quota":500000}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	provider := &SupplierProvider{
		ID:               42,
		Code:             "supplier-newapi-access-token",
		ProviderType:     "newapi",
		BaseURL:          server.URL,
		LoginURL:         "/api/user/login",
		BalanceURL:       "/api/user/self",
		Username:         "root",
		TurnstileEnabled: true,
	}
	client := NewSupplierNewAPIClient(server.Client(), nil, supplierTurnstileSolverStub{token: "turnstile-token"})

	firstBalance, err := client.FetchBalance(context.Background(), provider, "secret")
	require.NoError(t, err)
	require.Equal(t, float64(1), firstBalance)
	secondBalance, err := client.FetchBalance(context.Background(), provider, "secret")
	require.NoError(t, err)
	require.Equal(t, float64(1), secondBalance)
	require.Equal(t, 1, loginCalls)
}

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type supplierSub2APIFakeTokenCache struct {
	mu sync.Mutex

	tokens map[int64]SupplierProviderAuthToken
	locks  map[int64]string

	getErr     error
	setErr     error
	deleteErr  error
	lockErr    error
	releaseErr error

	getCalls     int
	setCalls     int
	deleteCalls  int
	lockCalls    int
	releaseCalls int

	setTTLs        []time.Duration
	lockTTLs       []time.Duration
	acquiredOwners []string
	releasedOwners []string
}

type supplierProviderAuthAuditorSpy struct {
	mu     sync.Mutex
	events []SupplierProviderAuthEventInput
}

func (s *supplierProviderAuthAuditorSpy) Record(_ context.Context, event SupplierProviderAuthEventInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *supplierProviderAuthAuditorSpy) eventTypes() []SupplierProviderAuthEventType {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]SupplierProviderAuthEventType, 0, len(s.events))
	for _, event := range s.events {
		items = append(items, event.EventType)
	}
	return items
}

func TestSupplierSub2APIClientRedactsRefreshTokenBeforeAuditing(t *testing.T) {
	auditor := &supplierProviderAuthAuditorSpy{}
	client := NewSupplierSub2APIClient(nil, nil, nil)
	client.SetAuthAuditor(auditor)
	token := SupplierProviderAuthToken{
		AccessToken:  "access-token",
		RefreshToken: "unredacted-refresh",
	}

	client.recordAuthEvent(context.Background(), &SupplierProvider{ID: 42, Code: "supplier-sub2api"}, SupplierProviderAuthEventInput{
		EventType: SupplierProviderAuthEventRefreshFailed,
		Error:     errors.New("upstream refresh failed: refresh_token=unredacted-refresh"),
		Token:     &token,
	})

	auditor.mu.Lock()
	events := append([]SupplierProviderAuthEventInput(nil), auditor.events...)
	auditor.mu.Unlock()
	require.Len(t, events, 1)
	require.Error(t, events[0].Error)
	require.NotContains(t, events[0].Error.Error(), "unredacted-refresh")
	require.NotNil(t, events[0].Token)
	require.Empty(t, events[0].Token.RefreshToken)
}
func newSupplierSub2APIFakeTokenCache() *supplierSub2APIFakeTokenCache {
	return &supplierSub2APIFakeTokenCache{
		tokens: make(map[int64]SupplierProviderAuthToken),
		locks:  make(map[int64]string),
	}
}

func (c *supplierSub2APIFakeTokenCache) Get(_ context.Context, providerID int64) (SupplierProviderAuthToken, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.getCalls++
	if c.getErr != nil {
		return SupplierProviderAuthToken{}, false, c.getErr
	}
	token, found := c.tokens[providerID]
	return token, found, nil
}

func (c *supplierSub2APIFakeTokenCache) Set(_ context.Context, providerID int64, token SupplierProviderAuthToken, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.setCalls++
	c.setTTLs = append(c.setTTLs, ttl)
	if c.setErr != nil {
		return c.setErr
	}
	c.tokens[providerID] = token
	return nil
}

func (c *supplierSub2APIFakeTokenCache) Delete(_ context.Context, providerID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.deleteCalls++
	if c.deleteErr != nil {
		return c.deleteErr
	}
	delete(c.tokens, providerID)
	return nil
}

func (c *supplierSub2APIFakeTokenCache) TryAcquireLoginLock(_ context.Context, providerID int64, owner string, ttl time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lockCalls++
	c.lockTTLs = append(c.lockTTLs, ttl)
	if c.lockErr != nil {
		return false, c.lockErr
	}
	if _, locked := c.locks[providerID]; locked {
		return false, nil
	}
	c.locks[providerID] = owner
	c.acquiredOwners = append(c.acquiredOwners, owner)
	return true, nil
}

func (c *supplierSub2APIFakeTokenCache) ReleaseLoginLock(_ context.Context, providerID int64, owner string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.releaseCalls++
	c.releasedOwners = append(c.releasedOwners, owner)
	if c.releaseErr != nil {
		return c.releaseErr
	}
	if c.locks[providerID] == owner {
		delete(c.locks, providerID)
	}
	return nil
}

func (c *supplierSub2APIFakeTokenCache) preload(providerID int64, token SupplierProviderAuthToken) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tokens[providerID] = token
}

func (c *supplierSub2APIFakeTokenCache) holdLock(providerID int64, owner string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.locks[providerID] = owner
}

func supplierSub2APITestProvider(baseURL string) *SupplierProvider {
	return &SupplierProvider{
		ID:                 42,
		Code:               "supplier-a",
		BaseURL:            baseURL,
		Email:              "admin@example.com",
		Username:           "must-not-be-used",
		APIKeysURL:         "/accounts",
		GroupsURL:          "/groups",
		AvailableGroupsURL: "/available-groups",
		BalanceURL:         "/balance",
		UsageCostURL:       "/cost",
	}
}

func supplierSub2APIWriteJSON(w http.ResponseWriter, status int, payload string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(payload))
}

func TestSupplierSub2APIClientUsesThirtySecondDefaultTimeout(t *testing.T) {
	client := NewSupplierSub2APIClient(nil, newSupplierSub2APIFakeTokenCache(), nil)

	require.Equal(t, 30*time.Second, client.httpClient.Timeout)
}

func TestSupplierSub2APIClientLoginUsesEmailAndCachesToken(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	var loginPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			require.Equal(t, http.MethodPost, r.Method)
			require.NoError(t, json.NewDecoder(r.Body).Decode(&loginPayload))
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"fresh-token"},"expires_in":3600}`)
		case "/accounts":
			require.Equal(t, "Bearer fresh-token", r.Header.Get("Authorization"))
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"items":[]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(nil, cache, nil)
	_, err := client.FetchAccounts(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"email":    "admin@example.com",
		"password": "secret",
	}, loginPayload)
	require.NotContains(t, loginPayload, "username")

	cache.mu.Lock()
	defer cache.mu.Unlock()
	require.Equal(t, 1, cache.setCalls)
	require.Equal(t, []time.Duration{0}, cache.setTTLs)
	require.Equal(t, []time.Duration{supplierSub2APILoginLockTTL}, cache.lockTTLs)
	require.Len(t, cache.acquiredOwners, 1)
	require.Equal(t, cache.acquiredOwners, cache.releasedOwners)
	require.Equal(t, "fresh-token", cache.tokens[42].AccessToken)
	require.Equal(t, "Bearer", cache.tokens[42].TokenType)
}

func TestSupplierSub2APIClientFetchMonitorItemsUsesConfiguredMonitorURLAndBearerToken(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{AccessToken: "cached-token", TokenType: "Bearer", ExpiresAt: time.Now().Add(time.Hour)})
	var authorization string
	var requestURI string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		requestURI = r.URL.RequestURI()
		require.Equal(t, http.MethodGet, r.Method)
		supplierSub2APIWriteJSON(w, http.StatusOK, `{
			"code": 0,
			"message": "success",
			"data": {
				"items": [{
					"id": 39,
					"name": "grok对话",
					"provider": "grok",
					"group_name": "",
					"primary_model": "grok-4.5",
					"primary_status": "degraded",
					"primary_latency_ms": 10122,
					"primary_ping_latency_ms": 56,
					"availability_7d": 83.28358208955224,
					"timeline": [
						{"status":"degraded","latency_ms":10122,"ping_latency_ms":56,"checked_at":"2026-08-08T03:09:00Z"},
						{"status":"operational","latency_ms":4331,"ping_latency_ms":23,"checked_at":"2026-08-08T02:59:01Z"}
					]
				}]
			}
		}`)
	}))
	defer server.Close()

	provider := supplierSub2APITestProvider(server.URL)
	provider.MonitorURL = "/api/v1/channel-monitors?timezone=Asia%2FShanghai"
	client := NewSupplierSub2APIClient(server.Client(), cache, nil)

	items, err := client.FetchMonitorItems(context.Background(), provider, "secret")

	require.NoError(t, err)
	require.Equal(t, "Bearer cached-token", authorization)
	require.Equal(t, "/api/v1/channel-monitors?timezone=Asia%2FShanghai", requestURI)
	require.Len(t, items, 1)
	require.Equal(t, "39", items[0].Key)
	require.Equal(t, "grok对话", items[0].Name)
	require.Equal(t, "grok", items[0].Provider)
	require.Equal(t, "grok-4.5", items[0].PrimaryModel)
	require.Equal(t, "degraded", items[0].PrimaryStatus)
	require.Equal(t, int64(10122), items[0].PrimaryLatencyMS)
	require.Equal(t, int64(56), items[0].PrimaryPingLatencyMS)
	require.InDelta(t, 83.2835, items[0].Availability7D, 0.0001)
	require.Len(t, items[0].Timeline, 2)
	require.Equal(t, "operational", items[0].Timeline[1].Status)
	require.Equal(t, time.Date(2026, 8, 8, 3, 9, 0, 0, time.UTC), items[0].Timeline[0].CheckedAt)
}

func TestSupplierSub2APIClientMonitorProbeBlockedIsNotAuthFailure(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{AccessToken: "cached-token", TokenType: "Bearer", ExpiresAt: time.Now().Add(time.Hour)})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"login-token","token_type":"Bearer","expires_in":3600}}`)
		case "/api/v1/channel-monitors":
			require.Equal(t, "/api/v1/channel-monitors", r.URL.Path)
			require.Equal(t, "Bearer cached-token", r.Header.Get("Authorization"))
			supplierSub2APIWriteJSON(w, http.StatusForbidden, `{"error":{"message":"Probe, monitoring, and test traffic are disabled by site policy.","type":"probe_blocked"}}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	provider := supplierSub2APITestProvider(server.URL)
	provider.MonitorURL = "/api/v1/channel-monitors?timezone=Asia%2FShanghai"
	client := NewSupplierSub2APIClient(server.Client(), cache, nil)

	_, err := client.FetchMonitorItems(context.Background(), provider, "secret")

	require.Error(t, err)
	require.False(t, IsSupplierProviderAuthFailure(err))
	require.Contains(t, err.Error(), "probe_blocked")
	require.Equal(t, 0, cache.deleteCalls)
}

func TestSupplierSub2APIClientForbiddenInvalidTokenIsAuthFailure(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{AccessToken: "cached-token", TokenType: "Bearer", ExpiresAt: time.Now().Add(time.Hour)})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		supplierSub2APIWriteJSON(w, http.StatusForbidden, `{"code":403,"message":"invalid token"}`)
	}))
	defer server.Close()

	provider := supplierSub2APITestProvider(server.URL)
	client := NewSupplierSub2APIClient(server.Client(), cache, nil)

	_, err := client.FetchAccounts(context.Background(), provider, "secret")

	require.Error(t, err)
	require.True(t, IsSupplierProviderAuthFailure(err))
}

func TestSupplierSub2APIProbeBlockedClassifier(t *testing.T) {
	require.True(t, supplierSub2APIProbeBlocked([]byte(`{"error":{"message":"Probe, monitoring, and test traffic are disabled by site policy.","type":"probe_blocked"}}`)))
	require.False(t, supplierSub2APIProbeBlocked([]byte(`{"error":{"message":"invalid token","type":"auth_error"}}`)))
}

func TestSupplierSub2APIClientExtractsSupportedLoginTokenShapes(t *testing.T) {
	tests := []struct {
		name              string
		loginResponse     string
		wantAuthorization string
		wantTTL           time.Duration
	}{
		{
			name:              "nested access token and expiry",
			loginResponse:     `{"code":0,"data":{"access_token":"nested-access","token_type":"Token","expires_in":100}}`,
			wantAuthorization: "Token nested-access",
			wantTTL:           0,
		},
		{
			name:              "nested token",
			loginResponse:     `{"code":0,"data":{"token":"nested-token"},"expires_in":120}`,
			wantAuthorization: "Bearer nested-token",
			wantTTL:           0,
		},
		{
			name:              "top level access token",
			loginResponse:     `{"code":0,"access_token":"top-access","token_type":"JWT","expires_in":121}`,
			wantAuthorization: "JWT top-access",
			wantTTL:           0,
		},
		{
			name:              "top level token with fallback expiry",
			loginResponse:     `{"code":0,"token":"top-token"}`,
			wantAuthorization: "Bearer top-token",
			wantTTL:           0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newSupplierSub2APIFakeTokenCache()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/login":
					supplierSub2APIWriteJSON(w, http.StatusOK, tt.loginResponse)
				case "/balance":
					require.Equal(t, tt.wantAuthorization, r.Header.Get("Authorization"))
					supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"balance":12.5}}`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			provider := supplierSub2APITestProvider(server.URL)
			provider.LoginURL = "/login"
			client := NewSupplierSub2APIClient(nil, cache, nil)
			balance, err := client.FetchBalance(context.Background(), provider, "secret")

			require.NoError(t, err)
			require.Equal(t, 12.5, balance)
			cache.mu.Lock()
			defer cache.mu.Unlock()
			require.Equal(t, []time.Duration{tt.wantTTL}, cache.setTTLs)
		})
	}
}

func TestSupplierSub2APIClientReusesCachedToken(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{AccessToken: "cached-token", TokenType: "Bearer"})
	var loginCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			loginCalls.Add(1)
			supplierSub2APIWriteJSON(w, http.StatusInternalServerError, `{"message":"login must not be called"}`)
		case "/accounts":
			require.Equal(t, "Bearer cached-token", r.Header.Get("Authorization"))
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"items":[]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(nil, cache, nil)
	_, err := client.FetchAccounts(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

	require.NoError(t, err)
	require.Equal(t, int32(0), loginCalls.Load())
	cache.mu.Lock()
	defer cache.mu.Unlock()
	require.Equal(t, 0, cache.lockCalls)
	require.Equal(t, 0, cache.setCalls)
}

func TestSupplierSub2APIClientConcurrentRequestsLoginOnce(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	var loginCalls atomic.Int32
	var accountCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			loginCalls.Add(1)
			time.Sleep(150 * time.Millisecond)
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"shared-token"},"expires_in":3600}`)
		case "/accounts":
			accountCalls.Add(1)
			if r.Header.Get("Authorization") != "Bearer shared-token" {
				t.Errorf("unexpected authorization header: %q", r.Header.Get("Authorization"))
			}
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"items":[]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(nil, cache, nil)
	provider := supplierSub2APITestProvider(server.URL)
	const requestCount = 8
	start := make(chan struct{})
	errs := make(chan error, requestCount)

	for range requestCount {
		go func() {
			<-start
			_, err := client.FetchAccounts(context.Background(), provider, "secret")
			errs <- err
		}()
	}
	close(start)

	for range requestCount {
		require.NoError(t, <-errs)
	}
	require.Equal(t, int32(1), loginCalls.Load())
	require.Equal(t, int32(requestCount), accountCalls.Load())
}

func TestSupplierSub2APIClientDoesNotLoginWhileAnotherOwnerHoldsLock(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.holdLock(42, "other-owner")
	var loginCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/login" {
			loginCalls.Add(1)
		}
		supplierSub2APIWriteJSON(w, http.StatusInternalServerError, `{"message":"unexpected request"}`)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	client := NewSupplierSub2APIClient(nil, cache, nil)
	_, err := client.FetchBalance(ctx, supplierSub2APITestProvider(server.URL), "secret")

	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, int32(0), loginCalls.Load())
}

func TestSupplierSub2APIClientRetriesOnceAfterUnauthorized(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	var loginCalls atomic.Int32
	var accountCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			loginNumber := loginCalls.Add(1)
			supplierSub2APIWriteJSON(w, http.StatusOK, fmt.Sprintf(`{"code":0,"data":{"access_token":"token-%d"}}`, loginNumber))
		case "/accounts":
			accountNumber := accountCalls.Add(1)
			if accountNumber == 1 {
				require.Equal(t, "Bearer token-1", r.Header.Get("Authorization"))
				supplierSub2APIWriteJSON(w, http.StatusUnauthorized, `{"message":"unauthorized"}`)
				return
			}
			require.Equal(t, "Bearer token-2", r.Header.Get("Authorization"))
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"items":[]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(nil, cache, nil)
	_, err := client.FetchAccounts(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

	require.NoError(t, err)
	require.Equal(t, int32(2), loginCalls.Load())
	require.Equal(t, int32(2), accountCalls.Load())
	cache.mu.Lock()
	defer cache.mu.Unlock()
	require.Equal(t, 1, cache.deleteCalls)
}

func TestSupplierSub2APIClientFallsBackWhenConfiguredAccountsEndpoint404s(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	var paths []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" {
			paths = append(paths, r.URL.Path)
			require.Equal(t, "Bearer fallback-token", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/api/v1/auth/login":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"fallback-token"}}`)
		case "/api/token/":
			http.NotFound(w, r)
		case "/api/v1/user/keys":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"items":[{"id":"key-1","name":"Primary","status":"active"}]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := supplierSub2APITestProvider(server.URL)
	provider.APIKeysURL = "/api/token/"
	client := NewSupplierSub2APIClient(nil, cache, nil)
	accounts, err := client.FetchAccounts(context.Background(), provider, "secret")

	require.NoError(t, err)
	require.Equal(t, []SupplierProviderRemoteAccount{{
		Key:       "key-1",
		Name:      "Primary",
		Status:    "active",
		RawStatus: "active",
	}}, accounts)
	require.Equal(t, []string{"/api/token/", "/api/v1/user/keys"}, paths)
}

func TestSupplierSub2APIClientStopsAfterSecondUnauthorized(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	var loginCalls atomic.Int32
	var accountCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			loginNumber := loginCalls.Add(1)
			supplierSub2APIWriteJSON(w, http.StatusOK, fmt.Sprintf(`{"code":0,"data":{"access_token":"token-%d"}}`, loginNumber))
		case "/accounts":
			accountCalls.Add(1)
			supplierSub2APIWriteJSON(w, http.StatusUnauthorized, `{"message":"unauthorized"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(nil, cache, nil)
	_, err := client.FetchAccounts(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

	require.Error(t, err)
	require.True(t, IsSupplierProviderAuthFailure(err))
	require.Equal(t, int32(2), loginCalls.Load())
	require.Equal(t, int32(2), accountCalls.Load())
	cache.mu.Lock()
	defer cache.mu.Unlock()
	require.Equal(t, 2, cache.deleteCalls)
	_, tokenRemains := cache.tokens[42]
	require.False(t, tokenRemains)
}

func TestSupplierSub2APIClientRetriesBusinessTokenFailure(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	var loginCalls atomic.Int32
	var groupCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			loginNumber := loginCalls.Add(1)
			supplierSub2APIWriteJSON(w, http.StatusOK, fmt.Sprintf(`{"code":0,"data":{"access_token":"token-%d"}}`, loginNumber))
		case "/groups":
			if groupCalls.Add(1) == 1 {
				supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":"AUTH_FAILED","message":"Session Expired","data":[]}`)
				return
			}
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":[{"id":"group-1","name":"VIP","rate_multiplier":"2.5"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(nil, cache, nil)
	groups, err := client.FetchGroups(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

	require.NoError(t, err)
	require.Equal(t, []SupplierProviderRemoteGroup{{
		Key:            "group-1",
		Name:           "VIP",
		RateMultiplier: 2.5,
	}}, groups)
	require.Equal(t, int32(2), loginCalls.Load())
	require.Equal(t, int32(2), groupCalls.Load())
}

func TestSupplierSub2APIClientStopsWhenRedisIsUnavailable(t *testing.T) {
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
			expectedLoginCalls: 0,
		},
		{
			name: "lock failure",
			configure: func(cache *supplierSub2APIFakeTokenCache) {
				cache.lockErr = errors.New("redis lock unavailable")
			},
			expectedLoginCalls: 0,
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
				switch r.URL.Path {
				case "/api/v1/auth/login":
					loginCalls.Add(1)
					supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"degraded-token"}}`)
				case "/balance":
					require.Equal(t, "Bearer degraded-token", r.Header.Get("Authorization"))
					supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"balance":88.75}}`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			client := NewSupplierSub2APIClient(nil, cache, nil)
			_, err := client.FetchBalance(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

			require.Error(t, err)
			require.False(t, IsSupplierProviderAuthFailure(err))
			require.True(t, IsSupplierProviderSessionFailure(err))
			require.Equal(t, tt.expectedLoginCalls, loginCalls.Load())
		})
	}
}

func TestSupplierSub2APIClientMarksLoginCredentialFailureAsAuthFailure(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			supplierSub2APIWriteJSON(w, http.StatusUnauthorized, `{"message":"invalid username or password"}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(nil, cache, nil)
	_, err := client.FetchBalance(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

	require.Error(t, err)
	require.True(t, IsSupplierProviderAuthFailure(err))
}

func TestSupplierSub2APIClientRecordsCacheMissAndLoginSuccess(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	auditor := &supplierProviderAuthAuditorSpy{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"audited-token"}}`)
		case "/balance":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"balance":12.5}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(server.Client(), cache, nil)
	client.SetAuthAuditor(auditor)
	_, err := client.FetchBalance(context.Background(), supplierSub2APITestProvider(server.URL), "secret")
	require.NoError(t, err)
	require.Contains(t, auditor.eventTypes(), SupplierProviderAuthEventCacheMiss)
	require.Contains(t, auditor.eventTypes(), SupplierProviderAuthEventLoginSuccess)
}

func TestSupplierSub2APIClientParsesAccountsGroupsBalanceAndCost(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	var loginCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" {
			require.Equal(t, "Bearer parser-token", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/api/v1/auth/login":
			loginCalls.Add(1)
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"parser-token"}}`)
		case "/accounts":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{
				"code":0,
				"data":{"items":[{
					"id":123,
					"name":"Primary Account",
					"status":"active",
					"api_key":"sk-secret-must-not-return",
					"group":{"id":"group-1","name":"VIP","rate_multiplier":"2.5"}
				}]}
			}`)
		case "/accounts-array":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{
				"code":0,
				"data":[{
					"key":"account-two",
					"name":"Second Account",
					"status":"disabled",
					"group_key":"group-2",
					"group_name":"Trial",
					"rate_multiplier":0.75
				}]
			}`)
		case "/groups":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":[{"id":"group-1","name":"VIP","status":"active","rate_multiplier":2.5}]}`)
		case "/groups-items":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"items":[{"key":"group-2","name":"Trial","status":"disabled","rate_multiplier":"0.75"}]}}`)
		case "/balance":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"balance":321.5}}`)
		case "/cost":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"today_actual_cost":45.625}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := supplierSub2APITestProvider(server.URL)
	client := NewSupplierSub2APIClient(nil, cache, nil)

	accounts, err := client.FetchAccounts(context.Background(), provider, "secret")
	require.NoError(t, err)
	require.Equal(t, []SupplierProviderRemoteAccount{{
		Key:            "123",
		Name:           "Primary Account",
		Status:         "active",
		GroupKey:       "group-1",
		GroupName:      "VIP",
		RateMultiplier: 2.5,
		RawStatus:      "active",
	}}, accounts)
	serializedAccounts, err := json.Marshal(accounts)
	require.NoError(t, err)
	require.NotContains(t, string(serializedAccounts), "sk-secret-must-not-return")

	provider.APIKeysURL = "/accounts-array"
	accounts, err = client.FetchAccounts(context.Background(), provider, "secret")
	require.NoError(t, err)
	require.Equal(t, []SupplierProviderRemoteAccount{{
		Key:            "account-two",
		Name:           "Second Account",
		Status:         "disabled",
		GroupKey:       "group-2",
		GroupName:      "Trial",
		RateMultiplier: 0.75,
		RawStatus:      "disabled",
	}}, accounts)

	groups, err := client.FetchGroups(context.Background(), provider, "secret")
	require.NoError(t, err)
	require.Equal(t, []SupplierProviderRemoteGroup{{
		Key:            "group-1",
		Name:           "VIP",
		RateMultiplier: 2.5,
		RawStatus:      "active",
	}}, groups)

	provider.GroupsURL = ""
	provider.AvailableGroupsURL = "/groups-items"
	groups, err = client.FetchGroups(context.Background(), provider, "secret")
	require.NoError(t, err)
	require.Empty(t, groups)

	balance, err := client.FetchBalance(context.Background(), provider, "secret")
	require.NoError(t, err)
	require.Equal(t, 321.5, balance)

	cost, err := client.FetchCost(context.Background(), provider, "secret", time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Equal(t, 45.625, cost)
	require.Equal(t, int32(1), loginCalls.Load())
}

func TestSupplierSub2APIClientUsesNormalizedNameWhenAccountKeyMissing(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"name-token"}}`)
		case "/accounts":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{
				"code":0,
				"data":[
					{"name":"  Foo   BAR  ","status":"active"},
					{"key":"key-only"},
					{"status":"missing-both"}
				]
			}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(nil, cache, nil)
	accounts, err := client.FetchAccounts(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

	require.NoError(t, err)
	require.Len(t, accounts, 2)
	require.Equal(t, "foo bar", accounts[0].Key)
	require.Equal(t, "Foo   BAR", accounts[0].Name)
	require.Equal(t, "key-only", accounts[1].Key)
	require.Empty(t, accounts[1].Name)
}

func TestSupplierSub2APIClientFetchRechargeAmountFiltersStatDay(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"recharge-token"}}`)
		case "/api/v1/redeem/history":
			require.Equal(t, "Asia/Shanghai", r.URL.Query().Get("timezone"))
			supplierSub2APIWriteJSON(w, http.StatusOK, `{
				"code":0,
				"message":"success",
				"data":[
					{"id":1,"type":"balance","value":100,"status":"used","used_at":"2026-08-19T22:15:32.45021+08:00"},
					{"id":2,"type":"admin_balance","value":50,"status":"used","used_at":"2026-08-19T17:38:37.682027+08:00"},
					{"id":3,"type":"balance","value":25,"status":"unused","used_at":"2026-08-19T15:36:32.673927+08:00"},
					{"id":4,"type":"other","value":30,"status":"used","used_at":"2026-08-19T12:00:00+08:00"},
					{"id":5,"type":"balance","value":200,"status":"used","used_at":"2026-08-18T16:16:04.172276+08:00"}
				]
			}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(server.Client(), cache, nil)
	amount, err := client.FetchRechargeAmount(context.Background(), supplierSub2APITestProvider(server.URL), "secret", time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC))

	require.NoError(t, err)
	require.Equal(t, 150.0, amount)
}

func TestSupplierSub2APIClientFetchRechargeRecordsFallsBackToCanonicalPathAfterLegacy404(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	var legacyCalls atomic.Int32
	var canonicalCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"recharge-token"}}`)
		case "/v1/redeem/history":
			legacyCalls.Add(1)
			http.NotFound(w, r)
		case "/api/v1/redeem/history":
			canonicalCalls.Add(1)
			require.Equal(t, "Asia/Shanghai", r.URL.Query().Get("timezone"))
			supplierSub2APIWriteJSON(w, http.StatusOK, `{
				"code":0,
				"message":"success",
				"data":[
					{"id":5362,"code":"PAY-4636-84024","type":"balance","value":100,"status":"used","used_by":906,"used_at":"2026-08-19T22:15:32.45021+08:00","created_at":"2026-08-19T22:15:32.442416+08:00"}
				]
			}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := supplierSub2APITestProvider(server.URL)
	provider.RechargeURL = "/v1/redeem/history?timezone=Asia%2FShanghai"
	client := NewSupplierSub2APIClient(server.Client(), cache, nil)
	location := time.FixedZone("CST", 8*3600)
	records, err := client.FetchRechargeRecords(
		context.Background(),
		provider,
		"secret",
		time.Date(2026, 8, 19, 0, 0, 0, 0, location),
		time.Date(2026, 8, 19, 23, 59, 59, 0, location),
	)

	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, "5362", records[0].ExternalID)
	require.Equal(t, int32(1), legacyCalls.Load())
	require.Equal(t, int32(1), canonicalCalls.Load())
}

func TestSupplierSub2APIClientRejectsMalformedEnvelope(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"malformed-token"}}`)
		case "/accounts":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"unexpected":[]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(nil, cache, nil)
	_, err := client.FetchAccounts(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

	require.ErrorContains(t, err, "data.items")
}

func TestSupplierSub2APIClientRejectsOversizedResponse(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"large-token"}}`)
		case "/accounts":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(bytes.Repeat([]byte("x"), (4<<20)+1))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(nil, cache, nil)
	_, err := client.FetchAccounts(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

	require.ErrorContains(t, err, "4 MiB")
}

func TestSupplierSub2APIClientRejectsCrossHostRedirect(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	var redirectedCalls atomic.Int32

	redirectedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectedCalls.Add(1)
		supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"balance":1}}`)
	}))
	defer redirectedServer.Close()

	sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"redirect-token"}}`)
		case "/balance":
			http.Redirect(w, r, redirectedServer.URL+"/balance", http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer sourceServer.Close()

	client := NewSupplierSub2APIClient(nil, cache, nil)
	_, err := client.FetchBalance(context.Background(), supplierSub2APITestProvider(sourceServer.URL), "secret")

	require.ErrorContains(t, err, "redirect")
	require.Equal(t, int32(0), redirectedCalls.Load())
}

func TestSupplierSub2APIClientRejectsUnsupportedURLScheme(t *testing.T) {
	client := NewSupplierSub2APIClient(nil, newSupplierSub2APIFakeTokenCache(), nil)
	provider := supplierSub2APITestProvider("file:///tmp/sub2api")

	_, err := client.FetchBalance(context.Background(), provider, "secret")

	require.ErrorContains(t, err, "http or https")
}

func TestSupplierSub2APIClientNormalizesAuthorizationTokenType(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{AccessToken: "cached-token", TokenType: "  bearer  "})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "bearer cached-token", r.Header.Get("Authorization"))
		supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"balance":1}}`)
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(nil, cache, nil)
	_, err := client.FetchBalance(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

	require.NoError(t, err)
}

func TestSupplierSub2APIClientUsesNetURLComposition(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	var pathsMu sync.Mutex
	var paths []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathsMu.Lock()
		paths = append(paths, r.URL.RequestURI())
		pathsMu.Unlock()
		if strings.HasSuffix(r.URL.Path, "/login") {
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"url-token"}}`)
			return
		}
		supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"balance":9}}`)
	}))
	defer server.Close()

	provider := supplierSub2APITestProvider(server.URL + "/root/")
	provider.LoginURL = "auth/login"
	provider.BalanceURL = "stats/balance?timezone=Asia%2FShanghai"

	client := NewSupplierSub2APIClient(nil, cache, nil)
	balance, err := client.FetchBalance(context.Background(), provider, "secret")

	require.NoError(t, err)
	require.Equal(t, 9.0, balance)
	pathsMu.Lock()
	defer pathsMu.Unlock()
	require.Equal(t, []string{
		"/root/auth/login",
		"/root/stats/balance?timezone=Asia%2FShanghai",
	}, paths)
}

func TestSupplierSub2APISafeResponseSummaryRedactsEmbeddedPlaintextSecrets(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "plain text",
			raw:  "upstream failed: refresh_token=unredacted-refresh-token access_token=unredacted-access-token",
		},
		{
			name: "JSON message",
			raw:  `{"success":false,"message":"upstream failed: refresh_token=unredacted-refresh-token access_token=unredacted-access-token"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := supplierSub2APISafeResponseSummary([]byte(tt.raw))

			require.NotContains(t, summary, "unredacted-refresh-token")
			require.NotContains(t, summary, "unredacted-access-token")
			require.Contains(t, summary, "refresh_token=[REDACTED]")
			require.Contains(t, summary, "access_token=[REDACTED]")
		})
	}
}

func TestSupplierSub2APISafeResponseSummaryRedactsNewAPIRefreshCookie(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "plain text",
			raw:  "upstream failed: new_api_refresh=unredacted-new-api-refresh-cookie",
		},
		{
			name: "JSON message",
			raw:  `{"success":false,"message":"upstream failed: new_api_refresh=unredacted-new-api-refresh-cookie"}`,
		},
		{
			name: "JSON refresh cookie field",
			raw:  `{"success":false,"new_api_refresh":"unredacted-new-api-refresh-cookie"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := supplierSub2APISafeResponseSummary([]byte(tt.raw))

			require.NotContains(t, summary, "unredacted-new-api-refresh-cookie")
			if tt.name == "JSON refresh cookie field" {
				require.Contains(t, summary, `"new_api_refresh":"[redacted]"`)
				return
			}
			require.Contains(t, summary, "new_api_refresh=[REDACTED]")
		})
	}
}

func TestSupplierSub2APISafeResponseSummaryRedactsAndTruncates(t *testing.T) {
	raw := []byte(`{"code":0,"message":"ok","data":{"access_token":"secret-access-token-value","refresh_token":"secret-refresh-token-value","password":"secret-password-value","items":[{"id":"a"}]},"extra":"` + strings.Repeat("abcdefghijklmnopqrstuvwxyz", 40) + `"}`)

	summary := supplierSub2APISafeResponseSummary(raw)

	require.Contains(t, summary, `"code":0`)
	require.Contains(t, summary, `"message":"ok"`)
	require.NotContains(t, summary, "secret-access-token-value")
	require.NotContains(t, summary, "secret-refresh-token-value")
	require.NotContains(t, summary, "secret-password-value")
	require.Contains(t, summary, `"access_token":"[redacted]"`)
	require.Contains(t, summary, "...")
	require.LessOrEqual(t, len(summary), supplierSub2APILogResponseSummaryLimit+3)
}

func TestSupplierSub2APIClientRefreshesNearExpiredCachedSession(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{
		AccessToken:  "stale-access",
		RefreshToken: "old-refresh",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(time.Minute),
	})
	var refreshCalls atomic.Int32
	var loginCalls atomic.Int32
	var accountCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			refreshCalls.Add(1)
			require.Equal(t, http.MethodPost, r.Method)
			var payload map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			require.Equal(t, "old-refresh", payload["refresh_token"])
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"fresh-access","refresh_token":"rotated-refresh","token_type":"Bearer","expires_in":3600}}`)
		case "/api/v1/auth/login":
			loginCalls.Add(1)
			http.Error(w, "login must not be called", http.StatusInternalServerError)
		case "/accounts":
			accountCalls.Add(1)
			require.Equal(t, "Bearer fresh-access", r.Header.Get("Authorization"))
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"items":[]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(server.Client(), cache, nil)
	_, err := client.FetchAccounts(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

	require.NoError(t, err)
	require.Equal(t, int32(1), refreshCalls.Load())
	require.Zero(t, loginCalls.Load())
	require.Equal(t, int32(1), accountCalls.Load())
	cache.mu.Lock()
	cached, found := cache.tokens[42]
	setTTLs := append([]time.Duration(nil), cache.setTTLs...)
	lockCalls := cache.lockCalls
	cache.mu.Unlock()
	require.True(t, found)
	require.Equal(t, "fresh-access", cached.AccessToken)
	require.Equal(t, "rotated-refresh", cached.RefreshToken)
	require.True(t, cached.ExpiresAt.After(time.Now()))
	require.Equal(t, []time.Duration{0}, setTTLs)
	require.GreaterOrEqual(t, lockCalls, 1)
}

func TestSupplierSub2APIClientKeepsRefreshTokenWhenRefreshDoesNotRotate(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{
		AccessToken:  "stale-access",
		RefreshToken: "old-refresh",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(time.Minute),
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"fresh-access","expires_in":3600}}`)
		case "/api/v1/auth/login":
			http.Error(w, "login must not be called", http.StatusInternalServerError)
		case "/accounts":
			require.Equal(t, "Bearer fresh-access", r.Header.Get("Authorization"))
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"items":[]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(server.Client(), cache, nil)
	_, err := client.FetchAccounts(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

	require.NoError(t, err)
	cache.mu.Lock()
	cached := cache.tokens[42]
	cache.mu.Unlock()
	require.Equal(t, "fresh-access", cached.AccessToken)
	require.Equal(t, "old-refresh", cached.RefreshToken)
}

func TestSupplierSub2APIClientFallsBackToLoginWhenRefreshFails(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{
		AccessToken:  "stale-access",
		RefreshToken: "old-refresh",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(time.Minute),
	})
	var refreshCalls atomic.Int32
	var loginCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			refreshCalls.Add(1)
			supplierSub2APIWriteJSON(w, http.StatusUnauthorized, `{"code":"AUTH_FAILED","message":"refresh expired"}`)
		case "/api/v1/auth/login":
			loginCalls.Add(1)
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"login-access","refresh_token":"login-refresh","expires_in":3600}}`)
		case "/accounts":
			require.Equal(t, "Bearer login-access", r.Header.Get("Authorization"))
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"items":[]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(server.Client(), cache, nil)
	_, err := client.FetchAccounts(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

	require.NoError(t, err)
	require.Equal(t, int32(1), refreshCalls.Load())
	require.Equal(t, int32(1), loginCalls.Load())
	cache.mu.Lock()
	cached := cache.tokens[42]
	cache.mu.Unlock()
	require.Equal(t, "login-access", cached.AccessToken)
	require.Equal(t, "login-refresh", cached.RefreshToken)
}

func TestSupplierSub2APIClientRefreshesAfterAuthFailureBeforeLogin(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{
		AccessToken:  "expired-access",
		RefreshToken: "old-refresh",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(time.Hour),
	})
	var refreshCalls atomic.Int32
	var loginCalls atomic.Int32
	var accountCalls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			refreshCalls.Add(1)
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"fresh-access","refresh_token":"rotated-refresh","expires_in":3600}}`)
		case "/api/v1/auth/login":
			loginCalls.Add(1)
			http.Error(w, "login must not be called", http.StatusInternalServerError)
		case "/accounts":
			if accountCalls.Add(1) == 1 {
				require.Equal(t, "Bearer expired-access", r.Header.Get("Authorization"))
				supplierSub2APIWriteJSON(w, http.StatusUnauthorized, `{"code":"AUTH_FAILED","message":"session expired"}`)
				return
			}
			require.Equal(t, "Bearer fresh-access", r.Header.Get("Authorization"))
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"items":[]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(server.Client(), cache, nil)
	_, err := client.FetchAccounts(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

	require.NoError(t, err)
	require.Equal(t, int32(1), refreshCalls.Load())
	require.Zero(t, loginCalls.Load())
	require.Equal(t, int32(2), accountCalls.Load())
}

func TestSupplierSub2APIClientRecordsRefreshAuditWithoutRefreshToken(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	cache.preload(42, SupplierProviderAuthToken{
		AccessToken:  "stale-access",
		RefreshToken: "old-refresh",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(time.Minute),
	})
	auditor := &supplierProviderAuthAuditorSpy{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"access_token":"fresh-access","refresh_token":"rotated-refresh","expires_in":3600}}`)
		case "/accounts":
			supplierSub2APIWriteJSON(w, http.StatusOK, `{"code":0,"data":{"items":[]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSupplierSub2APIClient(server.Client(), cache, nil)
	client.SetAuthAuditor(auditor)
	_, err := client.FetchAccounts(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

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
	require.Equal(t, "fresh-access", refreshEvent.Token.AccessToken)
	require.Empty(t, refreshEvent.Token.RefreshToken)
}

func TestParseSupplierSub2APIRechargeRecordsPreservesUsedBalanceEntries(t *testing.T) {
	raw := []byte(`{"code":0,"message":"success","data":[{"id":5362,"code":"PAY-4636","type":"balance","value":100,"status":"used","used_at":"2026-08-18T22:15:32.45021+08:00"},{"id":1,"code":"skip","type":"balance","value":50,"status":"unused","used_at":"2026-08-18T12:00:00+08:00"},{"id":2,"code":"admin","type":"admin_balance","value":25,"status":"used","used_at":"2026-08-17T12:00:00+08:00"}]}`)

	records, err := parseSupplierSub2APIRechargeRecords(raw, time.Date(2026, 8, 18, 0, 0, 0, 0, time.FixedZone("CST", 8*3600)))

	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, "5362", records[0].ExternalID)
	require.Equal(t, "PAY-4636", records[0].ExternalCode)
	require.Equal(t, "balance", records[0].RechargeType)
	require.Equal(t, 100.0, records[0].Amount)
}

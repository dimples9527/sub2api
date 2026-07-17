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
	client := NewSupplierSub2APIClient(nil, newSupplierSub2APIFakeTokenCache())

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

	client := NewSupplierSub2APIClient(nil, cache)
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
	require.Equal(t, []time.Duration{59 * time.Minute}, cache.setTTLs)
	require.Equal(t, []time.Duration{15 * time.Second}, cache.lockTTLs)
	require.Len(t, cache.acquiredOwners, 1)
	require.Equal(t, cache.acquiredOwners, cache.releasedOwners)
	require.Equal(t, "fresh-token", cache.tokens[42].AccessToken)
	require.Equal(t, "Bearer", cache.tokens[42].TokenType)
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
			wantTTL:           90 * time.Second,
		},
		{
			name:              "nested token",
			loginResponse:     `{"code":0,"data":{"token":"nested-token"},"expires_in":120}`,
			wantAuthorization: "Bearer nested-token",
			wantTTL:           108 * time.Second,
		},
		{
			name:              "top level access token",
			loginResponse:     `{"code":0,"access_token":"top-access","token_type":"JWT","expires_in":121}`,
			wantAuthorization: "JWT top-access",
			wantTTL:           61 * time.Second,
		},
		{
			name:              "top level token with fallback expiry",
			loginResponse:     `{"code":0,"token":"top-token"}`,
			wantAuthorization: "Bearer top-token",
			wantTTL:           30 * time.Minute,
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
			client := NewSupplierSub2APIClient(nil, cache)
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

	client := NewSupplierSub2APIClient(nil, cache)
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

	client := NewSupplierSub2APIClient(nil, cache)
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

	client := NewSupplierSub2APIClient(nil, cache)
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

	client := NewSupplierSub2APIClient(nil, cache)
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
	client := NewSupplierSub2APIClient(nil, cache)
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

	client := NewSupplierSub2APIClient(nil, cache)
	_, err := client.FetchAccounts(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

	require.Error(t, err)
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

	client := NewSupplierSub2APIClient(nil, cache)
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

func TestSupplierSub2APIClientContinuesWhenRedisIsUnavailable(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*supplierSub2APIFakeTokenCache)
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

			client := NewSupplierSub2APIClient(nil, cache)
			balance, err := client.FetchBalance(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

			require.NoError(t, err)
			require.Equal(t, 88.75, balance)
			require.Equal(t, int32(1), loginCalls.Load())
		})
	}
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
	client := NewSupplierSub2APIClient(nil, cache)

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
	require.Equal(t, []SupplierProviderRemoteGroup{{
		Key:            "group-2",
		Name:           "Trial",
		RateMultiplier: 0.75,
		RawStatus:      "disabled",
	}}, groups)

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

	client := NewSupplierSub2APIClient(nil, cache)
	accounts, err := client.FetchAccounts(context.Background(), supplierSub2APITestProvider(server.URL), "secret")

	require.NoError(t, err)
	require.Len(t, accounts, 2)
	require.Equal(t, "foo bar", accounts[0].Key)
	require.Equal(t, "Foo   BAR", accounts[0].Name)
	require.Equal(t, "key-only", accounts[1].Key)
	require.Empty(t, accounts[1].Name)
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

	client := NewSupplierSub2APIClient(nil, cache)
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

	client := NewSupplierSub2APIClient(nil, cache)
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

	client := NewSupplierSub2APIClient(nil, cache)
	_, err := client.FetchBalance(context.Background(), supplierSub2APITestProvider(sourceServer.URL), "secret")

	require.ErrorContains(t, err, "redirect")
	require.Equal(t, int32(0), redirectedCalls.Load())
}

func TestSupplierSub2APIClientRejectsUnsupportedURLScheme(t *testing.T) {
	client := NewSupplierSub2APIClient(nil, newSupplierSub2APIFakeTokenCache())
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

	client := NewSupplierSub2APIClient(nil, cache)
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

	client := NewSupplierSub2APIClient(nil, cache)
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

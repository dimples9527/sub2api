package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestSupplierNewAPIClientReusesSessionFromSharedTokenCache(t *testing.T) {
	cache := newSupplierSub2APIFakeTokenCache()
	var loginCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			loginCalls.Add(1)
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "cached-session"})
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
	require.Equal(t, int64(42), cachedToken.UserID)
	require.Equal(t, "session=cached-session", cachedToken.CookieHeader)
	require.False(t, cachedToken.ExpiresAt.IsZero())
	require.Len(t, setTTLs, 1)
	require.Greater(t, setTTLs[0], time.Duration(0))
}

func TestSupplierNewAPIClientRefreshesExpiredCachedSession(t *testing.T) {
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
	cache.mu.Lock()
	setCalls := cache.setCalls
	cache.mu.Unlock()
	require.Equal(t, 1, setCalls)
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

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
				"Trial":{"id":8,"ratio":0.75}
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
	client := NewSupplierNewAPIClient(server.Client())

	accounts, err := client.FetchAccounts(context.Background(), provider, "secret")
	require.NoError(t, err)
	require.Equal(t, []SupplierProviderRemoteAccount{{
		Key:            "key-1",
		Name:           "key-1",
		Status:         "1",
		GroupKey:       "7",
		GroupName:      "VIP",
		RateMultiplier: 3.25,
		RawStatus:      "1",
	}, {
		Key:            "key-2",
		Name:           "key-2",
		Status:         "2",
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
	client := NewSupplierNewAPIClient(server.Client())

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

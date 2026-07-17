package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpstreamProviderServiceFetchDefaultModelSquareUsesSub2APIToken(t *testing.T) {
	var loginRequests int
	var modelRequests int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/login":
			loginRequests++
			if r.Method != http.MethodPost {
				t.Fatalf("login method = %s, want POST", r.Method)
			}
			_, _ = w.Write([]byte(`{"code":0,"data":{"access_token":"model-token","token_type":"Bearer","expires_in":3600}}`))
		case "/api/v1/model-square":
			modelRequests++
			if r.Header.Get("Authorization") != "Bearer model-token" {
				t.Fatalf("Authorization = %q, want Bearer model-token", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"groups":[{"id":1,"name":"vip","rate_multiplier":2}],"models":[{"id":"gpt-5.2","group_ids":[1]}]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	repo := newUpstreamProviderMemorySettingRepo()
	svc := NewUpstreamProviderServiceWithHTTPClient(repo, upstream.Client())
	if _, err := svc.CreateProvider(context.Background(), UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "primary",
		Name:       "Primary",
		Enabled:    true,
		IsDefault:  true,
		BaseURL:    upstream.URL,
		LoginURL:   "/api/v1/auth/login",
		APIKeysURL: "/api/v1/keys",
		Email:      "admin@example.com",
		Password:   "secret",
	}); err != nil {
		t.Fatalf("CreateProvider returned error: %v", err)
	}

	payload, provider, err := svc.FetchDefaultModelSquare(context.Background())
	if err != nil {
		t.Fatalf("FetchDefaultModelSquare returned error: %v", err)
	}
	if provider.Slug != "primary" {
		t.Fatalf("provider slug = %q, want primary", provider.Slug)
	}
	if loginRequests != 1 || modelRequests != 1 {
		t.Fatalf("requests login/model = %d/%d, want 1/1", loginRequests, modelRequests)
	}
	var body struct {
		Groups []struct {
			Name string `json:"name"`
		} `json:"groups"`
		Models []struct {
			ID string `json:"id"`
		} `json:"models"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("payload should be JSON: %v", err)
	}
	if len(body.Groups) != 1 || body.Groups[0].Name != "vip" || len(body.Models) != 1 || body.Models[0].ID != "gpt-5.2" {
		t.Fatalf("unexpected model-square payload: %s", string(payload))
	}
}

func TestUpstreamProviderServiceFetchDefaultModelSquareUsesNewAPISession(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			if r.Method != http.MethodPost {
				t.Fatalf("login method = %s, want POST", r.Method)
			}
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":42}}`))
		case "/api/v1/model-square":
			if r.Header.Get("New-Api-User") != "42" {
				t.Fatalf("New-Api-User = %q, want 42", r.Header.Get("New-Api-User"))
			}
			if !strings.Contains(r.Header.Get("Cookie"), "session=abc") {
				t.Fatalf("Cookie header = %q, want session cookie", r.Header.Get("Cookie"))
			}
			_, _ = w.Write([]byte(`{"groups":[],"models":[]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	repo := newUpstreamProviderMemorySettingRepo()
	svc := NewUpstreamProviderServiceWithHTTPClient(repo, upstream.Client())
	if _, err := svc.CreateProvider(context.Background(), UpstreamProviderConfig{
		Type:       UpstreamProviderTypeNewAPI,
		Slug:       "newapi",
		Name:       "NewAPI",
		Enabled:    true,
		IsDefault:  true,
		BaseURL:    upstream.URL,
		LoginURL:   "/api/user/login",
		APIKeysURL: "/api/token/",
		GroupsURL:  "/api/user/self/groups",
		Username:   "root",
		Password:   "secret",
	}); err != nil {
		t.Fatalf("CreateProvider returned error: %v", err)
	}

	payload, provider, err := svc.FetchDefaultModelSquare(context.Background())
	if err != nil {
		t.Fatalf("FetchDefaultModelSquare returned error: %v", err)
	}
	if provider.Type != UpstreamProviderTypeNewAPI {
		t.Fatalf("provider type = %q, want newapi", provider.Type)
	}
	if strings.TrimSpace(string(payload)) != `{"groups":[],"models":[]}` {
		t.Fatalf("payload = %s", string(payload))
	}
}

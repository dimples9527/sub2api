package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type upstreamProviderMemorySettingRepo struct {
	mu     sync.Mutex
	values map[string]string
}

func newUpstreamProviderMemorySettingRepo() *upstreamProviderMemorySettingRepo {
	return &upstreamProviderMemorySettingRepo{values: map[string]string{}}
}

func (r *upstreamProviderMemorySettingRepo) Get(ctx context.Context, key string) (*Setting, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.values[key]
	if !ok {
		return nil, ErrSettingNotFound
	}
	return &Setting{Key: key, Value: value}, nil
}

func (r *upstreamProviderMemorySettingRepo) GetValue(ctx context.Context, key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (r *upstreamProviderMemorySettingRepo) Set(ctx context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[key] = value
	return nil
}

func (r *upstreamProviderMemorySettingRepo) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (r *upstreamProviderMemorySettingRepo) SetMultiple(ctx context.Context, settings map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *upstreamProviderMemorySettingRepo) GetAll(ctx context.Context) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]string, len(r.values))
	for key, value := range r.values {
		out[key] = value
	}
	return out, nil
}

func (r *upstreamProviderMemorySettingRepo) Delete(ctx context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.values, key)
	return nil
}

func TestUpstreamProviderServiceCreateAndListRedactsPassword(t *testing.T) {
	ctx := context.Background()
	repo := newUpstreamProviderMemorySettingRepo()
	svc := NewUpstreamProviderService(repo)

	created, err := svc.CreateProvider(ctx, UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "primary",
		Name:       "Primary upstream",
		Enabled:    true,
		BaseURL:    "https://upstream.example.com",
		APIKeysURL: "/api/admin/keys",
		Email:      "admin@example.com",
		Password:   "super-secret",
	})
	if err != nil {
		t.Fatalf("CreateProvider returned error: %v", err)
	}
	if created.Password != "" {
		t.Fatalf("returned provider leaked password %q", created.Password)
	}
	if !created.PasswordConfigured {
		t.Fatalf("created provider should report password_configured")
	}

	providers, err := svc.ListProviders(ctx)
	if err != nil {
		t.Fatalf("ListProviders returned error: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("provider count = %d, want 1", len(providers))
	}
	if providers[0].Password != "" {
		t.Fatalf("listed provider leaked password %q", providers[0].Password)
	}
	if !providers[0].PasswordConfigured {
		t.Fatalf("listed provider should report password_configured")
	}

	raw := repo.values[SettingKeyUpstreamProviderConfigs]
	if !strings.Contains(raw, "super-secret") {
		t.Fatalf("stored provider config should retain encrypted/plain setting payload for later use, got %s", raw)
	}
}

func TestUpstreamProviderServiceUpdateKeepsPasswordWhenBlank(t *testing.T) {
	ctx := context.Background()
	repo := newUpstreamProviderMemorySettingRepo()
	svc := NewUpstreamProviderService(repo)

	if _, err := svc.CreateProvider(ctx, UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "primary",
		Name:       "Primary upstream",
		Enabled:    true,
		BaseURL:    "https://upstream.example.com",
		APIKeysURL: "/api/admin/keys",
		Password:   "keep-me",
	}); err != nil {
		t.Fatalf("CreateProvider returned error: %v", err)
	}

	updated, err := svc.UpdateProvider(ctx, "primary", UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "primary",
		Name:       "Renamed upstream",
		Enabled:    false,
		BaseURL:    "https://upstream.example.com",
		APIKeysURL: "/api/admin/keys",
	})
	if err != nil {
		t.Fatalf("UpdateProvider returned error: %v", err)
	}
	if updated.Password != "" {
		t.Fatalf("returned provider leaked password %q", updated.Password)
	}
	if !updated.PasswordConfigured {
		t.Fatalf("updated provider should retain password_configured")
	}

	raw := repo.values[SettingKeyUpstreamProviderConfigs]
	if !strings.Contains(raw, "keep-me") {
		t.Fatalf("blank update should keep existing password, got %s", raw)
	}
}

func TestUpstreamProviderServiceCreateDefaultClearsOtherDefaults(t *testing.T) {
	ctx := context.Background()
	repo := newUpstreamProviderMemorySettingRepo()
	svc := NewUpstreamProviderService(repo)

	if _, err := svc.CreateProvider(ctx, UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "primary",
		Name:       "Primary upstream",
		Enabled:    true,
		IsDefault:  true,
		BaseURL:    "https://primary.example.com",
		APIKeysURL: "/api/admin/keys",
	}); err != nil {
		t.Fatalf("CreateProvider primary returned error: %v", err)
	}
	if _, err := svc.CreateProvider(ctx, UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "secondary",
		Name:       "Secondary upstream",
		Enabled:    true,
		IsDefault:  true,
		BaseURL:    "https://secondary.example.com",
		APIKeysURL: "/api/admin/keys",
	}); err != nil {
		t.Fatalf("CreateProvider secondary returned error: %v", err)
	}

	providers, err := svc.ListProviders(ctx)
	if err != nil {
		t.Fatalf("ListProviders returned error: %v", err)
	}
	if len(providers) != 2 {
		t.Fatalf("provider count = %d, want 2", len(providers))
	}
	defaultSlug := ""
	for _, provider := range providers {
		if provider.IsDefault {
			if defaultSlug != "" {
				t.Fatalf("multiple default providers: %s and %s", defaultSlug, provider.Slug)
			}
			defaultSlug = provider.Slug
		}
	}
	if defaultSlug != "secondary" {
		t.Fatalf("default provider = %q, want secondary", defaultSlug)
	}
}

func TestUpstreamProviderServiceSetDefaultProvider(t *testing.T) {
	ctx := context.Background()
	repo := newUpstreamProviderMemorySettingRepo()
	svc := NewUpstreamProviderService(repo)

	if _, err := svc.CreateProvider(ctx, UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "primary",
		Name:       "Primary upstream",
		Enabled:    true,
		BaseURL:    "https://primary.example.com",
		APIKeysURL: "/api/admin/keys",
	}); err != nil {
		t.Fatalf("CreateProvider primary returned error: %v", err)
	}
	if _, err := svc.CreateProvider(ctx, UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "secondary",
		Name:       "Secondary upstream",
		Enabled:    true,
		IsDefault:  true,
		BaseURL:    "https://secondary.example.com",
		APIKeysURL: "/api/admin/keys",
	}); err != nil {
		t.Fatalf("CreateProvider secondary returned error: %v", err)
	}

	updated, err := svc.SetDefaultProvider(ctx, "primary")
	if err != nil {
		t.Fatalf("SetDefaultProvider returned error: %v", err)
	}
	if !updated.IsDefault {
		t.Fatalf("updated provider should be default")
	}
	providers, err := svc.ListProviders(ctx)
	if err != nil {
		t.Fatalf("ListProviders returned error: %v", err)
	}
	for _, provider := range providers {
		if provider.Slug == "primary" && !provider.IsDefault {
			t.Fatalf("primary should be default")
		}
		if provider.Slug == "secondary" && provider.IsDefault {
			t.Fatalf("secondary should no longer be default")
		}
	}
}

func TestUpstreamProviderServiceUpdateCanSetDefaultProvider(t *testing.T) {
	ctx := context.Background()
	repo := newUpstreamProviderMemorySettingRepo()
	svc := NewUpstreamProviderService(repo)

	if _, err := svc.CreateProvider(ctx, UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "primary",
		Name:       "Primary upstream",
		Enabled:    true,
		IsDefault:  true,
		BaseURL:    "https://primary.example.com",
		APIKeysURL: "/api/admin/keys",
	}); err != nil {
		t.Fatalf("CreateProvider primary returned error: %v", err)
	}
	updated, err := svc.CreateProvider(ctx, UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "secondary",
		Name:       "Secondary upstream",
		Enabled:    true,
		BaseURL:    "https://secondary.example.com",
		APIKeysURL: "/api/admin/keys",
	})
	if err != nil {
		t.Fatalf("CreateProvider secondary returned error: %v", err)
	}
	if updated.IsDefault {
		t.Fatalf("secondary should not start as default")
	}

	updated, err = svc.UpdateProvider(ctx, "secondary", UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "secondary",
		Name:       "Secondary upstream",
		Enabled:    true,
		IsDefault:  true,
		BaseURL:    "https://secondary.example.com",
		APIKeysURL: "/api/admin/keys",
	})
	if err != nil {
		t.Fatalf("UpdateProvider returned error: %v", err)
	}
	if !updated.IsDefault {
		t.Fatalf("secondary should become default")
	}
	providers, err := svc.ListProviders(ctx)
	if err != nil {
		t.Fatalf("ListProviders returned error: %v", err)
	}
	for _, provider := range providers {
		if provider.Slug == "primary" && provider.IsDefault {
			t.Fatalf("primary should no longer be default")
		}
		if provider.Slug == "secondary" && !provider.IsDefault {
			t.Fatalf("secondary should be default")
		}
	}
}

func TestSub2APIProviderAdapterFetchKeysUsesSingleEndpoint(t *testing.T) {
	var keysRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/admin/keys" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		keysRequests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"code": 0,
			"data": {
				"items": [
					{"name": "sk-live-1", "group": {"name": "vip", "rate_multiplier": 2.5}}
				]
			}
		}`))
	}))
	defer server.Close()

	adapter := NewSub2APIProviderAdapter(server.Client())
	keys, warnings, err := adapter.FetchKeys(context.Background(), UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "sub2api-main",
		Name:       "Sub2API main",
		BaseURL:    server.URL,
		APIKeysURL: "/api/admin/keys",
	})
	if err != nil {
		t.Fatalf("FetchKeys returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if keysRequests != 1 {
		t.Fatalf("keys endpoint requests = %d, want 1", keysRequests)
	}
	if len(keys) != 1 {
		t.Fatalf("key count = %d, want 1", len(keys))
	}
	if keys[0].KeyName != "sk-live-1" || keys[0].GroupName != "vip" || keys[0].RateMultiplier != 2.5 {
		t.Fatalf("unexpected normalized key: %+v", keys[0])
	}
}

func TestSub2APIProviderAdapterAppliesAccountNamePrefix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"code": 0,
			"data": {
				"items": [
					{"name": "sk-live-1", "group": {"name": "vip", "rate_multiplier": 2.5}}
				]
			}
		}`))
	}))
	defer server.Close()

	adapter := NewSub2APIProviderAdapter(server.Client())
	keys, _, err := adapter.FetchKeys(context.Background(), UpstreamProviderConfig{
		Type:              UpstreamProviderTypeSub2API,
		Slug:              "sub2api-main",
		Name:              "Sub2API main",
		BaseURL:           server.URL,
		APIKeysURL:        "/api/admin/keys",
		AccountNamePrefix: "sub-",
	})
	if err != nil {
		t.Fatalf("FetchKeys returned error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("key count = %d, want 1", len(keys))
	}
	if keys[0].KeyName != "sub-sk-live-1" {
		t.Fatalf("key name = %q, want prefixed name", keys[0].KeyName)
	}
}

func TestSub2APIProviderAdapterLoginAcceptsAccessTokenResponse(t *testing.T) {
	var keysAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/login":
			if r.Method != http.MethodPost {
				t.Fatalf("login method = %s, want POST", r.Method)
			}
			_, _ = w.Write([]byte(`{
				"code": 0,
				"data": {
					"access_token": "access-123",
					"token_type": "Bearer"
				}
			}`))
		case "/api/admin/keys":
			keysAuthorization = r.Header.Get("Authorization")
			_, _ = w.Write([]byte(`{
				"code": 0,
				"data": {
					"items": [
						{"name": "sk-live-1", "group": {"name": "vip", "rate_multiplier": 2.5}}
					]
				}
			}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewSub2APIProviderAdapter(server.Client())
	result := adapter.Test(context.Background(), UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "sub2api-main",
		Name:       "Sub2API main",
		BaseURL:    server.URL,
		LoginURL:   "/api/v1/auth/login",
		APIKeysURL: "/api/admin/keys",
		Email:      "admin@example.com",
		Password:   "secret",
	})

	if !result.Login.OK {
		t.Fatalf("login should pass, got error: %s", result.Login.Error)
	}
	if !result.Keys.OK {
		t.Fatalf("keys should pass, got error: %s", result.Keys.Error)
	}
	if keysAuthorization != "Bearer access-123" {
		t.Fatalf("Authorization = %q, want Bearer access-123", keysAuthorization)
	}
}

func TestNewAPIProviderAdapterFetchKeysMergesKeysAndGroups(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			if r.Method != http.MethodPost {
				t.Fatalf("login method = %s, want POST", r.Method)
			}
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			_, _ = w.Write([]byte(`{"success": true, "data": {"id": 42}}`))
		case "/api/token/":
			if r.Header.Get("New-Api-User") != "42" {
				t.Fatalf("New-Api-User = %q, want 42", r.Header.Get("New-Api-User"))
			}
			if !strings.Contains(r.Header.Get("Cookie"), "session=abc") {
				t.Fatalf("Cookie header = %q, want session cookie", r.Header.Get("Cookie"))
			}
			_, _ = w.Write([]byte(`{
				"success": true,
				"data": {
					"items": [
						{"name": "newapi-key-1", "group": "VIP"}
					]
				}
			}`))
		case "/api/group/":
			_, _ = w.Write([]byte(`{
				"success": true,
				"data": {
					"VIP": {"ratio": 3.25}
				}
			}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewNewAPIProviderAdapter(server.Client())
	keys, warnings, err := adapter.FetchKeys(context.Background(), UpstreamProviderConfig{
		Type:       UpstreamProviderTypeNewAPI,
		Slug:       "newapi-main",
		Name:       "NewAPI main",
		BaseURL:    server.URL,
		LoginURL:   "/api/user/login",
		APIKeysURL: "/api/token/",
		GroupsURL:  "/api/group/",
		Username:   "root",
		Password:   "secret",
	})
	if err != nil {
		t.Fatalf("FetchKeys returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if len(keys) != 1 {
		t.Fatalf("key count = %d, want 1", len(keys))
	}
	if keys[0].KeyName != "newapi-key-1" || keys[0].GroupName != "VIP" || keys[0].RateMultiplier != 3.25 {
		t.Fatalf("unexpected normalized key: %+v", keys[0])
	}
}

func TestNewAPIProviderAdapterAppliesAccountNamePrefix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			_, _ = w.Write([]byte(`{"success": true, "data": {"id": 42}}`))
		case "/api/token/":
			_, _ = w.Write([]byte(`{"success": true, "data": {"items": [{"name": "key-1", "group": "VIP"}]}}`))
		case "/api/group/":
			_, _ = w.Write([]byte(`{"success": true, "data": {"VIP": {"ratio": 3.25}}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewNewAPIProviderAdapter(server.Client())
	keys, _, err := adapter.FetchKeys(context.Background(), UpstreamProviderConfig{
		Type:              UpstreamProviderTypeNewAPI,
		Slug:              "newapi-main",
		Name:              "NewAPI main",
		BaseURL:           server.URL,
		LoginURL:          "/api/user/login",
		APIKeysURL:        "/api/token/",
		GroupsURL:         "/api/group/",
		Username:          "root",
		Password:          "secret",
		AccountNamePrefix: "new-",
	})
	if err != nil {
		t.Fatalf("FetchKeys returned error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("key count = %d, want 1", len(keys))
	}
	if keys[0].KeyName != "new-key-1" {
		t.Fatalf("key name = %q, want prefixed name", keys[0].KeyName)
	}
}

func TestNewAPIProviderAdapterWarnsWhenKeyGroupHasNoRatio(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			_, _ = w.Write([]byte(`{"success": true, "data": {"id": 42}}`))
		case "/api/token/":
			_, _ = w.Write([]byte(`{"success": true, "data": {"items": [{"name": "orphan-key", "group": "missing"}]}}`))
		case "/api/group/":
			_, _ = w.Write([]byte(`{"success": true, "data": {"VIP": {"ratio": 2}}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewNewAPIProviderAdapter(server.Client())
	keys, warnings, err := adapter.FetchKeys(context.Background(), UpstreamProviderConfig{
		Type:       UpstreamProviderTypeNewAPI,
		Slug:       "newapi-main",
		Name:       "NewAPI main",
		BaseURL:    server.URL,
		LoginURL:   "/api/user/login",
		APIKeysURL: "/api/token/",
		GroupsURL:  "/api/group/",
		Username:   "root",
		Password:   "secret",
	})
	if err != nil {
		t.Fatalf("FetchKeys returned error: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("key count = %d, want 0", len(keys))
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "missing") {
		t.Fatalf("warnings = %v, want missing group warning", warnings)
	}
}

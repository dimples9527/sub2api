package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
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

func TestUpstreamProviderServicePersistsAvailableGroupsURL(t *testing.T) {
	ctx := context.Background()
	repo := newUpstreamProviderMemorySettingRepo()
	svc := NewUpstreamProviderService(repo)

	created, err := svc.CreateProvider(ctx, UpstreamProviderConfig{
		Type:               UpstreamProviderTypeSub2API,
		Slug:               "primary",
		Name:               "Primary upstream",
		Enabled:            true,
		BaseURL:            "https://upstream.example.com",
		APIKeysURL:         "/api/admin/keys",
		AvailableGroupsURL: " /api/v1/groups/available?timezone=Asia%2FShanghai ",
	})
	if err != nil {
		t.Fatalf("CreateProvider returned error: %v", err)
	}
	if created.AvailableGroupsURL != "/api/v1/groups/available?timezone=Asia%2FShanghai" {
		t.Fatalf("available groups url = %q", created.AvailableGroupsURL)
	}

	providers, err := svc.ListProviders(ctx)
	if err != nil {
		t.Fatalf("ListProviders returned error: %v", err)
	}
	if len(providers) != 1 || providers[0].AvailableGroupsURL != "/api/v1/groups/available?timezone=Asia%2FShanghai" {
		t.Fatalf("providers = %+v, want available groups url persisted", providers)
	}
}

func TestUpstreamProviderServicePersistsBalanceURL(t *testing.T) {
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
		BalanceURL: " /api/custom/balance ",
	})
	if err != nil {
		t.Fatalf("CreateProvider returned error: %v", err)
	}
	if created.BalanceURL != "/api/custom/balance" {
		t.Fatalf("balance url = %q", created.BalanceURL)
	}

	providers, err := svc.ListProviders(ctx)
	if err != nil {
		t.Fatalf("ListProviders returned error: %v", err)
	}
	if len(providers) != 1 || providers[0].BalanceURL != "/api/custom/balance" {
		t.Fatalf("providers = %+v, want balance url persisted", providers)
	}
}

func TestUpstreamProviderServicePersistsUsageCostURL(t *testing.T) {
	ctx := context.Background()
	repo := newUpstreamProviderMemorySettingRepo()
	svc := NewUpstreamProviderService(repo)

	created, err := svc.CreateProvider(ctx, UpstreamProviderConfig{
		Type:         UpstreamProviderTypeSub2API,
		Slug:         "primary",
		Name:         "Primary upstream",
		Enabled:      true,
		BaseURL:      "https://upstream.example.com",
		APIKeysURL:   "/api/admin/keys",
		UsageCostURL: " /api/custom/usage-cost ",
	})
	if err != nil {
		t.Fatalf("CreateProvider returned error: %v", err)
	}
	if created.UsageCostURL != "/api/custom/usage-cost" {
		t.Fatalf("usage cost url = %q", created.UsageCostURL)
	}

	providers, err := svc.ListProviders(ctx)
	if err != nil {
		t.Fatalf("ListProviders returned error: %v", err)
	}
	if len(providers) != 1 || providers[0].UsageCostURL != "/api/custom/usage-cost" {
		t.Fatalf("providers = %+v, want usage cost url persisted", providers)
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

func TestUpstreamProviderServiceListsProvidersBySortOrder(t *testing.T) {
	ctx := context.Background()
	repo := newUpstreamProviderMemorySettingRepo()
	repo.values[SettingKeyUpstreamProviderConfigs] = `[
		{
			"type":"sub2api",
			"slug":"z-last",
			"name":"Last upstream",
			"enabled":true,
			"base_url":"https://last.example.com",
			"api_keys_url":"/api/admin/keys",
			"sort_order":10
		},
		{
			"type":"sub2api",
			"slug":"a-first",
			"name":"First upstream",
			"enabled":true,
			"base_url":"https://first.example.com",
			"api_keys_url":"/api/admin/keys",
			"sort_order":20
		},
		{
			"type":"sub2api",
			"slug":"default-upstream",
			"name":"Default upstream",
			"enabled":true,
			"is_default":true,
			"base_url":"https://default.example.com",
			"api_keys_url":"/api/admin/keys",
			"sort_order":99
		}
	]`
	svc := NewUpstreamProviderService(repo)

	providers, err := svc.ListProviders(ctx)
	if err != nil {
		t.Fatalf("ListProviders returned error: %v", err)
	}
	if len(providers) != 3 {
		t.Fatalf("provider count = %d, want 3", len(providers))
	}
	if providers[0].Slug != "default-upstream" || providers[1].Slug != "z-last" || providers[2].Slug != "a-first" {
		t.Fatalf("providers = %+v, want default first then sort_order ascending", providers)
	}
}

func TestUpstreamProviderServicePersistsSortOrder(t *testing.T) {
	ctx := context.Background()
	repo := newUpstreamProviderMemorySettingRepo()
	svc := NewUpstreamProviderService(repo)

	created, err := svc.CreateProvider(ctx, UpstreamProviderConfig{
		Type:                       UpstreamProviderTypeSub2API,
		Slug:                       "primary",
		Name:                       "Primary upstream",
		Enabled:                    true,
		BaseURL:                    "https://upstream.example.com",
		APIKeysURL:                 "/api/admin/keys",
		SortOrder:                  18,
		AccountRateMultiplierScale: 1.2,
	})
	if err != nil {
		t.Fatalf("CreateProvider returned error: %v", err)
	}
	if created.SortOrder != 18 {
		t.Fatalf("created provider sort order = %d, want 18", created.SortOrder)
	}

	updated, err := svc.UpdateProvider(ctx, "primary", UpstreamProviderConfig{
		Type:                       UpstreamProviderTypeSub2API,
		Slug:                       "primary",
		Name:                       "Primary upstream",
		Enabled:                    true,
		BaseURL:                    "https://upstream.example.com",
		APIKeysURL:                 "/api/admin/keys",
		SortOrder:                  7,
		AccountRateMultiplierScale: 1.2,
	})
	if err != nil {
		t.Fatalf("UpdateProvider returned error: %v", err)
	}
	if updated.SortOrder != 7 {
		t.Fatalf("updated provider sort order = %d, want 7", updated.SortOrder)
	}

	providers, err := svc.ListProviders(ctx)
	if err != nil {
		t.Fatalf("ListProviders returned error: %v", err)
	}
	if len(providers) != 1 || providers[0].SortOrder != 7 {
		t.Fatalf("providers = %+v, want persisted sort order", providers)
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

func TestSub2APIProviderAdapterPreservesRawKeyName(t *testing.T) {
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
	if keys[0].KeyName != "sk-live-1" {
		t.Fatalf("key name = %q, want raw key name", keys[0].KeyName)
	}
}

func TestSub2APIProviderAdapterFetchGroupsUsesAvailableGroupsEndpoint(t *testing.T) {
	var groupsRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/groups/available" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("timezone") != "Asia/Shanghai" {
			t.Fatalf("timezone query = %q, want Asia/Shanghai", r.URL.Query().Get("timezone"))
		}
		groupsRequests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"code": 0,
			"data": [
				{"id": 2, "name": "codex福利", "platform": "openai", "rate_multiplier": 0.15, "status": "active"},
				{"id": 5, "name": "claude 福利", "platform": "anthropic", "rate_multiplier": "0.6", "status": "active"}
			]
		}`))
	}))
	defer server.Close()

	adapter := NewSub2APIProviderAdapter(server.Client())
	groups, warnings, err := adapter.FetchGroups(context.Background(), UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "sub2api-main",
		Name:       "Sub2API main",
		BaseURL:    server.URL,
		APIKeysURL: "/api/admin/keys",
	})
	if err != nil {
		t.Fatalf("FetchGroups returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if groupsRequests != 1 {
		t.Fatalf("groups endpoint requests = %d, want 1", groupsRequests)
	}
	if len(groups) != 2 {
		t.Fatalf("group count = %d, want 2", len(groups))
	}
	if groups[0].GroupName != "codex福利" || groups[0].RateMultiplier != 0.15 || groups[0].RawGroupID != "2" || groups[0].RawStatus != "active" {
		t.Fatalf("unexpected first group: %+v", groups[0])
	}
	if groups[1].GroupName != "claude 福利" || groups[1].RateMultiplier != 0.6 || groups[1].RawGroupID != "5" {
		t.Fatalf("unexpected second group: %+v", groups[1])
	}
}

func TestSub2APIProviderAdapterFetchGroupsPrefersAvailableGroupsURL(t *testing.T) {
	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":[{"id":2,"name":"vip","rate_multiplier":0.15}]}`))
	}))
	defer server.Close()

	adapter := NewSub2APIProviderAdapter(server.Client())
	_, _, err := adapter.FetchGroups(context.Background(), UpstreamProviderConfig{
		Type:               UpstreamProviderTypeSub2API,
		Slug:               "sub2api-main",
		Name:               "Sub2API main",
		BaseURL:            server.URL,
		APIKeysURL:         "/api/admin/keys",
		GroupsURL:          "/legacy/groups",
		AvailableGroupsURL: "/api/v1/groups/available?timezone=Asia%2FShanghai",
	})
	if err != nil {
		t.Fatalf("FetchGroups returned error: %v", err)
	}
	if requestedPath != "/api/v1/groups/available?timezone=Asia%2FShanghai" {
		t.Fatalf("requested path = %q, want available groups URL", requestedPath)
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

func TestNewAPIProviderAdapterPreservesRawKeyName(t *testing.T) {
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
	if keys[0].KeyName != "key-1" {
		t.Fatalf("key name = %q, want raw key name", keys[0].KeyName)
	}
}

func TestNewAPIProviderAdapterParsesStringGroupRatio(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			_, _ = w.Write([]byte(`{"success": true, "data": {"id": 42}}`))
		case "/api/token/":
			_, _ = w.Write([]byte(`{"success": true, "data": {"items": [{"name": "key-1", "group": "VIP"}]}}`))
		case "/api/group/":
			_, _ = w.Write([]byte(`{"success": true, "data": {"VIP": {"ratio": "3.25"}}}`))
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
	if len(keys) != 1 || keys[0].RateMultiplier != 3.25 {
		t.Fatalf("keys = %+v, want one key with ratio 3.25", keys)
	}
}

func TestNewAPIProviderAdapterFetchGroupsUsesGroupsEndpoint(t *testing.T) {
	var groupsRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			_, _ = w.Write([]byte(`{"success": true, "data": {"id": 42}}`))
		case "/api/group/":
			groupsRequests++
			if r.Header.Get("New-Api-User") != "42" {
				t.Fatalf("New-Api-User = %q, want 42", r.Header.Get("New-Api-User"))
			}
			if !strings.Contains(r.Header.Get("Cookie"), "session=abc") {
				t.Fatalf("Cookie header = %q, want session cookie", r.Header.Get("Cookie"))
			}
			_, _ = w.Write([]byte(`{
				"success": true,
				"data": {
					"VIP": {"id": 7, "ratio": "3.25"},
					"No Key Group": {"id": 8, "ratio": 0.15}
				}
			}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewNewAPIProviderAdapter(server.Client())
	groups, warnings, err := adapter.FetchGroups(context.Background(), UpstreamProviderConfig{
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
		t.Fatalf("FetchGroups returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if groupsRequests != 1 {
		t.Fatalf("groups endpoint requests = %d, want 1", groupsRequests)
	}
	byName := map[string]UpstreamProviderGroup{}
	for _, group := range groups {
		byName[group.GroupName] = group
	}
	if len(byName) != 2 {
		t.Fatalf("groups = %+v, want 2 groups", groups)
	}
	if byName["VIP"].RateMultiplier != 3.25 || byName["VIP"].RawGroupID != "7" {
		t.Fatalf("VIP group = %+v, want ratio 3.25 id 7", byName["VIP"])
	}
	if byName["No Key Group"].RateMultiplier != 0.15 || byName["No Key Group"].RawGroupID != "8" {
		t.Fatalf("No Key Group = %+v, want ratio 0.15 id 8", byName["No Key Group"])
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

func TestSub2APIProviderAdapterReusesCachedTokenAcrossKeysAndModelSquare(t *testing.T) {
	var loginRequests int
	var keyRequests int
	var modelRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/login":
			loginRequests++
			_, _ = w.Write([]byte(`{"code":0,"data":{"access_token":"shared-token","token_type":"Bearer"}}`))
		case "/api/admin/keys":
			keyRequests++
			if r.Header.Get("Authorization") != "Bearer shared-token" {
				t.Fatalf("Authorization = %q, want Bearer shared-token", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"code":0,"data":{"items":[{"name":"sk-live-1","group":{"name":"vip","rate_multiplier":2.5}}]}}`))
		case "/api/v1/model-square":
			modelRequests++
			if r.Header.Get("Authorization") != "Bearer shared-token" {
				t.Fatalf("Authorization = %q, want Bearer shared-token", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"groups":[],"models":[]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewSub2APIProviderAdapter(server.Client())
	provider := UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "sub2api-main",
		Name:       "Sub2API main",
		BaseURL:    server.URL,
		LoginURL:   "/api/v1/auth/login",
		APIKeysURL: "/api/admin/keys",
		Email:      "admin@example.com",
		Password:   "secret",
	}
	if _, _, err := adapter.FetchKeys(context.Background(), provider); err != nil {
		t.Fatalf("FetchKeys returned error: %v", err)
	}
	if _, err := adapter.FetchModelSquare(context.Background(), provider); err != nil {
		t.Fatalf("FetchModelSquare returned error: %v", err)
	}
	if loginRequests != 1 {
		t.Fatalf("login requests = %d, want 1", loginRequests)
	}
	if keyRequests != 1 || modelRequests != 1 {
		t.Fatalf("key/model requests = %d/%d, want 1/1", keyRequests, modelRequests)
	}
}

func TestSub2APIProviderAdapterFetchBalanceUsesCachedToken(t *testing.T) {
	var loginRequests int
	var balanceRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/login":
			loginRequests++
			_, _ = w.Write([]byte(`{"code":0,"data":{"access_token":"cached-token","token_type":"Bearer"}}`))
		case "/api/v1/auth/me":
			balanceRequests++
			if r.URL.Query().Get("timezone") != "Asia/Shanghai" {
				t.Fatalf("timezone query = %q, want Asia/Shanghai", r.URL.Query().Get("timezone"))
			}
			if r.Header.Get("Authorization") != "Bearer cached-token" {
				t.Fatalf("Authorization = %q, want Bearer cached-token", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"balance":334.74079414}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewSub2APIProviderAdapter(server.Client())
	provider := UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "sub2api-main",
		Name:       "Sub2API main",
		BaseURL:    server.URL,
		LoginURL:   "/api/v1/auth/login",
		APIKeysURL: "/api/admin/keys",
		Email:      "admin@example.com",
		Password:   "secret",
	}
	balance, err := adapter.FetchBalance(context.Background(), provider)
	if err != nil {
		t.Fatalf("FetchBalance returned error: %v", err)
	}
	if balance.ProviderSlug != provider.Slug || balance.Balance != 334.74079414 {
		t.Fatalf("balance = %+v, want provider slug and raw sub2api balance", balance)
	}
	if loginRequests != 1 || balanceRequests != 1 {
		t.Fatalf("login/balance requests = %d/%d, want 1/1", loginRequests, balanceRequests)
	}
}

func TestSub2APIProviderAdapterFetchBalanceUsesConfiguredURL(t *testing.T) {
	var requestedURI string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/login":
			_, _ = w.Write([]byte(`{"code":0,"data":{"access_token":"cached-token","token_type":"Bearer"}}`))
		case "/api/custom/balance":
			requestedURI = r.URL.RequestURI()
			if r.Header.Get("Authorization") != "Bearer cached-token" {
				t.Fatalf("Authorization = %q, want Bearer cached-token", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"balance":42.5}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewSub2APIProviderAdapter(server.Client())
	balance, err := adapter.FetchBalance(context.Background(), UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "sub2api-main",
		Name:       "Sub2API main",
		BaseURL:    server.URL,
		LoginURL:   "/api/v1/auth/login",
		APIKeysURL: "/api/admin/keys",
		BalanceURL: "/api/custom/balance?source=config",
		Email:      "admin@example.com",
		Password:   "secret",
	})
	if err != nil {
		t.Fatalf("FetchBalance returned error: %v", err)
	}
	if balance.Balance != 42.5 {
		t.Fatalf("balance = %+v, want configured endpoint balance", balance)
	}
	if requestedURI != "/api/custom/balance?source=config" {
		t.Fatalf("requested URI = %q, want configured balance URL", requestedURI)
	}
}

func TestSub2APIProviderAdapterFetchTodayCostUsesConfiguredURL(t *testing.T) {
	var requestedURI string
	day := time.Date(2026, 6, 17, 0, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/login":
			_, _ = w.Write([]byte(`{"code":0,"data":{"access_token":"cached-token","token_type":"Bearer"}}`))
		case "/api/custom/usage":
			requestedURI = r.URL.RequestURI()
			if r.Header.Get("Authorization") != "Bearer cached-token" {
				t.Fatalf("Authorization = %q, want Bearer cached-token", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"today_actual_cost":70.45062742}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewSub2APIProviderAdapter(server.Client())
	cost, err := adapter.FetchTodayCost(context.Background(), UpstreamProviderConfig{
		Type:         UpstreamProviderTypeSub2API,
		Slug:         "sub2api-main",
		Name:         "Sub2API main",
		BaseURL:      server.URL,
		LoginURL:     "/api/v1/auth/login",
		APIKeysURL:   "/api/admin/keys",
		UsageCostURL: "/api/custom/usage?timezone=Asia%2FShanghai",
		Email:        "admin@example.com",
		Password:     "secret",
	}, day)
	if err != nil {
		t.Fatalf("FetchTodayCost returned error: %v", err)
	}
	if cost.ProviderSlug != "sub2api-main" || cost.TodayCost != 70.45062742 {
		t.Fatalf("cost = %+v, want parsed today_actual_cost", cost)
	}
	if requestedURI != "/api/custom/usage?timezone=Asia%2FShanghai" {
		t.Fatalf("requested URI = %q, want configured usage cost URL", requestedURI)
	}
}

func TestSub2APIProviderAdapterRefreshesCachedTokenAfterUnauthorized(t *testing.T) {
	var loginRequests int
	var keyRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/login":
			loginRequests++
			token := "token-1"
			if loginRequests > 1 {
				token = "token-2"
			}
			_, _ = w.Write([]byte(`{"code":0,"data":{"access_token":"` + token + `","token_type":"Bearer"}}`))
		case "/api/admin/keys":
			keyRequests++
			switch r.Header.Get("Authorization") {
			case "Bearer token-1":
				if keyRequests == 1 {
					_, _ = w.Write([]byte(`{"code":0,"data":{"items":[{"name":"sk-live-1","group":{"name":"vip","rate_multiplier":2.5}}]}}`))
					return
				}
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"code":401,"message":"token expired"}`))
			case "Bearer token-2":
				_, _ = w.Write([]byte(`{"code":0,"data":{"items":[{"name":"sk-live-2","group":{"name":"vip","rate_multiplier":2.5}}]}}`))
			default:
				t.Fatalf("unexpected Authorization %q", r.Header.Get("Authorization"))
			}
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewSub2APIProviderAdapter(server.Client())
	provider := UpstreamProviderConfig{
		Type:       UpstreamProviderTypeSub2API,
		Slug:       "sub2api-main",
		Name:       "Sub2API main",
		BaseURL:    server.URL,
		LoginURL:   "/api/v1/auth/login",
		APIKeysURL: "/api/admin/keys",
		Email:      "admin@example.com",
		Password:   "secret",
	}
	if _, _, err := adapter.FetchKeys(context.Background(), provider); err != nil {
		t.Fatalf("first FetchKeys returned error: %v", err)
	}
	keys, _, err := adapter.FetchKeys(context.Background(), provider)
	if err != nil {
		t.Fatalf("second FetchKeys returned error: %v", err)
	}
	if loginRequests != 2 {
		t.Fatalf("login requests = %d, want 2", loginRequests)
	}
	if keyRequests != 3 {
		t.Fatalf("key requests = %d, want 3", keyRequests)
	}
	if len(keys) != 1 || keys[0].KeyName != "sk-live-2" {
		t.Fatalf("keys = %+v, want refreshed token result", keys)
	}
}

func TestNewAPIProviderAdapterReusesCachedSessionAcrossKeysAndModelSquare(t *testing.T) {
	var loginRequests int
	var keyRequests int
	var modelRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			loginRequests++
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":42}}`))
		case "/api/token/":
			keyRequests++
			if r.Header.Get("New-Api-User") != "42" || !strings.Contains(r.Header.Get("Cookie"), "session=abc") {
				t.Fatalf("unexpected key request auth user=%q cookie=%q", r.Header.Get("New-Api-User"), r.Header.Get("Cookie"))
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"items":[{"name":"key-1","group":"VIP"}]}}`))
		case "/api/group/":
			_, _ = w.Write([]byte(`{"success":true,"data":{"VIP":{"ratio":3.25}}}`))
		case "/api/v1/model-square":
			modelRequests++
			if r.Header.Get("New-Api-User") != "42" || !strings.Contains(r.Header.Get("Cookie"), "session=abc") {
				t.Fatalf("unexpected model request auth user=%q cookie=%q", r.Header.Get("New-Api-User"), r.Header.Get("Cookie"))
			}
			_, _ = w.Write([]byte(`{"groups":[],"models":[]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewNewAPIProviderAdapter(server.Client())
	provider := UpstreamProviderConfig{
		Type:       UpstreamProviderTypeNewAPI,
		Slug:       "newapi-main",
		Name:       "NewAPI main",
		BaseURL:    server.URL,
		LoginURL:   "/api/user/login",
		APIKeysURL: "/api/token/",
		GroupsURL:  "/api/group/",
		Username:   "root",
		Password:   "secret",
	}
	if _, _, err := adapter.FetchKeys(context.Background(), provider); err != nil {
		t.Fatalf("FetchKeys returned error: %v", err)
	}
	if _, err := adapter.FetchModelSquare(context.Background(), provider); err != nil {
		t.Fatalf("FetchModelSquare returned error: %v", err)
	}
	if loginRequests != 1 {
		t.Fatalf("login requests = %d, want 1", loginRequests)
	}
	if keyRequests != 1 || modelRequests != 1 {
		t.Fatalf("key/model requests = %d/%d, want 1/1", keyRequests, modelRequests)
	}
}

func TestNewAPIProviderAdapterFetchBalanceConvertsQuota(t *testing.T) {
	var loginRequests int
	var balanceRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			loginRequests++
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":42}}`))
		case "/api/user/self":
			balanceRequests++
			if r.Header.Get("New-Api-User") != "42" {
				t.Fatalf("New-Api-User = %q, want 42", r.Header.Get("New-Api-User"))
			}
			if !strings.Contains(r.Header.Get("Cookie"), "session=abc") {
				t.Fatalf("Cookie header = %q, want session cookie", r.Header.Get("Cookie"))
			}
			_, _ = w.Write([]byte(`{"success":true,"message":"","data":{"quota":9402397}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewNewAPIProviderAdapter(server.Client())
	provider := UpstreamProviderConfig{
		Type:       UpstreamProviderTypeNewAPI,
		Slug:       "newapi-main",
		Name:       "NewAPI main",
		BaseURL:    server.URL,
		LoginURL:   "/api/user/login",
		APIKeysURL: "/api/token/",
		GroupsURL:  "/api/group/",
		Username:   "root",
		Password:   "secret",
	}
	balance, err := adapter.FetchBalance(context.Background(), provider)
	if err != nil {
		t.Fatalf("FetchBalance returned error: %v", err)
	}
	if balance.ProviderSlug != provider.Slug || balance.Balance != 18.804794 {
		t.Fatalf("balance = %+v, want converted newapi quota", balance)
	}
	if loginRequests != 1 || balanceRequests != 1 {
		t.Fatalf("login/balance requests = %d/%d, want 1/1", loginRequests, balanceRequests)
	}
}

func TestNewAPIProviderAdapterFetchBalanceUsesConfiguredURL(t *testing.T) {
	var requestedURI string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":42}}`))
		case "/api/custom/self":
			requestedURI = r.URL.RequestURI()
			if r.Header.Get("New-Api-User") != "42" {
				t.Fatalf("New-Api-User = %q, want 42", r.Header.Get("New-Api-User"))
			}
			if !strings.Contains(r.Header.Get("Cookie"), "session=abc") {
				t.Fatalf("Cookie header = %q, want session cookie", r.Header.Get("Cookie"))
			}
			_, _ = w.Write([]byte(`{"success":true,"message":"","data":{"quota":1000000}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewNewAPIProviderAdapter(server.Client())
	balance, err := adapter.FetchBalance(context.Background(), UpstreamProviderConfig{
		Type:       UpstreamProviderTypeNewAPI,
		Slug:       "newapi-main",
		Name:       "NewAPI main",
		BaseURL:    server.URL,
		LoginURL:   "/api/user/login",
		APIKeysURL: "/api/token/",
		GroupsURL:  "/api/group/",
		BalanceURL: "/api/custom/self?source=config",
		Username:   "root",
		Password:   "secret",
	})
	if err != nil {
		t.Fatalf("FetchBalance returned error: %v", err)
	}
	if balance.Balance != 2 {
		t.Fatalf("balance = %+v, want converted configured endpoint quota", balance)
	}
	if requestedURI != "/api/custom/self?source=config" {
		t.Fatalf("requested URI = %q, want configured balance URL", requestedURI)
	}
}

func TestNewAPIProviderAdapterFetchTodayCostUsesConfiguredURLWithDayTimestamps(t *testing.T) {
	var requestedURI string
	day := time.Date(2026, 6, 17, 12, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":42}}`))
		case "/api/log/self/stat":
			requestedURI = r.URL.RequestURI()
			if r.Header.Get("New-Api-User") != "42" {
				t.Fatalf("New-Api-User = %q, want 42", r.Header.Get("New-Api-User"))
			}
			if !strings.Contains(r.Header.Get("Cookie"), "session=abc") {
				t.Fatalf("Cookie header = %q, want session cookie", r.Header.Get("Cookie"))
			}
			if r.URL.Query().Get("start_timestamp") != "1781625600" {
				t.Fatalf("start_timestamp = %q, want 1781625600", r.URL.Query().Get("start_timestamp"))
			}
			if r.URL.Query().Get("end_timestamp") != "1781711999" {
				t.Fatalf("end_timestamp = %q, want 1781711999", r.URL.Query().Get("end_timestamp"))
			}
			_, _ = w.Write([]byte(`{"success":true,"message":"","data":{"quota":1306899,"rpm":0,"tpm":0}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewNewAPIProviderAdapter(server.Client())
	cost, err := adapter.FetchTodayCost(context.Background(), UpstreamProviderConfig{
		Type:         UpstreamProviderTypeNewAPI,
		Slug:         "newapi-main",
		Name:         "NewAPI main",
		BaseURL:      server.URL,
		LoginURL:     "/api/user/login",
		APIKeysURL:   "/api/token/",
		GroupsURL:    "/api/group/",
		UsageCostURL: "/api/log/self/stat?type=0&token_name=&model_name=&start_timestamp={start_timestamp}&end_timestamp={end_timestamp}&group=",
		Username:     "root",
		Password:     "secret",
	}, day)
	if err != nil {
		t.Fatalf("FetchTodayCost returned error: %v", err)
	}
	if cost.ProviderSlug != "newapi-main" || cost.TodayCost != 2.613798 {
		t.Fatalf("cost = %+v, want quota converted to cost", cost)
	}
	if !strings.Contains(requestedURI, "type=0") || !strings.Contains(requestedURI, "group=") {
		t.Fatalf("requested URI = %q, want configured query preserved", requestedURI)
	}
}

func TestNewAPIProviderAdapterRefreshesCachedSessionAfterUnauthorized(t *testing.T) {
	var loginRequests int
	var keyRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/login":
			loginRequests++
			cookieValue := "session-1"
			if loginRequests > 1 {
				cookieValue = "session-2"
			}
			http.SetCookie(w, &http.Cookie{Name: "session", Value: cookieValue})
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":42}}`))
		case "/api/token/":
			keyRequests++
			cookie := r.Header.Get("Cookie")
			switch {
			case strings.Contains(cookie, "session=session-1"):
				if keyRequests == 1 {
					_, _ = w.Write([]byte(`{"success":true,"data":{"items":[{"name":"key-1","group":"VIP"}]}}`))
					return
				}
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"success":false,"message":"session expired"}`))
			case strings.Contains(cookie, "session=session-2"):
				_, _ = w.Write([]byte(`{"success":true,"data":{"items":[{"name":"key-2","group":"VIP"}]}}`))
			default:
				t.Fatalf("unexpected Cookie %q", cookie)
			}
		case "/api/group/":
			_, _ = w.Write([]byte(`{"success":true,"data":{"VIP":{"ratio":3.25}}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewNewAPIProviderAdapter(server.Client())
	provider := UpstreamProviderConfig{
		Type:       UpstreamProviderTypeNewAPI,
		Slug:       "newapi-main",
		Name:       "NewAPI main",
		BaseURL:    server.URL,
		LoginURL:   "/api/user/login",
		APIKeysURL: "/api/token/",
		GroupsURL:  "/api/group/",
		Username:   "root",
		Password:   "secret",
	}
	if _, _, err := adapter.FetchKeys(context.Background(), provider); err != nil {
		t.Fatalf("first FetchKeys returned error: %v", err)
	}
	keys, _, err := adapter.FetchKeys(context.Background(), provider)
	if err != nil {
		t.Fatalf("second FetchKeys returned error: %v", err)
	}
	if loginRequests != 2 {
		t.Fatalf("login requests = %d, want 2", loginRequests)
	}
	if keyRequests != 3 {
		t.Fatalf("key requests = %d, want 3", keyRequests)
	}
	if len(keys) != 1 || keys[0].KeyName != "key-2" {
		t.Fatalf("keys = %+v, want refreshed session result", keys)
	}
}

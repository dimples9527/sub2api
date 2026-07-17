package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type upstreamProviderHandlerSettingRepo struct {
	mu     sync.Mutex
	values map[string]string
}

func newUpstreamProviderHandlerSettingRepo() *upstreamProviderHandlerSettingRepo {
	return &upstreamProviderHandlerSettingRepo{values: map[string]string{}}
}

func (r *upstreamProviderHandlerSettingRepo) Get(ctx context.Context, key string) (*service.Setting, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.values[key]
	if !ok {
		return nil, service.ErrSettingNotFound
	}
	return &service.Setting{Key: key, Value: value}, nil
}

func (r *upstreamProviderHandlerSettingRepo) GetValue(ctx context.Context, key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (r *upstreamProviderHandlerSettingRepo) Set(ctx context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[key] = value
	return nil
}

func (r *upstreamProviderHandlerSettingRepo) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
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

func (r *upstreamProviderHandlerSettingRepo) SetMultiple(ctx context.Context, settings map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *upstreamProviderHandlerSettingRepo) GetAll(ctx context.Context) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]string, len(r.values))
	for key, value := range r.values {
		out[key] = value
	}
	return out, nil
}

func (r *upstreamProviderHandlerSettingRepo) Delete(ctx context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.values, key)
	return nil
}

func newUpstreamProviderHandlerTestRouter(svc *service.UpstreamProviderService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewUpstreamProviderHandler(svc)
	router.GET("/admin/upstream-management/providers", handler.List)
	router.POST("/admin/upstream-management/providers", handler.Create)
	router.PUT("/admin/upstream-management/providers/:slug", handler.Update)
	router.DELETE("/admin/upstream-management/providers/:slug", handler.Delete)
	router.POST("/admin/upstream-management/providers/:slug/default", handler.SetDefault)
	router.POST("/admin/upstream-management/providers/:slug/test", handler.TestSaved)
	router.GET("/admin/upstream-management/providers/:slug/keys", handler.Keys)
	router.GET("/admin/upstream-management/providers/:slug/balance", handler.Balance)
	return router
}

func TestUpstreamProviderHandlerCreateListAndUpdate(t *testing.T) {
	repo := newUpstreamProviderHandlerSettingRepo()
	svc := service.NewUpstreamProviderService(repo)
	router := newUpstreamProviderHandlerTestRouter(svc)

	body := []byte(`{
		"type": "sub2api",
		"slug": "primary",
		"name": "Primary",
		"enabled": true,
		"base_url": "https://upstream.example.com",
		"api_keys_url": "/api/admin/keys",
		"password": "secret"
	}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/upstream-management/providers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var createResp struct {
		Code int                            `json:"code"`
		Data service.UpstreamProviderConfig `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	require.Equal(t, "primary", createResp.Data.Slug)
	require.Empty(t, createResp.Data.Password)
	require.True(t, createResp.Data.PasswordConfigured)

	update := []byte(`{
		"type": "sub2api",
		"slug": "primary",
		"name": "Renamed",
		"enabled": false,
		"base_url": "https://upstream.example.com",
		"api_keys_url": "/api/admin/keys"
	}`)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/admin/upstream-management/providers/primary", bytes.NewReader(update))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin/upstream-management/providers", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var listResp struct {
		Code int                              `json:"code"`
		Data []service.UpstreamProviderConfig `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listResp))
	require.Len(t, listResp.Data, 1)
	require.Equal(t, "Renamed", listResp.Data[0].Name)
	require.Empty(t, listResp.Data[0].Password)
	require.True(t, listResp.Data[0].PasswordConfigured)
}

func TestUpstreamProviderHandlerTestSavedProvider(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/admin/keys", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"items":[{"name":"key-a","group":{"name":"vip","rate_multiplier":2}}]}}`))
	}))
	defer upstream.Close()

	repo := newUpstreamProviderHandlerSettingRepo()
	svc := service.NewUpstreamProviderServiceWithHTTPClient(repo, upstream.Client())
	_, err := svc.CreateProvider(context.Background(), service.UpstreamProviderConfig{
		Type:       service.UpstreamProviderTypeSub2API,
		Slug:       "primary",
		Name:       "Primary",
		Enabled:    true,
		BaseURL:    upstream.URL,
		APIKeysURL: "/api/admin/keys",
	})
	require.NoError(t, err)
	router := newUpstreamProviderHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/upstream-management/providers/primary/test", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int                                `json:"code"`
		Data service.UpstreamProviderTestResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.True(t, resp.Data.Keys.OK)
	require.Len(t, resp.Data.ParsedKeys, 1)
	require.Equal(t, "key-a", resp.Data.ParsedKeys[0].KeyName)
}

func TestUpstreamProviderHandlerSetDefaultProvider(t *testing.T) {
	repo := newUpstreamProviderHandlerSettingRepo()
	svc := service.NewUpstreamProviderService(repo)
	_, err := svc.CreateProvider(context.Background(), service.UpstreamProviderConfig{
		Type:       service.UpstreamProviderTypeSub2API,
		Slug:       "primary",
		Name:       "Primary",
		Enabled:    true,
		BaseURL:    "https://primary.example.com",
		APIKeysURL: "/api/admin/keys",
	})
	require.NoError(t, err)
	router := newUpstreamProviderHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/upstream-management/providers/primary/default", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int                            `json:"code"`
		Data service.UpstreamProviderConfig `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.True(t, resp.Data.IsDefault)
	require.Equal(t, "primary", resp.Data.Slug)
}

func TestUpstreamProviderHandlerFetchBalance(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/login":
			_, _ = w.Write([]byte(`{"code":0,"data":{"access_token":"token","token_type":"Bearer"}}`))
		case "/api/v1/auth/me":
			require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{"code":0,"data":{"balance":12.5}}`))
		case "/api/v1/usage/dashboard/stats":
			require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{"code":0,"data":{"today_actual_cost":3.25}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	repo := newUpstreamProviderHandlerSettingRepo()
	svc := service.NewUpstreamProviderServiceWithHTTPClient(repo, upstream.Client())
	_, err := svc.CreateProvider(context.Background(), service.UpstreamProviderConfig{
		Type:       service.UpstreamProviderTypeSub2API,
		Slug:       "primary",
		Name:       "Primary",
		Enabled:    true,
		BaseURL:    upstream.URL,
		LoginURL:   "/api/v1/auth/login",
		APIKeysURL: "/api/admin/keys",
		Email:      "admin@example.com",
		Password:   "secret",
	})
	require.NoError(t, err)
	router := newUpstreamProviderHandlerTestRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/providers/primary/balance", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int                                   `json:"code"`
		Data service.UpstreamProviderBalanceStatus `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "primary", resp.Data.ProviderSlug)
	require.Equal(t, 12.5, resp.Data.Balance)
	require.Equal(t, 3.25, resp.Data.TodayCost)
}

package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMaskFindCGEmail(t *testing.T) {
	require.Equal(t, "a***@example.com", maskFindCGEmail("a@example.com"))
	require.Equal(t, "te***st@example.com", maskFindCGEmail("test@example.com"))
	require.Equal(t, "***", maskFindCGEmail(""))
}

func TestResolveConfigReportsSource(t *testing.T) {
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{ModelSquare: config.ModelSquareConfig{
			BaseURL:  "https://yaml.example.com",
			Email:    "yaml@example.com",
			Password: "yaml-secret",
		}},
	})

	baseURL, _, _, _, _, email, password, source := h.resolveConfigLocked(context.Background())

	require.Equal(t, "https://yaml.example.com", baseURL)
	require.Equal(t, "yaml@example.com", email)
	require.Equal(t, "yaml-secret", password)
	require.Equal(t, "config_or_env", source)
}

func TestResolveConfigReportsSettingsSource(t *testing.T) {
	settingSvc := service.NewSettingService(&modelSquareSettingRepo{values: map[string]string{
		service.SettingKeyModelSquareBaseURL:  "https://db.example.com",
		service.SettingKeyModelSquareEmail:    "db@example.com",
		service.SettingKeyModelSquarePassword: "db-secret",
	}}, &config.Config{})
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig:      &config.Config{ModelSquare: config.ModelSquareConfig{BaseURL: "https://yaml.example.com", Email: "yaml@example.com", Password: "yaml-secret"}},
		SettingService: settingSvc,
	})

	baseURL, _, _, _, _, email, password, source := h.resolveConfigLocked(context.Background())

	require.Equal(t, "https://db.example.com", baseURL)
	require.Equal(t, "db@example.com", email)
	require.Equal(t, "db-secret", password)
	require.Equal(t, "settings", source)
}

func TestLogFindCGLoginAttemptDoesNotExposePassword(t *testing.T) {
	recorder := &slogRecordHandler{}
	original := slog.Default()
	slog.SetDefault(slog.New(recorder))
	defer slog.SetDefault(original)

	logFindCGLoginAttempt("https://www.findcg.com", "test@example.com", "secret", "settings")

	require.Len(t, recorder.records, 1)
	require.NotContains(t, recorder.records[0], "secret")
	require.Contains(t, recorder.records[0], "te***st@example.com")
	require.Contains(t, recorder.records[0], "password_configured=true")
}

type slogRecordHandler struct {
	records []string
}

func (h *slogRecordHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *slogRecordHandler) WithAttrs([]slog.Attr) slog.Handler       { return h }
func (h *slogRecordHandler) WithGroup(string) slog.Handler            { return h }
func (h *slogRecordHandler) Handle(_ context.Context, r slog.Record) error {
	msg := r.Message
	r.Attrs(func(a slog.Attr) bool {
		msg += " " + a.Key + "=" + a.Value.String()
		return true
	})
	h.records = append(h.records, msg)
	return nil
}

func TestModelSquareHandlerFetchesWithLoginToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var modelSquareAuth string
	var loginBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			require.Equal(t, http.MethodPost, r.Method)
			buf := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(buf)
			loginBody = string(buf)
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"token-1","expires_in":86400,"token_type":"Bearer"}}`))
		case "/api/v1/model-square":
			modelSquareAuth = r.Header.Get("Authorization")
			_, _ = w.Write([]byte(`{"groups":[],"models":[{"id":"gpt-test","provider":"openai","available":true}],"updated_at":"2026-05-30T22:02:54+08:00"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{ModelSquare: config.ModelSquareConfig{
			BaseURL:  server.URL,
			Email:    "configured@example.com",
			Password: "configured-secret",
		}},
		HTTPClient: server.Client(),
	})
	router := gin.New()
	router.GET("/admin/model-square", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/admin/model-square", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"email":"configured@example.com","password":"configured-secret"}`, loginBody)
	require.Equal(t, "Bearer token-1", modelSquareAuth)
	require.Contains(t, rec.Body.String(), `"id":"gpt-test"`)
}

func TestModelSquareHandlerUsesDBBackedCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var loginBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			buf := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(buf)
			loginBody = string(buf)
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"token-db","expires_in":86400,"token_type":"Bearer"}}`))
		case "/api/v1/model-square":
			require.Equal(t, "Bearer token-db", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{"groups":[],"models":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	settingSvc := service.NewSettingService(&modelSquareSettingRepo{values: map[string]string{
		service.SettingKeyModelSquareBaseURL:  server.URL,
		service.SettingKeyModelSquareEmail:    "db@example.com",
		service.SettingKeyModelSquarePassword: "db-secret",
	}}, &config.Config{})
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig:      &config.Config{ModelSquare: config.ModelSquareConfig{Email: "yaml@example.com", Password: "yaml-secret"}},
		SettingService: settingSvc,
		HTTPClient:     server.Client(),
	})
	router := gin.New()
	router.GET("/admin/model-square", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/admin/model-square", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"email":"db@example.com","password":"db-secret"}`, loginBody)
}

func TestModelSquareHandlerUsesConfiguredEndpointURLs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/custom/login":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"token-custom","expires_in":86400,"token_type":"Bearer"}}`))
		case "/custom/models":
			require.Equal(t, "Bearer token-custom", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{"groups":[],"models":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{ModelSquare: config.ModelSquareConfig{
			BaseURL:        "https://unused.example.com",
			LoginURL:       server.URL + "/custom/login",
			ModelSquareURL: server.URL + "/custom/models",
			Email:          "configured@example.com",
			Password:       "configured-secret",
		}},
		HTTPClient: server.Client(),
	})
	router := gin.New()
	router.GET("/admin/model-square", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/admin/model-square", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []string{"/custom/login", "/custom/models"}, paths)
}

func TestModelSquareHandlerUsesSettingsEndpointURLs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/settings/login":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"token-settings-url","expires_in":86400,"token_type":"Bearer"}}`))
		case "/settings/models":
			require.Equal(t, "Bearer token-settings-url", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{"groups":[],"models":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	settingSvc := service.NewSettingService(&modelSquareSettingRepo{values: map[string]string{
		service.SettingKeyModelSquareBaseURL:  "https://unused.example.com",
		service.SettingKeyModelSquareLoginURL: server.URL + "/settings/login",
		service.SettingKeyModelSquareModelURL: server.URL + "/settings/models",
		service.SettingKeyModelSquareEmail:    "db@example.com",
		service.SettingKeyModelSquarePassword: "db-secret",
	}}, &config.Config{})
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig:      &config.Config{ModelSquare: config.ModelSquareConfig{Email: "yaml@example.com", Password: "yaml-secret"}},
		SettingService: settingSvc,
		HTTPClient:     server.Client(),
	})
	router := gin.New()
	router.GET("/admin/model-square", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/admin/model-square", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []string{"/settings/login", "/settings/models"}, paths)
}

func TestModelSquareHandlerGetsAvailableGroupsFromConfiguredURLWithLocalMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/custom/login":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"token-groups","expires_in":86400,"token_type":"Bearer"}}`))
		case "/custom/groups":
			require.Equal(t, "Bearer token-groups", r.Header.Get("Authorization"))
			require.Equal(t, "Asia/Shanghai", r.URL.Query().Get("timezone"))
			_, _ = w.Write([]byte(`{
				"code":0,
				"message":"success",
				"data":[
					{"id":2,"name":"codex福利","platform":"openai","rate_multiplier":0.15,"status":"active"},
					{"id":5,"name":"unmatched","platform":"anthropic","rate_multiplier":0.75,"status":"active"}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	groupProvider := &modelSquareGroupProviderStub{groups: []service.Group{
		{ID: 10, Name: "codex 福利", RateMultiplier: 0.2},
	}}
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{ModelSquare: config.ModelSquareConfig{
			BaseURL:   "https://unused.example.com",
			LoginURL:  server.URL + "/custom/login",
			GroupsURL: server.URL + "/custom/groups?timezone=Asia%2FShanghai",
			Email:     "configured@example.com",
			Password:  "configured-secret",
		}},
		GroupProvider: groupProvider,
		HTTPClient:    server.Client(),
	})
	router := gin.New()
	router.GET("/admin/upstream-management/groups", h.GetAvailableGroups)

	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/groups", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []string{"/custom/login", "/custom/groups"}, paths)
	require.JSONEq(t, `{
		"code":0,
		"message":"success",
		"data":[
			{"id":2,"name":"codex福利","platform":"openai","rate_multiplier":0.15,"status":"active","local_group_id":10,"local_group_name":"codex 福利","local_rate_multiplier":0.2},
			{"id":5,"name":"unmatched","platform":"anthropic","rate_multiplier":0.75,"status":"active","local_group_id":null,"local_group_name":"","local_rate_multiplier":null}
		]
	}`, rec.Body.String())
}

func TestModelSquareHandlerMergesGroupsByNormalizedName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	createdAt := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"token-merge","expires_in":86400,"token_type":"Bearer"}}`))
		case "/api/v1/model-square":
			require.Equal(t, "Bearer token-merge", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{
				"groups":[
					{"id":"remote-1","name":" Codex 特价 ","rate_multiplier":9.9,"description":"upstream"},
					{"id":2002,"name":"unmatched","rate_multiplier":2.2}
				],
				"models":[{"id":"gpt-test","group_ids":["remote-1",2002]}]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{ModelSquare: config.ModelSquareConfig{
			BaseURL:  server.URL,
			Email:    "configured@example.com",
			Password: "configured-secret",
		}},
		GroupProvider: &modelSquareGroupProviderStub{groups: []service.Group{
			{
				ID:             99,
				Name:           "codex特价",
				Description:    "local group",
				Platform:       service.PlatformOpenAI,
				RateMultiplier: 0.25,
				Status:         service.StatusActive,
				CreatedAt:      createdAt,
				UpdatedAt:      updatedAt,
			},
		}},
		HTTPClient: server.Client(),
	})
	router := gin.New()
	router.GET("/admin/model-square", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/admin/model-square", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Groups []map[string]any `json:"groups"`
		Models []struct {
			GroupIDs []any `json:"group_ids"`
		} `json:"models"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Groups, 1)
	require.Equal(t, "remote-1", body.Groups[0]["id"])
	require.Equal(t, "codex特价", body.Groups[0]["name"])
	require.Equal(t, "local group", body.Groups[0]["description"])
	require.Equal(t, service.PlatformOpenAI, body.Groups[0]["platform"])
	require.Equal(t, 0.25, body.Groups[0]["rate_multiplier"])
	require.Equal(t, service.StatusActive, body.Groups[0]["status"])
	require.Equal(t, []any{"remote-1"}, body.Models[0].GroupIDs)
}

func TestModelSquareHandlerManualSyncRaisesLocalGroupRateFromKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var keysAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"token-keys","expires_in":86400,"token_type":"Bearer"}}`))
		case "/api/v1/keys":
			keysAuth = r.Header.Get("Authorization")
			require.Equal(t, "1", r.URL.Query().Get("page"))
			require.Equal(t, "100", r.URL.Query().Get("page_size"))
			require.Equal(t, "Asia/Shanghai", r.URL.Query().Get("timezone"))
			_, _ = w.Write([]byte(`{
				"code":0,
				"message":"success",
				"data":{
					"items":[
						{"id":1,"name":" Codex Special ","group":{"rate_multiplier":0.6}},
						{"id":2,"name":"Stable Group","group":{"rate_multiplier":0.25}},
						{"id":3,"name":"Unmatched Group","group":{"rate_multiplier":9.9}},
						{"id":4,"name":"codexspecial","group":{"rate_multiplier":0.7}}
					],
					"total":4,
					"page":1,
					"page_size":100,
					"pages":1
				}
			}`))
		case "/api/v1/model-square":
			require.Equal(t, "Bearer token-keys", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{"groups":[],"models":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	groupProvider := &modelSquareGroupProviderStub{groups: []service.Group{
		{ID: 10, Name: "codex special", RateMultiplier: 0.5},
		{ID: 20, Name: "StableGroup", RateMultiplier: 0.4},
	}}
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{ModelSquare: config.ModelSquareConfig{
			BaseURL:  server.URL,
			Email:    "configured@example.com",
			Password: "configured-secret",
		}},
		GroupProvider: groupProvider,
		HTTPClient:    server.Client(),
	})
	router := gin.New()
	router.POST("/admin/model-square/sync", h.SyncKeys)

	req := httptest.NewRequest(http.MethodPost, "/admin/model-square/sync", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "Bearer token-keys", keysAuth)
	require.Equal(t, []modelSquareGroupRateUpdate{{id: 10, rate: 0.7}}, groupProvider.updatesSnapshot())
	require.JSONEq(t, `{
		"code":0,
		"message":"success",
		"data":{
			"checked_count":4,
			"matched_count":3,
			"updated_count":1,
			"rate_warnings":[
				{"group_id":10,"group_name":"codex special","local_rate_multiplier":0.5,"upstream_rate_multiplier":0.7}
			]
		}
	}`, rec.Body.String())
}

func TestModelSquareHandlerGetsKeySummaryFromConfiguredAPIKeysURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/custom/login":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"token-key-summary","expires_in":86400,"token_type":"Bearer"}}`))
		case "/custom/keys":
			require.Equal(t, "Bearer token-key-summary", r.Header.Get("Authorization"))
			require.Equal(t, "1", r.URL.Query().Get("page"))
			_, _ = w.Write([]byte(`{
				"code":0,
				"message":"success",
				"data":{
					"items":[
						{"id":1,"name":"key-a","group":{"name":"Codex Special","rate_multiplier":0.6}},
						{"id":2,"name":"key-b","group":{"name":"Codex Special","rate_multiplier":0.7}},
						{"id":3,"name":"key-c","group":{"name":"Stable Group","rate_multiplier":0.4}},
						{"id":4,"name":"sk-secret-no-group"}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{ModelSquare: config.ModelSquareConfig{
			BaseURL:    "https://unused.example.com",
			LoginURL:   server.URL + "/custom/login",
			APIKeysURL: server.URL + "/custom/keys?page=1",
			Email:      "configured@example.com",
			Password:   "configured-secret",
		}},
		HTTPClient: server.Client(),
	})
	router := gin.New()
	router.GET("/admin/upstream-management/key-summary", h.KeySummary)

	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/key-summary", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []string{"/custom/login", "/custom/keys"}, paths)
	require.JSONEq(t, `{
		"code":0,
		"message":"success",
		"data":{
			"groups":[
				{"name":"Codex Special","normalized_name":"codexspecial","key_count":2,"keys":[{"name":"key-a"},{"name":"key-b"}]},
				{"name":"Stable Group","normalized_name":"stablegroup","key_count":1,"keys":[{"name":"key-c"}]}
			]
		}
	}`, rec.Body.String())
	require.NotContains(t, rec.Body.String(), "sk-secret-no-group")
}

func TestModelSquareHandlerAccountRateGuardUnbindsLowRateGroupsWhenLocalGroupRateIsLower(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/custom/login":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"token-guard","expires_in":86400,"token_type":"Bearer"}}`))
		case "/custom/keys":
			require.Equal(t, "Bearer token-guard", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{
				"code":0,
				"message":"success",
				"data":{
					"items":[
						{"id":1,"name":"key-a","group":{"name":"VIP","rate_multiplier":0.8}},
						{"id":2,"name":"key-b","group":{"name":"VIP","rate_multiplier":0.4}}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	groupProvider := &modelSquareGroupProviderStub{accounts: []service.Account{
		{
			ID:       101,
			Name:     "findcg-key-a",
			GroupIDs: []int64{10, 11},
			Groups: []*service.Group{
				{ID: 10, Name: "cheap", RateMultiplier: 0.5},
				{ID: 11, Name: "expensive", RateMultiplier: 1.2},
			},
		},
		{
			ID:       102,
			Name:     "findcg-key-b",
			GroupIDs: []int64{12},
			Groups: []*service.Group{
				{ID: 12, Name: "enough", RateMultiplier: 0.5},
			},
		},
		{
			ID:       103,
			Name:     "other-key-a",
			GroupIDs: []int64{13},
			Groups: []*service.Group{
				{ID: 13, Name: "wrong-upstream", RateMultiplier: 0.1},
			},
		},
	}}
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{UpstreamManagement: config.UpstreamManagementConfig{
			Providers: []config.UpstreamManagementProviderConfig{
				{
					Slug:              "findcg",
					Name:              "FindCG",
					Enabled:           true,
					BaseURL:           "https://unused.example.com",
					LoginURL:          server.URL + "/custom/login",
					APIKeysURL:        server.URL + "/custom/keys",
					Email:             "admin@example.com",
					Password:          "secret",
					AccountNamePrefix: "findcg-",
				},
			},
		}},
		GroupProvider: groupProvider,
		HTTPClient:    server.Client(),
	})
	router := gin.New()
	router.POST("/admin/upstream-management/account-rate-guard/run", h.RunAccountRateGuard)
	router.GET("/admin/upstream-management/account-rate-guard/status", h.GetAccountRateGuardStatus)
	router.GET("/admin/upstream-management/account-rate-guard/audits", h.ListAccountRateGuardAudits)

	req := httptest.NewRequest(http.MethodPost, "/admin/upstream-management/account-rate-guard/run", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []string{"/custom/login", "/custom/keys"}, paths)
	accountUpdates := groupProvider.accountUpdatesSnapshot()
	require.Len(t, accountUpdates, 1)
	require.Equal(t, int64(101), accountUpdates[0].id)
	require.NotNil(t, accountUpdates[0].groupIDs)
	require.Equal(t, []int64{11}, *accountUpdates[0].groupIDs)
	var resp struct {
		Code int                    `json:"code"`
		Data accountRateGuardResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.False(t, resp.Data.DryRun)
	require.Equal(t, 2, resp.Data.CheckedKeyCount)
	require.Equal(t, 2, resp.Data.MatchedAccountCount)
	require.Equal(t, 1, resp.Data.ViolationCount)
	require.Equal(t, 1, resp.Data.UnboundCount)
	require.Equal(t, []accountRateGuardProvider{{Slug: "findcg", Name: "FindCG", AccountNamePrefix: "findcg-", CheckedKeyCount: 2}}, resp.Data.Providers)
	require.Len(t, resp.Data.Violations, 1)
	require.Equal(t, accountRateGuardViolation{
		ProviderSlug:            "findcg",
		ProviderName:            "FindCG",
		UpstreamKeyName:         "key-a",
		MatchedLocalAccountID:   101,
		MatchedLocalAccountName: "findcg-key-a",
		UpstreamGroupName:       "VIP",
		UpstreamRateMultiplier:  0.8,
		LocalMinRateMultiplier:  0.5,
		UnboundGroupIDs:         []int64{10},
		UnboundGroupNames:       []string{"cheap"},
	}, resp.Data.Violations[0])

	statusReq := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/account-rate-guard/status", nil)
	statusRec := httptest.NewRecorder()
	router.ServeHTTP(statusRec, statusReq)
	require.Equal(t, http.StatusOK, statusRec.Code)
	var statusResp struct {
		Code int                    `json:"code"`
		Data accountRateGuardStatus `json:"data"`
	}
	require.NoError(t, json.Unmarshal(statusRec.Body.Bytes(), &statusResp))
	require.NotNil(t, statusResp.Data.LastRun)
	require.Equal(t, int64(1), statusResp.Data.LastRun.RunID)
	require.False(t, statusResp.Data.LastRun.Result.DryRun)
	require.Equal(t, 1, statusResp.Data.LastRun.Result.UnboundCount)

	auditReq := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/account-rate-guard/audits", nil)
	auditRec := httptest.NewRecorder()
	router.ServeHTTP(auditRec, auditReq)
	require.Equal(t, http.StatusOK, auditRec.Code)
	var auditResp struct {
		Code int                          `json:"code"`
		Data []accountRateGuardAuditEntry `json:"data"`
	}
	require.NoError(t, json.Unmarshal(auditRec.Body.Bytes(), &auditResp))
	require.Len(t, auditResp.Data, 1)
	require.Equal(t, int64(1), auditResp.Data[0].RunID)
	require.Equal(t, int64(101), auditResp.Data[0].MatchedLocalAccountID)
	require.Equal(t, []int64{10}, auditResp.Data[0].UnboundGroupIDs)
	require.Equal(t, []string{"cheap"}, auditResp.Data[0].UnboundGroupNames)
	require.Equal(t, []int64{11}, auditResp.Data[0].RemainingGroupIDs)
}

func TestModelSquareHandlerAccountRateGuardDryRunDoesNotUnbindGroups(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/custom/keys":
			_, _ = w.Write([]byte(`{
				"code":0,
				"message":"success",
				"data":{"items":[{"id":1,"name":"key-a","group":{"name":"VIP","rate_multiplier":0.8}}]}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	groupProvider := &modelSquareGroupProviderStub{accounts: []service.Account{
		{
			ID:       101,
			Name:     "findcg-key-a",
			GroupIDs: []int64{10},
			Groups:   []*service.Group{{ID: 10, Name: "cheap", RateMultiplier: 0.5}},
		},
	}}
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{UpstreamManagement: config.UpstreamManagementConfig{
			Providers: []config.UpstreamManagementProviderConfig{
				{
					Slug:              "findcg",
					Name:              "FindCG",
					Enabled:           true,
					APIKeysURL:        server.URL + "/custom/keys",
					AccountNamePrefix: "findcg-",
				},
			},
		}},
		GroupProvider: groupProvider,
		HTTPClient:    server.Client(),
	})
	router := gin.New()
	router.POST("/admin/upstream-management/account-rate-guard/run", h.RunAccountRateGuard)
	router.GET("/admin/upstream-management/account-rate-guard/audits", h.ListAccountRateGuardAudits)

	req := httptest.NewRequest(http.MethodPost, "/admin/upstream-management/account-rate-guard/run", strings.NewReader(`{"dry_run":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, groupProvider.accountUpdatesSnapshot())
	require.Contains(t, rec.Body.String(), `"dry_run":true`)
	require.Contains(t, rec.Body.String(), `"violation_count":1`)
	require.Contains(t, rec.Body.String(), `"unbound_count":0`)

	auditReq := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/account-rate-guard/audits", nil)
	auditRec := httptest.NewRecorder()
	router.ServeHTTP(auditRec, auditReq)
	require.Equal(t, http.StatusOK, auditRec.Code)
	require.Contains(t, auditRec.Body.String(), `"data":[]`)
}

func TestModelSquareHandlerAccountRateGuardContinuesWhenOneProviderFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bad/keys":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"message":"bad gateway"}`))
		case "/good/keys":
			_, _ = w.Write([]byte(`{
				"code":0,
				"message":"success",
				"data":{"items":[{"id":1,"name":"key-a","group":{"name":"VIP","rate_multiplier":0.8}}]}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	groupProvider := &modelSquareGroupProviderStub{accounts: []service.Account{
		{
			ID:       201,
			Name:     "good-key-a",
			GroupIDs: []int64{20},
			Groups:   []*service.Group{{ID: 20, Name: "cheap", RateMultiplier: 0.5}},
		},
	}}
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{UpstreamManagement: config.UpstreamManagementConfig{
			Providers: []config.UpstreamManagementProviderConfig{
				{
					Slug:              "bad",
					Name:              "Bad",
					Enabled:           true,
					APIKeysURL:        server.URL + "/bad/keys",
					AccountNamePrefix: "bad-",
				},
				{
					Slug:              "good",
					Name:              "Good",
					Enabled:           true,
					APIKeysURL:        server.URL + "/good/keys",
					AccountNamePrefix: "good-",
				},
			},
		}},
		GroupProvider: groupProvider,
		HTTPClient:    server.Client(),
	})

	result, err := h.runAccountRateGuard(context.Background(), false)

	require.NoError(t, err)
	require.Len(t, result.Providers, 2)
	require.Equal(t, "bad", result.Providers[0].Slug)
	require.Contains(t, result.Providers[0].Error, "HTTP 502")
	require.Equal(t, "good", result.Providers[1].Slug)
	require.Empty(t, result.Providers[1].Error)
	require.Equal(t, 1, result.UnboundCount)
	accountUpdates := groupProvider.accountUpdatesSnapshot()
	require.Len(t, accountUpdates, 1)
	require.Equal(t, int64(201), accountUpdates[0].id)
	require.NotNil(t, accountUpdates[0].groupIDs)
	require.Empty(t, *accountUpdates[0].groupIDs)
}

func TestModelSquareHandlerAccountRateGuardSupportsNewAPIProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var loginBody string
	var tokenUserHeader string
	var tokenCookieHeader string
	var groupsUserHeader string
	var groupsCookieHeader string
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/api/user/login":
			require.Equal(t, http.MethodPost, r.Method)
			buf := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(buf)
			loginBody = string(buf)
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "newapi-session"})
			_, _ = w.Write([]byte(`{
				"success":true,
				"message":"",
				"data":{"id":18,"username":"zhongyj","display_name":"zhongyj","group":"default","role":1,"status":1}
			}`))
		case "/api/token/":
			tokenUserHeader = r.Header.Get("New-Api-User")
			tokenCookieHeader = r.Header.Get("Cookie")
			require.Equal(t, "1", r.URL.Query().Get("p"))
			require.Equal(t, "200", r.URL.Query().Get("size"))
			_, _ = w.Write([]byte(`{
				"success":true,
				"message":"",
				"data":{
					"page":1,
					"page_size":200,
					"total":1,
					"items":[
						{"id":16,"name":"codex【福利】","group":"CodeX【福利】","status":1}
					]
				}
			}`))
		case "/api/user/self/groups":
			groupsUserHeader = r.Header.Get("New-Api-User")
			groupsCookieHeader = r.Header.Get("Cookie")
			_, _ = w.Write([]byte(`{
				"success":true,
				"message":"",
				"data":{
					"CodeX【福利】":{"desc":"Codex 福利渠道","ratio":0.001},
					"default":{"desc":"用户分组","ratio":1}
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	groupProvider := &modelSquareGroupProviderStub{accounts: []service.Account{
		{
			ID:       301,
			Name:     "newapi-codex【福利】",
			GroupIDs: []int64{30, 31},
			Groups: []*service.Group{
				{ID: 30, Name: "below-newapi-ratio", RateMultiplier: 0.0005},
				{ID: 31, Name: "above-newapi-ratio", RateMultiplier: 0.01},
			},
		},
	}}
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{UpstreamManagement: config.UpstreamManagementConfig{
			Providers: []config.UpstreamManagementProviderConfig{
				{
					Type:              "newapi",
					Slug:              "newapi",
					Name:              "NewAPI",
					Enabled:           true,
					BaseURL:           server.URL,
					LoginURL:          server.URL + "/api/user/login",
					APIKeysURL:        server.URL + "/api/token/?p=1&size=200",
					GroupsURL:         server.URL + "/api/user/self/groups",
					Username:          "zhongyj",
					Password:          "zhong960216",
					AccountNamePrefix: "newapi-",
				},
			},
		}},
		GroupProvider: groupProvider,
		HTTPClient:    server.Client(),
	})

	result, err := h.runAccountRateGuard(context.Background(), false)

	require.NoError(t, err)
	require.JSONEq(t, `{"username":"zhongyj","password":"zhong960216"}`, loginBody)
	require.Equal(t, []string{"/api/user/login", "/api/token/", "/api/user/self/groups"}, paths)
	require.Equal(t, "18", tokenUserHeader)
	require.Contains(t, tokenCookieHeader, "session=newapi-session")
	require.Equal(t, "18", groupsUserHeader)
	require.Contains(t, groupsCookieHeader, "session=newapi-session")
	require.Equal(t, 1, result.CheckedKeyCount)
	require.Equal(t, 1, result.MatchedAccountCount)
	require.Equal(t, 1, result.ViolationCount)
	require.Equal(t, 1, result.UnboundCount)
	require.Len(t, result.Violations, 1)
	require.Equal(t, "codex【福利】", result.Violations[0].UpstreamKeyName)
	require.Equal(t, "CodeX【福利】", result.Violations[0].UpstreamGroupName)
	require.Equal(t, 0.001, result.Violations[0].UpstreamRateMultiplier)
	require.Equal(t, []int64{30}, result.Violations[0].UnboundGroupIDs)
	accountUpdates := groupProvider.accountUpdatesSnapshot()
	require.Len(t, accountUpdates, 1)
	require.Equal(t, int64(301), accountUpdates[0].id)
	require.NotNil(t, accountUpdates[0].groupIDs)
	require.Equal(t, []int64{31}, *accountUpdates[0].groupIDs)
}

func TestModelSquareHandlerTestUpstreamProviderReportsNewAPIStages(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var loginBody string
	var tokenUserHeader string
	var tokenCookieHeader string
	var groupsUserHeader string
	var groupsCookieHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/login":
			require.Equal(t, http.MethodPost, r.Method)
			buf := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(buf)
			loginBody = string(buf)
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "newapi-session"})
			_, _ = w.Write([]byte(`{"success":true,"message":"","data":{"id":18,"username":"zhongyj"}}`))
		case "/api/token/":
			tokenUserHeader = r.Header.Get("New-Api-User")
			tokenCookieHeader = r.Header.Get("Cookie")
			_, _ = w.Write([]byte(`{
				"success":true,
				"message":"",
				"data":{"items":[{"id":16,"name":"codex-key","group":"CodeX福利","status":1}]}
			}`))
		case "/api/user/self/groups":
			groupsUserHeader = r.Header.Get("New-Api-User")
			groupsCookieHeader = r.Header.Get("Cookie")
			_, _ = w.Write([]byte(`{
				"success":true,
				"message":"",
				"data":{"CodeX福利":{"desc":"Codex 福利渠道","ratio":0.001},"default":{"desc":"用户分组","ratio":1}}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	h := newModelSquareHandler(ModelSquareHandlerConfig{HTTPClient: server.Client()})
	router := gin.New()
	router.POST("/test", h.TestUpstreamProvider)

	body := strings.NewReader(fmt.Sprintf(`{
		"type":"newapi",
		"slug":"newapi-main",
		"name":"NewAPI Main",
		"base_url":"%s",
		"login_url":"/api/user/login",
		"api_keys_url":"/api/token/?p=1&size=200",
		"groups_url":"/api/user/self/groups",
		"username":"zhongyj",
		"password":"secret",
		"account_name_prefix":"newapi-"
	}`, server.URL))
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"username":"zhongyj","password":"secret"}`, loginBody)
	require.Equal(t, "18", tokenUserHeader)
	require.Contains(t, tokenCookieHeader, "session=newapi-session")
	require.Equal(t, "18", groupsUserHeader)
	require.Contains(t, groupsCookieHeader, "session=newapi-session")

	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Type  string `json:"type"`
			Slug  string `json:"slug"`
			Login struct {
				OK            bool  `json:"ok"`
				StatusCode    int   `json:"status_code"`
				UserID        int64 `json:"user_id"`
				CookiePresent bool  `json:"cookie_present"`
			} `json:"login"`
			Keys struct {
				OK        bool `json:"ok"`
				ItemCount int  `json:"item_count"`
			} `json:"keys"`
			Groups struct {
				OK         bool `json:"ok"`
				GroupCount int  `json:"group_count"`
			} `json:"groups"`
			ParsedKeys []struct {
				Name           string  `json:"name"`
				GroupName      string  `json:"group_name"`
				RateMultiplier float64 `json:"rate_multiplier"`
			} `json:"parsed_keys"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	require.Equal(t, "newapi", envelope.Data.Type)
	require.Equal(t, "newapi-main", envelope.Data.Slug)
	require.True(t, envelope.Data.Login.OK)
	require.Equal(t, http.StatusOK, envelope.Data.Login.StatusCode)
	require.Equal(t, int64(18), envelope.Data.Login.UserID)
	require.True(t, envelope.Data.Login.CookiePresent)
	require.True(t, envelope.Data.Keys.OK)
	require.Equal(t, 1, envelope.Data.Keys.ItemCount)
	require.True(t, envelope.Data.Groups.OK)
	require.Equal(t, 2, envelope.Data.Groups.GroupCount)
	require.Len(t, envelope.Data.ParsedKeys, 1)
	require.Equal(t, "codex-key", envelope.Data.ParsedKeys[0].Name)
	require.Equal(t, "CodeX福利", envelope.Data.ParsedKeys[0].GroupName)
	require.Equal(t, 0.001, envelope.Data.ParsedKeys[0].RateMultiplier)
	require.NotContains(t, rec.Body.String(), "secret")
	require.NotContains(t, rec.Body.String(), "newapi-session")
}

func TestModelSquareHandlerAccountRateGuardLogsProviderFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := &slogRecordHandler{}
	original := slog.Default()
	slog.SetDefault(slog.New(recorder))
	defer slog.SetDefault(original)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "newapi-session"})
			_, _ = w.Write([]byte(`{"success":true,"message":"","data":{"id":18}}`))
		case "/api/token/":
			_, _ = w.Write([]byte(`{
				"success":true,
				"message":"",
				"data":{"items":[{"id":16,"name":"codex【福利】","group":"CodeX【福利】","status":1}]}
			}`))
		case "/api/user/self/groups":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"success":false,"message":"bad gateway"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{UpstreamManagement: config.UpstreamManagementConfig{
			Providers: []config.UpstreamManagementProviderConfig{
				{
					Type:              "newapi",
					Slug:              "newapi-main",
					Name:              "NewAPI Main",
					Enabled:           true,
					BaseURL:           server.URL,
					Username:          "zhongyj",
					Password:          "secret",
					AccountNamePrefix: "newapi-",
				},
			},
		}},
		GroupProvider: &modelSquareGroupProviderStub{accounts: []service.Account{
			{ID: 301, Name: "newapi-codex【福利】", GroupIDs: []int64{30}, Groups: []*service.Group{{ID: 30, Name: "cheap", RateMultiplier: 0.0005}}},
		}},
		HTTPClient: server.Client(),
	})

	result, err := h.runAccountRateGuard(context.Background(), false)

	require.NoError(t, err)
	require.Len(t, result.Providers, 1)
	require.Contains(t, result.Providers[0].Error, "newapi provider groups failed: HTTP 502")
	require.Len(t, recorder.records, 1)
	require.Contains(t, recorder.records[0], "upstream account rate guard provider failed")
	require.Contains(t, recorder.records[0], "provider_slug=newapi-main")
	require.Contains(t, recorder.records[0], "provider_type=newapi")
	require.Contains(t, recorder.records[0], "stage=groups")
	require.Contains(t, recorder.records[0], "newapi provider groups failed")
}

func TestModelSquareHandlerManualSyncMatchesKeysByGroupName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"token-group-name","expires_in":86400,"token_type":"Bearer"}}`))
		case "/api/v1/keys":
			_, _ = w.Write([]byte(`{
				"code":0,
				"message":"success",
				"data":{
					"items":[
						{"id":1,"name":"Production Key A","group":{"name":"Codex Special","rate_multiplier":0.8}}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	groupProvider := &modelSquareGroupProviderStub{groups: []service.Group{
		{ID: 10, Name: "codex special", RateMultiplier: 0.5},
	}}
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{ModelSquare: config.ModelSquareConfig{
			BaseURL:  server.URL,
			Email:    "configured@example.com",
			Password: "configured-secret",
		}},
		GroupProvider: groupProvider,
		HTTPClient:    server.Client(),
	})
	router := gin.New()
	router.POST("/admin/model-square/sync", h.SyncKeys)

	req := httptest.NewRequest(http.MethodPost, "/admin/model-square/sync", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []modelSquareGroupRateUpdate{{id: 10, rate: 0.8}}, groupProvider.updatesSnapshot())
	require.JSONEq(t, `{
		"code":0,
		"message":"success",
		"data":{
			"checked_count":1,
			"matched_count":1,
			"updated_count":1,
			"rate_warnings":[
				{"group_id":10,"group_name":"codex special","local_rate_multiplier":0.5,"upstream_rate_multiplier":0.8}
			]
		}
	}`, rec.Body.String())
}

func TestModelSquareHandlerManualSyncReportsUpstreamRatesNotLowerThanLocal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"token-warning","expires_in":86400,"token_type":"Bearer"}}`))
		case "/api/v1/keys":
			_, _ = w.Write([]byte(`{
				"code":0,
				"message":"success",
				"data":{
					"items":[
						{"id":1,"name":"Codex Special","group":{"name":"Codex Special","rate_multiplier":0.8}},
						{"id":2,"name":"Stable Group","group":{"name":"Stable Group","rate_multiplier":0.4}},
						{"id":3,"name":"Discount Group","group":{"name":"Discount Group","rate_multiplier":0.2}}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	groupProvider := &modelSquareGroupProviderStub{groups: []service.Group{
		{ID: 10, Name: "codex special", RateMultiplier: 0.5},
		{ID: 20, Name: "Stable Group", RateMultiplier: 0.4},
		{ID: 30, Name: "Discount Group", RateMultiplier: 0.3},
	}}
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{ModelSquare: config.ModelSquareConfig{
			BaseURL:  server.URL,
			Email:    "configured@example.com",
			Password: "configured-secret",
		}},
		GroupProvider: groupProvider,
		HTTPClient:    server.Client(),
	})
	router := gin.New()
	router.POST("/admin/model-square/sync", h.SyncKeys)

	req := httptest.NewRequest(http.MethodPost, "/admin/model-square/sync", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []modelSquareGroupRateUpdate{{id: 10, rate: 0.8}}, groupProvider.updatesSnapshot())
	require.JSONEq(t, `{
		"code":0,
		"message":"success",
		"data":{
			"checked_count":3,
			"matched_count":3,
			"updated_count":1,
			"rate_warnings":[
				{"group_id":10,"group_name":"codex special","local_rate_multiplier":0.5,"upstream_rate_multiplier":0.8},
				{"group_id":20,"group_name":"Stable Group","local_rate_multiplier":0.4,"upstream_rate_multiplier":0.4}
			]
		}
	}`, rec.Body.String())
}

func TestModelSquareHandlerRateWarningsDoesNotUpdateLocalGroups(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"token-readonly","expires_in":86400,"token_type":"Bearer"}}`))
		case "/api/v1/keys":
			_, _ = w.Write([]byte(`{
				"code":0,
				"message":"success",
				"data":{
					"items":[
						{"id":1,"name":"Codex Special","group":{"name":"Codex Special","rate_multiplier":0.8}},
						{"id":2,"name":"Discount Group","group":{"name":"Discount Group","rate_multiplier":0.2}}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	groupProvider := &modelSquareGroupProviderStub{groups: []service.Group{
		{ID: 10, Name: "codex special", RateMultiplier: 0.5},
		{ID: 20, Name: "Discount Group", RateMultiplier: 0.3},
	}}
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{ModelSquare: config.ModelSquareConfig{
			BaseURL:  server.URL,
			Email:    "configured@example.com",
			Password: "configured-secret",
		}},
		GroupProvider: groupProvider,
		HTTPClient:    server.Client(),
	})
	router := gin.New()
	router.GET("/admin/model-square/rate-warnings", h.RateWarnings)

	req := httptest.NewRequest(http.MethodGet, "/admin/model-square/rate-warnings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, groupProvider.updatesSnapshot())
	require.JSONEq(t, `{
		"code":0,
		"message":"success",
		"data":{
			"checked_count":2,
			"matched_count":2,
			"updated_count":0,
			"rate_warnings":[
				{"group_id":10,"group_name":"codex special","local_rate_multiplier":0.5,"upstream_rate_multiplier":0.8}
			]
		}
	}`, rec.Body.String())
}

func TestModelSquareHandlerBackgroundSyncRunsOnInterval(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"token-bg","expires_in":86400,"token_type":"Bearer"}}`))
		case "/api/v1/keys":
			require.Equal(t, "Bearer token-bg", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{
				"code":0,
				"message":"success",
				"data":{
					"items":[{"id":1,"name":"Codex Special","group":{"rate_multiplier":0.3}}],
					"total":1,
					"page":1,
					"page_size":100,
					"pages":1
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	groupProvider := &modelSquareGroupProviderStub{groups: []service.Group{
		{ID: 10, Name: "codexspecial", RateMultiplier: 0.2},
	}}
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{ModelSquare: config.ModelSquareConfig{
			BaseURL:                 server.URL,
			Email:                   "configured@example.com",
			Password:                "configured-secret",
			KeysSyncIntervalSeconds: 1,
		}},
		SyncInterval:  10 * time.Millisecond,
		GroupProvider: groupProvider,
		HTTPClient:    server.Client(),
	})
	h.StartBackgroundSync()
	defer h.StopBackgroundSync()

	require.Eventually(t, func() bool {
		return len(groupProvider.updatesSnapshot()) == 1
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, []modelSquareGroupRateUpdate{{id: 10, rate: 0.3}}, groupProvider.updatesSnapshot())
}

func TestModelSquareHandlerBackgroundAccountRateGuardRunsOnInterval(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/custom/keys":
			_, _ = w.Write([]byte(`{
				"code":0,
				"message":"success",
				"data":{"items":[{"id":1,"name":"key-a","group":{"name":"VIP","rate_multiplier":0.8}}]}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	groupProvider := &modelSquareGroupProviderStub{accounts: []service.Account{
		{
			ID:       101,
			Name:     "findcg-key-a",
			GroupIDs: []int64{10},
			Groups:   []*service.Group{{ID: 10, Name: "cheap", RateMultiplier: 0.5}},
		},
	}}
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{UpstreamManagement: config.UpstreamManagementConfig{
			Providers: []config.UpstreamManagementProviderConfig{
				{
					Slug:              "findcg",
					Name:              "FindCG",
					Enabled:           true,
					APIKeysURL:        server.URL + "/custom/keys",
					AccountNamePrefix: "findcg-",
				},
			},
		}},
		GroupProvider:            groupProvider,
		HTTPClient:               server.Client(),
		AccountRateGuardInterval: 10 * time.Millisecond,
	})
	h.StartAccountRateGuard()
	defer h.StopAccountRateGuard()

	require.Eventually(t, func() bool {
		return len(groupProvider.accountUpdatesSnapshot()) > 0
	}, time.Second, 10*time.Millisecond)
	accountUpdates := groupProvider.accountUpdatesSnapshot()
	require.Equal(t, int64(101), accountUpdates[0].id)
	require.NotNil(t, accountUpdates[0].groupIDs)
	require.Empty(t, *accountUpdates[0].groupIDs)
}

func TestModelSquareHandlerUsesSettingsSyncInterval(t *testing.T) {
	settingSvc := service.NewSettingService(&modelSquareSettingRepo{values: map[string]string{
		service.SettingKeyModelSquareKeysSyncIntervalSeconds: "12",
	}}, &config.Config{ModelSquare: config.ModelSquareConfig{KeysSyncIntervalSeconds: 30}})
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig:      &config.Config{ModelSquare: config.ModelSquareConfig{KeysSyncIntervalSeconds: 30}},
		SettingService: settingSvc,
		SyncInterval:   time.Second,
	})

	require.Equal(t, 12*time.Second, h.currentSyncInterval(context.Background()))
}

func TestModelSquareHandlerUsesSettingsAccountRateGuardInterval(t *testing.T) {
	settingSvc := service.NewSettingService(&modelSquareSettingRepo{values: map[string]string{
		service.SettingKeyUpstreamManagementAccountRateGuardIntervalSeconds: "12",
	}}, &config.Config{UpstreamManagement: config.UpstreamManagementConfig{AccountRateGuardIntervalSeconds: 30}})
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{UpstreamManagement: config.UpstreamManagementConfig{
			AccountRateGuardIntervalSeconds: 30,
		}},
		SettingService:           settingSvc,
		AccountRateGuardInterval: time.Second,
	})

	require.Equal(t, 12*time.Second, h.currentAccountRateGuardInterval(context.Background()))
}

func TestModelSquareHandlerManualSyncIgnoresFloatTailDifference(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"token-float","expires_in":86400,"token_type":"Bearer"}}`))
		case "/api/v1/keys":
			_, _ = w.Write([]byte(`{
				"code":0,
				"message":"success",
				"data":{
					"items":[{"id":1,"name":"Float Group","group":{"rate_multiplier":0.30000000000000004}}],
					"total":1,
					"page":1,
					"page_size":100,
					"pages":1
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	groupProvider := &modelSquareGroupProviderStub{groups: []service.Group{
		{ID: 10, Name: "floatgroup", RateMultiplier: 0.3},
	}}
	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{ModelSquare: config.ModelSquareConfig{
			BaseURL:  server.URL,
			Email:    "configured@example.com",
			Password: "configured-secret",
		}},
		GroupProvider: groupProvider,
		HTTPClient:    server.Client(),
	})
	router := gin.New()
	router.POST("/admin/model-square/sync", h.SyncKeys)

	req := httptest.NewRequest(http.MethodPost, "/admin/model-square/sync", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, groupProvider.updatesSnapshot())
	require.JSONEq(t, `{
		"code":0,
		"message":"success",
		"data":{
			"checked_count":1,
			"matched_count":1,
			"updated_count":0,
			"rate_warnings":[
				{"group_id":10,"group_name":"floatgroup","local_rate_multiplier":0.3,"upstream_rate_multiplier":0.30000000000000004}
			]
		}
	}`, rec.Body.String())
}

type modelSquareGroupRateUpdate struct {
	id   int64
	rate float64
}

type modelSquareAccountUpdateCall struct {
	id       int64
	groupIDs *[]int64
}

type modelSquareGroupProviderStub struct {
	mu             sync.Mutex
	groups         []service.Group
	accounts       []service.Account
	err            error
	updates        []modelSquareGroupRateUpdate
	accountUpdates []modelSquareAccountUpdateCall
}

func (s *modelSquareGroupProviderStub) GetAllGroups(context.Context) ([]service.Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	groups := append([]service.Group(nil), s.groups...)
	return groups, s.err
}

func (s *modelSquareGroupProviderStub) UpdateGroup(_ context.Context, id int64, input *service.UpdateGroupInput) (*service.Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if input == nil || input.RateMultiplier == nil {
		return nil, nil
	}
	s.updates = append(s.updates, modelSquareGroupRateUpdate{id: id, rate: *input.RateMultiplier})
	for i := range s.groups {
		if s.groups[i].ID == id {
			s.groups[i].RateMultiplier = *input.RateMultiplier
			return &s.groups[i], nil
		}
	}
	return nil, nil
}

func (s *modelSquareGroupProviderStub) updatesSnapshot() []modelSquareGroupRateUpdate {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]modelSquareGroupRateUpdate(nil), s.updates...)
}

func (s *modelSquareGroupProviderStub) ListAccounts(context.Context, int, int, string, string, string, string, int64, string, string, string) ([]service.Account, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts := append([]service.Account(nil), s.accounts...)
	return accounts, int64(len(accounts)), s.err
}

func (s *modelSquareGroupProviderStub) UpdateAccount(_ context.Context, id int64, input *service.UpdateAccountInput) (*service.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var groupIDsCopy *[]int64
	if input != nil && input.GroupIDs != nil {
		copied := append([]int64(nil), (*input.GroupIDs)...)
		groupIDsCopy = &copied
	}
	s.accountUpdates = append(s.accountUpdates, modelSquareAccountUpdateCall{id: id, groupIDs: groupIDsCopy})
	for i := range s.accounts {
		if s.accounts[i].ID == id {
			if groupIDsCopy != nil {
				s.accounts[i].GroupIDs = append([]int64(nil), (*groupIDsCopy)...)
			}
			return &s.accounts[i], s.err
		}
	}
	return &service.Account{ID: id}, s.err
}

func (s *modelSquareGroupProviderStub) accountUpdatesSnapshot() []modelSquareAccountUpdateCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]modelSquareAccountUpdateCall, 0, len(s.accountUpdates))
	for _, call := range s.accountUpdates {
		copied := modelSquareAccountUpdateCall{id: call.id}
		if call.groupIDs != nil {
			groupIDs := append([]int64(nil), (*call.groupIDs)...)
			copied.groupIDs = &groupIDs
		}
		out = append(out, copied)
	}
	return out
}

type modelSquareSettingRepo struct {
	values map[string]string
}

func (r *modelSquareSettingRepo) Get(_ context.Context, key string) (*service.Setting, error) {
	value, ok := r.values[key]
	if !ok {
		return nil, service.ErrSettingNotFound
	}
	return &service.Setting{Key: key, Value: value}, nil
}

func (r *modelSquareSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	value, ok := r.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (r *modelSquareSettingRepo) Set(_ context.Context, key, value string) error {
	r.values[key] = value
	return nil
}

func (r *modelSquareSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (r *modelSquareSettingRepo) SetMultiple(_ context.Context, settings map[string]string) error {
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *modelSquareSettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	result := make(map[string]string, len(r.values))
	for key, value := range r.values {
		result[key] = value
	}
	return result, nil
}

func (r *modelSquareSettingRepo) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func TestModelSquareHandlerReloginsAfterUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	loginCount := 0
	modelSquareCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			loginCount++
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"token-` + string(rune('0'+loginCount)) + `","expires_in":86400,"token_type":"Bearer"}}`))
		case "/api/v1/model-square":
			modelSquareCount++
			if modelSquareCount == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"message":"expired"}`))
				return
			}
			require.Equal(t, "Bearer token-2", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{"groups":[],"models":[],"updated_at":"2026-05-30T22:02:54+08:00"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	h := newModelSquareHandler(ModelSquareHandlerConfig{
		AppConfig: &config.Config{ModelSquare: config.ModelSquareConfig{
			BaseURL:  server.URL,
			Email:    "admin@example.com",
			Password: "secret",
		}},
		HTTPClient: server.Client(),
	})
	router := gin.New()
	router.GET("/admin/model-square", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/admin/model-square", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 2, loginCount)
	require.Equal(t, 2, modelSquareCount)
}

func TestModelSquareHandlerRequiresCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("FINDCG_EMAIL", "")
	t.Setenv("FINDCG_PASSWORD", "")
	t.Setenv("MODEL_SQUARE_EMAIL", "")
	t.Setenv("MODEL_SQUARE_PASSWORD", "")

	h := newModelSquareHandler(ModelSquareHandlerConfig{AppConfig: &config.Config{}})
	router := gin.New()
	router.GET("/admin/model-square", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/admin/model-square", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), "upstream_management.email")
}

package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

type modelSquareGroupProviderStub struct {
	mu      sync.Mutex
	groups  []service.Group
	err     error
	updates []modelSquareGroupRateUpdate
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

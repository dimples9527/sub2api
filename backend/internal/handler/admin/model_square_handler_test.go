package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

	baseURL, email, password, source := h.resolveConfigLocked(context.Background())

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

	baseURL, email, password, source := h.resolveConfigLocked(context.Background())

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
	require.Len(t, body.Groups, 2)
	require.Equal(t, "remote-1", body.Groups[0]["id"])
	require.Equal(t, "codex特价", body.Groups[0]["name"])
	require.Equal(t, "local group", body.Groups[0]["description"])
	require.Equal(t, service.PlatformOpenAI, body.Groups[0]["platform"])
	require.Equal(t, 0.25, body.Groups[0]["rate_multiplier"])
	require.Equal(t, service.StatusActive, body.Groups[0]["status"])
	require.Equal(t, float64(2002), body.Groups[1]["id"])
	require.Equal(t, "unmatched", body.Groups[1]["name"])
	require.Equal(t, []any{"remote-1", float64(2002)}, body.Models[0].GroupIDs)
}

type modelSquareGroupProviderStub struct {
	groups []service.Group
	err    error
}

func (s *modelSquareGroupProviderStub) GetAllGroups(context.Context) ([]service.Group, error) {
	return s.groups, s.err
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
	require.Contains(t, rec.Body.String(), "model_square.email")
}

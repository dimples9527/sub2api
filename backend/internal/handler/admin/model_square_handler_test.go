package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

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

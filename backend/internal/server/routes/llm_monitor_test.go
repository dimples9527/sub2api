package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type llmMonitorSettingsStub struct {
	statusAPIURL string
}

func (s llmMonitorSettingsStub) GetPublicSettings(context.Context) (*service.PublicSettings, error) {
	return &service.PublicSettings{LLMMonitorStatusAPIURL: s.statusAPIURL}, nil
}

func TestLLMMonitorStatusProxyUsesConfiguredUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var upstreamQuery string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"groups":[{"provider":"demo"}]}`))
	}))
	defer upstream.Close()

	router := gin.New()
	RegisterLLMMonitorRoutes(router, llmMonitorSettingsStub{statusAPIURL: upstream.URL + "/api/status"})

	req := httptest.NewRequest(http.MethodGet, "/api/llm-monitor/status?period=24h&board=hot", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q", got)
	}
	if upstreamQuery != "board=hot&period=24h" && upstreamQuery != "period=24h&board=hot" {
		t.Fatalf("upstream query = %q", upstreamQuery)
	}
	if rec.Body.String() != `{"groups":[{"provider":"demo"}]}` {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestLLMMonitorStatusProxyRejectsInvalidConfiguredURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	RegisterLLMMonitorRoutes(router, llmMonitorSettingsStub{statusAPIURL: "://bad-url"})

	req := httptest.NewRequest(http.MethodGet, "/api/llm-monitor/status", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status code = %d, body = %s", rec.Code, rec.Body.String())
	}
}

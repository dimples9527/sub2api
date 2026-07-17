package routes

import (
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminLLMMonitorStatusProxySupportsRequiredGzipResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotAcceptable)
			_, _ = w.Write([]byte(`{"error":{"message":"gzip required"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		writer := gzip.NewWriter(w)
		_, _ = writer.Write([]byte(`{"groups":[{"provider":"gzip provider"}]}`))
		require.NoError(t, writer.Close())
	}))
	defer upstream.Close()

	router := gin.New()
	admin := router.Group("/admin")
	RegisterAdminLLMMonitorRoutes(admin, llmMonitorSettingsStub{statusAPIURL: upstream.URL + "/api/status"})

	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/monitor-status", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "gzip provider")
}

type llmMonitorSettingsStub struct {
	statusAPIURL string
}

func (s llmMonitorSettingsStub) GetLLMMonitorSettings(context.Context) (*service.LLMMonitorSettings, error) {
	return &service.LLMMonitorSettings{StatusAPIURL: s.statusAPIURL}, nil
}

type llmMonitorGroupStub struct {
	groups []service.Group
}

func (s llmMonitorGroupStub) GetAllGroups(context.Context) ([]service.Group, error) {
	return s.groups, nil
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
	RegisterLLMMonitorRoutes(router, llmMonitorSettingsStub{statusAPIURL: upstream.URL + "/api/status"}, llmMonitorGroupStub{
		groups: []service.Group{{Name: "demo"}},
	})

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

func TestLLMMonitorStatusProxyFiltersGroupsAndScrubsFindCGURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"groups": [
				{"provider":" Demo Provider ","provider_slug":"findcg-ai","provider_url":"https://www.findcg.com","probe_url":"https://www.findcg.com","layers":[]},
				{"provider":"hidden provider","provider_slug":"findcg-ai","provider_url":"https://www.findcg.com","layers":[]}
			],
			"meta": {
				"all_monitor_ids": ["Demo Provider-cc-vip1", "hidden provider-cc-vip1"],
				"source_url": "https://www.findcg.com",
				"source_slug": "findcg-ai"
			}
		}`))
	}))
	defer upstream.Close()

	router := gin.New()
	RegisterLLMMonitorRoutes(router, llmMonitorSettingsStub{statusAPIURL: upstream.URL + "/api/status"}, llmMonitorGroupStub{
		groups: []service.Group{{Name: "demoprovider"}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/llm-monitor/status", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "hidden provider") {
		t.Fatalf("unexpected hidden provider in body = %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "https://www.findcg.com") {
		t.Fatalf("unexpected findcg url in body = %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "findcg-ai") {
		t.Fatalf("unexpected findcg slug in body = %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"provider":" Demo Provider "`) {
		t.Fatalf("expected matched provider in body = %s", rec.Body.String())
	}
}

func TestAdminLLMMonitorStatusProxyDoesNotFilterGroups(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var upstreamQuery string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"groups": [
				{"provider":"matched provider"},
				{"provider":"upstream only provider"}
			]
		}`))
	}))
	defer upstream.Close()

	router := gin.New()
	admin := router.Group("/admin")
	RegisterAdminLLMMonitorRoutes(admin, llmMonitorSettingsStub{statusAPIURL: upstream.URL + "/api/status?period=30m&board=cold"})

	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/monitor-status?period=90m&board=hot", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, body = %s", rec.Code, rec.Body.String())
	}
	if upstreamQuery != "board=hot&period=90m" && upstreamQuery != "period=90m&board=hot" {
		t.Fatalf("upstream query = %q", upstreamQuery)
	}
	if !strings.Contains(rec.Body.String(), "matched provider") {
		t.Fatalf("expected matched provider in body = %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "upstream only provider") {
		t.Fatalf("expected upstream-only provider in body = %s", rec.Body.String())
	}
}

func TestLLMMonitorStatusProxyRejectsInvalidConfiguredURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	RegisterLLMMonitorRoutes(router, llmMonitorSettingsStub{statusAPIURL: "://bad-url"}, llmMonitorGroupStub{})

	req := httptest.NewRequest(http.MethodGet, "/api/llm-monitor/status", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status code = %d, body = %s", rec.Code, rec.Body.String())
	}
}

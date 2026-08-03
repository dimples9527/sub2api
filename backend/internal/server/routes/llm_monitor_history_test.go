package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type llmMonitorHistoryStoreStub struct {
	snapshots []service.LLMMonitorHistorySnapshot
}

func (s *llmMonitorHistoryStoreStub) SaveSnapshot(_ context.Context, snapshot service.LLMMonitorHistorySnapshot) error {
	s.snapshots = append(s.snapshots, snapshot)
	return nil
}

func (s *llmMonitorHistoryStoreStub) LoadLatestSnapshot(_ context.Context, sourceKey, period, board string) (*service.LLMMonitorHistorySnapshot, error) {
	for i := len(s.snapshots) - 1; i >= 0; i-- {
		snapshot := s.snapshots[i]
		if snapshot.SourceKey == sourceKey && snapshot.Period == period && snapshot.Board == board {
			return &snapshot, nil
		}
	}
	return nil, nil
}

func TestLLMMonitorStatusProxySavesValidSnapshotAndRestoresEmptyTimeline(t *testing.T) {
	gin.SetMode(gin.TestMode)

	requestCount := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			_, _ = w.Write([]byte(`{"groups":[{"provider":"demo","layers":[{"timeline":[{"availability":0,"latency":900,"timestamp":1722513000}]}]}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"groups":[{"provider":"demo","layers":[{"timeline":[]}]}]}`))
	}))
	defer upstream.Close()

	history := &llmMonitorHistoryStoreStub{}
	router := gin.New()
	RegisterLLMMonitorRoutes(router, llmMonitorSettingsStub{statusAPIURL: upstream.URL + "/api/status"}, llmMonitorGroupStub{
		groups: []service.Group{{Name: "demo"}},
	}, history)

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/llm-monitor/status?period=24h&board=hot", nil))
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())
	require.Len(t, history.snapshots, 1)

	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/llm-monitor/status?period=24h&board=hot", nil))
	require.Equal(t, http.StatusOK, second.Code, second.Body.String())
	require.Contains(t, second.Body.String(), `"availability":0`)

	var payload struct {
		Groups []map[string]any `json:"groups"`
	}
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &payload))
	require.Len(t, payload.Groups[0]["layers"].([]any)[0].(map[string]any)["timeline"].([]any), 1)
}

func TestLocalLLMMonitorStatusRestoresRawRemoteSnapshotAfterFilteredRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	requestCount := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			_, _ = w.Write([]byte(`{"groups":[{"provider":"Upstream VIP","layers":[{"timeline":[{"availability":88,"latency":120,"timestamp":1722513000}]}]}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"groups":[{"provider":"Upstream VIP","layers":[{"timeline":[]}]}]}`))
	}))
	defer upstream.Close()

	localGroupID := int64(1)
	history := &llmMonitorHistoryStoreStub{}
	settings := llmMonitorSettingsStub{statusAPIURL: upstream.URL + "/api/status"}
	groups := llmMonitorGroupStub{groups: []service.Group{{ID: localGroupID, Name: "本地 VIP", Status: "active"}}}

	router := gin.New()
	RegisterLLMMonitorRoutes(router, settings, groups, history)
	RegisterLocalLLMMonitorRoutes(router, settings, groups, localLLMMonitorDataStub{
		mappings: []service.SupplierProviderGroup{{
			LocalGroupID:        &localGroupID,
			MatchedUpstreamName: "Upstream VIP",
		}},
	}, history)

	seed := httptest.NewRecorder()
	router.ServeHTTP(seed, httptest.NewRequest(http.MethodGet, "/api/llm-monitor/status?period=24h&board=hot", nil))
	require.Equal(t, http.StatusOK, seed.Code, seed.Body.String())
	require.Len(t, history.snapshots, 1)
	require.Contains(t, string(history.snapshots[0].Payload), "Upstream VIP")

	recovered := httptest.NewRecorder()
	router.ServeHTTP(recovered, httptest.NewRequest(http.MethodGet, "/api/llm-monitor/local-status?period=24h&board=hot", nil))
	require.Equal(t, http.StatusOK, recovered.Code, recovered.Body.String())

	var payload struct {
		Groups []map[string]any `json:"groups"`
	}
	require.NoError(t, json.Unmarshal(recovered.Body.Bytes(), &payload))
	require.Len(t, payload.Groups, 1)
	layer := payload.Groups[0]["layers"].([]any)[0].(map[string]any)
	point := layer["timeline"].([]any)[0].(map[string]any)
	require.Equal(t, 88.0, point["availability"])
}

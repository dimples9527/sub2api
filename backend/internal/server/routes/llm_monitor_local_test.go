package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type localLLMMonitorDataStub struct {
	mappings []service.SupplierProviderGroup
	trends   []service.SupplierProviderGroupHealthTrend
}

func (s localLLMMonitorDataStub) ListMappingsByLocalGroup(context.Context, []int64) ([]service.SupplierProviderGroup, error) {
	return s.mappings, nil
}

func (s localLLMMonitorDataStub) ListLocalGroupHealthTrends(context.Context, service.SupplierProviderGroupHealthTrendParams) ([]service.SupplierProviderGroupHealthTrend, error) {
	return s.trends, nil
}

func TestLocalLLMMonitorStatusMergesActiveLocalGroupsAndHidesSources(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"groups": [
				{"provider":"Upstream VIP","layers":[{"timeline":[{"availability":70,"latency":300,"timestamp":1722513000},{"availability":80,"latency":200,"timestamp":1722513300}]}]},
				{"provider":"Local Upstream","layers":[{"timeline":[{"availability":88,"latency":120,"timestamp":1722513000}]}]}
			]
		}`))
	}))
	defer upstream.Close()

	localID1 := int64(1)
	localID2 := int64(2)
	localID3 := int64(3)
	router := gin.New()
	RegisterLocalLLMMonitorRoutes(
		router,
		llmMonitorSettingsStub{statusAPIURL: upstream.URL},
		llmMonitorGroupStub{groups: []service.Group{
			{ID: localID1, Name: "本地 VIP", Status: "active"},
			{ID: localID2, Name: "健康守护组", Status: "active"},
			{ID: localID3, Name: "Local Upstream", Status: "active"},
			{ID: 4, Name: "Disabled", Status: "disabled"},
		}},
		localLLMMonitorDataStub{
			mappings: []service.SupplierProviderGroup{
				{LocalGroupID: &localID1, Name: "供应商 VIP", UpstreamKey: "vip", MatchedUpstreamName: "Upstream VIP"},
				{LocalGroupID: &localID3, Name: "Local Upstream", UpstreamKey: "local-upstream"},
			},
			trends: []service.SupplierProviderGroupHealthTrend{
				{GroupID: localID1, Trend: []service.SupplierProviderGroupHealthTrendPoint{
					{Time: now.Add(-time.Minute), Availability: 95, Latency: 100, TestedAccountCount: 1, Tone: "green"},
				}},
				{GroupID: localID2, Trend: []service.SupplierProviderGroupHealthTrendPoint{
					{Time: now.Add(-time.Minute), Availability: 80, Latency: 180, TestedAccountCount: 1, Tone: "yellow"},
				}},
			},
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/llm-monitor/local-status?period=24h&board=hot", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var payload struct {
		Groups []map[string]any `json:"groups"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Len(t, payload.Groups, 3)
	require.NotContains(t, rec.Body.String(), `"source"`)
	require.NotContains(t, rec.Body.String(), "supplier_account")

	byProvider := make(map[string]map[string]any, len(payload.Groups))
	for _, group := range payload.Groups {
		byProvider[group["provider"].(string)] = group
	}
	require.Contains(t, byProvider, "本地 VIP")
	require.Contains(t, byProvider, "健康守护组")
	require.Contains(t, byProvider, "Local Upstream")
	require.NotContains(t, byProvider, "Disabled")

	localVIP := byProvider["本地 VIP"]
	localVIPLayer := localVIP["layers"].([]any)[0].(map[string]any)
	require.Equal(t, 95.0, localVIPLayer["timeline"].([]any)[0].(map[string]any)["availability"])

	healthOnly := byProvider["健康守护组"]
	healthLayer := healthOnly["layers"].([]any)[0].(map[string]any)
	require.Equal(t, 80.0, healthLayer["timeline"].([]any)[0].(map[string]any)["availability"])

	upstreamOnly := byProvider["Local Upstream"]
	upstreamLayer := upstreamOnly["layers"].([]any)[0].(map[string]any)
	require.Equal(t, 88.0, upstreamLayer["timeline"].([]any)[0].(map[string]any)["availability"])
}

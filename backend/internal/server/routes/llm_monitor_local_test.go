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
	mappings          []service.SupplierProviderGroup
	trends            []service.SupplierProviderGroupHealthTrend
	healthTrendParams *[]service.SupplierProviderGroupHealthTrendParams
}

func (s localLLMMonitorDataStub) ListMappingsByLocalGroup(context.Context, []int64) ([]service.SupplierProviderGroup, error) {
	return s.mappings, nil
}

func (s localLLMMonitorDataStub) ListLocalGroupHealthTrends(_ context.Context, params service.SupplierProviderGroupHealthTrendParams) ([]service.SupplierProviderGroupHealthTrend, error) {
	if s.healthTrendParams != nil {
		*s.healthTrendParams = append(*s.healthTrendParams, params)
	}
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
	var healthTrendParams []service.SupplierProviderGroupHealthTrendParams
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
			healthTrendParams: &healthTrendParams,
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
	require.Len(t, healthTrendParams, 1)
	require.True(t, healthTrendParams[0].PreferRawMonitorTimeline)
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

func TestParseLocalLLMMonitorPeriodUsesNinetyMinutesForDefaultSupplierTimeline(t *testing.T) {
	period, ok := parseLocalLLMMonitorPeriod("")

	require.True(t, ok)
	require.Equal(t, 90*time.Minute, period.Duration)
	require.Equal(t, 18, period.BucketCount)
	_, ok = parseLocalLLMMonitorPeriod("3h")
	require.False(t, ok)
}
func TestLocalLLMMonitorToneUsesFortyPercentGreenThreshold(t *testing.T) {
	require.Equal(t, "green", localLLMMonitorTone(nil, 40))
	require.Equal(t, "yellow", localLLMMonitorTone(nil, 39.99))
	require.Equal(t, "red", localLLMMonitorTone(nil, 0))
}

func TestLocalLLMMonitorStatusPrefersLatestTone(t *testing.T) {
	greenTail := []service.LocalModelMonitorTrendPoint{{Time: time.Now(), Availability: 100, Tone: "green", Valid: true}}

	// 最新一刻被调度账号快照覆盖后，灯色以那条账号为准，不能再看趋势尾巴（全组聚合）。
	require.Equal(t, 0, localLLMMonitorStatus("red", 0, greenTail))
	require.Equal(t, 2, localLLMMonitorStatus("yellow", 100, greenTail))
	require.Equal(t, 1, localLLMMonitorStatus("green", 100, greenTail))
	// 没有快照的分组留空，沿用原来的判断。
	require.Equal(t, 1, localLLMMonitorStatus("", 100, greenTail))
}

func TestBuildLocalLLMMonitorPayloadCurrentStatusUsesSchedulingAccountMoment(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	groupID := int64(9)
	snapshotTime := now.Add(-30 * time.Second)

	payload := buildLocalLLMMonitorPayload(
		[]service.Group{{ID: groupID, Name: "调度账号组", Status: "active"}},
		nil,
		[]service.SupplierProviderGroupHealthTrend{{
			GroupID: groupID,
			// 最新一刻已被调度账号快照覆盖：这条账号挂了，但趋势线里仍是全组聚合出来的绿点。
			Availability: 0,
			Latency:      4321,
			Time:         snapshotTime,
			LatestTone:   "red",
			Trend: []service.SupplierProviderGroupHealthTrendPoint{
				{Time: now.Add(-time.Minute), Availability: 100, Latency: 100, TestedAccountCount: 2, Tone: "green"},
			},
		}},
		nil,
	)

	require.Len(t, payload, 1)
	layer := payload[0]["layers"].([]any)[0].(map[string]any)
	current := layer["current_status"].(map[string]any)
	// 状态 / 耗时 / 时间必须同源：都出自调度账号那一条记录。
	require.Equal(t, 0, current["status"])
	require.Equal(t, int64(4321), current["latency"])
	require.Equal(t, snapshotTime.UnixMilli(), current["timestamp"])
	// 历史不动：趋势线里仍然是全组聚合算出来的 100。
	timeline := layer["timeline"].([]any)
	require.Len(t, timeline, 1)
	require.Equal(t, 100.0, timeline[0].(map[string]any)["availability"])
}

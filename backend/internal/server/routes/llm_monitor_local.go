package routes

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type llmMonitorLocalDataProvider interface {
	ListMappingsByLocalGroup(ctx context.Context, localGroupIDs []int64) ([]service.SupplierProviderGroup, error)
	ListLocalGroupHealthTrends(ctx context.Context, params service.SupplierProviderGroupHealthTrendParams) ([]service.SupplierProviderGroupHealthTrend, error)
}

type localLLMMonitorPeriod struct {
	Duration    time.Duration
	BucketCount int
	AllHistory  bool
}

type localLLMMonitorParsedTrend struct {
	Trend    service.LocalModelMonitorTrend
	Service  string
	Category string
	Model    string
}

// RegisterLocalLLMMonitorRoutes 注册面向本地分组的模型监控聚合接口。
func RegisterLocalLLMMonitorRoutes(
	r gin.IRouter,
	settingsProvider llmMonitorSettingsProvider,
	groupProvider llmMonitorGroupProvider,
	dataProvider llmMonitorLocalDataProvider,
) {
	r.GET("/api/llm-monitor/local-status", func(c *gin.Context) {
		period, ok := parseLocalLLMMonitorPeriod(c.Query("period"))
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid monitor period"})
			return
		}
		if groupProvider == nil || dataProvider == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "local monitor dependencies unavailable"})
			return
		}

		ctx := c.Request.Context()
		groups, err := groupProvider.GetAllGroups(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load local groups"})
			return
		}
		activeGroups := activeLocalMonitorGroups(groups)
		localGroupIDs := make([]int64, 0, len(activeGroups))
		for _, group := range activeGroups {
			if group.ID > 0 {
				localGroupIDs = append(localGroupIDs, group.ID)
			}
		}

		mappings, err := dataProvider.ListMappingsByLocalGroup(ctx, localGroupIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load local group mappings"})
			return
		}
		healthTrends, err := dataProvider.ListLocalGroupHealthTrends(ctx, service.SupplierProviderGroupHealthTrendParams{
			GroupIDs:    localGroupIDs,
			Period:      period.Duration,
			BucketCount: period.BucketCount,
			Now:         time.Now().UTC(),
			AllHistory:  period.AllHistory,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load local health trends"})
			return
		}

		upstreamGroups := loadLocalLLMMonitorUpstream(ctx, settingsProvider, c.Query("period"), c.Query("board"))
		payload := buildLocalLLMMonitorPayload(activeGroups, mappings, healthTrends, upstreamGroups)
		c.JSON(http.StatusOK, gin.H{"groups": payload})
	})
}

func parseLocalLLMMonitorPeriod(raw string) (localLLMMonitorPeriod, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "90m":
		return localLLMMonitorPeriod{Duration: 90 * time.Minute, BucketCount: 18}, true
	case "24h":
		return localLLMMonitorPeriod{Duration: 24 * time.Hour, BucketCount: 18}, true
	case "7d":
		return localLLMMonitorPeriod{Duration: 7 * 24 * time.Hour, BucketCount: 18}, true
	case "30d":
		return localLLMMonitorPeriod{Duration: 30 * 24 * time.Hour, BucketCount: 18}, true
	case "all":
		return localLLMMonitorPeriod{Duration: 90 * time.Minute, BucketCount: 18, AllHistory: true}, true
	default:
		return localLLMMonitorPeriod{}, false
	}
}

func activeLocalMonitorGroups(groups []service.Group) []service.Group {
	active := make([]service.Group, 0, len(groups))
	for _, group := range groups {
		if group.ID <= 0 || (strings.TrimSpace(group.Status) != "" && !strings.EqualFold(strings.TrimSpace(group.Status), "active")) {
			continue
		}
		active = append(active, group)
	}
	return active
}

func loadLocalLLMMonitorUpstream(ctx context.Context, settingsProvider llmMonitorSettingsProvider, period, board string) []map[string]any {
	if settingsProvider == nil {
		return nil
	}
	settings, err := settingsProvider.GetLLMMonitorSettings(ctx)
	if err != nil || settings == nil || strings.TrimSpace(settings.StatusAPIURL) == "" {
		return nil
	}
	targetURL, err := llmMonitorTargetURL(settings.StatusAPIURL, period, board)
	if err != nil {
		return nil
	}

	requestCtx, cancel := context.WithTimeout(ctx, llmMonitorProxyTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("User-Agent", "sub2api-llm-monitor-local/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil
	}

	bodyReader := io.Reader(resp.Body)
	if strings.EqualFold(strings.TrimSpace(resp.Header.Get("Content-Encoding")), "gzip") {
		gzipReader, gzipErr := gzip.NewReader(resp.Body)
		if gzipErr != nil {
			return nil
		}
		defer gzipReader.Close()
		bodyReader = gzipReader
	}
	body, err := io.ReadAll(bodyReader)
	if err != nil {
		return nil
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	return extractLocalLLMMonitorGroups(payload)
}

func extractLocalLLMMonitorGroups(value any) []map[string]any {
	switch typed := value.(type) {
	case []any:
		groups := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			object, ok := item.(map[string]any)
			if ok && isLocalLLMMonitorGroupObject(object) {
				groups = append(groups, object)
				continue
			}
			groups = append(groups, extractLocalLLMMonitorGroups(item)...)
		}
		return groups
	case map[string]any:
		for _, key := range []string{"groups", "data", "items", "list", "status", "statuses", "providers", "services", "result"} {
			if child, ok := typed[key]; ok {
				if groups := extractLocalLLMMonitorGroups(child); len(groups) > 0 {
					return groups
				}
			}
		}
		if isLocalLLMMonitorGroupObject(typed) {
			return []map[string]any{typed}
		}
		for _, child := range typed {
			if groups := extractLocalLLMMonitorGroups(child); len(groups) > 0 {
				return groups
			}
		}
	}
	return nil
}

func isLocalLLMMonitorGroupObject(object map[string]any) bool {
	for _, key := range []string{"provider", "provider_name", "providerName", "name", "title", "service_provider", "layers", "timeline"} {
		if _, ok := object[key]; ok {
			return true
		}
	}
	return false
}

func buildLocalLLMMonitorPayload(
	groups []service.Group,
	mappings []service.SupplierProviderGroup,
	healthTrends []service.SupplierProviderGroupHealthTrend,
	upstreamGroups []map[string]any,
) []map[string]any {
	mappingsByLocalGroup := make(map[int64][]service.SupplierProviderGroup)
	for _, mapping := range mappings {
		if mapping.LocalGroupID == nil || *mapping.LocalGroupID <= 0 {
			continue
		}
		mappingsByLocalGroup[*mapping.LocalGroupID] = append(mappingsByLocalGroup[*mapping.LocalGroupID], mapping)
	}

	healthByLocalGroup := make(map[int64]service.LocalModelMonitorTrend, len(healthTrends))
	for _, healthTrend := range healthTrends {
		if trend, ok := localModelMonitorTrendFromHealth(healthTrend); ok {
			healthByLocalGroup[healthTrend.GroupID] = trend
		}
	}

	upstreamByKey := make(map[string]localLLMMonitorParsedTrend)
	for _, upstreamGroup := range upstreamGroups {
		parsed, ok := parseLocalLLMMonitorUpstreamGroup(upstreamGroup, time.Now().UTC())
		if !ok {
			continue
		}
		for _, key := range localLLMMonitorUpstreamKeys(upstreamGroup) {
			if _, exists := upstreamByKey[key]; !exists {
				upstreamByKey[key] = parsed
			}
		}
	}

	payload := make([]map[string]any, 0, len(groups))
	for _, group := range groups {
		upstreamParsed, upstreamOK := findLocalLLMMonitorUpstreamTrend(group, mappingsByLocalGroup[group.ID], upstreamByKey)
		upstream := upstreamParsed.Trend
		health, healthOK := healthByLocalGroup[group.ID]
		selected, selectedOK := service.SelectLocalModelMonitorTrend(localModelMonitorTrendPointer(upstream, upstreamOK), localModelMonitorTrendPointer(health, healthOK))

		item := map[string]any{
			"provider": group.Name,
			"service":  "CC",
			"layers":   []any{map[string]any{"timeline": []any{}}},
		}
		if !selectedOK {
			payload = append(payload, item)
			continue
		}

		selectedFromHealth := healthOK && (!upstreamOK || localModelMonitorTrendAverageOrZero(&health) >= localModelMonitorTrendAverageOrZero(&upstream))
		metadata := localLLMMonitorParsedTrend{}
		if !selectedFromHealth && upstreamOK {
			metadata = upstreamParsed
		}
		item["service"] = firstNonEmptyLocalLLMMonitorValue(metadata.Service, "CC")
		if metadata.Category != "" {
			item["category"] = metadata.Category
		}
		layer := map[string]any{
			"timeline": localLLMMonitorTimeline(selected.Trend),
			"current_status": map[string]any{
				"status":    localLLMMonitorStatus(selected.Availability, selected.Trend),
				"latency":   selected.Latency,
				"timestamp": localLLMMonitorTimestamp(selected.Time),
			},
		}
		if metadata.Model != "" {
			layer["model"] = metadata.Model
		}
		item["layers"] = []any{layer}
		payload = append(payload, item)
	}
	return payload
}

func localModelMonitorTrendPointer(trend service.LocalModelMonitorTrend, ok bool) *service.LocalModelMonitorTrend {
	if !ok {
		return nil
	}
	return &trend
}

func localModelMonitorTrendFromHealth(health service.SupplierProviderGroupHealthTrend) (service.LocalModelMonitorTrend, bool) {
	trend := service.LocalModelMonitorTrend{
		Availability: health.Availability,
		Latency:      health.Latency,
		Time:         health.Time,
		Trend:        make([]service.LocalModelMonitorTrendPoint, 0, len(health.Trend)),
	}
	for _, point := range health.Trend {
		trend.Trend = append(trend.Trend, service.LocalModelMonitorTrendPoint{
			Time:         point.Time,
			Availability: point.Availability,
			Latency:      point.Latency,
			Tone:         point.Tone,
			Valid:        point.TestedAccountCount > 0,
		})
	}
	if _, ok := trend.AverageAvailability(); !ok {
		return service.LocalModelMonitorTrend{}, false
	}
	if trend.Time.IsZero() {
		trend.Time = trend.LatestTime()
	}
	return trend, true
}

func findLocalLLMMonitorUpstreamTrend(
	group service.Group,
	mappings []service.SupplierProviderGroup,
	upstreamByKey map[string]localLLMMonitorParsedTrend,
) (localLLMMonitorParsedTrend, bool) {
	candidates := make([]string, 0, len(mappings)*3+1)
	for _, mapping := range mappings {
		candidates = append(candidates, mapping.MatchedUpstreamName)
	}
	for _, mapping := range mappings {
		candidates = append(candidates, mapping.Name, mapping.UpstreamKey)
	}
	candidates = append(candidates, group.Name)
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		key := normalizeLocalLLMMonitorKey(candidate)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		if parsed, ok := upstreamByKey[key]; ok {
			return parsed, true
		}
	}
	return localLLMMonitorParsedTrend{}, false
}

func localLLMMonitorUpstreamKeys(object map[string]any) []string {
	keys := make([]string, 0, 6)
	for _, key := range []string{"provider", "provider_name", "providerName", "name", "title", "service_provider"} {
		if value, ok := object[key]; ok {
			if text := localLLMMonitorString(value); text != "" {
				keys = append(keys, normalizeLocalLLMMonitorKey(text))
			}
		}
	}
	return uniqueLocalLLMMonitorKeys(keys)
}

func normalizeLocalLLMMonitorKey(value string) string {
	var builder strings.Builder
	for _, char := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsSpace(char) || char == '_' || char == '-' {
			continue
		}
		builder.WriteRune(char)
	}
	return builder.String()
}

func uniqueLocalLLMMonitorKeys(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func parseLocalLLMMonitorUpstreamGroup(object map[string]any, now time.Time) (localLLMMonitorParsedTrend, bool) {
	layer := object
	if layers, ok := object["layers"].([]any); ok && len(layers) > 0 {
		if first, ok := layers[0].(map[string]any); ok {
			layer = first
		}
	}

	rawTimeline := localLLMMonitorValue(layer, "timeline")
	if rawTimeline == nil {
		rawTimeline = localLLMMonitorValue(object, "timeline")
	}
	points := make([]service.LocalModelMonitorTrendPoint, 0)
	if timeline, ok := rawTimeline.([]any); ok {
		for index, rawPoint := range timeline {
			point, valid := parseLocalLLMMonitorPoint(rawPoint, now, index, len(timeline))
			if valid {
				points = append(points, point)
			}
		}
	}
	if len(points) == 0 {
		if point, valid := parseLocalLLMMonitorPoint(layer, now, 0, 1); valid {
			points = append(points, point)
		} else if point, valid := parseLocalLLMMonitorPoint(object, now, 0, 1); valid {
			points = append(points, point)
		}
	}
	if len(points) == 0 {
		return localLLMMonitorParsedTrend{}, false
	}

	trend := service.LocalModelMonitorTrend{Trend: points}
	latest := points[0]
	for _, point := range points[1:] {
		if point.Time.After(latest.Time) || latest.Time.IsZero() {
			latest = point
		}
	}
	trend.Availability = latest.Availability
	trend.Latency = latest.Latency
	trend.Time = latest.Time
	return localLLMMonitorParsedTrend{
		Trend:    trend,
		Service:  localLLMMonitorString(localLLMMonitorValue(object, "service")),
		Category: localLLMMonitorString(localLLMMonitorValue(object, "category")),
		Model:    localLLMMonitorString(firstLocalLLMMonitorValue(localLLMMonitorValue(layer, "model"), localLLMMonitorValue(layer, "request_model"))),
	}, true
}

func parseLocalLLMMonitorPoint(value any, now time.Time, index, total int) (service.LocalModelMonitorTrendPoint, bool) {
	object, ok := value.(map[string]any)
	if !ok {
		return service.LocalModelMonitorTrendPoint{}, false
	}
	availability, availabilityOK := localLLMMonitorNumber(firstLocalLLMMonitorValue(
		localLLMMonitorValue(object, "availability"),
		localLLMMonitorValue(object, "available_rate"),
		localLLMMonitorValue(object, "availableRate"),
		localLLMMonitorValue(object, "success_rate"),
		localLLMMonitorValue(object, "successRate"),
		localLLMMonitorValue(object, "rate_percent"),
		localLLMMonitorValue(object, "uptime"),
	))
	statusValue := firstLocalLLMMonitorValue(localLLMMonitorValue(object, "status"), localLLMMonitorValue(object, "state"))
	if !availabilityOK {
		availability, availabilityOK = localLLMMonitorStatusAvailability(statusValue)
	}
	if !availabilityOK {
		return service.LocalModelMonitorTrendPoint{}, false
	}
	pointTime := localLLMMonitorTime(firstLocalLLMMonitorValue(
		localLLMMonitorValue(object, "timestamp"),
		localLLMMonitorValue(object, "time"),
		localLLMMonitorValue(object, "checked_at"),
		localLLMMonitorValue(object, "checkedAt"),
	), time.Time{})
	if pointTime.IsZero() {
		step := time.Duration(total-index-1) * time.Minute
		pointTime = now.Add(-step)
	}
	latency, _ := localLLMMonitorNumber(firstLocalLLMMonitorValue(
		localLLMMonitorValue(object, "latency"),
		localLLMMonitorValue(object, "latency_ms"),
		localLLMMonitorValue(object, "latencyMs"),
		localLLMMonitorValue(object, "response_time"),
		localLLMMonitorValue(object, "responseTime"),
	))
	tone := localLLMMonitorTone(firstLocalLLMMonitorValue(localLLMMonitorValue(object, "tone"), statusValue), availability)
	return service.LocalModelMonitorTrendPoint{
		Time:         pointTime,
		Availability: availability,
		Latency:      int64(latency),
		Tone:         tone,
		Valid:        true,
	}, true
}

func localLLMMonitorTimeline(trend []service.LocalModelMonitorTrendPoint) []any {
	points := make([]any, 0, len(trend))
	for _, point := range trend {
		if !point.Valid {
			continue
		}
		points = append(points, map[string]any{
			"availability": point.Availability,
			"latency":      point.Latency,
			"tone":         point.Tone,
			"timestamp":    localLLMMonitorTimestamp(point.Time),
		})
	}
	return points
}

func localLLMMonitorTimestamp(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.UnixMilli()
}

func localLLMMonitorStatus(availability float64, trend []service.LocalModelMonitorTrendPoint) int {
	for index := len(trend) - 1; index >= 0; index-- {
		if trend[index].Valid {
			switch trend[index].Tone {
			case "green":
				return 1
			case "yellow":
				return 2
			case "red":
				return 0
			}
			break
		}
	}
	if availability >= 40 {
		return 1
	}
	if availability > 0 {
		return 2
	}
	return 0
}

func localLLMMonitorTone(value any, availability float64) string {
	text := strings.ToLower(strings.TrimSpace(localLLMMonitorString(value)))
	switch text {
	case "green", "healthy", "available", "ok", "success", "1", "true":
		return "green"
	case "yellow", "warn", "warning", "slow", "degraded", "partial", "2":
		return "yellow"
	case "red", "failed", "failure", "unavailable", "error", "0", "false":
		return "red"
	}
	if availability >= 40 {
		return "green"
	}
	if availability > 0 {
		return "yellow"
	}
	return "red"
}

func localLLMMonitorStatusAvailability(value any) (float64, bool) {
	if number, ok := localLLMMonitorNumber(value); ok {
		switch int(number) {
		case 1:
			return 100, true
		case 2:
			return 70, true
		case 0:
			return 0, true
		}
	}
	text := strings.ToLower(strings.TrimSpace(localLLMMonitorString(value)))
	switch text {
	case "healthy", "available", "ok", "success":
		return 100, true
	case "slow", "degraded", "warn", "warning", "partial":
		return 70, true
	case "failed", "failure", "unavailable", "error":
		return 0, true
	default:
		return 0, false
	}
}

func localLLMMonitorNumber(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		number, err := typed.Float64()
		return number, err == nil
	case string:
		number, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(typed, "%")), 64)
		return number, err == nil
	default:
		return 0, false
	}
}

func localLLMMonitorTime(value any, fallback time.Time) time.Time {
	if number, ok := localLLMMonitorNumber(value); ok && number > 0 {
		if number < 10000000000 {
			return time.Unix(int64(number), 0).UTC()
		}
		return time.UnixMilli(int64(number)).UTC()
	}
	if text := strings.TrimSpace(localLLMMonitorString(value)); text != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, text); err == nil {
			return parsed.UTC()
		}
		if parsed, err := time.Parse("2006-01-02 15:04:05", text); err == nil {
			return parsed.UTC()
		}
	}
	return fallback
}

func localLLMMonitorString(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return fmt.Sprint(value)
}

func localLLMMonitorValue(object map[string]any, key string) any {
	if object == nil {
		return nil
	}
	return object[key]
}

func firstLocalLLMMonitorValue(values ...any) any {
	for _, value := range values {
		if value == nil {
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			continue
		}
		return value
	}
	return nil
}

func firstNonEmptyLocalLLMMonitorValue(values ...string) string {
	for _, value := range values {
		if text := strings.TrimSpace(value); text != "" {
			return text
		}
	}
	return ""
}

func localModelMonitorTrendAverageOrZero(trend *service.LocalModelMonitorTrend) float64 {
	if trend == nil {
		return 0
	}
	average, _ := trend.AverageAvailability()
	return average
}

package routes

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const llmMonitorProxyTimeout = 15 * time.Second

var llmMonitorScrubbedValues = map[string]struct{}{
	"https://www.findcg.com": {},
	"findcg-ai":              {},
}

type llmMonitorSettingsProvider interface {
	GetLLMMonitorSettings(ctx context.Context) (*service.LLMMonitorSettings, error)
}

type llmMonitorGroupProvider interface {
	GetAllGroups(ctx context.Context) ([]service.Group, error)
}

func RegisterLLMMonitorRoutes(
	r gin.IRouter,
	settingsProvider llmMonitorSettingsProvider,
	groupProvider llmMonitorGroupProvider,
	historyStores ...service.LLMMonitorHistoryStore,
) {
	var historyStore service.LLMMonitorHistoryStore
	if len(historyStores) > 0 {
		historyStore = historyStores[0]
	}
	r.GET("/api/llm-monitor/status", func(c *gin.Context) {
		proxyLLMMonitorStatus(c, settingsProvider, func(ctx context.Context, body []byte) ([]byte, error) {
			return filterLLMMonitorStatusPayload(ctx, body, groupProvider)
		}, false, historyStore)
	})
}

func RegisterAdminLLMMonitorRoutes(r gin.IRouter, settingsProvider llmMonitorSettingsProvider) {
	r.GET("/upstream-management/monitor-status", func(c *gin.Context) {
		proxyLLMMonitorStatus(c, settingsProvider, nil, true, nil)
	})
}

func proxyLLMMonitorStatus(
	c *gin.Context,
	settingsProvider llmMonitorSettingsProvider,
	transform func(context.Context, []byte) ([]byte, error),
	standardResponse bool,
	historyStore service.LLMMonitorHistoryStore,
) {
	settings, err := settingsProvider.GetLLMMonitorSettings(c.Request.Context())
	if err != nil || settings == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load monitor settings"})
		return
	}

	targetURL, err := llmMonitorTargetURL(settings.StatusAPIURL, c.Query("period"), c.Query("board"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid monitor upstream url"})
		return
	}

	body, contentType, statusCode, fetchErr := fetchLLMMonitorStatus(c.Request.Context(), targetURL, "sub2api-llm-monitor/1.0")
	sourceKey, period, board, historyKeyOK := llmMonitorHistoryRequestKey(settings.StatusAPIURL, c.Query("period"), c.Query("board"))
	if fetchErr != nil {
		if recovered, ok := loadLLMMonitorHistory(c.Request.Context(), historyStore, sourceKey, period, board, historyKeyOK); ok {
			body = recovered
			contentType = "application/json"
			statusCode = http.StatusOK
		} else {
			c.JSON(http.StatusBadGateway, gin.H{"error": "monitor upstream request failed"})
			return
		}
	}

	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		if recovered, ok := loadLLMMonitorHistory(c.Request.Context(), historyStore, sourceKey, period, board, historyKeyOK); ok {
			body = recovered
			contentType = "application/json"
			statusCode = http.StatusOK
		} else {
			c.Data(statusCode, contentType, body)
			return
		}
	}

	freshSnapshot := llmMonitorPayloadHasTimeline(body)
	if !freshSnapshot {
		if recovered, ok := loadLLMMonitorHistory(c.Request.Context(), historyStore, sourceKey, period, board, historyKeyOK); ok {
			body = recovered
			contentType = "application/json"
			statusCode = http.StatusOK
		}
	}
	if historyStore != nil && freshSnapshot {
		// 保存未过滤的上游快照，供本地监控按供应商映射恢复；页面响应再单独过滤。
		persistLLMMonitorHistory(c.Request.Context(), historyStore, sourceKey, period, board, body)
	}
	if transform != nil {
		body, err = transform(c.Request.Context(), body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to filter monitor response"})
			return
		}
	}
	if standardResponse {
		var payload json.RawMessage = body
		c.JSON(statusCode, gin.H{
			"code":    0,
			"message": "success",
			"data":    payload,
		})
		return
	}
	c.Data(statusCode, contentType, body)
}

func fetchLLMMonitorStatus(ctx context.Context, targetURL, userAgent string) ([]byte, string, int, error) {
	requestCtx, cancel := context.WithTimeout(ctx, llmMonitorProxyTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, "application/json", 0, fmt.Errorf("创建模型监控上游请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "application/json", 0, err
	}
	defer resp.Body.Close()
	responseBody := io.Reader(resp.Body)
	if strings.EqualFold(strings.TrimSpace(resp.Header.Get("Content-Encoding")), "gzip") {
		gzipReader, gzipErr := gzip.NewReader(resp.Body)
		if gzipErr != nil {
			return nil, "application/json", 0, fmt.Errorf("解压模型监控上游响应失败: %w", gzipErr)
		}
		defer gzipReader.Close()
		responseBody = gzipReader
	}
	body, err := io.ReadAll(responseBody)
	if err != nil {
		return nil, "application/json", 0, fmt.Errorf("读取模型监控上游响应失败: %w", err)
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/json"
	}
	return body, contentType, resp.StatusCode, nil
}

func llmMonitorHistoryRequestKey(rawURL, period, board string) (string, string, string, bool) {
	targetURL, err := llmMonitorTargetURL(rawURL, period, board)
	if err != nil {
		return "", "", "", false
	}
	u, err := url.Parse(targetURL)
	if err != nil {
		return "", "", "", false
	}
	query := u.Query()
	return service.LLMMonitorHistorySourceKey(rawURL), query.Get("period"), query.Get("board"), true
}

func loadLLMMonitorHistory(
	ctx context.Context,
	historyStore service.LLMMonitorHistoryStore,
	sourceKey, period, board string,
	keyOK bool,
) ([]byte, bool) {
	if historyStore == nil || !keyOK {
		return nil, false
	}
	snapshot, err := historyStore.LoadLatestSnapshot(ctx, sourceKey, period, board)
	if err != nil {
		slog.Warn("读取模型监控历史快照失败", "error", err)
		return nil, false
	}
	if snapshot == nil || !json.Valid(snapshot.Payload) || !llmMonitorPayloadHasTimeline(snapshot.Payload) {
		return nil, false
	}
	return append([]byte(nil), snapshot.Payload...), true
}

func persistLLMMonitorHistory(
	ctx context.Context,
	historyStore service.LLMMonitorHistoryStore,
	sourceKey, period, board string,
	body []byte,
) {
	if historyStore == nil || sourceKey == "" || period == "" || board == "" || !llmMonitorPayloadHasTimeline(body) {
		return
	}
	sanitized, err := scrubLLMMonitorPayload(body)
	if err != nil || !llmMonitorPayloadHasTimeline(sanitized) {
		return
	}
	if err := historyStore.SaveSnapshot(ctx, service.LLMMonitorHistorySnapshot{
		SourceKey:  sourceKey,
		Period:     period,
		Board:      board,
		Payload:    sanitized,
		CapturedAt: time.Now().UTC(),
	}); err != nil {
		slog.Warn("保存模型监控历史快照失败", "error", err)
	}
}

func llmMonitorPayloadHasTimeline(body []byte) bool {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	return llmMonitorValueHasTimeline(payload)
}

func llmMonitorValueHasTimeline(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key == "timeline" {
				if timeline, ok := child.([]any); ok && len(timeline) > 0 {
					return true
				}
			}
			if llmMonitorValueHasTimeline(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if llmMonitorValueHasTimeline(child) {
				return true
			}
		}
	}
	return false
}

func filterLLMMonitorStatusPayload(ctx context.Context, body []byte, groupProvider llmMonitorGroupProvider) ([]byte, error) {
	if groupProvider == nil {
		return scrubLLMMonitorPayload(body)
	}

	groups, err := groupProvider.GetAllGroups(ctx)
	if err != nil {
		return nil, err
	}
	allowedProviders := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		if key := normalizeLLMMonitorProviderKey(group.Name); key != "" {
			allowedProviders[key] = struct{}{}
		}
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body, nil
	}
	filtered := filterLLMMonitorPayloadValue(payload, allowedProviders)
	cleaned := scrubLLMMonitorValue(filtered)
	out, err := json.Marshal(cleaned)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func scrubLLMMonitorPayload(body []byte) ([]byte, error) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body, nil
	}
	out, err := json.Marshal(scrubLLMMonitorValue(payload))
	if err != nil {
		return nil, err
	}
	return out, nil
}

func filterLLMMonitorPayloadValue(value any, allowedProviders map[string]struct{}) any {
	object, ok := value.(map[string]any)
	if !ok {
		return value
	}

	if groups, ok := object["groups"].([]any); ok {
		object["groups"] = filterLLMMonitorGroups(groups, allowedProviders)
	}
	if meta, ok := object["meta"].(map[string]any); ok {
		if ids, ok := meta["all_monitor_ids"].([]any); ok {
			meta["all_monitor_ids"] = filterLLMMonitorIDs(ids, allowedProviders)
		}
		object["meta"] = meta
	}
	return object
}

func filterLLMMonitorGroups(groups []any, allowedProviders map[string]struct{}) []any {
	filtered := make([]any, 0, len(groups))
	for _, item := range groups {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		provider, _ := object["provider"].(string)
		if _, ok := allowedProviders[normalizeLLMMonitorProviderKey(provider)]; ok {
			filtered = append(filtered, object)
		}
	}
	return filtered
}

func filterLLMMonitorIDs(ids []any, allowedProviders map[string]struct{}) []any {
	filtered := make([]any, 0, len(ids))
	for _, item := range ids {
		text, ok := item.(string)
		if !ok {
			continue
		}
		provider := text
		if idx := strings.LastIndex(provider, "-"); idx > -1 {
			provider = provider[:idx]
		}
		if idx := strings.LastIndex(provider, "-"); idx > -1 {
			provider = provider[:idx]
		}
		if _, ok := allowedProviders[normalizeLLMMonitorProviderKey(provider)]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func scrubLLMMonitorValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		cleaned := make(map[string]any, len(typed))
		for key, child := range typed {
			if text, ok := child.(string); ok && shouldScrubLLMMonitorString(text) {
				continue
			}
			cleaned[key] = scrubLLMMonitorValue(child)
		}
		return cleaned
	case []any:
		cleaned := make([]any, 0, len(typed))
		for _, child := range typed {
			if text, ok := child.(string); ok && shouldScrubLLMMonitorString(text) {
				continue
			}
			cleaned = append(cleaned, scrubLLMMonitorValue(child))
		}
		return cleaned
	default:
		return value
	}
}

func shouldScrubLLMMonitorString(value string) bool {
	_, ok := llmMonitorScrubbedValues[strings.TrimSpace(value)]
	return ok
}

func normalizeLLMMonitorProviderKey(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), ""))
}

func llmMonitorTargetURL(rawURL, period, board string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("empty url")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if !u.IsAbs() || (u.Scheme != "http" && u.Scheme != "https") || strings.TrimSpace(u.Host) == "" || u.Fragment != "" {
		return "", fmt.Errorf("invalid url")
	}
	q := u.Query()
	if strings.TrimSpace(period) == "" {
		period = "90m"
	}
	if strings.TrimSpace(board) == "" {
		board = "hot"
	}
	q.Set("period", strings.TrimSpace(period))
	q.Set("board", strings.TrimSpace(board))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

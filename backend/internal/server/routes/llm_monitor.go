package routes

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

func RegisterLLMMonitorRoutes(r gin.IRouter, settingsProvider llmMonitorSettingsProvider, groupProvider llmMonitorGroupProvider) {
	r.GET("/api/llm-monitor/status", func(c *gin.Context) {
		proxyLLMMonitorStatus(c, settingsProvider, func(ctx context.Context, body []byte) ([]byte, error) {
			return filterLLMMonitorStatusPayload(ctx, body, groupProvider)
		}, false)
	})
}

func RegisterAdminLLMMonitorRoutes(r gin.IRouter, settingsProvider llmMonitorSettingsProvider) {
	r.GET("/upstream-management/monitor-status", func(c *gin.Context) {
		proxyLLMMonitorStatus(c, settingsProvider, nil, true)
	})
}

func proxyLLMMonitorStatus(
	c *gin.Context,
	settingsProvider llmMonitorSettingsProvider,
	transform func(context.Context, []byte) ([]byte, error),
	standardResponse bool,
) {
	settings, err := settingsProvider.GetLLMMonitorSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load monitor settings"})
		return
	}

	targetURL, err := llmMonitorTargetURL(settings.StatusAPIURL, c.Query("period"), c.Query("board"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid monitor upstream url"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), llmMonitorProxyTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upstream request"})
		return
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("User-Agent", "sub2api-llm-monitor/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "monitor upstream request failed"})
		return
	}
	defer resp.Body.Close()
	responseBody := io.Reader(resp.Body)
	if strings.EqualFold(strings.TrimSpace(resp.Header.Get("Content-Encoding")), "gzip") {
		gzipReader, gzipErr := gzip.NewReader(resp.Body)
		if gzipErr != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to decompress monitor upstream response"})
			return
		}
		defer gzipReader.Close()
		responseBody = gzipReader
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/json"
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		c.DataFromReader(resp.StatusCode, -1, contentType, responseBody, map[string]string{})
		return
	}

	body, err := io.ReadAll(responseBody)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read monitor upstream response"})
		return
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
		c.JSON(resp.StatusCode, gin.H{
			"code":    0,
			"message": "success",
			"data":    payload,
		})
		return
	}
	c.Data(resp.StatusCode, contentType, body)
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

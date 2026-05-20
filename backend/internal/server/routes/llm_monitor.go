package routes

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const llmMonitorProxyTimeout = 15 * time.Second

type llmMonitorSettingsProvider interface {
	GetPublicSettings(ctx context.Context) (*service.PublicSettings, error)
}

func RegisterLLMMonitorRoutes(r gin.IRouter, settingsProvider llmMonitorSettingsProvider) {
	r.GET("/api/llm-monitor/status", func(c *gin.Context) {
		settings, err := settingsProvider.GetPublicSettings(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load monitor settings"})
			return
		}

		targetURL, err := llmMonitorTargetURL(settings.LLMMonitorStatusAPIURL, c.Query("period"), c.Query("board"))
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
		req.Header.Set("User-Agent", "sub2api-llm-monitor/1.0")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "monitor upstream request failed"})
			return
		}
		defer resp.Body.Close()

		contentType := resp.Header.Get("Content-Type")
		if strings.TrimSpace(contentType) == "" {
			contentType = "application/json"
		}
		c.DataFromReader(resp.StatusCode, resp.ContentLength, contentType, resp.Body, map[string]string{})
	})
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

package captcha

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// yesCaptcha 对接 YesCaptcha 的 JSON API，负责求解 Cloudflare Turnstile。
type yesCaptcha struct {
	apiKey   string
	endpoint string
	http     *http.Client
}

func newYesCaptcha(cfg Config) *yesCaptcha {
	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	if endpoint == "" {
		endpoint = defaultYesCaptchaEndpoint
	}
	return &yesCaptcha{
		apiKey:   strings.TrimSpace(cfg.APIKey),
		endpoint: endpoint,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type yesCaptchaCreateResp struct {
	ErrorID          int    `json:"errorId"`
	ErrorCode        string `json:"errorCode"`
	ErrorDescription string `json:"errorDescription"`
	TaskID           any    `json:"taskId"`
}

type yesCaptchaResultResp struct {
	ErrorID          int    `json:"errorId"`
	ErrorCode        string `json:"errorCode"`
	ErrorDescription string `json:"errorDescription"`
	Status           string `json:"status"`
	Solution         struct {
		Token string `json:"token"`
	} `json:"solution"`
}

func (p *yesCaptcha) SolveTurnstile(ctx context.Context, siteKey, pageURL string) (string, error) {
	if p.apiKey == "" {
		return "", errors.New("yescaptcha: api key is empty")
	}
	if strings.TrimSpace(siteKey) == "" {
		return "", errors.New("yescaptcha: siteKey is empty")
	}
	if strings.TrimSpace(pageURL) == "" {
		return "", errors.New("yescaptcha: pageURL is empty")
	}

	taskID, err := p.createTask(ctx, siteKey, pageURL)
	if err != nil {
		return "", err
	}

	deadline := time.Now().Add(120 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return "", errors.New("yescaptcha: timed out waiting for solution")
			}
			token, ready, err := p.fetchResult(ctx, taskID)
			if err != nil {
				return "", err
			}
			if ready {
				return token, nil
			}
		}
	}
}

func (p *yesCaptcha) createTask(ctx context.Context, siteKey, pageURL string) (string, error) {
	body := map[string]any{
		"clientKey": p.apiKey,
		"task": map[string]any{
			"type":       "TurnstileTaskProxyless",
			"websiteURL": pageURL,
			"websiteKey": siteKey,
		},
	}
	raw, err := p.postJSON(ctx, p.endpoint+"/createTask", body)
	if err != nil {
		return "", fmt.Errorf("yescaptcha createTask http: %w", err)
	}
	var resp yesCaptchaCreateResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("yescaptcha createTask decode: %w", err)
	}
	if resp.ErrorID != 0 || resp.TaskID == nil {
		return "", fmt.Errorf("yescaptcha createTask: %s %s", resp.ErrorCode, resp.ErrorDescription)
	}
	switch v := resp.TaskID.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return "", errors.New("yescaptcha createTask: empty taskId")
		}
		return v, nil
	case float64:
		return fmt.Sprintf("%.0f", v), nil
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" || s == "<nil>" {
			return "", errors.New("yescaptcha createTask: empty taskId")
		}
		return s, nil
	}
}

func (p *yesCaptcha) fetchResult(ctx context.Context, taskID string) (string, bool, error) {
	raw, err := p.postJSON(ctx, p.endpoint+"/getTaskResult", map[string]any{
		"clientKey": p.apiKey,
		"taskId":    taskID,
	})
	if err != nil {
		return "", false, fmt.Errorf("yescaptcha getTaskResult http: %w", err)
	}
	var resp yesCaptchaResultResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", false, fmt.Errorf("yescaptcha getTaskResult decode: %w", err)
	}
	if resp.ErrorID != 0 {
		return "", false, fmt.Errorf("yescaptcha getTaskResult: %s %s", resp.ErrorCode, resp.ErrorDescription)
	}
	switch strings.ToLower(strings.TrimSpace(resp.Status)) {
	case "ready":
		if strings.TrimSpace(resp.Solution.Token) == "" {
			return "", false, errors.New("yescaptcha: ready but empty token")
		}
		return resp.Solution.Token, true, nil
	case "failed", "error":
		reason := strings.TrimSpace(resp.ErrorDescription)
		if reason == "" {
			reason = strings.TrimSpace(resp.ErrorCode)
		}
		if reason == "" {
			reason = "task failed"
		}
		return "", false, fmt.Errorf("yescaptcha task failed: %s", reason)
	}
	return "", false, nil
}

func (p *yesCaptcha) postJSON(ctx context.Context, url string, body any) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := p.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return raw, nil
}

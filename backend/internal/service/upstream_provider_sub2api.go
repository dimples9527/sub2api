package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Sub2APIProviderAdapter struct {
	httpClient *http.Client
}

func NewSub2APIProviderAdapter(httpClient *http.Client) *Sub2APIProviderAdapter {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Sub2APIProviderAdapter{httpClient: httpClient}
}

func (a *Sub2APIProviderAdapter) FetchKeys(ctx context.Context, provider UpstreamProviderConfig) ([]UpstreamProviderKey, []string, error) {
	token := ""
	if provider.Email != "" || provider.Password != "" {
		nextToken, _, _, err := a.login(ctx, provider)
		if err != nil {
			return nil, nil, err
		}
		token = nextToken
	}
	payload, status, err := a.request(ctx, provider, token, provider.APIKeysURL, "sub2api provider keys")
	if err != nil {
		return nil, nil, err
	}
	if status < 200 || status >= 300 {
		return nil, nil, upstreamProviderHTTPError("sub2api provider keys", status, payload)
	}
	keys, err := parseSub2APIProviderKeys(provider, payload)
	if err != nil {
		return nil, nil, err
	}
	return keys, nil, nil
}

func (a *Sub2APIProviderAdapter) Test(ctx context.Context, provider UpstreamProviderConfig) UpstreamProviderTestResult {
	result := newUpstreamProviderTestResult(provider)
	token := ""
	if provider.Email != "" || provider.Password != "" {
		nextToken, _, status, err := a.login(ctx, provider)
		result.Login.StatusCode = status
		if err != nil {
			result.Login.Error = err.Error()
			return result
		}
		token = nextToken
		result.Login.OK = true
	} else {
		result.Login.OK = true
	}
	payload, status, err := a.request(ctx, provider, token, provider.APIKeysURL, "sub2api provider keys")
	result.Keys.StatusCode = status
	if err != nil {
		result.Keys.Error = err.Error()
		return result
	}
	if status < 200 || status >= 300 {
		result.Keys.Error = upstreamProviderHTTPError("sub2api provider keys", status, payload).Error()
		return result
	}
	keys, err := parseSub2APIProviderKeys(provider, payload)
	if err != nil {
		result.Keys.Error = err.Error()
		return result
	}
	result.Keys.OK = true
	result.Keys.ItemCount = len(keys)
	result.ParsedKeys = limitUpstreamProviderKeys(keys, 20)
	return result
}

func (a *Sub2APIProviderAdapter) login(ctx context.Context, provider UpstreamProviderConfig) (string, []byte, int, error) {
	if provider.LoginURL == "" {
		return "", nil, 0, fmt.Errorf("sub2api provider login_url is required when credentials are configured")
	}
	body, err := json.Marshal(map[string]string{
		"email":    provider.Email,
		"password": provider.Password,
	})
	if err != nil {
		return "", nil, 0, fmt.Errorf("marshal sub2api login payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamProviderURL(provider, provider.LoginURL), bytes.NewReader(body))
	if err != nil {
		return "", nil, 0, fmt.Errorf("create sub2api login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", nil, 0, fmt.Errorf("sub2api login request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, resp.StatusCode, fmt.Errorf("read sub2api login response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", raw, resp.StatusCode, upstreamProviderHTTPError("sub2api login", resp.StatusCode, raw)
	}
	var parsed struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			AccessToken string `json:"access_token"`
			Token       string `json:"token"`
			TokenType   string `json:"token_type"`
		} `json:"data"`
		AccessToken string `json:"access_token"`
		Token       string `json:"token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", raw, resp.StatusCode, fmt.Errorf("decode sub2api login response: %w", err)
	}
	if parsed.Code != 0 {
		if parsed.Message == "" {
			parsed.Message = "unknown error"
		}
		return "", raw, resp.StatusCode, fmt.Errorf("sub2api login failed: %s", parsed.Message)
	}
	token := firstNonEmptySub2APIString(parsed.Data.AccessToken, parsed.Data.Token, parsed.AccessToken, parsed.Token)
	if token == "" {
		return "", raw, resp.StatusCode, fmt.Errorf("sub2api login failed: missing token")
	}
	return token, raw, resp.StatusCode, nil
}

func firstNonEmptySub2APIString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (a *Sub2APIProviderAdapter) request(ctx context.Context, provider UpstreamProviderConfig, token, path, label string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstreamProviderURL(provider, path), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create %s request: %w", label, err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%s request failed: %w", label, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read %s response: %w", label, err)
	}
	return raw, resp.StatusCode, nil
}

func parseSub2APIProviderKeys(provider UpstreamProviderConfig, payload []byte) ([]UpstreamProviderKey, error) {
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Items []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
				Group  *struct {
					ID             any      `json:"id"`
					Name           string   `json:"name"`
					RateMultiplier *float64 `json:"rate_multiplier"`
				} `json:"group"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return nil, fmt.Errorf("decode sub2api provider keys response: %w", err)
	}
	if resp.Code != 0 {
		if resp.Message == "" {
			resp.Message = "unknown error"
		}
		return nil, fmt.Errorf("sub2api provider keys failed: %s", resp.Message)
	}
	keys := make([]UpstreamProviderKey, 0, len(resp.Data.Items))
	for _, item := range resp.Data.Items {
		if item.Name == "" || item.Group == nil || item.Group.Name == "" || item.Group.RateMultiplier == nil {
			continue
		}
		keys = append(keys, UpstreamProviderKey{
			ProviderSlug:   provider.Slug,
			ProviderName:   provider.Name,
			ProviderType:   provider.Type,
			KeyName:        upstreamProviderKeyName(provider, item.Name),
			GroupName:      item.Group.Name,
			RateMultiplier: *item.Group.RateMultiplier,
			RawStatus:      item.Status,
			RawGroupID:     fmt.Sprint(item.Group.ID),
		})
	}
	return keys, nil
}

func limitUpstreamProviderKeys(keys []UpstreamProviderKey, limit int) []UpstreamProviderKey {
	if limit <= 0 || len(keys) <= limit {
		return keys
	}
	out := make([]UpstreamProviderKey, limit)
	copy(out, keys[:limit])
	return out
}

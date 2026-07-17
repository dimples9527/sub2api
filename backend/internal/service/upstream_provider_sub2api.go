package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultSub2APIProviderGroupsURL = "/api/v1/groups/available?timezone=Asia%2FShanghai"
const defaultSub2APIProviderBalanceURL = "/api/v1/auth/me?timezone=Asia%2FShanghai"
const defaultSub2APIProviderUsageCostURL = "/api/v1/usage/dashboard/stats?timezone=Asia%2FShanghai"

type upstreamSub2APIAuth struct {
	Token     string
	TokenType string
}

type sub2APIProviderRateMultiplier struct {
	value float64
	valid bool
}

func (r *sub2APIProviderRateMultiplier) UnmarshalJSON(raw []byte) error {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" {
		return nil
	}
	if strings.HasPrefix(value, `"`) {
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return err
		}
		value = strings.TrimSpace(text)
	}
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	r.value = parsed
	r.valid = true
	return nil
}

type Sub2APIProviderAdapter struct {
	httpClient *http.Client

	mu         sync.Mutex
	authBySlug map[string]upstreamSub2APIAuth
}

func NewSub2APIProviderAdapter(httpClient *http.Client) *Sub2APIProviderAdapter {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Sub2APIProviderAdapter{
		httpClient: httpClient,
		authBySlug: map[string]upstreamSub2APIAuth{},
	}
}

func (a *Sub2APIProviderAdapter) FetchKeys(ctx context.Context, provider UpstreamProviderConfig) ([]UpstreamProviderKey, []string, error) {
	for attempt := 0; attempt < 2; attempt++ {
		auth, err := a.ensureAuth(ctx, provider)
		if err != nil {
			return nil, nil, err
		}
		payload, status, err := a.request(ctx, provider, auth, provider.APIKeysURL, "sub2api provider keys")
		if err != nil {
			return nil, nil, err
		}
		if status < 200 || status >= 300 {
			requestErr := upstreamProviderHTTPError("sub2api provider keys", status, payload)
			if attempt == 0 && hasSub2APICredentials(provider) && upstreamProviderAuthFailureHint(status, payload, requestErr) {
				a.clearAuth(provider.Slug)
				continue
			}
			return nil, nil, requestErr
		}
		keys, err := parseSub2APIProviderKeys(provider, payload)
		if err != nil {
			if attempt == 0 && hasSub2APICredentials(provider) && upstreamProviderAuthFailureHint(status, payload, err) {
				a.clearAuth(provider.Slug)
				continue
			}
			return nil, nil, err
		}
		return keys, nil, nil
	}
	return nil, nil, fmt.Errorf("sub2api provider keys failed after auth retry")
}

func (a *Sub2APIProviderAdapter) FetchGroups(ctx context.Context, provider UpstreamProviderConfig) ([]UpstreamProviderGroup, []string, error) {
	groupsURL := strings.TrimSpace(provider.AvailableGroupsURL)
	if groupsURL == "" {
		groupsURL = strings.TrimSpace(provider.GroupsURL)
	}
	if groupsURL == "" {
		groupsURL = defaultSub2APIProviderGroupsURL
	}
	for attempt := 0; attempt < 2; attempt++ {
		auth, err := a.ensureAuth(ctx, provider)
		if err != nil {
			return nil, nil, err
		}
		payload, status, err := a.request(ctx, provider, auth, groupsURL, "sub2api provider groups")
		if err != nil {
			return nil, nil, err
		}
		if status < 200 || status >= 300 {
			requestErr := upstreamProviderHTTPError("sub2api provider groups", status, payload)
			if attempt == 0 && hasSub2APICredentials(provider) && upstreamProviderAuthFailureHint(status, payload, requestErr) {
				a.clearAuth(provider.Slug)
				continue
			}
			return nil, nil, requestErr
		}
		groups, err := parseSub2APIProviderGroups(provider, payload)
		if err != nil {
			if attempt == 0 && hasSub2APICredentials(provider) && upstreamProviderAuthFailureHint(status, payload, err) {
				a.clearAuth(provider.Slug)
				continue
			}
			return nil, nil, err
		}
		return groups, nil, nil
	}
	return nil, nil, fmt.Errorf("sub2api provider groups failed after auth retry")
}

func (a *Sub2APIProviderAdapter) FetchBalance(ctx context.Context, provider UpstreamProviderConfig) (UpstreamProviderBalance, error) {
	balanceURL := strings.TrimSpace(provider.BalanceURL)
	if balanceURL == "" {
		balanceURL = defaultSub2APIProviderBalanceURL
	}
	for attempt := 0; attempt < 2; attempt++ {
		auth, err := a.ensureAuth(ctx, provider)
		if err != nil {
			return UpstreamProviderBalance{}, err
		}
		payload, status, err := a.request(ctx, provider, auth, balanceURL, "sub2api provider balance")
		if err != nil {
			return UpstreamProviderBalance{}, err
		}
		if status < 200 || status >= 300 {
			requestErr := upstreamProviderHTTPError("sub2api provider balance", status, payload)
			if attempt == 0 && hasSub2APICredentials(provider) && upstreamProviderAuthFailureHint(status, payload, requestErr) {
				a.clearAuth(provider.Slug)
				continue
			}
			return UpstreamProviderBalance{}, requestErr
		}
		balance, err := parseSub2APIProviderBalance(provider, payload)
		if err != nil {
			if attempt == 0 && hasSub2APICredentials(provider) && upstreamProviderAuthFailureHint(status, payload, err) {
				a.clearAuth(provider.Slug)
				continue
			}
			return UpstreamProviderBalance{}, err
		}
		return balance, nil
	}
	return UpstreamProviderBalance{}, fmt.Errorf("sub2api provider balance failed after auth retry")
}

func (a *Sub2APIProviderAdapter) FetchTodayCost(ctx context.Context, provider UpstreamProviderConfig, day time.Time) (UpstreamProviderCost, error) {
	usageCostURL := strings.TrimSpace(provider.UsageCostURL)
	if usageCostURL == "" {
		usageCostURL = defaultSub2APIProviderUsageCostURL
	}
	for attempt := 0; attempt < 2; attempt++ {
		auth, err := a.ensureAuth(ctx, provider)
		if err != nil {
			return UpstreamProviderCost{}, err
		}
		payload, status, err := a.request(ctx, provider, auth, usageCostURL, "sub2api provider usage cost")
		if err != nil {
			return UpstreamProviderCost{}, err
		}
		if status < 200 || status >= 300 {
			requestErr := upstreamProviderHTTPError("sub2api provider usage cost", status, payload)
			if attempt == 0 && hasSub2APICredentials(provider) && upstreamProviderAuthFailureHint(status, payload, requestErr) {
				a.clearAuth(provider.Slug)
				continue
			}
			return UpstreamProviderCost{}, requestErr
		}
		cost, err := parseSub2APIProviderTodayCost(provider, payload)
		if err != nil {
			if attempt == 0 && hasSub2APICredentials(provider) && upstreamProviderAuthFailureHint(status, payload, err) {
				a.clearAuth(provider.Slug)
				continue
			}
			return UpstreamProviderCost{}, err
		}
		return cost, nil
	}
	return UpstreamProviderCost{}, fmt.Errorf("sub2api provider usage cost failed after auth retry")
}

func (a *Sub2APIProviderAdapter) Test(ctx context.Context, provider UpstreamProviderConfig) UpstreamProviderTestResult {
	result := newUpstreamProviderTestResult(provider)
	auth := upstreamSub2APIAuth{}
	if hasSub2APICredentials(provider) {
		nextAuth, _, status, err := a.login(ctx, provider)
		result.Login.StatusCode = status
		if err != nil {
			result.Login.Error = err.Error()
			return result
		}
		auth = nextAuth
		result.Login.OK = true
	} else {
		result.Login.OK = true
	}
	payload, status, err := a.request(ctx, provider, auth, provider.APIKeysURL, "sub2api provider keys")
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

func (a *Sub2APIProviderAdapter) ensureAuth(ctx context.Context, provider UpstreamProviderConfig) (upstreamSub2APIAuth, error) {
	if !hasSub2APICredentials(provider) {
		return upstreamSub2APIAuth{}, nil
	}
	if auth, ok := a.cachedAuth(provider.Slug); ok {
		return auth, nil
	}
	auth, _, _, err := a.login(ctx, provider)
	if err != nil {
		return upstreamSub2APIAuth{}, err
	}
	a.storeAuth(provider.Slug, auth)
	return auth, nil
}

func (a *Sub2APIProviderAdapter) cachedAuth(slug string) (upstreamSub2APIAuth, bool) {
	if a == nil {
		return upstreamSub2APIAuth{}, false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	auth, ok := a.authBySlug[strings.TrimSpace(slug)]
	if !ok || strings.TrimSpace(auth.Token) == "" {
		return upstreamSub2APIAuth{}, false
	}
	if strings.TrimSpace(auth.TokenType) == "" {
		auth.TokenType = "Bearer"
	}
	return auth, true
}

func (a *Sub2APIProviderAdapter) storeAuth(slug string, auth upstreamSub2APIAuth) {
	if a == nil || strings.TrimSpace(slug) == "" || strings.TrimSpace(auth.Token) == "" {
		return
	}
	if strings.TrimSpace(auth.TokenType) == "" {
		auth.TokenType = "Bearer"
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.authBySlug[strings.TrimSpace(slug)] = auth
}

func (a *Sub2APIProviderAdapter) clearAuth(slug string) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.authBySlug, strings.TrimSpace(slug))
}

func hasSub2APICredentials(provider UpstreamProviderConfig) bool {
	return strings.TrimSpace(provider.Email) != "" || strings.TrimSpace(provider.Password) != ""
}

func (a *Sub2APIProviderAdapter) login(ctx context.Context, provider UpstreamProviderConfig) (upstreamSub2APIAuth, []byte, int, error) {
	loginPath := provider.LoginURL
	if strings.TrimSpace(loginPath) == "" {
		loginPath = "/api/v1/auth/login"
	}
	body, err := json.Marshal(map[string]string{
		"email":    provider.Email,
		"password": provider.Password,
	})
	if err != nil {
		return upstreamSub2APIAuth{}, nil, 0, fmt.Errorf("marshal sub2api login payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamProviderURL(provider, loginPath), bytes.NewReader(body))
	if err != nil {
		return upstreamSub2APIAuth{}, nil, 0, fmt.Errorf("create sub2api login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return upstreamSub2APIAuth{}, nil, 0, fmt.Errorf("sub2api login request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return upstreamSub2APIAuth{}, nil, resp.StatusCode, fmt.Errorf("read sub2api login response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return upstreamSub2APIAuth{}, raw, resp.StatusCode, upstreamProviderHTTPError("sub2api login", resp.StatusCode, raw)
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
		return upstreamSub2APIAuth{}, raw, resp.StatusCode, fmt.Errorf("decode sub2api login response: %w", err)
	}
	if parsed.Code != 0 {
		if parsed.Message == "" {
			parsed.Message = "unknown error"
		}
		return upstreamSub2APIAuth{}, raw, resp.StatusCode, fmt.Errorf("sub2api login failed: %s", parsed.Message)
	}
	token := firstNonEmptySub2APIString(parsed.Data.AccessToken, parsed.Data.Token, parsed.AccessToken, parsed.Token)
	if token == "" {
		return upstreamSub2APIAuth{}, raw, resp.StatusCode, fmt.Errorf("sub2api login failed: missing token")
	}
	tokenType := firstNonEmptySub2APIString(parsed.Data.TokenType, parsed.TokenType, "Bearer")
	return upstreamSub2APIAuth{Token: token, TokenType: tokenType}, raw, resp.StatusCode, nil
}

func firstNonEmptySub2APIString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (a *Sub2APIProviderAdapter) request(ctx context.Context, provider UpstreamProviderConfig, auth upstreamSub2APIAuth, path, label string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstreamProviderURL(provider, path), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create %s request: %w", label, err)
	}
	if strings.TrimSpace(auth.Token) != "" {
		tokenType := strings.TrimSpace(auth.TokenType)
		if tokenType == "" {
			tokenType = "Bearer"
		}
		req.Header.Set("Authorization", tokenType+" "+auth.Token)
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
			KeyName:        strings.TrimSpace(item.Name),
			GroupName:      item.Group.Name,
			RateMultiplier: *item.Group.RateMultiplier,
			RawStatus:      item.Status,
			RawGroupID:     fmt.Sprint(item.Group.ID),
		})
	}
	return keys, nil
}

func parseSub2APIProviderGroups(provider UpstreamProviderConfig, payload []byte) ([]UpstreamProviderGroup, error) {
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    []struct {
			ID             any                           `json:"id"`
			Name           string                        `json:"name"`
			Status         string                        `json:"status"`
			RateMultiplier sub2APIProviderRateMultiplier `json:"rate_multiplier"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return nil, fmt.Errorf("decode sub2api provider groups response: %w", err)
	}
	if resp.Code != 0 {
		if resp.Message == "" {
			resp.Message = "unknown error"
		}
		return nil, fmt.Errorf("sub2api provider groups failed: %s", resp.Message)
	}
	groups := make([]UpstreamProviderGroup, 0, len(resp.Data))
	for _, item := range resp.Data {
		name := strings.TrimSpace(item.Name)
		if name == "" || !item.RateMultiplier.valid {
			continue
		}
		groups = append(groups, UpstreamProviderGroup{
			ProviderSlug:   provider.Slug,
			ProviderName:   provider.Name,
			ProviderType:   provider.Type,
			GroupName:      name,
			RateMultiplier: item.RateMultiplier.value,
			RawStatus:      item.Status,
			RawGroupID:     fmt.Sprint(item.ID),
		})
	}
	return groups, nil
}

func parseSub2APIProviderBalance(provider UpstreamProviderConfig, payload []byte) (UpstreamProviderBalance, error) {
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Balance float64 `json:"balance"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return UpstreamProviderBalance{}, fmt.Errorf("decode sub2api provider balance response: %w", err)
	}
	if resp.Code != 0 {
		if resp.Message == "" {
			resp.Message = "unknown error"
		}
		return UpstreamProviderBalance{}, fmt.Errorf("sub2api provider balance failed: %s", resp.Message)
	}
	return UpstreamProviderBalance{
		ProviderSlug: provider.Slug,
		ProviderName: provider.Name,
		ProviderType: provider.Type,
		Balance:      resp.Data.Balance,
	}, nil
}

func parseSub2APIProviderTodayCost(provider UpstreamProviderConfig, payload []byte) (UpstreamProviderCost, error) {
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			TodayActualCost float64 `json:"today_actual_cost"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return UpstreamProviderCost{}, fmt.Errorf("decode sub2api provider usage cost response: %w", err)
	}
	if resp.Code != 0 {
		if resp.Message == "" {
			resp.Message = "unknown error"
		}
		return UpstreamProviderCost{}, fmt.Errorf("sub2api provider usage cost failed: %s", resp.Message)
	}
	return UpstreamProviderCost{
		ProviderSlug: provider.Slug,
		ProviderName: provider.Name,
		ProviderType: provider.Type,
		TodayCost:    resp.Data.TodayActualCost,
	}, nil
}

func limitUpstreamProviderKeys(keys []UpstreamProviderKey, limit int) []UpstreamProviderKey {
	if limit <= 0 || len(keys) <= limit {
		return keys
	}
	out := make([]UpstreamProviderKey, limit)
	copy(out, keys[:limit])
	return out
}

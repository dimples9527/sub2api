package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type NewAPIProviderAdapter struct {
	httpClient *http.Client

	mu            sync.Mutex
	sessionBySlug map[string]newAPIProviderSession
}

type newAPIProviderSession struct {
	UserID       int64
	CookieHeader string
}

type newAPIProviderGroupRatio struct {
	value float64
	valid bool
}

const defaultNewAPIProviderBalanceURL = "/api/user/self"
const defaultNewAPIProviderUsageCostURL = "/api/log/self/stat?type=0&token_name=&model_name=&start_timestamp={start_timestamp}&end_timestamp={end_timestamp}&group="

func (r *newAPIProviderGroupRatio) UnmarshalJSON(raw []byte) error {
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

func NewNewAPIProviderAdapter(httpClient *http.Client) *NewAPIProviderAdapter {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &NewAPIProviderAdapter{
		httpClient:    httpClient,
		sessionBySlug: map[string]newAPIProviderSession{},
	}
}

func (a *NewAPIProviderAdapter) FetchKeys(ctx context.Context, provider UpstreamProviderConfig) ([]UpstreamProviderKey, []string, error) {
	for attempt := 0; attempt < 2; attempt++ {
		session, err := a.ensureSession(ctx, provider)
		if err != nil {
			return nil, nil, err
		}
		keysPayload, status, err := a.request(ctx, provider, session, provider.APIKeysURL, "newapi provider keys")
		if err != nil {
			return nil, nil, err
		}
		if status < 200 || status >= 300 {
			requestErr := upstreamProviderHTTPError("newapi provider keys", status, keysPayload)
			if attempt == 0 && upstreamProviderAuthFailureHint(status, keysPayload, requestErr) {
				a.clearSession(provider.Slug)
				continue
			}
			return nil, nil, requestErr
		}
		groupsPayload, groupStatus, err := a.request(ctx, provider, session, provider.GroupsURL, "newapi provider groups")
		if err != nil {
			return nil, nil, err
		}
		if groupStatus < 200 || groupStatus >= 300 {
			requestErr := upstreamProviderHTTPError("newapi provider groups", groupStatus, groupsPayload)
			if attempt == 0 && upstreamProviderAuthFailureHint(groupStatus, groupsPayload, requestErr) {
				a.clearSession(provider.Slug)
				continue
			}
			return nil, nil, requestErr
		}
		keys, warnings, err := parseNewAPIProviderKeys(provider, keysPayload, groupsPayload)
		if err != nil {
			if attempt == 0 && upstreamProviderAuthFailureHint(status, keysPayload, err) {
				a.clearSession(provider.Slug)
				continue
			}
			return nil, nil, err
		}
		return keys, warnings, nil
	}
	return nil, nil, fmt.Errorf("newapi provider keys failed after auth retry")
}

func (a *NewAPIProviderAdapter) FetchGroups(ctx context.Context, provider UpstreamProviderConfig) ([]UpstreamProviderGroup, []string, error) {
	groupsURL := strings.TrimSpace(provider.AvailableGroupsURL)
	if groupsURL == "" {
		groupsURL = provider.GroupsURL
	}
	for attempt := 0; attempt < 2; attempt++ {
		session, err := a.ensureSession(ctx, provider)
		if err != nil {
			return nil, nil, err
		}
		groupsPayload, status, err := a.request(ctx, provider, session, groupsURL, "newapi provider groups")
		if err != nil {
			return nil, nil, err
		}
		if status < 200 || status >= 300 {
			requestErr := upstreamProviderHTTPError("newapi provider groups", status, groupsPayload)
			if attempt == 0 && upstreamProviderAuthFailureHint(status, groupsPayload, requestErr) {
				a.clearSession(provider.Slug)
				continue
			}
			return nil, nil, requestErr
		}
		groups, err := parseNewAPIProviderGroups(provider, groupsPayload)
		if err != nil {
			if attempt == 0 && upstreamProviderAuthFailureHint(status, groupsPayload, err) {
				a.clearSession(provider.Slug)
				continue
			}
			return nil, nil, err
		}
		return groups, nil, nil
	}
	return nil, nil, fmt.Errorf("newapi provider groups failed after auth retry")
}

func (a *NewAPIProviderAdapter) FetchBalance(ctx context.Context, provider UpstreamProviderConfig) (UpstreamProviderBalance, error) {
	balanceURL := strings.TrimSpace(provider.BalanceURL)
	if balanceURL == "" {
		balanceURL = defaultNewAPIProviderBalanceURL
	}
	for attempt := 0; attempt < 2; attempt++ {
		session, err := a.ensureSession(ctx, provider)
		if err != nil {
			return UpstreamProviderBalance{}, err
		}
		payload, status, err := a.request(ctx, provider, session, balanceURL, "newapi provider balance")
		if err != nil {
			return UpstreamProviderBalance{}, err
		}
		if status < 200 || status >= 300 {
			requestErr := upstreamProviderHTTPError("newapi provider balance", status, payload)
			if attempt == 0 && upstreamProviderAuthFailureHint(status, payload, requestErr) {
				a.clearSession(provider.Slug)
				continue
			}
			return UpstreamProviderBalance{}, requestErr
		}
		balance, err := parseNewAPIProviderBalance(provider, payload)
		if err != nil {
			if attempt == 0 && upstreamProviderAuthFailureHint(status, payload, err) {
				a.clearSession(provider.Slug)
				continue
			}
			return UpstreamProviderBalance{}, err
		}
		return balance, nil
	}
	return UpstreamProviderBalance{}, fmt.Errorf("newapi provider balance failed after auth retry")
}

func (a *NewAPIProviderAdapter) FetchTodayCost(ctx context.Context, provider UpstreamProviderConfig, day time.Time) (UpstreamProviderCost, error) {
	usageCostURL := strings.TrimSpace(provider.UsageCostURL)
	if usageCostURL == "" {
		usageCostURL = defaultNewAPIProviderUsageCostURL
	}
	usageCostURL = upstreamProviderUsageCostURL(usageCostURL, day, upstreamBalanceStatsLocation())
	for attempt := 0; attempt < 2; attempt++ {
		session, err := a.ensureSession(ctx, provider)
		if err != nil {
			return UpstreamProviderCost{}, err
		}
		payload, status, err := a.request(ctx, provider, session, usageCostURL, "newapi provider usage cost")
		if err != nil {
			return UpstreamProviderCost{}, err
		}
		if status < 200 || status >= 300 {
			requestErr := upstreamProviderHTTPError("newapi provider usage cost", status, payload)
			if attempt == 0 && upstreamProviderAuthFailureHint(status, payload, requestErr) {
				a.clearSession(provider.Slug)
				continue
			}
			return UpstreamProviderCost{}, requestErr
		}
		cost, err := parseNewAPIProviderTodayCost(provider, payload)
		if err != nil {
			if attempt == 0 && upstreamProviderAuthFailureHint(status, payload, err) {
				a.clearSession(provider.Slug)
				continue
			}
			return UpstreamProviderCost{}, err
		}
		return cost, nil
	}
	return UpstreamProviderCost{}, fmt.Errorf("newapi provider usage cost failed after auth retry")
}

func (a *NewAPIProviderAdapter) Test(ctx context.Context, provider UpstreamProviderConfig) UpstreamProviderTestResult {
	result := newUpstreamProviderTestResult(provider)
	groupStage := UpstreamProviderTestStage{}
	result.Groups = &groupStage

	session, _, status, err := a.login(ctx, provider)
	result.Login.StatusCode = status
	if err != nil {
		result.Login.Error = err.Error()
		return result
	}
	result.Login.OK = true
	result.Login.UserID = session.UserID
	result.Login.CookiePresent = session.CookieHeader != ""

	keysPayload, status, err := a.request(ctx, provider, session, provider.APIKeysURL, "newapi provider keys")
	result.Keys.StatusCode = status
	if err != nil {
		result.Keys.Error = err.Error()
		return result
	}
	if status < 200 || status >= 300 {
		result.Keys.Error = upstreamProviderHTTPError("newapi provider keys", status, keysPayload).Error()
		return result
	}
	keyCount, err := countNewAPIProviderKeys(keysPayload)
	if err != nil {
		result.Keys.Error = err.Error()
		return result
	}
	result.Keys.OK = true
	result.Keys.ItemCount = keyCount

	groupsPayload, status, err := a.request(ctx, provider, session, provider.GroupsURL, "newapi provider groups")
	result.Groups.StatusCode = status
	if err != nil {
		result.Groups.Error = err.Error()
		return result
	}
	if status < 200 || status >= 300 {
		result.Groups.Error = upstreamProviderHTTPError("newapi provider groups", status, groupsPayload).Error()
		return result
	}
	groupCount, err := countNewAPIProviderGroups(groupsPayload)
	if err != nil {
		result.Groups.Error = err.Error()
		return result
	}
	result.Groups.OK = true
	result.Groups.GroupCount = groupCount

	keys, warnings, err := parseNewAPIProviderKeys(provider, keysPayload, groupsPayload)
	if err != nil {
		result.Warnings = append(result.Warnings, err.Error())
		return result
	}
	result.ParsedKeys = limitUpstreamProviderKeys(keys, 20)
	result.Warnings = append(result.Warnings, warnings...)
	return result
}

func (a *NewAPIProviderAdapter) ensureSession(ctx context.Context, provider UpstreamProviderConfig) (newAPIProviderSession, error) {
	if session, ok := a.cachedSession(provider.Slug); ok {
		return session, nil
	}
	session, _, _, err := a.login(ctx, provider)
	if err != nil {
		return newAPIProviderSession{}, err
	}
	a.storeSession(provider.Slug, session)
	return session, nil
}

func (a *NewAPIProviderAdapter) cachedSession(slug string) (newAPIProviderSession, bool) {
	if a == nil {
		return newAPIProviderSession{}, false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	session, ok := a.sessionBySlug[strings.TrimSpace(slug)]
	if !ok || session.UserID <= 0 || strings.TrimSpace(session.CookieHeader) == "" {
		return newAPIProviderSession{}, false
	}
	return session, true
}

func (a *NewAPIProviderAdapter) storeSession(slug string, session newAPIProviderSession) {
	if a == nil || strings.TrimSpace(slug) == "" || session.UserID <= 0 || strings.TrimSpace(session.CookieHeader) == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessionBySlug[strings.TrimSpace(slug)] = session
}

func (a *NewAPIProviderAdapter) clearSession(slug string) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessionBySlug, strings.TrimSpace(slug))
}

func (a *NewAPIProviderAdapter) login(ctx context.Context, provider UpstreamProviderConfig) (newAPIProviderSession, []byte, int, error) {
	username := provider.Username
	if username == "" {
		username = provider.Email
	}
	body, err := json.Marshal(map[string]string{
		"username": username,
		"password": provider.Password,
	})
	if err != nil {
		return newAPIProviderSession{}, nil, 0, fmt.Errorf("marshal newapi login payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamProviderURL(provider, provider.LoginURL), bytes.NewReader(body))
	if err != nil {
		return newAPIProviderSession{}, nil, 0, fmt.Errorf("create newapi login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return newAPIProviderSession{}, nil, 0, fmt.Errorf("newapi login request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return newAPIProviderSession{}, nil, resp.StatusCode, fmt.Errorf("read newapi login response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newAPIProviderSession{}, raw, resp.StatusCode, upstreamProviderHTTPError("newapi login", resp.StatusCode, raw)
	}
	var parsed struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return newAPIProviderSession{}, raw, resp.StatusCode, fmt.Errorf("decode newapi login response: %w", err)
	}
	if !parsed.Success {
		if parsed.Message == "" {
			parsed.Message = "unknown error"
		}
		return newAPIProviderSession{}, raw, resp.StatusCode, fmt.Errorf("newapi login failed: %s", parsed.Message)
	}
	if parsed.Data.ID <= 0 {
		return newAPIProviderSession{}, raw, resp.StatusCode, fmt.Errorf("newapi login failed: missing user id")
	}
	cookieHeader := upstreamProviderCookiesHeader(resp.Cookies())
	if cookieHeader == "" {
		return newAPIProviderSession{}, raw, resp.StatusCode, fmt.Errorf("newapi login failed: missing cookie")
	}
	return newAPIProviderSession{UserID: parsed.Data.ID, CookieHeader: cookieHeader}, raw, resp.StatusCode, nil
}

func (a *NewAPIProviderAdapter) request(ctx context.Context, provider UpstreamProviderConfig, session newAPIProviderSession, path, label string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstreamProviderURL(provider, path), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create %s request: %w", label, err)
	}
	req.Header.Set("New-Api-User", strconv.FormatInt(session.UserID, 10))
	if session.CookieHeader != "" {
		req.Header.Set("Cookie", session.CookieHeader)
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

func parseNewAPIProviderKeys(provider UpstreamProviderConfig, keysPayload, groupsPayload []byte) ([]UpstreamProviderKey, []string, error) {
	var keysResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Items []struct {
				Name   string `json:"name"`
				Group  string `json:"group"`
				Status any    `json:"status"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(keysPayload, &keysResp); err != nil {
		return nil, nil, fmt.Errorf("decode newapi provider keys response: %w", err)
	}
	if !keysResp.Success {
		if keysResp.Message == "" {
			keysResp.Message = "unknown error"
		}
		return nil, nil, fmt.Errorf("newapi provider keys failed: %s", keysResp.Message)
	}
	var groupsResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    map[string]struct {
			Ratio newAPIProviderGroupRatio `json:"ratio"`
			ID    any                      `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(groupsPayload, &groupsResp); err != nil {
		return nil, nil, fmt.Errorf("decode newapi provider groups response: %w", err)
	}
	if !groupsResp.Success {
		if groupsResp.Message == "" {
			groupsResp.Message = "unknown error"
		}
		return nil, nil, fmt.Errorf("newapi provider groups failed: %s", groupsResp.Message)
	}
	type groupInfo struct {
		name  string
		ratio float64
		id    string
	}
	ratioByGroup := make(map[string]groupInfo, len(groupsResp.Data)*2)
	for name, group := range groupsResp.Data {
		if !group.Ratio.valid || group.Ratio.value < 0 {
			continue
		}
		info := groupInfo{name: name, ratio: group.Ratio.value, id: fmt.Sprint(group.ID)}
		ratioByGroup[name] = info
		ratioByGroup[normalizeUpstreamProviderGroupName(name)] = info
	}
	keys := make([]UpstreamProviderKey, 0, len(keysResp.Data.Items))
	warnings := []string{}
	for _, item := range keysResp.Data.Items {
		if item.Name == "" || item.Group == "" {
			continue
		}
		info, ok := ratioByGroup[item.Group]
		if !ok {
			info, ok = ratioByGroup[normalizeUpstreamProviderGroupName(item.Group)]
		}
		if !ok {
			warnings = append(warnings, fmt.Sprintf("newapi key %s group %s has no matching group ratio", item.Name, item.Group))
			continue
		}
		keys = append(keys, UpstreamProviderKey{
			ProviderSlug:   provider.Slug,
			ProviderName:   provider.Name,
			ProviderType:   provider.Type,
			KeyName:        item.Name,
			GroupName:      item.Group,
			RateMultiplier: info.ratio,
			RawStatus:      fmt.Sprint(item.Status),
			RawGroupID:     info.id,
		})
	}
	return keys, warnings, nil
}

func parseNewAPIProviderGroups(provider UpstreamProviderConfig, groupsPayload []byte) ([]UpstreamProviderGroup, error) {
	var groupsResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    map[string]struct {
			Ratio newAPIProviderGroupRatio `json:"ratio"`
			ID    any                      `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(groupsPayload, &groupsResp); err != nil {
		return nil, fmt.Errorf("decode newapi provider groups response: %w", err)
	}
	if !groupsResp.Success {
		if groupsResp.Message == "" {
			groupsResp.Message = "unknown error"
		}
		return nil, fmt.Errorf("newapi provider groups failed: %s", groupsResp.Message)
	}
	names := make([]string, 0, len(groupsResp.Data))
	for name := range groupsResp.Data {
		names = append(names, name)
	}
	sort.Strings(names)
	groups := make([]UpstreamProviderGroup, 0, len(names))
	for _, name := range names {
		group := groupsResp.Data[name]
		if strings.TrimSpace(name) == "" || !group.Ratio.valid || group.Ratio.value < 0 {
			continue
		}
		groups = append(groups, UpstreamProviderGroup{
			ProviderSlug:   provider.Slug,
			ProviderName:   provider.Name,
			ProviderType:   provider.Type,
			GroupName:      name,
			RateMultiplier: group.Ratio.value,
			RawGroupID:     fmt.Sprint(group.ID),
		})
	}
	return groups, nil
}

func parseNewAPIProviderBalance(provider UpstreamProviderConfig, payload []byte) (UpstreamProviderBalance, error) {
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Quota float64 `json:"quota"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return UpstreamProviderBalance{}, fmt.Errorf("decode newapi provider balance response: %w", err)
	}
	if !resp.Success {
		if resp.Message == "" {
			resp.Message = "unknown error"
		}
		return UpstreamProviderBalance{}, fmt.Errorf("newapi provider balance failed: %s", resp.Message)
	}
	return UpstreamProviderBalance{
		ProviderSlug: provider.Slug,
		ProviderName: provider.Name,
		ProviderType: provider.Type,
		Balance:      resp.Data.Quota / 500000,
	}, nil
}

func parseNewAPIProviderTodayCost(provider UpstreamProviderConfig, payload []byte) (UpstreamProviderCost, error) {
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Quota float64 `json:"quota"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return UpstreamProviderCost{}, fmt.Errorf("decode newapi provider usage cost response: %w", err)
	}
	if !resp.Success {
		if resp.Message == "" {
			resp.Message = "unknown error"
		}
		return UpstreamProviderCost{}, fmt.Errorf("newapi provider usage cost failed: %s", resp.Message)
	}
	return UpstreamProviderCost{
		ProviderSlug: provider.Slug,
		ProviderName: provider.Name,
		ProviderType: provider.Type,
		TodayCost:    resp.Data.Quota / 500000,
	}, nil
}

func countNewAPIProviderKeys(payload []byte) (int, error) {
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Items []json.RawMessage `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return 0, fmt.Errorf("decode newapi provider keys response: %w", err)
	}
	if !resp.Success {
		if resp.Message == "" {
			resp.Message = "unknown error"
		}
		return 0, fmt.Errorf("newapi provider keys failed: %s", resp.Message)
	}
	return len(resp.Data.Items), nil
}

func countNewAPIProviderGroups(payload []byte) (int, error) {
	var resp struct {
		Success bool                       `json:"success"`
		Message string                     `json:"message"`
		Data    map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return 0, fmt.Errorf("decode newapi provider groups response: %w", err)
	}
	if !resp.Success {
		if resp.Message == "" {
			resp.Message = "unknown error"
		}
		return 0, fmt.Errorf("newapi provider groups failed: %s", resp.Message)
	}
	return len(resp.Data), nil
}

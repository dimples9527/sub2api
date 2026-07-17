package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultSupplierNewAPILoginPath    = "/api/user/login"
	defaultSupplierNewAPIKeysPath     = "/api/token/"
	defaultSupplierNewAPIGroupsPath   = "/api/group/"
	defaultSupplierNewAPIBalancePath  = "/api/user/self"
	defaultSupplierNewAPIUsageCostURL = "/api/log/self/stat?type=0&token_name=&model_name=&start_timestamp={start_timestamp}&end_timestamp={end_timestamp}&group="
	supplierNewAPIQuotaUnit           = 500000
)

type SupplierNewAPIClient struct {
	httpClient *http.Client

	sessionMu sync.Mutex
	sessions  map[string]supplierNewAPISession

	endpointResultMu sync.Mutex
	endpointResults  map[string]SupplierProviderEndpointResult
}

type supplierNewAPISession struct {
	UserID       int64
	CookieHeader string
}

type supplierNewAPIGroupRatio struct {
	value float64
	valid bool
}

type supplierNewAPIGroupInfo struct {
	Key            string
	Name           string
	RateMultiplier float64
	RawStatus      string
}

func NewSupplierNewAPIClient(httpClient *http.Client) *SupplierNewAPIClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultSupplierSub2APIHTTPTimeout}
	}
	return &SupplierNewAPIClient{
		httpClient:      httpClient,
		sessions:        make(map[string]supplierNewAPISession),
		endpointResults: make(map[string]SupplierProviderEndpointResult),
	}
}

func (r *supplierNewAPIGroupRatio) UnmarshalJSON(raw []byte) error {
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
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	r.value = parsed
	r.valid = true
	return nil
}

func (c *SupplierNewAPIClient) FetchAccounts(ctx context.Context, provider *SupplierProvider, password string) ([]SupplierProviderRemoteAccount, error) {
	keysPath := strings.TrimSpace(provider.APIKeysURL)
	if keysPath == "" {
		keysPath = defaultSupplierNewAPIKeysPath
	}
	groupsPath := supplierNewAPIGroupsPath(provider)
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		session, err := c.ensureSession(ctx, provider, password)
		if err != nil {
			return nil, err
		}
		keysRaw, status, err := c.authenticatedGet(ctx, provider, session, keysPath, "accounts")
		if err != nil {
			if attempt == 0 && supplierNewAPIAuthFailure(status, keysRaw, err) {
				c.clearSession(provider)
				lastErr = err
				continue
			}
			return nil, err
		}
		groupsRaw, groupStatus, err := c.authenticatedGet(ctx, provider, session, groupsPath, "groups")
		if err != nil {
			if attempt == 0 && supplierNewAPIAuthFailure(groupStatus, groupsRaw, err) {
				c.clearSession(provider)
				lastErr = err
				continue
			}
			return nil, err
		}
		accounts, parseErr := parseSupplierNewAPIAccounts(keysRaw, groupsRaw)
		c.annotateEndpointParse(provider.ID, "accounts", map[string]any{"count": len(accounts)}, parseErr)
		return accounts, parseErr
	}
	return nil, fmt.Errorf("supplier newapi accounts failed after auth retry: %w", lastErr)
}

func (c *SupplierNewAPIClient) FetchGroups(ctx context.Context, provider *SupplierProvider, password string) ([]SupplierProviderRemoteGroup, error) {
	groupsPath := supplierNewAPIGroupsPath(provider)
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		session, err := c.ensureSession(ctx, provider, password)
		if err != nil {
			return nil, err
		}
		raw, status, err := c.authenticatedGet(ctx, provider, session, groupsPath, "groups")
		if err != nil {
			if attempt == 0 && supplierNewAPIAuthFailure(status, raw, err) {
				c.clearSession(provider)
				lastErr = err
				continue
			}
			return nil, err
		}
		groups, parseErr := parseSupplierNewAPIGroups(raw)
		c.annotateEndpointParse(provider.ID, "groups", map[string]any{"count": len(groups)}, parseErr)
		return groups, parseErr
	}
	return nil, fmt.Errorf("supplier newapi groups failed after auth retry: %w", lastErr)
}

func (c *SupplierNewAPIClient) FetchBalance(ctx context.Context, provider *SupplierProvider, password string) (float64, error) {
	path := strings.TrimSpace(provider.BalanceURL)
	if path == "" {
		path = defaultSupplierNewAPIBalancePath
	}
	raw, err := c.fetchJSONWithRetry(ctx, provider, password, path, "balance")
	if err != nil {
		return 0, err
	}
	balance, parseErr := parseSupplierNewAPINumber(raw, "balance")
	c.annotateEndpointParse(provider.ID, "balance", map[string]any{"balance": balance}, parseErr)
	return balance, parseErr
}

func (c *SupplierNewAPIClient) FetchCost(ctx context.Context, provider *SupplierProvider, password string, day time.Time) (float64, error) {
	path := strings.TrimSpace(provider.UsageCostURL)
	if path == "" {
		path = defaultSupplierNewAPIUsageCostURL
	}
	path = supplierNewAPIUsageCostURL(path, day)
	raw, err := c.fetchJSONWithRetry(ctx, provider, password, path, "cost")
	if err != nil {
		return 0, err
	}
	cost, parseErr := parseSupplierNewAPINumber(raw, "cost")
	c.annotateEndpointParse(provider.ID, "cost", map[string]any{"today_actual_cost": cost}, parseErr)
	return cost, parseErr
}

func (c *SupplierNewAPIClient) TestEndpoint(ctx context.Context, provider *SupplierProvider, password string, scope string) (SupplierProviderEndpointTestResult, error) {
	scope = strings.TrimSpace(scope)
	result := SupplierProviderEndpointTestResult{ProviderID: provider.ID, Scope: scope, Attempts: []SupplierProviderEndpointTestAttempt{}}
	session, err := c.ensureSession(ctx, provider, password)
	if err != nil {
		return result, err
	}
	endpoints := []string{}
	switch scope {
	case SupplierSyncScopeAccounts:
		endpoints = []string{firstSupplierSub2APIString(provider.APIKeysURL, defaultSupplierNewAPIKeysPath)}
	case SupplierSyncScopeGroups:
		endpoints = []string{supplierNewAPIGroupsPath(provider)}
	case SupplierSyncScopeBalance:
		endpoints = []string{firstSupplierSub2APIString(provider.BalanceURL, defaultSupplierNewAPIBalancePath)}
	case SupplierSyncScopeCost:
		endpoints = []string{supplierNewAPIUsageCostURL(firstSupplierSub2APIString(provider.UsageCostURL, defaultSupplierNewAPIUsageCostURL), time.Now())}
	default:
		return SupplierProviderEndpointTestResult{}, fmt.Errorf("unsupported supplier endpoint test scope: %s", scope)
	}
	for _, endpoint := range endpoints {
		startedAt := time.Now()
		raw, status, err := c.authenticatedGet(ctx, provider, session, endpoint, scope)
		attempt := SupplierProviderEndpointTestAttempt{
			Endpoint:        endpoint,
			HTTPStatus:      status,
			DurationMS:      time.Since(startedAt).Milliseconds(),
			ResponseBytes:   len(raw),
			ResponseSummary: supplierSub2APISafeResponseText(raw, supplierSub2APITestResponseSummaryLimit),
			Error:           supplierSub2APIErrorText(err),
		}
		if err == nil {
			attempt.ParsedData, attempt.ParseError = supplierNewAPIParsedDiagnostic(scope, raw, provider)
		}
		result.Attempts = append(result.Attempts, attempt)
		result.Endpoint = attempt.Endpoint
		result.HTTPStatus = attempt.HTTPStatus
		result.DurationMS = attempt.DurationMS
		result.ResponseBytes = attempt.ResponseBytes
		result.ResponseSummary = attempt.ResponseSummary
		result.ParsedData = attempt.ParsedData
		result.ParseError = attempt.ParseError
		result.Error = attempt.Error
		break
	}
	return result, nil
}

func (c *SupplierNewAPIClient) LastEndpointResult(providerID int64, scope string) *SupplierProviderEndpointResult {
	c.endpointResultMu.Lock()
	defer c.endpointResultMu.Unlock()
	result, ok := c.endpointResults[supplierSub2APIEndpointResultKey(providerID, scope)]
	if !ok {
		return nil
	}
	return &result
}

func (c *SupplierNewAPIClient) fetchJSONWithRetry(ctx context.Context, provider *SupplierProvider, password, path, label string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		session, err := c.ensureSession(ctx, provider, password)
		if err != nil {
			return nil, err
		}
		raw, status, err := c.authenticatedGet(ctx, provider, session, path, label)
		if err == nil {
			return raw, nil
		}
		if attempt == 0 && supplierNewAPIAuthFailure(status, raw, err) {
			c.clearSession(provider)
			lastErr = err
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("supplier newapi %s failed after auth retry: %w", label, lastErr)
}

func (c *SupplierNewAPIClient) authenticatedGet(ctx context.Context, provider *SupplierProvider, session supplierNewAPISession, path, label string) ([]byte, int, error) {
	startedAt := time.Now()
	raw, status, err := c.doJSON(ctx, http.MethodGet, provider, path, session, nil)
	c.recordEndpointResult(provider.ID, label, SupplierProviderEndpointResult{
		Endpoint:        path,
		HTTPStatus:      status,
		DurationMS:      time.Since(startedAt).Milliseconds(),
		ResponseBytes:   len(raw),
		ResponseSummary: supplierSub2APISafeResponseText(raw, supplierSub2APITestResponseSummaryLimit),
		Error:           supplierSub2APIErrorText(err),
	})
	if err != nil {
		return raw, status, fmt.Errorf("supplier newapi %s request failed: %w", label, err)
	}
	if status < 200 || status >= 300 {
		err := supplierSub2APIHTTPError("newapi "+label, status, raw)
		c.updateEndpointError(provider.ID, label, err)
		return raw, status, err
	}
	if err := supplierNewAPIEnvelopeOK(raw); err != nil {
		c.updateEndpointError(provider.ID, label, err)
		return raw, status, err
	}
	return raw, status, nil
}

func (c *SupplierNewAPIClient) ensureSession(ctx context.Context, provider *SupplierProvider, password string) (supplierNewAPISession, error) {
	if session, ok := c.cachedSession(provider); ok {
		return session, nil
	}
	session, err := c.login(ctx, provider, password)
	if err != nil {
		return supplierNewAPISession{}, err
	}
	c.storeSession(provider, session)
	return session, nil
}

func (c *SupplierNewAPIClient) cachedSession(provider *SupplierProvider) (supplierNewAPISession, bool) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	session, ok := c.sessions[supplierNewAPISessionKey(provider)]
	if !ok || session.UserID <= 0 || strings.TrimSpace(session.CookieHeader) == "" {
		return supplierNewAPISession{}, false
	}
	return session, true
}

func (c *SupplierNewAPIClient) storeSession(provider *SupplierProvider, session supplierNewAPISession) {
	if session.UserID <= 0 || strings.TrimSpace(session.CookieHeader) == "" {
		return
	}
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	c.sessions[supplierNewAPISessionKey(provider)] = session
}

func (c *SupplierNewAPIClient) clearSession(provider *SupplierProvider) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	delete(c.sessions, supplierNewAPISessionKey(provider))
}

func (c *SupplierNewAPIClient) login(ctx context.Context, provider *SupplierProvider, password string) (supplierNewAPISession, error) {
	loginPath := strings.TrimSpace(provider.LoginURL)
	if loginPath == "" {
		loginPath = defaultSupplierNewAPILoginPath
	}
	username := strings.TrimSpace(provider.Username)
	if username == "" {
		username = strings.TrimSpace(provider.Email)
	}
	body, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		return supplierNewAPISession{}, fmt.Errorf("marshal supplier newapi login: %w", err)
	}
	raw, status, cookies, err := c.doLogin(ctx, provider, loginPath, bytes.NewReader(body))
	if err != nil {
		return supplierNewAPISession{}, err
	}
	if status < 200 || status >= 300 {
		return supplierNewAPISession{}, supplierSub2APIHTTPError("newapi login", status, raw)
	}
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return supplierNewAPISession{}, fmt.Errorf("decode supplier newapi login response: %w", err)
	}
	if !resp.Success {
		return supplierNewAPISession{}, fmt.Errorf("supplier newapi login failed: %s", firstSupplierSub2APIString(resp.Message, "unknown error"))
	}
	if resp.Data.ID <= 0 {
		return supplierNewAPISession{}, fmt.Errorf("supplier newapi login failed: missing user id")
	}
	cookieHeader := supplierNewAPICookiesHeader(cookies)
	if cookieHeader == "" {
		return supplierNewAPISession{}, fmt.Errorf("supplier newapi login failed: missing cookie")
	}
	return supplierNewAPISession{UserID: resp.Data.ID, CookieHeader: cookieHeader}, nil
}

func (c *SupplierNewAPIClient) doLogin(ctx context.Context, provider *SupplierProvider, path string, body io.Reader) ([]byte, int, []*http.Cookie, error) {
	target, err := supplierSub2APIURL(provider.BaseURL, path)
	if err != nil {
		return nil, 0, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), body)
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("supplier newapi login request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := readSupplierSub2APIResponse(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, resp.Cookies(), err
	}
	return raw, resp.StatusCode, resp.Cookies(), nil
}

func (c *SupplierNewAPIClient) doJSON(ctx context.Context, method string, provider *SupplierProvider, path string, session supplierNewAPISession, body io.Reader) ([]byte, int, error) {
	target, err := supplierSub2APIURL(provider.BaseURL, path)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("New-Api-User", strconv.FormatInt(session.UserID, 10))
	if session.CookieHeader != "" {
		req.Header.Set("Cookie", session.CookieHeader)
	}
	httpClient := *c.httpClient
	originHost := target.Host
	httpClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if len(via) > 0 && !strings.EqualFold(next.URL.Host, originHost) {
			return fmt.Errorf("supplier newapi redirect to different host rejected")
		}
		return nil
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, err := readSupplierSub2APIResponse(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return raw, resp.StatusCode, nil
}

func parseSupplierNewAPIAccounts(keysPayload, groupsPayload []byte) ([]SupplierProviderRemoteAccount, error) {
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
		return nil, fmt.Errorf("decode supplier newapi accounts response: %w", err)
	}
	if !keysResp.Success {
		return nil, fmt.Errorf("supplier newapi accounts failed: %s", firstSupplierSub2APIString(keysResp.Message, "unknown error"))
	}
	_, byName, err := supplierNewAPIGroupIndexes(groupsPayload)
	if err != nil {
		return nil, err
	}
	out := make([]SupplierProviderRemoteAccount, 0, len(keysResp.Data.Items))
	for _, item := range keysResp.Data.Items {
		name := strings.TrimSpace(item.Name)
		groupName := strings.TrimSpace(item.Group)
		if name == "" {
			continue
		}
		group, ok := byName[groupName]
		if !ok {
			group, ok = byName[normalizeSupplierNewAPIGroupName(groupName)]
		}
		if !ok {
			continue
		}
		status := jsonString(item.Status)
		out = append(out, SupplierProviderRemoteAccount{
			Key:            name,
			Name:           name,
			Status:         status,
			GroupKey:       group.Key,
			GroupName:      groupName,
			RateMultiplier: group.RateMultiplier,
			RawStatus:      status,
		})
	}
	return out, nil
}

func parseSupplierNewAPIGroups(payload []byte) ([]SupplierProviderRemoteGroup, error) {
	groups, _, err := supplierNewAPIGroupIndexes(payload)
	if err != nil {
		return nil, err
	}
	out := make([]SupplierProviderRemoteGroup, 0, len(groups))
	for _, group := range groups {
		out = append(out, SupplierProviderRemoteGroup{
			Key:            group.Key,
			Name:           group.Name,
			RateMultiplier: group.RateMultiplier,
			RawStatus:      group.RawStatus,
		})
	}
	return out, nil
}

func supplierNewAPIGroupIndexes(payload []byte) ([]supplierNewAPIGroupInfo, map[string]supplierNewAPIGroupInfo, error) {
	var groupsResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    map[string]struct {
			Ratio  supplierNewAPIGroupRatio `json:"ratio"`
			ID     any                      `json:"id"`
			Status any                      `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &groupsResp); err != nil {
		return nil, nil, fmt.Errorf("decode supplier newapi groups response: %w", err)
	}
	if !groupsResp.Success {
		return nil, nil, fmt.Errorf("supplier newapi groups failed: %s", firstSupplierSub2APIString(groupsResp.Message, "unknown error"))
	}
	names := make([]string, 0, len(groupsResp.Data))
	for name := range groupsResp.Data {
		names = append(names, name)
	}
	sort.Strings(names)
	groups := make([]supplierNewAPIGroupInfo, 0, len(names))
	byName := make(map[string]supplierNewAPIGroupInfo, len(names)*2)
	for _, name := range names {
		item := groupsResp.Data[name]
		if strings.TrimSpace(name) == "" || !item.Ratio.valid || item.Ratio.value < 0 {
			continue
		}
		key := jsonString(item.ID)
		if key == "" {
			key = normalizeSupplierNewAPIGroupName(name)
		}
		group := supplierNewAPIGroupInfo{
			Key:            key,
			Name:           strings.TrimSpace(name),
			RateMultiplier: item.Ratio.value,
			RawStatus:      jsonString(item.Status),
		}
		groups = append(groups, group)
		byName[group.Name] = group
		byName[normalizeSupplierNewAPIGroupName(group.Name)] = group
	}
	return groups, byName, nil
}

func parseSupplierNewAPINumber(payload []byte, label string) (float64, error) {
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Quota float64 `json:"quota"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return 0, fmt.Errorf("decode supplier newapi %s response: %w", label, err)
	}
	if !resp.Success {
		return 0, fmt.Errorf("supplier newapi %s failed: %s", label, firstSupplierSub2APIString(resp.Message, "unknown error"))
	}
	return resp.Data.Quota / supplierNewAPIQuotaUnit, nil
}

func supplierNewAPIEnvelopeOK(raw []byte) error {
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return err
	}
	if resp.Success {
		return nil
	}
	return fmt.Errorf("supplier newapi request failed: %s", firstSupplierSub2APIString(resp.Message, "unknown error"))
}

func supplierNewAPIAuthFailure(status int, raw []byte, err error) bool {
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return true
	}
	if supplierNewAPIBusinessAuthFailure(raw) {
		return true
	}
	if err != nil {
		return supplierNewAPIAuthPhrase(strings.ToLower(err.Error()))
	}
	return false
}

func supplierNewAPIBusinessAuthFailure(raw []byte) bool {
	if len(raw) == 0 {
		return false
	}
	var resp struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return false
	}
	return supplierNewAPIAuthPhrase(strings.ToLower(resp.Message + " " + resp.Error))
}

func supplierNewAPIAuthPhrase(text string) bool {
	for _, phrase := range []string{"unauthorized", "forbidden", "token expired", "invalid token", "session expired", "auth failed", "未登录", "登录"} {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}

func supplierNewAPIUsageCostURL(path string, day time.Time) string {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	localDay := day.In(loc)
	start := time.Date(localDay.Year(), localDay.Month(), localDay.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour).Add(-time.Second)
	startText := strconv.FormatInt(start.Unix(), 10)
	endText := strconv.FormatInt(end.Unix(), 10)
	out := strings.TrimSpace(path)
	out = strings.ReplaceAll(out, "{start_timestamp}", startText)
	out = strings.ReplaceAll(out, "{end_timestamp}", endText)
	parsed, err := url.Parse(out)
	if err != nil {
		return out
	}
	query := parsed.Query()
	if query.Get("start_timestamp") == "" {
		query.Set("start_timestamp", startText)
	}
	if query.Get("end_timestamp") == "" {
		query.Set("end_timestamp", endText)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func supplierNewAPIGroupsPath(provider *SupplierProvider) string {
	return firstSupplierSub2APIString(provider.AvailableGroupsURL, provider.GroupsURL, defaultSupplierNewAPIGroupsPath)
}

func normalizeSupplierNewAPIGroupName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(name)), " "))
}

func supplierNewAPISessionKey(provider *SupplierProvider) string {
	if provider == nil {
		return ""
	}
	if provider.ID > 0 {
		return strconv.FormatInt(provider.ID, 10)
	}
	return strings.TrimSpace(provider.Code)
}

func supplierNewAPICookiesHeader(cookies []*http.Cookie) string {
	parts := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		if cookie == nil || strings.TrimSpace(cookie.Name) == "" {
			continue
		}
		parts = append(parts, cookie.Name+"="+cookie.Value)
	}
	return strings.Join(parts, "; ")
}

func (c *SupplierNewAPIClient) recordEndpointResult(providerID int64, scope string, result SupplierProviderEndpointResult) {
	c.endpointResultMu.Lock()
	defer c.endpointResultMu.Unlock()
	c.endpointResults[supplierSub2APIEndpointResultKey(providerID, scope)] = result
}

func (c *SupplierNewAPIClient) updateEndpointError(providerID int64, scope string, err error) {
	c.endpointResultMu.Lock()
	defer c.endpointResultMu.Unlock()
	key := supplierSub2APIEndpointResultKey(providerID, scope)
	result := c.endpointResults[key]
	result.Error = supplierSub2APIErrorText(err)
	c.endpointResults[key] = result
}

func (c *SupplierNewAPIClient) annotateEndpointParse(providerID int64, scope string, parsed any, parseErr error) {
	c.endpointResultMu.Lock()
	defer c.endpointResultMu.Unlock()
	key := supplierSub2APIEndpointResultKey(providerID, scope)
	result := c.endpointResults[key]
	if parseErr != nil {
		result.ParseError = parseErr.Error()
	} else if parsed != nil {
		raw, _ := json.Marshal(parsed)
		result.ParsedSummary = string(raw)
	}
	c.endpointResults[key] = result
}

func supplierNewAPIParsedDiagnostic(scope string, raw []byte, provider *SupplierProvider) (any, string) {
	switch scope {
	case SupplierSyncScopeAccounts:
		items, err := parseSupplierNewAPIAccountDiagnostic(raw)
		if err != nil {
			return nil, err.Error()
		}
		return map[string]any{"count": len(items), "items": items}, ""
	case SupplierSyncScopeGroups:
		items, err := parseSupplierNewAPIGroups(raw)
		if err != nil {
			return nil, err.Error()
		}
		return map[string]any{"count": len(items), "items": items}, ""
	case SupplierSyncScopeBalance:
		value, err := parseSupplierNewAPINumber(raw, "balance")
		if err != nil {
			return nil, err.Error()
		}
		return map[string]any{"balance": value}, ""
	case SupplierSyncScopeCost:
		value, err := parseSupplierNewAPINumber(raw, "cost")
		if err != nil {
			return nil, err.Error()
		}
		return map[string]any{"today_actual_cost": value}, ""
	default:
		return nil, "unsupported scope"
	}
}

func parseSupplierNewAPIAccountDiagnostic(payload []byte) ([]map[string]string, error) {
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Items []struct {
				Name  string `json:"name"`
				Group string `json:"group"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return nil, fmt.Errorf("decode supplier newapi accounts response: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("supplier newapi accounts failed: %s", firstSupplierSub2APIString(resp.Message, "unknown error"))
	}
	items := make([]map[string]string, 0, len(resp.Data.Items))
	for _, item := range resp.Data.Items {
		name := strings.TrimSpace(item.Name)
		group := strings.TrimSpace(item.Group)
		if name == "" && group == "" {
			continue
		}
		items = append(items, map[string]string{"name": name, "group": group})
	}
	return items, nil
}

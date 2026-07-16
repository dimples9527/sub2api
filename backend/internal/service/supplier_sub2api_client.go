package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
)

const (
	defaultSupplierSub2APILoginPath               = "/api/v1/auth/login"
	defaultSupplierSub2APIHTTPTimeout             = 30 * time.Second
	supplierSub2APIMaxResponseBytes         int64 = 4 << 20
	supplierSub2APILoginLockTTL                   = 15 * time.Second
	supplierSub2APILoginLockWait                  = 20 * time.Second
	supplierSub2APILoginLockPoll                  = 100 * time.Millisecond
	supplierSub2APILogResponseSummaryLimit        = 512
	supplierSub2APITestResponseSummaryLimit       = 8192
	supplierSub2APILogStringValueLimit            = 160
)

type SupplierProviderRemoteAccount struct {
	Key            string
	Name           string
	Status         string
	GroupKey       string
	GroupName      string
	RateMultiplier float64
	RawStatus      string
}

type SupplierProviderRemoteGroup struct {
	Key            string
	Name           string
	RateMultiplier float64
	RawStatus      string
}

type SupplierProviderRemoteClient interface {
	FetchAccounts(ctx context.Context, provider *SupplierProvider, password string) ([]SupplierProviderRemoteAccount, error)
	FetchGroups(ctx context.Context, provider *SupplierProvider, password string) ([]SupplierProviderRemoteGroup, error)
	FetchBalance(ctx context.Context, provider *SupplierProvider, password string) (float64, error)
	FetchCost(ctx context.Context, provider *SupplierProvider, password string, day time.Time) (float64, error)
}

type SupplierProviderRemoteTester interface {
	TestEndpoint(ctx context.Context, provider *SupplierProvider, password string, scope string) (SupplierProviderEndpointTestResult, error)
}

type SupplierSub2APIClient struct {
	httpClient *http.Client
	tokenCache SupplierProviderTokenCache
}

type supplierSub2APILoginResult struct {
	token SupplierProviderAuthToken
	ttl   time.Duration
}

func NewSupplierSub2APIClient(httpClient *http.Client, tokenCache SupplierProviderTokenCache) *SupplierSub2APIClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultSupplierSub2APIHTTPTimeout}
	}
	return &SupplierSub2APIClient{httpClient: httpClient, tokenCache: tokenCache}
}

func (c *SupplierSub2APIClient) FetchAccounts(ctx context.Context, provider *SupplierProvider, password string) ([]SupplierProviderRemoteAccount, error) {
	var lastErr error
	for _, endpoint := range supplierSub2APIAccountEndpoints(provider.APIKeysURL) {
		raw, err := c.authenticatedGet(ctx, provider, password, endpoint, "accounts")
		if err == nil {
			accounts, parseErr := parseSupplierSub2APIAccounts(raw)
			return accounts, parseErr
		}
		lastErr = err
		if !supplierSub2APIStatusIs(err, http.StatusNotFound) {
			return nil, err
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("supplier sub2api accounts endpoint is not configured")
}

func (c *SupplierSub2APIClient) FetchGroups(ctx context.Context, provider *SupplierProvider, password string) ([]SupplierProviderRemoteGroup, error) {
	groupsPath := strings.TrimSpace(provider.GroupsURL)
	if groupsPath == "" {
		groupsPath = strings.TrimSpace(provider.AvailableGroupsURL)
	}
	raw, err := c.authenticatedGet(ctx, provider, password, groupsPath, "groups")
	if err != nil {
		return nil, err
	}
	groups, parseErr := parseSupplierSub2APIGroups(raw)
	return groups, parseErr
}

func (c *SupplierSub2APIClient) FetchBalance(ctx context.Context, provider *SupplierProvider, password string) (float64, error) {
	raw, err := c.authenticatedGet(ctx, provider, password, provider.BalanceURL, "balance")
	if err != nil {
		return 0, err
	}
	balance, parseErr := parseSupplierSub2APINumberField(raw, "balance")
	return balance, parseErr
}

func (c *SupplierSub2APIClient) FetchCost(ctx context.Context, provider *SupplierProvider, password string, _ time.Time) (float64, error) {
	raw, err := c.authenticatedGet(ctx, provider, password, provider.UsageCostURL, "cost")
	if err != nil {
		return 0, err
	}
	cost, parseErr := parseSupplierSub2APINumberField(raw, "today_actual_cost")
	return cost, parseErr
}

func (c *SupplierSub2APIClient) TestEndpoint(ctx context.Context, provider *SupplierProvider, password string, scope string) (SupplierProviderEndpointTestResult, error) {
	scope = strings.TrimSpace(scope)
	result := SupplierProviderEndpointTestResult{
		ProviderID: provider.ID,
		Scope:      scope,
		Attempts:   []SupplierProviderEndpointTestAttempt{},
	}
	var endpoints []string
	switch scope {
	case SupplierSyncScopeAccounts:
		endpoints = supplierSub2APIAccountEndpoints(provider.APIKeysURL)
	case SupplierSyncScopeGroups:
		groupsPath := strings.TrimSpace(provider.GroupsURL)
		if groupsPath == "" {
			groupsPath = strings.TrimSpace(provider.AvailableGroupsURL)
		}
		endpoints = []string{groupsPath}
	case SupplierSyncScopeBalance:
		endpoints = []string{provider.BalanceURL}
	case SupplierSyncScopeCost:
		endpoints = []string{provider.UsageCostURL}
	default:
		return SupplierProviderEndpointTestResult{}, fmt.Errorf("unsupported supplier endpoint test scope: %s", scope)
	}
	token, err := c.ensureToken(ctx, provider, password)
	if err != nil {
		return SupplierProviderEndpointTestResult{}, err
	}
	for _, endpoint := range endpoints {
		startedAt := time.Now()
		raw, status, err := c.doJSON(ctx, http.MethodGet, provider, endpoint, scope, token, nil)
		attempt := SupplierProviderEndpointTestAttempt{
			Endpoint:        endpoint,
			HTTPStatus:      status,
			DurationMS:      time.Since(startedAt).Milliseconds(),
			ResponseBytes:   len(raw),
			ResponseSummary: supplierSub2APISafeResponseText(raw, supplierSub2APITestResponseSummaryLimit),
			Error:           supplierSub2APIErrorText(err),
		}
		if err == nil {
			attempt.ParsedData, attempt.ParseError = supplierSub2APIParsedDiagnostic(scope, raw)
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
		if err != nil || status < 200 || status >= 300 {
			if scope == SupplierSyncScopeAccounts && supplierSub2APIStatusCodeIs(status, http.StatusNotFound) {
				continue
			}
			break
		}
		break
	}
	return result, nil
}

func (c *SupplierSub2APIClient) authenticatedGet(ctx context.Context, provider *SupplierProvider, password, path, label string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		token, err := c.ensureToken(ctx, provider, password)
		if err != nil {
			return nil, err
		}
		raw, status, err := c.doJSON(ctx, http.MethodGet, provider, path, label, token, nil)
		if err != nil {
			return nil, fmt.Errorf("supplier sub2api %s request failed: %w", label, err)
		}
		if status >= 200 && status < 300 && !supplierSub2APIBusinessAuthFailure(raw) {
			if err := supplierSub2APIEnvelopeOK(raw); err != nil {
				if attempt == 0 && supplierSub2APIErrorLooksAuth(err) {
					c.deleteToken(ctx, provider)
					lastErr = err
					continue
				}
				return nil, err
			}
			return raw, nil
		}
		err = supplierSub2APIHTTPError(label, status, raw)
		if attempt == 0 && supplierSub2APIAuthFailure(status, raw, err) {
			c.deleteToken(ctx, provider)
			lastErr = err
			continue
		}
		c.deleteToken(ctx, provider)
		return nil, err
	}
	return nil, fmt.Errorf("supplier sub2api %s failed after auth retry: %w", label, lastErr)
}

func (c *SupplierSub2APIClient) ensureToken(ctx context.Context, provider *SupplierProvider, password string) (SupplierProviderAuthToken, error) {
	if c.tokenCache == nil {
		result, err := c.login(ctx, provider, password)
		return result.token, err
	}
	if token, found, err := c.tokenCache.Get(ctx, provider.ID); err != nil {
		c.logCacheError(provider, "get", err)
		result, loginErr := c.login(ctx, provider, password)
		return result.token, loginErr
	} else if found && strings.TrimSpace(token.AccessToken) != "" {
		return normalizeSupplierSub2APIToken(token), nil
	}

	owner := uuid.NewString()
	acquired, err := c.tokenCache.TryAcquireLoginLock(ctx, provider.ID, owner, supplierSub2APILoginLockTTL)
	if err != nil {
		c.logCacheError(provider, "acquire login lock", err)
		result, loginErr := c.login(ctx, provider, password)
		return result.token, loginErr
	}
	if acquired {
		return c.loginAndCache(ctx, provider, password, owner)
	}

	waitCtx, cancel := context.WithTimeout(ctx, supplierSub2APILoginLockWait)
	defer cancel()
	ticker := time.NewTicker(supplierSub2APILoginLockPoll)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			return SupplierProviderAuthToken{}, waitCtx.Err()
		case <-ticker.C:
			if token, found, err := c.tokenCache.Get(waitCtx, provider.ID); err != nil {
				c.logCacheError(provider, "poll token", err)
				result, loginErr := c.login(ctx, provider, password)
				return result.token, loginErr
			} else if found && strings.TrimSpace(token.AccessToken) != "" {
				return normalizeSupplierSub2APIToken(token), nil
			}
			acquired, err := c.tokenCache.TryAcquireLoginLock(waitCtx, provider.ID, owner, supplierSub2APILoginLockTTL)
			if err != nil {
				c.logCacheError(provider, "retry login lock", err)
				result, loginErr := c.login(ctx, provider, password)
				return result.token, loginErr
			}
			if acquired {
				return c.loginAndCache(ctx, provider, password, owner)
			}
		}
	}
}

func (c *SupplierSub2APIClient) loginAndCache(ctx context.Context, provider *SupplierProvider, password, owner string) (SupplierProviderAuthToken, error) {
	defer func() {
		if err := c.tokenCache.ReleaseLoginLock(context.Background(), provider.ID, owner); err != nil {
			c.logCacheError(provider, "release login lock", err)
		}
	}()
	result, err := c.login(ctx, provider, password)
	if err != nil {
		return SupplierProviderAuthToken{}, err
	}
	if err := c.tokenCache.Set(ctx, provider.ID, result.token, result.ttl); err != nil {
		c.logCacheError(provider, "set", err)
	}
	return result.token, nil
}

func (c *SupplierSub2APIClient) login(ctx context.Context, provider *SupplierProvider, password string) (supplierSub2APILoginResult, error) {
	loginPath := strings.TrimSpace(provider.LoginURL)
	if loginPath == "" {
		loginPath = defaultSupplierSub2APILoginPath
	}
	body, err := json.Marshal(map[string]string{
		"email":    strings.TrimSpace(provider.Email),
		"password": password,
	})
	if err != nil {
		return supplierSub2APILoginResult{}, fmt.Errorf("marshal supplier sub2api login: %w", err)
	}
	raw, status, err := c.doJSON(ctx, http.MethodPost, provider, loginPath, "login", SupplierProviderAuthToken{}, bytes.NewReader(body))
	if err != nil {
		return supplierSub2APILoginResult{}, fmt.Errorf("supplier sub2api login request failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return supplierSub2APILoginResult{}, supplierSub2APIHTTPError("login", status, raw)
	}
	token, expiresIn, err := parseSupplierSub2APILogin(raw)
	if err != nil {
		return supplierSub2APILoginResult{}, err
	}
	if token.TokenType == "" {
		token.TokenType = "Bearer"
	}
	ttl := SupplierProviderTokenTTL(expiresIn)
	token.ExpiresAt = time.Now().Add(ttl)
	return supplierSub2APILoginResult{token: normalizeSupplierSub2APIToken(token), ttl: ttl}, nil
}

func (c *SupplierSub2APIClient) doJSON(ctx context.Context, method string, provider *SupplierProvider, path string, label string, token SupplierProviderAuthToken, body io.Reader) ([]byte, int, error) {
	target, err := supplierSub2APIURL(provider.BaseURL, path)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	token = normalizeSupplierSub2APIToken(token)
	if token.AccessToken != "" {
		req.Header.Set("Authorization", token.TokenType+" "+token.AccessToken)
	}

	httpClient := *c.httpClient
	originHost := target.Host
	httpClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if len(via) > 0 && !strings.EqualFold(next.URL.Host, originHost) {
			return fmt.Errorf("supplier sub2api redirect to different host rejected")
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
	_ = label
	return raw, resp.StatusCode, nil
}

func readSupplierSub2APIResponse(reader io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, supplierSub2APIMaxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > supplierSub2APIMaxResponseBytes {
		return nil, fmt.Errorf("supplier sub2api response exceeds 4 MiB")
	}
	return raw, nil
}

func supplierSub2APIURL(baseURL, endpoint string) (*url.URL, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, fmt.Errorf("parse supplier sub2api base url: %w", err)
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return nil, fmt.Errorf("supplier sub2api URL scheme must be http or https")
	}
	if base.Host == "" {
		return nil, fmt.Errorf("supplier sub2api base url missing host")
	}
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = "/"
	}
	rel, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse supplier sub2api endpoint: %w", err)
	}
	if rel.IsAbs() {
		if rel.Scheme != "http" && rel.Scheme != "https" {
			return nil, fmt.Errorf("supplier sub2api URL scheme must be http or https")
		}
		return rel, nil
	}
	if strings.HasPrefix(endpoint, "/") {
		return base.ResolveReference(rel), nil
	}
	basePath := base.EscapedPath()
	if basePath == "" || !strings.HasSuffix(basePath, "/") {
		base.Path = strings.TrimSuffix(base.Path, "/") + "/"
	}
	return base.ResolveReference(rel), nil
}

func parseSupplierSub2APILogin(raw []byte) (SupplierProviderAuthToken, time.Duration, error) {
	var resp struct {
		Code        any    `json:"code"`
		Message     string `json:"message"`
		AccessToken string `json:"access_token"`
		Token       string `json:"token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   any    `json:"expires_in"`
		Data        struct {
			AccessToken string `json:"access_token"`
			Token       string `json:"token"`
			TokenType   string `json:"token_type"`
			ExpiresIn   any    `json:"expires_in"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return SupplierProviderAuthToken{}, 0, fmt.Errorf("decode supplier sub2api login response: %w", err)
	}
	if !supplierSub2APICodeOK(resp.Code) {
		return SupplierProviderAuthToken{}, 0, fmt.Errorf("supplier sub2api login failed: %s", firstSupplierSub2APIString(resp.Message, "unknown error"))
	}
	token := firstSupplierSub2APIString(resp.Data.AccessToken, resp.Data.Token, resp.AccessToken, resp.Token)
	if token == "" {
		return SupplierProviderAuthToken{}, 0, fmt.Errorf("supplier sub2api login failed: missing token")
	}
	expiresIn := supplierSub2APIDurationSeconds(resp.Data.ExpiresIn)
	if expiresIn <= 0 {
		expiresIn = supplierSub2APIDurationSeconds(resp.ExpiresIn)
	}
	return SupplierProviderAuthToken{
		AccessToken: token,
		TokenType:   firstSupplierSub2APIString(resp.Data.TokenType, resp.TokenType, "Bearer"),
	}, expiresIn, nil
}

func parseSupplierSub2APIAccounts(raw []byte) ([]SupplierProviderRemoteAccount, error) {
	items, err := supplierSub2APIItems(raw)
	if err != nil {
		return nil, err
	}
	out := make([]SupplierProviderRemoteAccount, 0, len(items))
	for _, item := range items {
		key := firstSupplierSub2APIString(jsonString(item["id"]), jsonString(item["key"]), jsonString(item["account_key"]), jsonString(item["api_key_id"]))
		name := strings.TrimSpace(jsonString(item["name"]))
		if key == "" && name != "" {
			key = normalizeSupplierSub2APIKeyFromName(name)
		}
		if key == "" && name == "" {
			continue
		}
		groupKey := firstSupplierSub2APIString(jsonString(item["group_key"]), jsonString(item["group_id"]))
		groupName := jsonString(item["group_name"])
		rate := jsonFloat(item["rate_multiplier"])
		if group, ok := item["group"].(map[string]any); ok {
			groupKey = firstSupplierSub2APIString(groupKey, jsonString(group["id"]), jsonString(group["key"]))
			groupName = firstSupplierSub2APIString(groupName, jsonString(group["name"]))
			if groupRate := jsonFloat(group["rate_multiplier"]); groupRate != 0 {
				rate = groupRate
			}
		}
		status := jsonString(item["status"])
		out = append(out, SupplierProviderRemoteAccount{
			Key:            strings.TrimSpace(key),
			Name:           name,
			Status:         strings.TrimSpace(status),
			GroupKey:       strings.TrimSpace(groupKey),
			GroupName:      strings.TrimSpace(groupName),
			RateMultiplier: rate,
			RawStatus:      strings.TrimSpace(status),
		})
	}
	return out, nil
}

func parseSupplierSub2APIGroups(raw []byte) ([]SupplierProviderRemoteGroup, error) {
	items, err := supplierSub2APIItems(raw)
	if err != nil {
		return nil, err
	}
	out := make([]SupplierProviderRemoteGroup, 0, len(items))
	for _, item := range items {
		key := firstSupplierSub2APIString(jsonString(item["id"]), jsonString(item["key"]), jsonString(item["group_key"]))
		name := strings.TrimSpace(jsonString(item["name"]))
		if key == "" && name == "" {
			continue
		}
		status := jsonString(item["status"])
		out = append(out, SupplierProviderRemoteGroup{
			Key:            strings.TrimSpace(key),
			Name:           name,
			RateMultiplier: jsonFloat(item["rate_multiplier"]),
			RawStatus:      strings.TrimSpace(status),
		})
	}
	return out, nil
}

func parseSupplierSub2APINumberField(raw []byte, field string) (float64, error) {
	var resp struct {
		Code    any            `json:"code"`
		Message string         `json:"message"`
		Data    map[string]any `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return 0, fmt.Errorf("decode supplier sub2api %s response: %w", field, err)
	}
	if !supplierSub2APICodeOK(resp.Code) {
		return 0, fmt.Errorf("supplier sub2api request failed: %s", firstSupplierSub2APIString(resp.Message, "unknown error"))
	}
	if resp.Data == nil {
		return 0, fmt.Errorf("supplier sub2api response missing data.%s", field)
	}
	return jsonFloat(resp.Data[field]), nil
}

func supplierSub2APIItems(raw []byte) ([]map[string]any, error) {
	var resp struct {
		Code    any             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("decode supplier sub2api response: %w", err)
	}
	if !supplierSub2APICodeOK(resp.Code) {
		return nil, fmt.Errorf("supplier sub2api request failed: %s", firstSupplierSub2APIString(resp.Message, "unknown error"))
	}
	var direct []map[string]any
	if err := json.Unmarshal(resp.Data, &direct); err == nil {
		return direct, nil
	}
	var wrapped struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(resp.Data, &wrapped); err == nil && wrapped.Items != nil {
		return wrapped.Items, nil
	}
	return nil, fmt.Errorf("supplier sub2api response must contain data array or data.items")
}

func supplierSub2APIEnvelopeOK(raw []byte) error {
	var resp struct {
		Code    any    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return err
	}
	if supplierSub2APICodeOK(resp.Code) {
		return nil
	}
	return fmt.Errorf("supplier sub2api request failed: %s", firstSupplierSub2APIString(resp.Message, fmt.Sprint(resp.Code)))
}

func supplierSub2APICodeOK(code any) bool {
	switch value := code.(type) {
	case nil:
		return true
	case float64:
		return value == 0
	case string:
		value = strings.TrimSpace(value)
		return value == "" || value == "0" || strings.EqualFold(value, "success") || strings.EqualFold(value, "ok")
	default:
		return fmt.Sprint(value) == "0"
	}
}

func supplierSub2APIAuthFailure(status int, raw []byte, err error) bool {
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return true
	}
	return supplierSub2APIBusinessAuthFailure(raw) || supplierSub2APIErrorLooksAuth(err)
}

func supplierSub2APIBusinessAuthFailure(raw []byte) bool {
	if len(raw) == 0 {
		return false
	}
	var resp struct {
		Code    any    `json:"code"`
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return false
	}
	text := strings.ToLower(strings.Join([]string{fmt.Sprint(resp.Code), resp.Message, resp.Error}, " "))
	return supplierSub2APIAuthPhrase(text)
}

func supplierSub2APIErrorLooksAuth(err error) bool {
	if err == nil {
		return false
	}
	return supplierSub2APIAuthPhrase(strings.ToLower(err.Error()))
}

func supplierSub2APIAuthPhrase(text string) bool {
	for _, phrase := range []string{"unauthorized", "forbidden", "token expired", "invalid token", "session expired", "auth failed"} {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}

func supplierSub2APIHTTPError(label string, status int, raw []byte) error {
	if status == 0 {
		return errors.New("supplier sub2api request failed")
	}
	msg := strings.TrimSpace(string(raw))
	if len(msg) > 256 {
		msg = msg[:256]
	}
	return supplierSub2APIHTTPStatusError{label: label, status: status, message: msg}
}

type supplierSub2APIHTTPStatusError struct {
	label   string
	status  int
	message string
}

func (e supplierSub2APIHTTPStatusError) Error() string {
	return fmt.Sprintf("supplier sub2api %s failed with HTTP %d: %s", e.label, e.status, e.message)
}

func supplierSub2APIStatusIs(err error, status int) bool {
	var statusErr supplierSub2APIHTTPStatusError
	return errors.As(err, &statusErr) && statusErr.status == status
}

func supplierSub2APIStatusCodeIs(actual int, expected int) bool {
	return actual == expected
}

func supplierSub2APIAccountEndpoints(configured string) []string {
	candidates := []string{
		strings.TrimSpace(configured),
		"/api/v1/user/keys",
		"/api/admin/keys",
	}
	out := make([]string, 0, len(candidates))
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		out = append(out, candidate)
	}
	return out
}

func normalizeSupplierSub2APIToken(token SupplierProviderAuthToken) SupplierProviderAuthToken {
	token.AccessToken = strings.TrimSpace(token.AccessToken)
	token.TokenType = strings.TrimSpace(token.TokenType)
	if token.TokenType == "" {
		token.TokenType = "Bearer"
	}
	return token
}

func normalizeSupplierSub2APIKeyFromName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(name)), " "))
}

func firstSupplierSub2APIString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func jsonString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case json.Number:
		return strings.TrimSpace(v.String())
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func jsonFloat(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case json.Number:
		parsed, _ := strconv.ParseFloat(v.String(), 64)
		return parsed
	case string:
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return parsed
	default:
		return 0
	}
}

func supplierSub2APIDurationSeconds(value any) time.Duration {
	seconds := jsonFloat(value)
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func (c *SupplierSub2APIClient) deleteToken(ctx context.Context, provider *SupplierProvider) {
	if c.tokenCache == nil {
		return
	}
	if err := c.tokenCache.Delete(ctx, provider.ID); err != nil {
		c.logCacheError(provider, "delete", err)
	}
}

func (c *SupplierSub2APIClient) logCacheError(provider *SupplierProvider, action string, err error) {
	if err == nil || provider == nil {
		return
	}
	logger.LegacyPrintf("supplier_sub2api_client", "supplier provider cache %s failed provider_id=%d provider_code=%s err=%v", action, provider.ID, provider.Code, err)
}

func supplierSub2APIErrorText(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}

func supplierSub2APISafeResponseSummary(raw []byte) string {
	return supplierSub2APISafeResponseText(raw, supplierSub2APILogResponseSummaryLimit)
}

func supplierSub2APISafeResponseText(raw []byte, limit int) string {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return ""
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err == nil {
		redacted := supplierSub2APIRedactSensitiveJSON(decoded)
		if encoded, err := json.Marshal(redacted); err == nil {
			text = string(encoded)
		}
	}
	return supplierSub2APITruncateLogText(text, limit)
}

func supplierSub2APIParsedDiagnostic(scope string, raw []byte) (any, string) {
	switch scope {
	case SupplierSyncScopeAccounts:
		items, err := parseSupplierSub2APIAccounts(raw)
		if err != nil {
			return nil, err.Error()
		}
		return map[string]any{"count": len(items), "items": items}, ""
	case SupplierSyncScopeGroups:
		items, err := parseSupplierSub2APIGroups(raw)
		if err != nil {
			return nil, err.Error()
		}
		return map[string]any{"count": len(items), "items": items}, ""
	case SupplierSyncScopeBalance:
		value, err := parseSupplierSub2APINumberField(raw, "balance")
		if err != nil {
			return nil, err.Error()
		}
		return map[string]any{"balance": value}, ""
	case SupplierSyncScopeCost:
		value, err := parseSupplierSub2APINumberField(raw, "today_actual_cost")
		if err != nil {
			return nil, err.Error()
		}
		return map[string]any{"today_actual_cost": value}, ""
	default:
		return nil, "unsupported scope"
	}
}

func supplierSub2APIRedactSensitiveJSON(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if supplierSub2APISensitiveLogKey(key) {
				out[key] = "[redacted]"
				continue
			}
			out[key] = supplierSub2APIRedactSensitiveJSON(item)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for idx, item := range typed {
			out[idx] = supplierSub2APIRedactSensitiveJSON(item)
		}
		return out
	case string:
		return supplierSub2APITruncateLogText(typed, supplierSub2APILogStringValueLimit)
	default:
		return value
	}
}

func supplierSub2APISensitiveLogKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	for _, marker := range []string{"token", "password", "secret", "authorization", "cookie"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func supplierSub2APITruncateLogText(text string, limit int) string {
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}

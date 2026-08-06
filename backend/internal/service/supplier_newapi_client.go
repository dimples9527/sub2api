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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
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
	httpClient      *http.Client
	tokenCache      SupplierProviderTokenCache
	authAuditor     SupplierProviderAuthAuditor
	turnstileSolver SupplierTurnstileSolver

	sessionMu sync.Mutex
	sessions  map[string]supplierNewAPISession

	endpointResultMu sync.Mutex
	endpointResults  map[string]SupplierProviderEndpointResult
}

func (c *SupplierNewAPIClient) SetAuthAuditor(auditor SupplierProviderAuthAuditor) {
	c.authAuditor = auditor
}

type supplierNewAPISession struct {
	UserID       int64
	AccessToken  string
	CookieHeader string
	ExpiresAt    time.Time
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

func NewSupplierNewAPIClient(httpClient *http.Client, tokenCache SupplierProviderTokenCache, turnstileSolver SupplierTurnstileSolver) *SupplierNewAPIClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultSupplierSub2APIHTTPTimeout}
	}
	if turnstileSolver == nil {
		turnstileSolver = noopSupplierTurnstileSolver{}
	}
	return &SupplierNewAPIClient{
		httpClient:      httpClient,
		tokenCache:      tokenCache,
		turnstileSolver: turnstileSolver,
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
				c.clearSession(ctx, provider)
				lastErr = err
				continue
			}
			return nil, err
		}
		groupsRaw, groupStatus, err := c.authenticatedGet(ctx, provider, session, groupsPath, "groups")
		if err != nil {
			if attempt == 0 && supplierNewAPIAuthFailure(groupStatus, groupsRaw, err) {
				c.clearSession(ctx, provider)
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
				c.clearSession(ctx, provider)
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
			c.clearSession(ctx, provider)
			lastErr = err
			continue
		}
		if supplierNewAPIAuthFailure(status, raw, err) {
			c.clearSession(ctx, provider)
			return nil, wrapSupplierProviderAuthFailure(fmt.Errorf("supplier newapi %s failed after auth retry: %w", label, err))
		}
		return nil, err
	}
	if supplierNewAPIAuthFailure(0, nil, lastErr) {
		return nil, wrapSupplierProviderAuthFailure(fmt.Errorf("supplier newapi %s failed after auth retry: %w", label, lastErr))
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

func (c *SupplierNewAPIClient) ensureSession(ctx context.Context, provider *SupplierProvider, password string) (result supplierNewAPISession, err error) {
	SupplierSyncProgress(ctx, SupplierSyncProgressStageSession, "正在检查上游登录会话", nil)
	defer func() {
		if err != nil {
			err = wrapSupplierProviderSessionFailure(err)
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStageSession, err)
			return
		}
		SupplierSyncProgressOK(ctx, SupplierSyncProgressStageSession, "上游登录会话已准备")
	}()
	if session, ok := c.cachedSession(provider); ok {
		token := supplierNewAPISessionToken(session)
		c.recordAuthEvent(ctx, provider, SupplierProviderAuthEventInput{EventType: SupplierProviderAuthEventCacheHit, Token: &token})
		return session, nil
	}
	if c.tokenCache == nil {
		c.recordAuthEvent(ctx, provider, SupplierProviderAuthEventInput{
			EventType: SupplierProviderAuthEventCacheError,
			Error:     errors.New("supplier provider token cache is unavailable"),
		})
		return c.loginAndStore(ctx, provider, password)
	}

	if token, found, err := c.tokenCache.Get(ctx, provider.ID); err != nil {
		return supplierNewAPISession{}, c.cacheFailure(ctx, provider, "get", err)
	} else if found {
		if session, ok := supplierNewAPISessionFromToken(token); ok {
			c.recordAuthEvent(ctx, provider, SupplierProviderAuthEventInput{EventType: SupplierProviderAuthEventCacheHit, Token: &token})
			c.storeSession(provider, session)
			return session, nil
		}
	}
	c.recordAuthEvent(ctx, provider, SupplierProviderAuthEventInput{EventType: SupplierProviderAuthEventCacheMiss})

	owner := uuid.NewString()
	acquired, err := c.tokenCache.TryAcquireLoginLock(ctx, provider.ID, owner, supplierSub2APILoginLockTTL)
	if err != nil {
		return supplierNewAPISession{}, c.cacheFailure(ctx, provider, "acquire login lock", err)
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
			return supplierNewAPISession{}, waitCtx.Err()
		case <-ticker.C:
			if token, found, err := c.tokenCache.Get(waitCtx, provider.ID); err != nil {
				return supplierNewAPISession{}, c.cacheFailure(ctx, provider, "poll token", err)
			} else if found {
				if session, ok := supplierNewAPISessionFromToken(token); ok {
					c.recordAuthEvent(ctx, provider, SupplierProviderAuthEventInput{EventType: SupplierProviderAuthEventCacheHit, Token: &token})
					c.storeSession(provider, session)
					return session, nil
				}
			}
			acquired, err := c.tokenCache.TryAcquireLoginLock(waitCtx, provider.ID, owner, supplierSub2APILoginLockTTL)
			if err != nil {
				return supplierNewAPISession{}, c.cacheFailure(ctx, provider, "retry login lock", err)
			}
			if acquired {
				return c.loginAndCache(ctx, provider, password, owner)
			}
		}
	}
}

func (c *SupplierNewAPIClient) cachedSession(provider *SupplierProvider) (supplierNewAPISession, bool) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	session, ok := c.sessions[supplierNewAPISessionKey(provider)]
	if !ok || !supplierNewAPISessionUsable(session) {
		return supplierNewAPISession{}, false
	}
	return session, true
}

func (c *SupplierNewAPIClient) storeSession(provider *SupplierProvider, session supplierNewAPISession) {
	if !supplierNewAPISessionUsable(session) {
		return
	}
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	c.sessions[supplierNewAPISessionKey(provider)] = session
}

func (c *SupplierNewAPIClient) clearSession(ctx context.Context, provider *SupplierProvider) {
	if provider == nil {
		return
	}
	c.sessionMu.Lock()
	delete(c.sessions, supplierNewAPISessionKey(provider))
	c.sessionMu.Unlock()
	if c.tokenCache == nil {
		return
	}
	if err := c.tokenCache.Delete(ctx, provider.ID); err != nil {
		_ = c.cacheFailure(ctx, provider, "delete", err)
		return
	}
	c.recordAuthEvent(ctx, provider, SupplierProviderAuthEventInput{EventType: SupplierProviderAuthEventCacheInvalidated})
}

func (c *SupplierNewAPIClient) loginAndStore(ctx context.Context, provider *SupplierProvider, password string) (supplierNewAPISession, error) {
	session, err := c.loginWithAudit(ctx, provider, password)
	if err != nil {
		return supplierNewAPISession{}, err
	}
	c.storeSession(provider, session)
	return session, nil
}

func (c *SupplierNewAPIClient) loginAndCache(ctx context.Context, provider *SupplierProvider, password, owner string) (supplierNewAPISession, error) {
	defer func() {
		if err := c.tokenCache.ReleaseLoginLock(context.Background(), provider.ID, owner); err != nil {
			c.logCacheError(provider, "release login lock", err)
		}
	}()

	if token, found, err := c.tokenCache.Get(ctx, provider.ID); err != nil {
		return supplierNewAPISession{}, c.cacheFailure(ctx, provider, "recheck token", err)
	} else if found {
		if session, ok := supplierNewAPISessionFromToken(token); ok {
			c.recordAuthEvent(ctx, provider, SupplierProviderAuthEventInput{EventType: SupplierProviderAuthEventCacheHit, Token: &token})
			c.storeSession(provider, session)
			return session, nil
		}
	}

	session, err := c.loginWithAudit(ctx, provider, password)
	if err != nil {
		return supplierNewAPISession{}, err
	}
	if err := c.tokenCache.Set(ctx, provider.ID, supplierNewAPISessionToken(session), supplierNewAPISessionTTL(session)); err != nil {
		return supplierNewAPISession{}, c.cacheFailure(ctx, provider, "set", err)
	}
	c.storeSession(provider, session)
	return session, nil
}

func (c *SupplierNewAPIClient) loginWithAudit(ctx context.Context, provider *SupplierProvider, password string) (supplierNewAPISession, error) {
	SupplierSyncProgress(ctx, SupplierSyncProgressStageLogin, "正在登录上游", nil)
	startedAt := time.Now()
	session, err := c.login(ctx, provider, password)
	if err != nil {
		SupplierSyncProgressFail(ctx, SupplierSyncProgressStageLogin, err)
	} else {
		SupplierSyncProgressOK(ctx, SupplierSyncProgressStageLogin, "上游登录成功")
	}
	event := SupplierProviderAuthEventInput{
		EventType:  SupplierProviderAuthEventLoginSuccess,
		StartedAt:  startedAt,
		FinishedAt: time.Now(),
		HTTPStatus: supplierProviderAuthHTTPStatus(err),
		Error:      err,
	}
	if err != nil {
		event.EventType = SupplierProviderAuthEventLoginFailed
	} else {
		token := supplierNewAPISessionToken(session)
		event.Token = &token
	}
	c.recordAuthEvent(ctx, provider, event)
	return session, err
}

func (c *SupplierNewAPIClient) cacheFailure(ctx context.Context, provider *SupplierProvider, action string, err error) error {
	c.logCacheError(provider, action, err)
	wrapped := fmt.Errorf("supplier provider token cache %s failed: %w", action, err)
	c.recordAuthEvent(ctx, provider, SupplierProviderAuthEventInput{EventType: SupplierProviderAuthEventCacheError, Error: wrapped})
	return wrapped
}

func (c *SupplierNewAPIClient) recordAuthEvent(ctx context.Context, provider *SupplierProvider, event SupplierProviderAuthEventInput) {
	if c == nil || c.authAuditor == nil || provider == nil {
		return
	}
	event.ProviderID = provider.ID
	if event.Source == "" {
		event.Source = supplierProviderAuthSourceFromContext(ctx)
	}
	if err := c.authAuditor.Record(ctx, event); err != nil {
		logger.LegacyPrintf("supplier_newapi_client", "record supplier provider auth event failed provider_id=%d provider_code=%s err=%v", provider.ID, provider.Code, err)
	}
}

func (c *SupplierNewAPIClient) logCacheError(provider *SupplierProvider, action string, err error) {
	if err == nil || provider == nil {
		return
	}
	logger.LegacyPrintf("supplier_newapi_client", "supplier provider cache %s failed provider_id=%d provider_code=%s err=%v", action, provider.ID, provider.Code, err)
}

func supplierNewAPISessionFromToken(token SupplierProviderAuthToken) (supplierNewAPISession, bool) {
	session := supplierNewAPISession{
		UserID:       token.UserID,
		AccessToken:  strings.TrimSpace(token.AccessToken),
		CookieHeader: strings.TrimSpace(token.CookieHeader),
		ExpiresAt:    token.ExpiresAt,
	}
	if !supplierNewAPISessionUsable(session) {
		return supplierNewAPISession{}, false
	}
	return session, true
}

func supplierNewAPISessionToken(session supplierNewAPISession) SupplierProviderAuthToken {
	return SupplierProviderAuthToken{
		AccessToken:  strings.TrimSpace(session.AccessToken),
		TokenType:    "Bearer",
		ExpiresAt:    session.ExpiresAt,
		UserID:       session.UserID,
		CookieHeader: strings.TrimSpace(session.CookieHeader),
	}
}

func supplierNewAPISessionTTL(session supplierNewAPISession) time.Duration {
	if !session.ExpiresAt.IsZero() {
		if ttl := time.Until(session.ExpiresAt); ttl > 0 {
			return ttl
		}
	}
	return SupplierProviderTokenTTL(0)
}

func supplierNewAPISessionUsable(session supplierNewAPISession) bool {
	if session.UserID <= 0 || !supplierNewAPIHasSessionAuth(session) {
		return false
	}
	return !session.ExpiresAt.IsZero() && time.Now().Before(session.ExpiresAt)
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
	if c.turnstileSolver != nil {
		token, err := c.turnstileSolver.PrepareToken(ctx, provider, strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/"), func(fetchCtx context.Context) (string, error) {
			return fetchSupplierNewAPITurnstileSiteKey(fetchCtx, c.httpClient, provider)
		})
		if err != nil {
			return supplierNewAPISession{}, wrapSupplierProviderAuthFailure(err)
		}
		if token != "" {
			loginPath = supplierNewAPIAppendQuery(loginPath, "turnstile", token)
		}
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
		loginErr := supplierSub2APIHTTPError("newapi login", status, raw)
		if supplierNewAPILoginAuthFailure(status, raw, loginErr) {
			return supplierNewAPISession{}, wrapSupplierProviderAuthFailure(loginErr)
		}
		return supplierNewAPISession{}, loginErr
	}
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			ID              int64  `json:"id"`
			AccessToken     string `json:"access_token"`
			AccessExpiresAt any    `json:"access_expires_at"`
			User            struct {
				ID int64 `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return supplierNewAPISession{}, fmt.Errorf("decode supplier newapi login response: %w", err)
	}
	if !resp.Success {
		loginErr := fmt.Errorf("supplier newapi login failed: %s", firstSupplierSub2APIString(resp.Message, "unknown error"))
		if supplierNewAPILoginAuthFailure(status, raw, loginErr) {
			return supplierNewAPISession{}, wrapSupplierProviderAuthFailure(loginErr)
		}
		return supplierNewAPISession{}, loginErr
	}
	expiresAt := supplierNewAPISessionExpiresAt(resp.Data.AccessExpiresAt)
	if accessToken := strings.TrimSpace(resp.Data.AccessToken); accessToken != "" {
		if resp.Data.User.ID <= 0 {
			return supplierNewAPISession{}, fmt.Errorf("supplier newapi login failed: missing user id")
		}
		return supplierNewAPISession{
			UserID:       resp.Data.User.ID,
			AccessToken:  accessToken,
			CookieHeader: supplierNewAPICookiesHeader(cookies),
			ExpiresAt:    expiresAt,
		}, nil
	}
	userID := resp.Data.User.ID
	if userID <= 0 {
		userID = resp.Data.ID
	}
	if userID <= 0 {
		return supplierNewAPISession{}, fmt.Errorf("supplier newapi login failed: missing user id")
	}
	cookieHeader := supplierNewAPICookiesHeader(cookies)
	if cookieHeader == "" {
		return supplierNewAPISession{}, fmt.Errorf("supplier newapi login failed: missing cookie")
	}
	return supplierNewAPISession{UserID: userID, CookieHeader: cookieHeader, ExpiresAt: expiresAt}, nil
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
	if accessToken := strings.TrimSpace(session.AccessToken); accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	} else if cookieHeader := strings.TrimSpace(session.CookieHeader); cookieHeader != "" {
		req.Header.Set("Cookie", cookieHeader)
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
		rawStatus := jsonString(item.Status)
		status := normalizeSupplierNewAPIKeyStatus(item.Status)
		out = append(out, SupplierProviderRemoteAccount{
			Key:            name,
			Name:           name,
			Status:         status,
			GroupKey:       group.Key,
			GroupName:      groupName,
			RateMultiplier: group.RateMultiplier,
			RawStatus:      rawStatus,
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
		rawStatus := jsonString(item.Status)
		if !supplierProviderGroupIsActive(rawStatus) {
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
			RawStatus:      rawStatus,
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

func supplierNewAPILoginAuthFailure(status int, raw []byte, err error) bool {
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return true
	}
	return supplierNewAPIBusinessAuthFailure(raw) || supplierNewAPIAuthPhraseFromError(err)
}

func supplierNewAPIAuthPhraseFromError(err error) bool {
	if err == nil {
		return false
	}
	return supplierNewAPIAuthPhrase(strings.ToLower(err.Error()))
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
	for _, phrase := range []string{
		"unauthorized",
		"forbidden",
		"token expired",
		"invalid token",
		"session expired",
		"auth failed",
		"authentication failed",
		"authentication error",
		"invalid credential",
		"invalid username",
		"invalid password",
		"username or password",
		"account or password",
		"incorrect password",
		"wrong password",
		"bad credential",
		"login failed",
		"failed to login",
		"login required",
		"not logged in",
		"未登录",
		"认证失败",
		"身份验证失败",
		"登录失败",
		"用户名或密码",
		"账号或密码",
		"密码错误",
		"账号错误",
		"凭证无效",
		"token无效",
		"会话过期",
	} {
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

func supplierNewAPISessionExpiresAt(raw any) time.Time {
	seconds := jsonFloat(raw)
	if seconds > 1e12 {
		seconds /= 1000
	}
	if seconds > 0 {
		expiresIn := time.Until(time.Unix(int64(seconds), 0))
		if expiresIn > 0 {
			return time.Now().Add(SupplierProviderTokenTTL(expiresIn))
		}
	}
	return time.Now().Add(SupplierProviderTokenTTL(0))
}

func supplierNewAPIHasSessionAuth(session supplierNewAPISession) bool {
	return strings.TrimSpace(session.AccessToken) != "" || strings.TrimSpace(session.CookieHeader) != ""
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

func supplierNewAPIAppendQuery(path, key, value string) string {
	path = strings.TrimSpace(path)
	if path == "" || strings.TrimSpace(key) == "" {
		return path
	}
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return path + sep + url.QueryEscape(key) + "=" + url.QueryEscape(value)
}

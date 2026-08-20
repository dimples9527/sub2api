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
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
)

const (
	defaultSupplierSub2APILoginPath    = "/api/v1/auth/login"
	defaultSupplierSub2APIRefreshPath  = "/api/v1/auth/refresh"
	DefaultSupplierSub2APIMonitorPath  = "/api/v1/channel-monitors?timezone=Asia%2FShanghai"
	defaultSupplierSub2APIRechargePath = "/api/v1/redeem/history?timezone=Asia%2FShanghai"
	legacySupplierSub2APIRechargePath  = "/v1/redeem/history?timezone=Asia%2FShanghai"
	defaultSupplierSub2APIHTTPTimeout  = 30 * time.Second
	// supplierSub2APICostRequestTimeout 限制成本接口单次请求耗时。
	// 上游成本接口在部分时段会长时间挂起或返回 500，
	// 用独立较短时间内让同步流程尽快回落到本地余额差保底估算，避免整个请求被拖满 30s。
	supplierSub2APICostRequestTimeout             = 15 * time.Second
	supplierSub2APIMaxResponseBytes         int64 = 4 << 20
	supplierSub2APILoginLockTTL                   = 120 * time.Second
	supplierSub2APILoginLockWait                  = 130 * time.Second
	supplierSub2APILoginLockPoll                  = 100 * time.Millisecond
	supplierSub2APISessionRefreshThreshold        = 5 * time.Minute
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

type SupplierProviderMonitorPoint struct {
	Status        string
	LatencyMS     int64
	PingLatencyMS int64
	CheckedAt     time.Time
}

type SupplierProviderMonitorItem struct {
	Key                  string
	Name                 string
	Provider             string
	GroupName            string
	PrimaryModel         string
	PrimaryStatus        string
	PrimaryLatencyMS     int64
	PrimaryPingLatencyMS int64
	Availability7D       float64
	Timeline             []SupplierProviderMonitorPoint
}

type SupplierProviderRemoteClient interface {
	FetchAccounts(ctx context.Context, provider *SupplierProvider, password string) ([]SupplierProviderRemoteAccount, error)
	FetchGroups(ctx context.Context, provider *SupplierProvider, password string) ([]SupplierProviderRemoteGroup, error)
	FetchBalance(ctx context.Context, provider *SupplierProvider, password string) (float64, error)
	FetchCost(ctx context.Context, provider *SupplierProvider, password string, day time.Time) (float64, error)
}

// SupplierProviderRemoteRechargeClient 提供指定统计日的上游充值金额。
type SupplierProviderRemoteRechargeClient interface {
	FetchRechargeAmount(ctx context.Context, provider *SupplierProvider, password string, day time.Time) (float64, error)
}

type SupplierProviderRemoteMonitorClient interface {
	FetchMonitorItems(ctx context.Context, provider *SupplierProvider, password string) ([]SupplierProviderMonitorItem, error)
}

type SupplierProviderRemoteTester interface {
	TestEndpoint(ctx context.Context, provider *SupplierProvider, password string, scope string) (SupplierProviderEndpointTestResult, error)
}

type SupplierProviderRemoteReauthenticator interface {
	Reauthenticate(ctx context.Context, provider *SupplierProvider, password string) (SupplierProviderAuthToken, error)
}

type SupplierProviderRemoteDiagnostics interface {
	LastEndpointResult(providerID int64, scope string) *SupplierProviderEndpointResult
}

type SupplierSub2APIClient struct {
	httpClient       *http.Client
	tokenCache       SupplierProviderTokenCache
	authAuditor      SupplierProviderAuthAuditor
	turnstileSolver  SupplierTurnstileSolver
	endpointResultMu sync.Mutex
	endpointResults  map[string]SupplierProviderEndpointResult
}

func (c *SupplierSub2APIClient) SetAuthAuditor(auditor SupplierProviderAuthAuditor) {
	c.authAuditor = auditor
}

type supplierSub2APILoginResult struct {
	token SupplierProviderAuthToken
	ttl   time.Duration
}

func NewSupplierSub2APIClient(httpClient *http.Client, tokenCache SupplierProviderTokenCache, turnstileSolver SupplierTurnstileSolver) *SupplierSub2APIClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultSupplierSub2APIHTTPTimeout}
	}
	if turnstileSolver == nil {
		turnstileSolver = noopSupplierTurnstileSolver{}
	}
	return &SupplierSub2APIClient{
		httpClient:      httpClient,
		tokenCache:      tokenCache,
		turnstileSolver: turnstileSolver,
		endpointResults: make(map[string]SupplierProviderEndpointResult),
	}
}

func (c *SupplierSub2APIClient) FetchAccounts(ctx context.Context, provider *SupplierProvider, password string) ([]SupplierProviderRemoteAccount, error) {
	var lastErr error
	for _, endpoint := range supplierSub2APIAccountEndpoints(provider.APIKeysURL) {
		raw, err := c.authenticatedGet(ctx, provider, password, endpoint, "accounts")
		if err == nil {
			accounts, parseErr := parseSupplierSub2APIAccounts(raw)
			c.annotateEndpointParse(provider.ID, "accounts", "accounts", raw, parseErr)
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
	c.annotateEndpointParse(provider.ID, "groups", "groups", raw, parseErr)
	return groups, parseErr
}

func (c *SupplierSub2APIClient) FetchBalance(ctx context.Context, provider *SupplierProvider, password string) (float64, error) {
	raw, err := c.authenticatedGet(ctx, provider, password, provider.BalanceURL, "balance")
	if err != nil {
		return 0, err
	}
	balance, parseErr := parseSupplierSub2APINumberField(raw, "balance")
	c.annotateEndpointParse(provider.ID, "balance", "balance", raw, parseErr)
	return balance, parseErr
}

func (c *SupplierSub2APIClient) FetchCost(ctx context.Context, provider *SupplierProvider, password string, _ time.Time) (float64, error) {
	// 上游成本接口不稳定时不应长时间挂起：使用独立较短超时，超时后由服务层走余额差保底估算。
	requestCtx, cancel := context.WithTimeout(ctx, supplierSub2APICostRequestTimeout)
	defer cancel()
	raw, err := c.authenticatedGet(requestCtx, provider, password, provider.UsageCostURL, "cost")
	if err != nil {
		return 0, err
	}
	cost, parseErr := parseSupplierSub2APINumberField(raw, "today_actual_cost")
	c.annotateEndpointParse(provider.ID, "cost", "cost", raw, parseErr)
	return cost, parseErr
}

func (c *SupplierSub2APIClient) FetchRechargeAmount(ctx context.Context, provider *SupplierProvider, password string, day time.Time) (float64, error) {
	var lastErr error
	for _, path := range supplierSub2APIRechargeEndpoints(provider.RechargeURL) {
		raw, err := c.authenticatedGet(ctx, provider, password, path, "recharge")
		if err == nil {
			return parseSupplierSub2APIRechargeAmount(raw, day)
		}
		lastErr = err
		if !supplierSub2APIStatusIs(err, http.StatusNotFound) {
			return 0, err
		}
	}
	return 0, lastErr
}

func (c *SupplierSub2APIClient) FetchRechargeRecords(ctx context.Context, provider *SupplierProvider, password string, start, end time.Time) ([]SupplierProviderRechargeRecord, error) {
	var lastErr error
	for _, path := range supplierSub2APIRechargeEndpoints(provider.RechargeURL) {
		raw, err := c.authenticatedGet(ctx, provider, password, path, "recharge")
		if err == nil {
			return parseSupplierSub2APIRechargeRecordsInRange(raw, start, end)
		}
		lastErr = err
		if !supplierSub2APIStatusIs(err, http.StatusNotFound) {
			return nil, err
		}
	}
	return nil, lastErr
}

func (c *SupplierSub2APIClient) FetchMonitorItems(ctx context.Context, provider *SupplierProvider, password string) ([]SupplierProviderMonitorItem, error) {
	monitorPath := strings.TrimSpace(provider.MonitorURL)
	if monitorPath == "" {
		monitorPath = DefaultSupplierSub2APIMonitorPath
	}
	raw, err := c.authenticatedGet(ctx, provider, password, monitorPath, SupplierSyncScopeMonitor)
	if err != nil {
		return nil, err
	}
	items, parseErr := parseSupplierSub2APIMonitorItems(raw)
	c.annotateEndpointParse(provider.ID, SupplierSyncScopeMonitor, SupplierSyncScopeMonitor, raw, parseErr)
	return items, parseErr
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
	case SupplierSyncScopeMonitor:
		monitorPath := strings.TrimSpace(provider.MonitorURL)
		if monitorPath == "" {
			monitorPath = DefaultSupplierSub2APIMonitorPath
		}
		endpoints = []string{monitorPath}
	default:
		return SupplierProviderEndpointTestResult{}, fmt.Errorf("unsupported supplier endpoint test scope: %s", scope)
	}
	for _, endpoint := range endpoints {
		startedAt := time.Now()
		raw, status, requestErr := c.authenticatedRequest(ctx, provider, password, http.MethodGet, endpoint, scope, nil)
		if requestErr != nil && IsSupplierProviderSessionFailure(requestErr) {
			return SupplierProviderEndpointTestResult{}, requestErr
		}
		attempt := SupplierProviderEndpointTestAttempt{
			Endpoint:        endpoint,
			HTTPStatus:      status,
			DurationMS:      time.Since(startedAt).Milliseconds(),
			ResponseBytes:   len(raw),
			ResponseSummary: supplierSub2APISafeResponseText(raw, supplierSub2APITestResponseSummaryLimit),
			Error:           supplierSub2APIErrorText(requestErr),
		}
		if requestErr == nil {
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
		if requestErr != nil || status < 200 || status >= 300 {
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
	startedAt := time.Now()
	raw, status, err := c.authenticatedRequest(ctx, provider, password, http.MethodGet, path, label, nil)
	c.recordEndpointResult(provider.ID, label, SupplierProviderEndpointResult{
		Endpoint:        path,
		HTTPStatus:      status,
		DurationMS:      time.Since(startedAt).Milliseconds(),
		ResponseBytes:   len(raw),
		ResponseSummary: supplierSub2APISafeResponseText(raw, supplierSub2APITestResponseSummaryLimit),
		Error:           supplierSub2APIErrorText(err),
	})
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *SupplierSub2APIClient) authenticatedRequest(ctx context.Context, provider *SupplierProvider, password, method, path, label string, body io.Reader) ([]byte, int, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		token, err := c.ensureToken(ctx, provider, password)
		if err != nil {
			return nil, 0, err
		}
		raw, status, err := c.doJSON(ctx, method, provider, path, label, token, body)
		if err != nil {
			c.logRequestDiagnosis(provider, label, attempt+1, status, raw, err, false)
			return raw, status, fmt.Errorf("supplier sub2api %s request failed: %w", label, err)
		}
		if status >= 200 && status < 300 && !supplierSub2APIBusinessAuthFailure(raw) {
			if envelopeErr := supplierSub2APIEnvelopeOK(raw); envelopeErr == nil {
				return raw, status, nil
			} else if attempt == 0 && supplierSub2APIAuthFailure(status, raw, envelopeErr) {
				c.logRequestDiagnosis(provider, label, attempt+1, status, raw, envelopeErr, true)
				lastErr = envelopeErr
				if _, recoverErr := c.recoverTokenAfterAuthFailure(ctx, provider, password, token); recoverErr != nil {
					return raw, status, recoverErr
				}
				continue
			} else {
				c.logRequestDiagnosis(provider, label, attempt+1, status, raw, envelopeErr, false)
				return raw, status, envelopeErr
			}
		}
		requestErr := supplierSub2APIHTTPError(label, status, raw)
		if attempt == 0 && supplierSub2APIAuthFailure(status, raw, requestErr) {
			c.logRequestDiagnosis(provider, label, attempt+1, status, raw, requestErr, true)
			lastErr = requestErr
			if _, recoverErr := c.recoverTokenAfterAuthFailure(ctx, provider, password, token); recoverErr != nil {
				return raw, status, recoverErr
			}
			continue
		}
		if supplierSub2APIAuthFailure(status, raw, requestErr) {
			c.deleteToken(ctx, provider)
			c.logRequestDiagnosis(provider, label, attempt+1, status, raw, requestErr, true)
			return raw, status, wrapSupplierProviderAuthFailure(fmt.Errorf("supplier sub2api %s failed after auth retry: %w", label, requestErr))
		}
		c.logRequestDiagnosis(provider, label, attempt+1, status, raw, requestErr, false)
		return raw, status, requestErr
	}
	if supplierSub2APIAuthFailure(0, nil, lastErr) {
		return nil, 0, wrapSupplierProviderAuthFailure(fmt.Errorf("supplier sub2api %s failed after auth retry: %w", label, lastErr))
	}
	return nil, 0, fmt.Errorf("supplier sub2api %s failed after auth retry: %w", label, lastErr)
}

func (c *SupplierSub2APIClient) logRequestDiagnosis(provider *SupplierProvider, label string, attempt int, status int, raw []byte, err error, authFailure bool) {
	if provider == nil {
		return
	}
	logger.LegacyPrintf(
		"supplier_sub2api_client",
		"supplier request diagnosis provider_id=%d provider_code=%s label=%s attempt=%d http_status=%d auth_failure=%t response=%s err=%v",
		provider.ID,
		provider.Code,
		label,
		attempt,
		status,
		authFailure,
		supplierSub2APISafeResponseSummary(raw),
		err,
	)
}

func (c *SupplierSub2APIClient) LastEndpointResult(providerID int64, scope string) *SupplierProviderEndpointResult {
	c.endpointResultMu.Lock()
	defer c.endpointResultMu.Unlock()
	if c.endpointResults == nil {
		return nil
	}
	result, ok := c.endpointResults[supplierSub2APIEndpointResultKey(providerID, scope)]
	if !ok {
		return nil
	}
	return &result
}

func (c *SupplierSub2APIClient) recordEndpointResult(providerID int64, scope string, result SupplierProviderEndpointResult) {
	c.endpointResultMu.Lock()
	defer c.endpointResultMu.Unlock()
	if c.endpointResults == nil {
		c.endpointResults = make(map[string]SupplierProviderEndpointResult)
	}
	c.endpointResults[supplierSub2APIEndpointResultKey(providerID, scope)] = result
}

func (c *SupplierSub2APIClient) updateEndpointError(providerID int64, scope string, err error) {
	c.endpointResultMu.Lock()
	defer c.endpointResultMu.Unlock()
	key := supplierSub2APIEndpointResultKey(providerID, scope)
	result := c.endpointResults[key]
	result.Error = supplierSub2APIErrorText(err)
	c.endpointResults[key] = result
}

func (c *SupplierSub2APIClient) annotateEndpointParse(providerID int64, scope, parseScope string, raw []byte, parseErr error) {
	c.endpointResultMu.Lock()
	defer c.endpointResultMu.Unlock()
	key := supplierSub2APIEndpointResultKey(providerID, scope)
	result := c.endpointResults[key]
	result.ParsedSummary, result.ParseError = supplierSub2APIParsedSummary(parseScope, raw, parseErr)
	c.endpointResults[key] = result
}

func supplierSub2APIEndpointResultKey(providerID int64, scope string) string {
	return fmt.Sprintf("%d:%s", providerID, strings.TrimSpace(scope))
}

func (c *SupplierSub2APIClient) ensureToken(ctx context.Context, provider *SupplierProvider, password string) (result SupplierProviderAuthToken, err error) {
	SupplierSyncProgress(ctx, SupplierSyncProgressStageSession, "正在检查上游登录会话", nil)
	defer func() {
		if err != nil {
			err = wrapSupplierProviderSessionFailure(err)
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStageSession, err)
			return
		}
		SupplierSyncProgressOK(ctx, SupplierSyncProgressStageSession, "上游登录会话已准备")
	}()
	if c.tokenCache == nil {
		c.recordAuthEvent(ctx, provider, SupplierProviderAuthEventInput{
			EventType: SupplierProviderAuthEventCacheError,
			Error:     errors.New("supplier provider token cache is unavailable"),
		})
		loginResult, loginErr := c.loginWithAudit(ctx, provider, password)
		return loginResult.token, loginErr
	}
	if token, found, getErr := c.tokenCache.Get(ctx, provider.ID); getErr != nil {
		return SupplierProviderAuthToken{}, c.cacheFailure(ctx, provider, "get", getErr)
	} else if found && supplierSub2APITokenUsable(token) {
		token = normalizeSupplierSub2APIToken(token)
		c.recordAuthEvent(ctx, provider, SupplierProviderAuthEventInput{EventType: SupplierProviderAuthEventCacheHit, Token: &token})
		if !supplierSub2APITokenNeedsRefresh(token, time.Now()) {
			return token, nil
		}
		return c.acquireTokenLockMode(ctx, provider, password, false, token)
	}
	c.recordAuthEvent(ctx, provider, SupplierProviderAuthEventInput{EventType: SupplierProviderAuthEventCacheMiss})
	return c.acquireTokenLockMode(ctx, provider, password, false, SupplierProviderAuthToken{})
}

func (c *SupplierSub2APIClient) recoverTokenAfterAuthFailure(ctx context.Context, provider *SupplierProvider, password string, failedToken SupplierProviderAuthToken) (SupplierProviderAuthToken, error) {
	if c.tokenCache == nil {
		if refreshed, refreshErr := c.refreshWithAudit(ctx, provider, failedToken); refreshErr == nil {
			return refreshed.token, nil
		}
		loginResult, loginErr := c.loginWithAudit(ctx, provider, password)
		if loginErr != nil {
			return SupplierProviderAuthToken{}, loginErr
		}
		return loginResult.token, nil
	}
	return c.acquireTokenLockMode(ctx, provider, password, true, failedToken)
}

func (c *SupplierSub2APIClient) acquireTokenLockMode(ctx context.Context, provider *SupplierProvider, password string, forceRefresh bool, fallbackToken SupplierProviderAuthToken) (SupplierProviderAuthToken, error) {
	owner := uuid.NewString()
	acquired, err := c.tokenCache.TryAcquireLoginLock(ctx, provider.ID, owner, supplierSub2APILoginLockTTL)
	if err != nil {
		return SupplierProviderAuthToken{}, c.cacheFailure(ctx, provider, "acquire login lock", err)
	}
	if acquired {
		return c.refreshOrLoginAndCache(ctx, provider, password, owner, forceRefresh, fallbackToken)
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
			if token, found, getErr := c.tokenCache.Get(waitCtx, provider.ID); getErr != nil {
				return SupplierProviderAuthToken{}, c.cacheFailure(ctx, provider, "poll token", getErr)
			} else if found && supplierSub2APITokenUsable(token) {
				token = normalizeSupplierSub2APIToken(token)
				canReuse := !supplierSub2APITokenNeedsRefresh(token, time.Now())
				if forceRefresh {
					canReuse = !supplierSub2APITokenAuthEqual(token, fallbackToken) && canReuse
				}
				if canReuse {
					c.recordAuthEvent(ctx, provider, SupplierProviderAuthEventInput{EventType: SupplierProviderAuthEventCacheHit, Token: &token})
					return token, nil
				}
			}
			acquired, err := c.tokenCache.TryAcquireLoginLock(waitCtx, provider.ID, owner, supplierSub2APILoginLockTTL)
			if err != nil {
				return SupplierProviderAuthToken{}, c.cacheFailure(ctx, provider, "retry login lock", err)
			}
			if acquired {
				return c.refreshOrLoginAndCache(ctx, provider, password, owner, forceRefresh, fallbackToken)
			}
		}
	}
}

func (c *SupplierSub2APIClient) refreshOrLoginAndCache(ctx context.Context, provider *SupplierProvider, password, owner string, forceRefresh bool, fallbackToken SupplierProviderAuthToken) (SupplierProviderAuthToken, error) {
	defer func() {
		if err := c.tokenCache.ReleaseLoginLock(context.Background(), provider.ID, owner); err != nil {
			c.logCacheError(provider, "release login lock", err)
		}
	}()

	var candidate SupplierProviderAuthToken
	foundCandidate := false
	if token, found, getErr := c.tokenCache.Get(ctx, provider.ID); getErr != nil {
		return SupplierProviderAuthToken{}, c.cacheFailure(ctx, provider, "recheck token", getErr)
	} else if found && supplierSub2APITokenUsable(token) {
		token = normalizeSupplierSub2APIToken(token)
		candidate = token
		foundCandidate = true
		canReuse := !supplierSub2APITokenNeedsRefresh(token, time.Now())
		if forceRefresh {
			canReuse = !supplierSub2APITokenAuthEqual(token, fallbackToken) && canReuse
		}
		if canReuse {
			c.recordAuthEvent(ctx, provider, SupplierProviderAuthEventInput{EventType: SupplierProviderAuthEventCacheHit, Token: &token})
			return token, nil
		}
	}
	if !foundCandidate && forceRefresh && supplierSub2APITokenUsable(fallbackToken) {
		candidate = normalizeSupplierSub2APIToken(fallbackToken)
		foundCandidate = true
	}
	if foundCandidate {
		shouldRefresh := supplierSub2APITokenNeedsRefresh(candidate, time.Now())
		if forceRefresh && supplierSub2APITokenAuthEqual(candidate, fallbackToken) {
			shouldRefresh = true
		}
		if shouldRefresh {
			if refreshed, refreshErr := c.refreshWithAudit(ctx, provider, candidate); refreshErr == nil {
				if err := c.tokenCache.Set(ctx, provider.ID, refreshed.token, refreshed.ttl); err != nil {
					return SupplierProviderAuthToken{}, c.cacheFailure(ctx, provider, "set refreshed token", err)
				}
				return refreshed.token, nil
			}
			c.deleteToken(ctx, provider)
		}
	}

	loginResult, loginErr := c.loginWithAudit(ctx, provider, password)
	if loginErr != nil {
		return SupplierProviderAuthToken{}, loginErr
	}
	if err := c.tokenCache.Set(ctx, provider.ID, loginResult.token, loginResult.ttl); err != nil {
		return SupplierProviderAuthToken{}, c.cacheFailure(ctx, provider, "set", err)
	}
	return loginResult.token, nil
}

func (c *SupplierSub2APIClient) loginWithAudit(ctx context.Context, provider *SupplierProvider, password string) (supplierSub2APILoginResult, error) {
	SupplierSyncProgress(ctx, SupplierSyncProgressStageLogin, "正在登录上游", nil)
	startedAt := time.Now()
	result, err := c.login(ctx, provider, password)
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
		event.Token = &result.token
	}
	c.recordAuthEvent(ctx, provider, event)
	return result, err
}

func (c *SupplierSub2APIClient) cacheFailure(ctx context.Context, provider *SupplierProvider, action string, err error) error {
	c.logCacheError(provider, action, err)
	wrapped := fmt.Errorf("supplier provider token cache %s failed: %w", action, err)
	c.recordAuthEvent(ctx, provider, SupplierProviderAuthEventInput{EventType: SupplierProviderAuthEventCacheError, Error: wrapped})
	return wrapped
}

func (c *SupplierSub2APIClient) recordAuthEvent(ctx context.Context, provider *SupplierProvider, event SupplierProviderAuthEventInput) {
	if c == nil || c.authAuditor == nil || provider == nil {
		return
	}
	if event.Token != nil {
		token := supplierProviderAuthAuditToken(*event.Token)
		event.Token = &token
	}
	if event.Error != nil {
		event.Error = supplierProviderAuthAuditError(event.Error)
	}
	event.ProviderID = provider.ID
	if event.Source == "" {
		event.Source = supplierProviderAuthSourceFromContext(ctx)
	}
	// 使用独立超时上下文，避免父请求取消后审计写入被静默丢弃。
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	if err := c.authAuditor.Record(auditCtx, event); err != nil {
		logger.LegacyPrintf("supplier_sub2api_client", "record supplier provider auth event failed provider_id=%d provider_code=%s err=%v", provider.ID, provider.Code, err)
	}
}

func (c *SupplierSub2APIClient) login(ctx context.Context, provider *SupplierProvider, password string) (supplierSub2APILoginResult, error) {
	loginPath := strings.TrimSpace(provider.LoginURL)
	if loginPath == "" {
		loginPath = defaultSupplierSub2APILoginPath
	}
	payload := map[string]string{
		"email":    strings.TrimSpace(provider.Email),
		"password": password,
	}
	if c.turnstileSolver != nil {
		token, err := c.turnstileSolver.PrepareToken(ctx, provider, strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/"), func(fetchCtx context.Context) (string, error) {
			return fetchSupplierSub2APITurnstileSiteKey(fetchCtx, c.httpClient, provider)
		})
		if err != nil {
			return supplierSub2APILoginResult{}, wrapSupplierProviderAuthFailure(err)
		}
		if token != "" {
			payload["turnstile_token"] = token
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return supplierSub2APILoginResult{}, fmt.Errorf("marshal supplier sub2api login: %w", err)
	}
	raw, status, err := c.doJSON(ctx, http.MethodPost, provider, loginPath, "login", SupplierProviderAuthToken{}, bytes.NewReader(body))
	if err != nil {
		return supplierSub2APILoginResult{}, fmt.Errorf("supplier sub2api login request failed: %w", err)
	}
	if status < 200 || status >= 300 {
		loginErr := supplierSub2APIHTTPError("login", status, raw)
		if supplierSub2APILoginAuthFailure(status, raw, loginErr) {
			return supplierSub2APILoginResult{}, wrapSupplierProviderAuthFailure(loginErr)
		}
		return supplierSub2APILoginResult{}, loginErr
	}
	token, expiresIn, err := parseSupplierSub2APILogin(raw)
	if err != nil {
		if supplierSub2APILoginAuthFailure(status, raw, err) {
			return supplierSub2APILoginResult{}, wrapSupplierProviderAuthFailure(err)
		}
		return supplierSub2APILoginResult{}, err
	}
	if expiresIn > 0 {
		token.ExpiresAt = time.Now().Add(expiresIn)
	}
	return supplierSub2APILoginResult{token: normalizeSupplierSub2APIToken(token), ttl: 0}, nil
}

func (c *SupplierSub2APIClient) refreshWithAudit(ctx context.Context, provider *SupplierProvider, previous SupplierProviderAuthToken) (supplierSub2APILoginResult, error) {
	startedAt := time.Now()
	result, status, err := c.refresh(ctx, provider, previous)
	var httpStatus *int
	if status > 0 {
		statusForAudit := status
		httpStatus = &statusForAudit
	}
	event := SupplierProviderAuthEventInput{
		EventType:  SupplierProviderAuthEventRefreshSuccess,
		StartedAt:  startedAt,
		FinishedAt: time.Now(),
		HTTPStatus: httpStatus,
		Error:      err,
	}
	if err != nil {
		event.EventType = SupplierProviderAuthEventRefreshFailed
		if supplierSub2APITokenUsable(previous) {
			token := previous
			event.Token = &token
		}
	} else {
		token := result.token
		event.Token = &token
	}
	c.recordAuthEvent(ctx, provider, event)
	return result, err
}

func (c *SupplierSub2APIClient) refresh(ctx context.Context, provider *SupplierProvider, previous SupplierProviderAuthToken) (supplierSub2APILoginResult, int, error) {
	refreshToken := strings.TrimSpace(previous.RefreshToken)
	if refreshToken == "" {
		return supplierSub2APILoginResult{}, 0, fmt.Errorf("supplier sub2api refresh failed: missing refresh token")
	}
	body, err := json.Marshal(map[string]string{"refresh_token": refreshToken})
	if err != nil {
		return supplierSub2APILoginResult{}, 0, fmt.Errorf("marshal supplier sub2api refresh: %w", err)
	}
	raw, status, err := c.doJSON(ctx, http.MethodPost, provider, defaultSupplierSub2APIRefreshPath, "refresh", SupplierProviderAuthToken{}, bytes.NewReader(body))
	if err != nil {
		return supplierSub2APILoginResult{}, status, fmt.Errorf("supplier sub2api refresh request failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return supplierSub2APILoginResult{}, status, supplierSub2APIHTTPError("refresh", status, raw)
	}
	token, expiresIn, err := parseSupplierSub2APILogin(raw)
	if err != nil {
		summary := supplierSub2APISafeResponseSummary(raw)
		if summary == "" {
			return supplierSub2APILoginResult{}, status, errors.New("supplier sub2api refresh response is invalid")
		}
		return supplierSub2APILoginResult{}, status, fmt.Errorf("supplier sub2api refresh response is invalid: %s", summary)
	}
	token.RefreshToken = firstSupplierSub2APIString(token.RefreshToken, refreshToken)
	if expiresIn > 0 {
		token.ExpiresAt = time.Now().Add(expiresIn)
	}
	return supplierSub2APILoginResult{token: normalizeSupplierSub2APIToken(token), ttl: 0}, status, nil
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
		Code         any    `json:"code"`
		Message      string `json:"message"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Token        string `json:"token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    any    `json:"expires_in"`
		Data         struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			Token        string `json:"token"`
			TokenType    string `json:"token_type"`
			ExpiresIn    any    `json:"expires_in"`
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
		AccessToken:  token,
		RefreshToken: firstSupplierSub2APIString(resp.Data.RefreshToken, resp.RefreshToken),
		TokenType:    firstSupplierSub2APIString(resp.Data.TokenType, resp.TokenType, "Bearer"),
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
		rawStatus := strings.TrimSpace(jsonString(item["status"]))
		status := normalizeSupplierSub2APIKeyStatus(rawStatus)
		out = append(out, SupplierProviderRemoteAccount{
			Key:            strings.TrimSpace(key),
			Name:           name,
			Status:         status,
			GroupKey:       strings.TrimSpace(groupKey),
			GroupName:      strings.TrimSpace(groupName),
			RateMultiplier: rate,
			RawStatus:      rawStatus,
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
		if !supplierProviderGroupIsActive(status) {
			continue
		}
		out = append(out, SupplierProviderRemoteGroup{
			Key:            strings.TrimSpace(key),
			Name:           name,
			RateMultiplier: jsonFloat(item["rate_multiplier"]),
			RawStatus:      strings.TrimSpace(status),
		})
	}
	return out, nil
}

func parseSupplierSub2APIMonitorItems(raw []byte) ([]SupplierProviderMonitorItem, error) {
	items, err := supplierSub2APIItems(raw)
	if err != nil {
		return nil, err
	}
	out := make([]SupplierProviderMonitorItem, 0, len(items))
	for _, item := range items {
		key := firstSupplierSub2APIString(jsonString(item["id"]), jsonString(item["key"]))
		name := strings.TrimSpace(jsonString(item["name"]))
		if key == "" && name != "" {
			key = normalizeSupplierSub2APIKeyFromName(name)
		}
		if key == "" && name == "" {
			continue
		}
		monitor := SupplierProviderMonitorItem{
			Key:                  strings.TrimSpace(key),
			Name:                 name,
			Provider:             strings.TrimSpace(jsonString(item["provider"])),
			GroupName:            strings.TrimSpace(jsonString(item["group_name"])),
			PrimaryModel:         strings.TrimSpace(jsonString(item["primary_model"])),
			PrimaryStatus:        strings.TrimSpace(jsonString(item["primary_status"])),
			PrimaryLatencyMS:     jsonInt64(item["primary_latency_ms"]),
			PrimaryPingLatencyMS: jsonInt64(item["primary_ping_latency_ms"]),
			Availability7D:       jsonFloat(item["availability_7d"]),
		}
		if timeline, ok := item["timeline"].([]any); ok {
			monitor.Timeline = make([]SupplierProviderMonitorPoint, 0, len(timeline))
			for _, rawPoint := range timeline {
				pointMap, ok := rawPoint.(map[string]any)
				if !ok {
					continue
				}
				checkedAt, _ := time.Parse(time.RFC3339, strings.TrimSpace(jsonString(pointMap["checked_at"])))
				monitor.Timeline = append(monitor.Timeline, SupplierProviderMonitorPoint{
					Status:        strings.TrimSpace(jsonString(pointMap["status"])),
					LatencyMS:     jsonInt64(pointMap["latency_ms"]),
					PingLatencyMS: jsonInt64(pointMap["ping_latency_ms"]),
					CheckedAt:     checkedAt,
				})
			}
		}
		out = append(out, monitor)
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

func parseSupplierSub2APIRechargeAmount(raw []byte, day time.Time) (float64, error) {
	items, err := supplierSub2APIItems(raw)
	if err != nil {
		return 0, fmt.Errorf("decode supplier sub2api recharge response: %w", err)
	}
	var amount float64
	for _, item := range items {
		if !strings.EqualFold(strings.TrimSpace(jsonString(item["status"])), "used") {
			continue
		}
		rechargeType := strings.ToLower(strings.TrimSpace(jsonString(item["type"])))
		if rechargeType != "balance" && rechargeType != "admin_balance" {
			continue
		}
		usedAt, parseErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(jsonString(item["used_at"])))
		if parseErr != nil || !sameSupplierCostStatDay(usedAt, day) {
			continue
		}
		value := jsonFloat(item["value"])
		if value > 0 {
			amount += value
		}
	}
	return amount, nil
}

func parseSupplierSub2APIRechargeRecords(raw []byte, day time.Time) ([]SupplierProviderRechargeRecord, error) {
	loc := day.Location()
	if loc == nil {
		loc = time.UTC
	}
	localDay := day.In(loc)
	start := time.Date(localDay.Year(), localDay.Month(), localDay.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)
	return parseSupplierSub2APIRechargeRecordsInRange(raw, start, end)
}

func parseSupplierSub2APIRechargeRecordsInRange(raw []byte, start, end time.Time) ([]SupplierProviderRechargeRecord, error) {
	var envelope struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    []struct {
			ID        json.RawMessage `json:"id"`
			Code      string          `json:"code"`
			Type      string          `json:"type"`
			Value     float64         `json:"value"`
			Status    string          `json:"status"`
			UsedAt    string          `json:"used_at"`
			CreatedAt string          `json:"created_at"`
			Raw       json.RawMessage `json:"-"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode supplier sub2api recharge response: %w", err)
	}
	if envelope.Code != 0 {
		return nil, fmt.Errorf("supplier sub2api recharge failed: %s", firstSupplierSub2APIString(envelope.Message, "unknown error"))
	}
	if end.IsZero() {
		end = time.Now()
	}
	if start.IsZero() {
		start = time.Unix(0, 0)
	}
	items := make([]map[string]any, 0, len(envelope.Data))
	var generic struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(raw, &generic); err == nil {
		items = generic.Data
	}
	records := make([]SupplierProviderRechargeRecord, 0, len(envelope.Data))
	for idx, item := range envelope.Data {
		if item.Status != "used" || (item.Type != "balance" && item.Type != "admin_balance") || item.Value <= 0 {
			continue
		}
		occurredAt, err := time.Parse(time.RFC3339Nano, firstSupplierSub2APIString(item.UsedAt, item.CreatedAt))
		if err != nil || occurredAt.Before(start) || !occurredAt.Before(end) {
			continue
		}
		externalID := supplierProviderJSONScalarText(item.ID)
		var payload json.RawMessage
		if idx < len(items) {
			payload, _ = json.Marshal(items[idx])
		}
		records = append(records, SupplierProviderRechargeRecord{
			ExternalID: externalID, ExternalCode: strings.TrimSpace(item.Code), RechargeType: strings.TrimSpace(item.Type),
			Amount: item.Value, Status: strings.TrimSpace(item.Status), OccurredAt: occurredAt, RawPayload: payload,
		})
	}
	return records, nil
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
	if supplierSub2APIProbeBlocked(raw) {
		return false
	}
	if status == http.StatusUnauthorized {
		return true
	}
	if status == http.StatusForbidden {
		return supplierSub2APIBusinessAuthFailure(raw) || supplierSub2APIErrorLooksAuth(err)
	}
	return supplierSub2APIBusinessAuthFailure(raw) || supplierSub2APIErrorLooksAuth(err)
}

func supplierSub2APILoginAuthFailure(status int, raw []byte, err error) bool {
	if supplierSub2APIProbeBlocked(raw) {
		return false
	}
	if status == http.StatusUnauthorized {
		return true
	}
	if status == http.StatusForbidden {
		return supplierSub2APIBusinessAuthFailure(raw) || supplierSub2APIErrorLooksAuth(err)
	}
	if supplierSub2APIBusinessAuthFailure(raw) || supplierSub2APIErrorLooksAuth(err) {
		return true
	}
	return false
}

func supplierSub2APIProbeBlocked(raw []byte) bool {
	if len(raw) == 0 {
		return false
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return false
	}
	return supplierSub2APIProbeBlockedValue(decoded)
}

func supplierSub2APIProbeBlockedValue(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if strings.EqualFold(strings.TrimSpace(key), "type") || strings.EqualFold(strings.TrimSpace(key), "code") {
				if supplierSub2APIProbeBlockedText(fmt.Sprint(item)) {
					return true
				}
			}
			if supplierSub2APIProbeBlockedValue(item) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if supplierSub2APIProbeBlockedValue(item) {
				return true
			}
		}
	case string:
		return supplierSub2APIProbeBlockedText(typed)
	}
	return false
}

func supplierSub2APIProbeBlockedText(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	return normalized == "probe_blocked" || strings.Contains(normalized, "probe, monitoring, and test traffic are disabled")
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

func supplierSub2APIHTTPError(label string, status int, raw []byte) error {
	if status == 0 {
		return errors.New("supplier sub2api request failed")
	}
	return supplierSub2APIHTTPStatusError{
		label:   label,
		status:  status,
		message: supplierSub2APISafeResponseText(raw, 256),
	}
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

func supplierSub2APIRechargeEndpoints(configured string) []string {
	candidates := []string{
		strings.TrimSpace(configured),
		defaultSupplierSub2APIRechargePath,
		legacySupplierSub2APIRechargePath,
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
	token.RefreshToken = strings.TrimSpace(token.RefreshToken)
	token.TokenType = strings.TrimSpace(token.TokenType)
	if token.TokenType == "" {
		token.TokenType = "Bearer"
	}
	return token
}

func supplierSub2APITokenUsable(token SupplierProviderAuthToken) bool {
	return strings.TrimSpace(token.AccessToken) != ""
}

func supplierSub2APITokenNeedsRefresh(token SupplierProviderAuthToken, now time.Time) bool {
	return !token.ExpiresAt.IsZero() && !token.ExpiresAt.After(now.Add(supplierSub2APISessionRefreshThreshold))
}

func supplierSub2APITokenAuthEqual(left, right SupplierProviderAuthToken) bool {
	return strings.TrimSpace(left.AccessToken) == strings.TrimSpace(right.AccessToken)
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

func jsonInt64(value any) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case json.Number:
		parsed, _ := strconv.ParseInt(v.String(), 10, 64)
		if parsed != 0 {
			return parsed
		}
		floatValue, _ := strconv.ParseFloat(v.String(), 64)
		return int64(floatValue)
	case string:
		parsed, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return parsed
	case int:
		return int64(v)
	case int64:
		return v
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
		_ = c.cacheFailure(ctx, provider, "delete", err)
		return
	}
	c.recordAuthEvent(ctx, provider, SupplierProviderAuthEventInput{EventType: SupplierProviderAuthEventCacheInvalidated})
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
	} else {
		text = supplierSub2APIRedactInlineSensitiveText(text)
	}
	return supplierSub2APITruncateLogText(text, limit)
}

func supplierSub2APIRedactInlineSensitiveText(text string) string {
	return supplierProviderAuthSecretPattern.ReplaceAllString(text, "$1[REDACTED]")
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
	case SupplierSyncScopeMonitor:
		items, err := parseSupplierSub2APIMonitorItems(raw)
		if err != nil {
			return nil, err.Error()
		}
		return map[string]any{"count": len(items), "items": items}, ""
	default:
		return nil, "unsupported scope"
	}
}

func supplierSub2APIParsedSummary(scope string, raw []byte, parseErr error) (string, string) {
	if parseErr != nil {
		return "", parseErr.Error()
	}
	switch scope {
	case SupplierSyncScopeAccounts:
		items, err := parseSupplierSub2APIAccounts(raw)
		if err != nil {
			return "", err.Error()
		}
		return fmt.Sprintf("账号 %d 个", len(items)), ""
	case SupplierSyncScopeGroups:
		items, err := parseSupplierSub2APIGroups(raw)
		if err != nil {
			return "", err.Error()
		}
		return fmt.Sprintf("分组 %d 个", len(items)), ""
	case SupplierSyncScopeBalance:
		value, err := parseSupplierSub2APINumberField(raw, "balance")
		if err != nil {
			return "", err.Error()
		}
		return fmt.Sprintf("余额 %.6f", value), ""
	case SupplierSyncScopeCost:
		value, err := parseSupplierSub2APINumberField(raw, "today_actual_cost")
		if err != nil {
			return "", err.Error()
		}
		return fmt.Sprintf("今日成本 %.6f", value), ""
	case SupplierSyncScopeMonitor:
		items, err := parseSupplierSub2APIMonitorItems(raw)
		if err != nil {
			return "", err.Error()
		}
		return fmt.Sprintf("监控 %d 个", len(items)), ""
	default:
		return "", "unsupported scope"
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
		return supplierSub2APITruncateLogText(supplierSub2APIRedactInlineSensitiveText(typed), supplierSub2APILogStringValueLimit)
	default:
		return value
	}
}

func supplierSub2APISensitiveLogKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if strings.ReplaceAll(strings.ReplaceAll(normalized, "_", ""), "-", "") == "newapirefresh" {
		return true
	}
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

type SupplierProviderRemoteTokenRefresher interface {
	RefreshToken(ctx context.Context, provider *SupplierProvider) (SupplierProviderAuthToken, error)
}

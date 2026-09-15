package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func supplierProviderJSONScalarText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return strings.Trim(strings.TrimSpace(string(raw)), `"`)
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

const (
	SupplierProviderTypeSub2API = "sub2api"
	SupplierProviderTypeNewAPI  = "newapi"
)

type SupplierProviderRemoteRegistry struct {
	sub2api *SupplierSub2APIClient
	newapi  *SupplierNewAPIClient
}

// SupplierProviderRechargeRecord 是上游充值记录在供应商模块内的统一表示。
type SupplierProviderRechargeRecord struct {
	ExternalID   string
	ExternalCode string
	RechargeType string
	Amount       float64
	Status       string
	OccurredAt   time.Time
	Description  string
	RawPayload   json.RawMessage
}

// SupplierProviderRemoteRechargeHistoryClient 提供上游充值历史拉取能力。
type SupplierProviderRemoteRechargeHistoryClient interface {
	FetchRechargeRecords(ctx context.Context, provider *SupplierProvider, password string, start, end time.Time) ([]SupplierProviderRechargeRecord, error)
}

// SupplierProviderUpstreamSession 描述上游一条登录会话的可展示信息。
// 上游不返回 is_current，Current 由本地已缓存会话的 sid 精确比对得出。
type SupplierProviderUpstreamSession struct {
	SID string `json:"sid"`
	// Current 表示这条会话正是当前同步正在使用的那条，清理时必须保留它。
	Current      bool      `json:"current"`
	Status       string    `json:"status"`
	LoginMethod  string    `json:"login_method"`
	IP           string    `json:"ip"`
	UserAgent    string    `json:"user_agent"`
	CreatedAt    time.Time `json:"created_at"`
	LastActiveAt time.Time `json:"last_active_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// SupplierProviderUpstreamSessionSummary 描述上游当前持有的登录会话。
// 上游的会话列表包含当前正在使用的会话，因此 Count 至少为 1；大于 1 即说明存在未被吊销的残留会话。
type SupplierProviderUpstreamSessionSummary struct {
	// Supported 表示上游是否提供会话管理接口，老版本 New API 没有该接口。
	Supported bool
	// CredentialAvailable 表示本地是否缓存了可用于查询的登录凭据。
	// 为 false 时 Count 一定是 0，但这个 0 表示「查不了」而不是「没有会话」，
	// 调用方必须据此显示明确的说明，绝不能把 0 当成「上游没有残留会话」。
	CredentialAvailable bool
	Count               int
	// Sessions 为会话明细，概览接口只取 Count，明细接口才把它透出给前端。
	Sessions []SupplierProviderUpstreamSession
}

// SupplierProviderRemoteSessionManager 提供上游登录会话的查询与清理能力。
// ListUpstreamSessions / RevokeOtherUpstreamSessions 复用当前已缓存的会话凭据，
// 不会触发新的登录，避免查询行为本身制造新会话；代价是本地没有凭据时查不了，
// 此时由 CredentialAvailable=false 如实反馈。
// ListUpstreamSessionsWithLogin 是唯一会在查询前主动登录的入口，供用户显式发起的操作使用。
type SupplierProviderRemoteSessionManager interface {
	ListUpstreamSessions(ctx context.Context, provider *SupplierProvider) (SupplierProviderUpstreamSessionSummary, error)
	// ListUpstreamSessionsWithLogin 在本地没有可用凭据时用给定密码登录一次再查询。
	// 登录会在上游新增一条会话，因此只应由用户显式确认的操作触发，不能用在自动巡检路径上。
	ListUpstreamSessionsWithLogin(ctx context.Context, provider *SupplierProvider, password string) (SupplierProviderUpstreamSessionSummary, error)
	RevokeOtherUpstreamSessions(ctx context.Context, provider *SupplierProvider) (int, error)
}

func NewSupplierProviderRemoteRegistry(httpClient *http.Client, tokenCache SupplierProviderTokenCache, turnstileSolver SupplierTurnstileSolver) *SupplierProviderRemoteRegistry {
	return &SupplierProviderRemoteRegistry{
		sub2api: NewSupplierSub2APIClient(httpClient, tokenCache, turnstileSolver),
		newapi:  NewSupplierNewAPIClient(httpClient, tokenCache, turnstileSolver),
	}
}

func (r *SupplierProviderRemoteRegistry) SetAuthAuditor(auditor SupplierProviderAuthAuditor) {
	if r == nil {
		return
	}
	r.sub2api.SetAuthAuditor(auditor)
	r.newapi.SetAuthAuditor(auditor)
}

func (r *SupplierProviderRemoteRegistry) FetchAccounts(ctx context.Context, provider *SupplierProvider, password string) ([]SupplierProviderRemoteAccount, error) {
	client, err := r.client(provider)
	if err != nil {
		return nil, err
	}
	return client.FetchAccounts(ctx, provider, password)
}

func (r *SupplierProviderRemoteRegistry) FetchGroups(ctx context.Context, provider *SupplierProvider, password string) ([]SupplierProviderRemoteGroup, error) {
	client, err := r.client(provider)
	if err != nil {
		return nil, err
	}
	return client.FetchGroups(ctx, provider, password)
}

func (r *SupplierProviderRemoteRegistry) FetchBalance(ctx context.Context, provider *SupplierProvider, password string) (float64, error) {
	client, err := r.client(provider)
	if err != nil {
		return 0, err
	}
	return client.FetchBalance(ctx, provider, password)
}

func (r *SupplierProviderRemoteRegistry) FetchCost(ctx context.Context, provider *SupplierProvider, password string, day time.Time) (float64, error) {
	client, err := r.client(provider)
	if err != nil {
		return 0, err
	}
	return client.FetchCost(ctx, provider, password, day)
}

func (r *SupplierProviderRemoteRegistry) FetchRechargeAmount(ctx context.Context, provider *SupplierProvider, password string, day time.Time) (float64, error) {
	client, err := r.client(provider)
	if err != nil {
		return 0, err
	}
	rechargeClient, ok := client.(SupplierProviderRemoteRechargeClient)
	if !ok {
		return 0, fmt.Errorf("supplier provider remote client does not support recharge history")
	}
	return rechargeClient.FetchRechargeAmount(ctx, provider, password, day)
}

func (r *SupplierProviderRemoteRegistry) FetchRechargeRecords(ctx context.Context, provider *SupplierProvider, password string, start, end time.Time) ([]SupplierProviderRechargeRecord, error) {
	client, err := r.client(provider)
	if err != nil {
		return nil, err
	}
	rechargeClient, ok := client.(SupplierProviderRemoteRechargeHistoryClient)
	if !ok {
		return nil, fmt.Errorf("supplier provider remote client does not support recharge history")
	}
	return rechargeClient.FetchRechargeRecords(ctx, provider, password, start, end)
}

func (r *SupplierProviderRemoteRegistry) FetchMonitorItems(ctx context.Context, provider *SupplierProvider, password string) ([]SupplierProviderMonitorItem, error) {
	client, err := r.client(provider)
	if err != nil {
		return nil, err
	}
	monitorClient, ok := client.(SupplierProviderRemoteMonitorClient)
	if !ok {
		return nil, ErrSupplierProviderInvalid
	}
	return monitorClient.FetchMonitorItems(ctx, provider, password)
}

func (r *SupplierProviderRemoteRegistry) TestEndpoint(ctx context.Context, provider *SupplierProvider, password string, scope string) (SupplierProviderEndpointTestResult, error) {
	client, err := r.client(provider)
	if err != nil {
		return SupplierProviderEndpointTestResult{}, err
	}
	tester, ok := client.(SupplierProviderRemoteTester)
	if !ok {
		return SupplierProviderEndpointTestResult{}, fmt.Errorf("supplier provider remote client does not support endpoint test")
	}
	return tester.TestEndpoint(ctx, provider, password, scope)
}

func (r *SupplierProviderRemoteRegistry) ListUpstreamSessions(ctx context.Context, provider *SupplierProvider) (SupplierProviderUpstreamSessionSummary, error) {
	client, err := r.client(provider)
	if err != nil {
		return SupplierProviderUpstreamSessionSummary{}, err
	}
	manager, ok := client.(SupplierProviderRemoteSessionManager)
	if !ok {
		// 会话管理是 New API 特有的能力，其它供应商类型不实现该接口。
		return SupplierProviderUpstreamSessionSummary{}, ErrSupplierProviderUpstreamSessionUnsupported
	}
	return manager.ListUpstreamSessions(ctx, provider)
}

func (r *SupplierProviderRemoteRegistry) ListUpstreamSessionsWithLogin(ctx context.Context, provider *SupplierProvider, password string) (SupplierProviderUpstreamSessionSummary, error) {
	client, err := r.client(provider)
	if err != nil {
		return SupplierProviderUpstreamSessionSummary{}, err
	}
	manager, ok := client.(SupplierProviderRemoteSessionManager)
	if !ok {
		return SupplierProviderUpstreamSessionSummary{}, ErrSupplierProviderUpstreamSessionUnsupported
	}
	return manager.ListUpstreamSessionsWithLogin(ctx, provider, password)
}

func (r *SupplierProviderRemoteRegistry) RevokeOtherUpstreamSessions(ctx context.Context, provider *SupplierProvider) (int, error) {
	client, err := r.client(provider)
	if err != nil {
		return 0, err
	}
	manager, ok := client.(SupplierProviderRemoteSessionManager)
	if !ok {
		return 0, ErrSupplierProviderUpstreamSessionUnsupported
	}
	return manager.RevokeOtherUpstreamSessions(ctx, provider)
}

func (r *SupplierProviderRemoteRegistry) LastEndpointResult(providerID int64, scope string) *SupplierProviderEndpointResult {
	if r == nil {
		return nil
	}
	if result := r.sub2api.LastEndpointResult(providerID, scope); result != nil {
		return result
	}
	return r.newapi.LastEndpointResult(providerID, scope)
}

func (r *SupplierProviderRemoteRegistry) client(provider *SupplierProvider) (SupplierProviderRemoteClient, error) {
	if r == nil || provider == nil {
		return nil, ErrSupplierProviderInvalid
	}
	switch normalizeSupplierProviderType(provider.ProviderType) {
	case SupplierProviderTypeNewAPI:
		return r.newapi, nil
	case SupplierProviderTypeSub2API:
		return r.sub2api, nil
	default:
		return nil, ErrSupplierProviderInvalid
	}
}

func normalizeSupplierProviderType(providerType string) string {
	switch strings.ToLower(strings.TrimSpace(providerType)) {
	case SupplierProviderTypeNewAPI:
		return SupplierProviderTypeNewAPI
	case SupplierProviderTypeSub2API:
		return SupplierProviderTypeSub2API
	default:
		return strings.ToLower(strings.TrimSpace(providerType))
	}
}

func supplierProviderGroupIsActive(rawStatus string) bool {
	switch strings.ToLower(strings.TrimSpace(rawStatus)) {
	case "", "active", "enabled", "enable", "valid", "available", "ok", "success", "true":
		return true
	case "inactive", "disabled", "disable", "invalid", "false", "deleted", "archived", "removed":
		return false
	default:
		return true
	}
}

// normalizeSupplierNewAPIKeyStatus 将 NewAPI 上游 token 的 int/string status 归一到本系统统一状态。
// 映射：1→active, 2→disabled, 3→expired, 4→quota_exhausted，其他→unknown。
func normalizeSupplierNewAPIKeyStatus(status any) string {
	if status == nil {
		return "unknown"
	}
	switch v := status.(type) {
	case float64:
		// JSON 数字默认解码为 float64
		return normalizeSupplierNewAPIKeyStatusInt(int(v))
	case int:
		return normalizeSupplierNewAPIKeyStatusInt(v)
	case int64:
		return normalizeSupplierNewAPIKeyStatusInt(int(v))
	case string:
		s := strings.ToLower(strings.TrimSpace(v))
		switch s {
		case "active", "1":
			return "active"
		case "disabled", "2":
			return "disabled"
		case "expired", "3":
			return "expired"
		case "quota_exhausted", "4":
			return "quota_exhausted"
		default:
			return "unknown"
		}
	default:
		return "unknown"
	}
}

func normalizeSupplierNewAPIKeyStatusInt(status int) string {
	switch status {
	case 1:
		return "active"
	case 2:
		return "disabled"
	case 3:
		return "expired"
	case 4:
		return "quota_exhausted"
	default:
		return "unknown"
	}
}

// normalizeSupplierSub2APIKeyStatus 将 Sub2API 上游 key 的字符串 status 归一到本系统统一状态。
// 重点：上游停用叫 inactive，本系统统一成 disabled。
func normalizeSupplierSub2APIKeyStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "inactive":
		return "disabled"
	case "active", "disabled", "expired", "quota_exhausted":
		return strings.ToLower(strings.TrimSpace(status))
	default:
		return "unknown"
	}
}

func (r *SupplierProviderRemoteRegistry) RefreshToken(ctx context.Context, provider *SupplierProvider) (SupplierProviderAuthToken, error) {
	client, err := r.client(provider)
	if err != nil {
		return SupplierProviderAuthToken{}, err
	}
	refresher, ok := client.(SupplierProviderRemoteTokenRefresher)
	if !ok {
		return SupplierProviderAuthToken{}, fmt.Errorf("supplier provider type %s does not support manual token refresh", provider.ProviderType)
	}
	return refresher.RefreshToken(ctx, provider)
}

func (r *SupplierProviderRemoteRegistry) Reauthenticate(ctx context.Context, provider *SupplierProvider, password string) (SupplierProviderAuthToken, error) {
	client, err := r.client(provider)
	if err != nil {
		return SupplierProviderAuthToken{}, err
	}
	reauthenticator, ok := client.(SupplierProviderRemoteReauthenticator)
	if !ok {
		return SupplierProviderAuthToken{}, fmt.Errorf("supplier provider type %s does not support manual reauthentication", provider.ProviderType)
	}
	return reauthenticator.Reauthenticate(ctx, provider, password)
}

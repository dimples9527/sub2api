package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	SupplierProviderTypeSub2API = "sub2api"
	SupplierProviderTypeNewAPI  = "newapi"
)

type SupplierProviderRemoteRegistry struct {
	sub2api *SupplierSub2APIClient
	newapi  *SupplierNewAPIClient
}

func NewSupplierProviderRemoteRegistry(httpClient *http.Client, tokenCache SupplierProviderTokenCache, turnstileSolver SupplierTurnstileSolver) *SupplierProviderRemoteRegistry {
	return &SupplierProviderRemoteRegistry{
		sub2api: NewSupplierSub2APIClient(httpClient, tokenCache, turnstileSolver),
		newapi:  NewSupplierNewAPIClient(httpClient, tokenCache, turnstileSolver),
	}
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

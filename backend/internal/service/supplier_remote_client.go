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

func NewSupplierProviderRemoteRegistry(httpClient *http.Client, tokenCache SupplierProviderTokenCache) *SupplierProviderRemoteRegistry {
	return &SupplierProviderRemoteRegistry{
		sub2api: NewSupplierSub2APIClient(httpClient, tokenCache),
		newapi:  NewSupplierNewAPIClient(httpClient),
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

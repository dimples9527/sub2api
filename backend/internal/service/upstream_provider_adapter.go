package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type UpstreamProviderAdapter interface {
	FetchKeys(ctx context.Context, provider UpstreamProviderConfig) ([]UpstreamProviderKey, []string, error)
	Test(ctx context.Context, provider UpstreamProviderConfig) UpstreamProviderTestResult
}

type UpstreamProviderAdapterRegistry struct {
	adapters map[string]UpstreamProviderAdapter
}

func NewUpstreamProviderAdapterRegistry(httpClient *http.Client) *UpstreamProviderAdapterRegistry {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &UpstreamProviderAdapterRegistry{
		adapters: map[string]UpstreamProviderAdapter{
			UpstreamProviderTypeSub2API: NewSub2APIProviderAdapter(httpClient),
			UpstreamProviderTypeNewAPI:  NewNewAPIProviderAdapter(httpClient),
		},
	}
}

func (r *UpstreamProviderAdapterRegistry) Get(providerType string) (UpstreamProviderAdapter, error) {
	if r == nil {
		return nil, infraerrors.InternalServer("UPSTREAM_PROVIDER_REGISTRY_UNAVAILABLE", "upstream provider registry unavailable")
	}
	adapter, ok := r.adapters[normalizeUpstreamProviderType(providerType)]
	if !ok {
		return nil, infraerrors.BadRequest("UPSTREAM_PROVIDER_TYPE_UNSUPPORTED", "upstream provider type is unsupported")
	}
	return adapter, nil
}

func newUpstreamProviderTestResult(provider UpstreamProviderConfig) UpstreamProviderTestResult {
	return UpstreamProviderTestResult{
		Type:              provider.Type,
		Slug:              provider.Slug,
		Name:              provider.Name,
		BaseURL:           provider.BaseURL,
		LoginURL:          provider.LoginURL,
		KeysURL:           provider.APIKeysURL,
		GroupsURL:         provider.GroupsURL,
		AccountNamePrefix: provider.AccountNamePrefix,
		ParsedKeys:        []UpstreamProviderKey{},
	}
}

func upstreamProviderURL(provider UpstreamProviderConfig, path string) string {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return strings.TrimRight(provider.BaseURL, "/")
	}
	if parsed, err := url.Parse(trimmedPath); err == nil && parsed.IsAbs() {
		return trimmedPath
	}
	base := strings.TrimRight(provider.BaseURL, "/")
	if strings.HasPrefix(trimmedPath, "/") {
		return base + trimmedPath
	}
	return base + "/" + trimmedPath
}

func upstreamProviderHTTPError(label string, status int, payload []byte) error {
	body := strings.TrimSpace(string(payload))
	if len(body) > 500 {
		body = body[:500]
	}
	return fmt.Errorf("%s failed: HTTP %d: %s", label, status, body)
}

func normalizeUpstreamProviderGroupName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(name)), " "))
}

func upstreamProviderCookiesHeader(cookies []*http.Cookie) string {
	parts := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		if cookie == nil || strings.TrimSpace(cookie.Name) == "" {
			continue
		}
		parts = append(parts, cookie.Name+"="+cookie.Value)
	}
	return strings.Join(parts, "; ")
}

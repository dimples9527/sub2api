package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type UpstreamProviderAdapter interface {
	FetchKeys(ctx context.Context, provider UpstreamProviderConfig) ([]UpstreamProviderKey, []string, error)
	FetchBalance(ctx context.Context, provider UpstreamProviderConfig) (UpstreamProviderBalance, error)
	FetchTodayCost(ctx context.Context, provider UpstreamProviderConfig, day time.Time) (UpstreamProviderCost, error)
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
		Type:               provider.Type,
		Slug:               provider.Slug,
		Name:               provider.Name,
		BaseURL:            provider.BaseURL,
		LoginURL:           provider.LoginURL,
		KeysURL:            provider.APIKeysURL,
		GroupsURL:          provider.GroupsURL,
		AvailableGroupsURL: provider.AvailableGroupsURL,
		AccountNamePrefix:  provider.AccountNamePrefix,
		ParsedKeys:         []UpstreamProviderKey{},
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

func upstreamProviderUsageCostURL(path string, day time.Time, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	localDay := day.In(loc)
	start := time.Date(localDay.Year(), localDay.Month(), localDay.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour).Add(-time.Second)
	startText := strconv.FormatInt(start.Unix(), 10)
	endText := strconv.FormatInt(end.Unix(), 10)

	out := strings.TrimSpace(path)
	out = strings.ReplaceAll(out, "{start_timestamp}", startText)
	out = strings.ReplaceAll(out, "{end_timestamp}", endText)
	if out == "" {
		return out
	}
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

func upstreamProviderKeyName(provider UpstreamProviderConfig, name string) string {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return ""
	}
	prefix := strings.TrimSpace(provider.AccountNamePrefix)
	if prefix == "" {
		return trimmedName
	}
	if strings.HasSuffix(prefix, "-") {
		return prefix + trimmedName
	}
	if strings.HasPrefix(trimmedName, "-") {
		return prefix + trimmedName
	}
	return prefix + "-" + trimmedName
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

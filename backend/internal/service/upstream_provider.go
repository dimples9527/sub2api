package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	SettingKeyUpstreamProviderConfigs = "upstream_provider_configs"

	UpstreamProviderTypeSub2API = "sub2api"
	UpstreamProviderTypeNewAPI  = "newapi"
)

var upstreamProviderSlugPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

type UpstreamProviderConfig struct {
	Type                       string  `json:"type"`
	Slug                       string  `json:"slug"`
	Name                       string  `json:"name"`
	SortOrder                  int     `json:"sort_order"`
	Enabled                    bool    `json:"enabled"`
	IsDefault                  bool    `json:"is_default"`
	BaseURL                    string  `json:"base_url"`
	LoginURL                   string  `json:"login_url"`
	APIKeysURL                 string  `json:"api_keys_url"`
	GroupsURL                  string  `json:"groups_url"`
	AvailableGroupsURL         string  `json:"available_groups_url"`
	BalanceURL                 string  `json:"balance_url"`
	UsageCostURL               string  `json:"usage_cost_url"`
	Email                      string  `json:"email"`
	Username                   string  `json:"username"`
	Password                   string  `json:"password,omitempty"`
	PasswordConfigured         bool    `json:"password_configured,omitempty"`
	AccountNamePrefix          string  `json:"account_name_prefix"`
	TempDisableMinutes         int     `json:"temp_disable_minutes,omitempty"`
	AccountRateMultiplierScale float64 `json:"account_rate_multiplier_scale"`
}

type UpstreamProviderKey struct {
	ProviderSlug   string  `json:"provider_slug"`
	ProviderName   string  `json:"provider_name"`
	ProviderType   string  `json:"provider_type"`
	KeyName        string  `json:"key_name"`
	APIKey         string  `json:"api_key,omitempty"`
	GroupName      string  `json:"group_name"`
	RateMultiplier float64 `json:"rate_multiplier"`
	RawStatus      string  `json:"raw_status,omitempty"`
	RawGroupID     string  `json:"raw_group_id,omitempty"`
}

type UpstreamProviderGroup struct {
	ProviderSlug   string  `json:"provider_slug"`
	ProviderName   string  `json:"provider_name"`
	ProviderType   string  `json:"provider_type"`
	GroupName      string  `json:"group_name"`
	RateMultiplier float64 `json:"rate_multiplier"`
	RawStatus      string  `json:"raw_status,omitempty"`
	RawGroupID     string  `json:"raw_group_id,omitempty"`
}

type UpstreamProviderBalance struct {
	ProviderSlug string  `json:"provider_slug"`
	ProviderName string  `json:"provider_name"`
	ProviderType string  `json:"provider_type"`
	Balance      float64 `json:"balance"`
}

type UpstreamProviderCost struct {
	ProviderSlug string  `json:"provider_slug"`
	ProviderName string  `json:"provider_name"`
	ProviderType string  `json:"provider_type"`
	TodayCost    float64 `json:"today_cost"`
}

type UpstreamProviderBalanceStatus struct {
	ProviderSlug string  `json:"provider_slug"`
	ProviderName string  `json:"provider_name"`
	ProviderType string  `json:"provider_type"`
	Balance      float64 `json:"balance"`
	TodayCost    float64 `json:"today_cost"`
}

type UpstreamProviderTestStage struct {
	OK            bool   `json:"ok"`
	StatusCode    int    `json:"status_code,omitempty"`
	UserID        int64  `json:"user_id,omitempty"`
	CookiePresent bool   `json:"cookie_present,omitempty"`
	ItemCount     int    `json:"item_count,omitempty"`
	GroupCount    int    `json:"group_count,omitempty"`
	Error         string `json:"error,omitempty"`
}

type UpstreamProviderTestResult struct {
	Type               string                     `json:"type"`
	Slug               string                     `json:"slug"`
	Name               string                     `json:"name"`
	BaseURL            string                     `json:"base_url"`
	LoginURL           string                     `json:"login_url"`
	KeysURL            string                     `json:"keys_url"`
	GroupsURL          string                     `json:"groups_url,omitempty"`
	AvailableGroupsURL string                     `json:"available_groups_url,omitempty"`
	AccountNamePrefix  string                     `json:"account_name_prefix"`
	Login              UpstreamProviderTestStage  `json:"login"`
	Keys               UpstreamProviderTestStage  `json:"keys"`
	Groups             *UpstreamProviderTestStage `json:"groups,omitempty"`
	ParsedKeys         []UpstreamProviderKey      `json:"parsed_keys"`
	Warnings           []string                   `json:"warnings,omitempty"`
}

type UpstreamProviderService struct {
	settingRepo SettingRepository
	registry    *UpstreamProviderAdapterRegistry
}

func NewUpstreamProviderService(settingRepo SettingRepository) *UpstreamProviderService {
	return NewUpstreamProviderServiceWithHTTPClient(settingRepo, http.DefaultClient)
}

func NewUpstreamProviderServiceWithHTTPClient(settingRepo SettingRepository, httpClient *http.Client) *UpstreamProviderService {
	return &UpstreamProviderService{
		settingRepo: settingRepo,
		registry:    NewUpstreamProviderAdapterRegistry(httpClient),
	}
}

func (s *UpstreamProviderService) ListProviders(ctx context.Context) ([]UpstreamProviderConfig, error) {
	providers, err := s.loadProviders(ctx)
	if err != nil {
		return nil, err
	}
	return redactUpstreamProviders(providers), nil
}

func (s *UpstreamProviderService) CreateProvider(ctx context.Context, input UpstreamProviderConfig) (UpstreamProviderConfig, error) {
	providers, err := s.loadProviders(ctx)
	if err != nil {
		return UpstreamProviderConfig{}, err
	}
	next := normalizeUpstreamProvider(input)
	if err := s.validateProvider(next, false); err != nil {
		return UpstreamProviderConfig{}, err
	}
	for _, provider := range providers {
		if provider.Slug == next.Slug {
			return UpstreamProviderConfig{}, infraerrors.Conflict("UPSTREAM_PROVIDER_EXISTS", "upstream provider slug already exists")
		}
	}
	providers = append(providers, next)
	normalizeUpstreamProviderDefaults(providers, next.Slug)
	if err := s.saveProviders(ctx, providers); err != nil {
		return UpstreamProviderConfig{}, err
	}
	return redactUpstreamProvider(next), nil
}

func (s *UpstreamProviderService) UpdateProvider(ctx context.Context, slug string, input UpstreamProviderConfig) (UpstreamProviderConfig, error) {
	slug = strings.TrimSpace(slug)
	providers, err := s.loadProviders(ctx)
	if err != nil {
		return UpstreamProviderConfig{}, err
	}
	index := -1
	for i := range providers {
		if providers[i].Slug == slug {
			index = i
			break
		}
	}
	if index < 0 {
		return UpstreamProviderConfig{}, infraerrors.NotFound("UPSTREAM_PROVIDER_NOT_FOUND", "upstream provider not found")
	}
	next := normalizeUpstreamProvider(input)
	if next.Slug == "" {
		next.Slug = slug
	}
	if next.Slug != slug {
		return UpstreamProviderConfig{}, infraerrors.BadRequest("UPSTREAM_PROVIDER_SLUG_IMMUTABLE", "upstream provider slug cannot be changed")
	}
	if next.Password == "" {
		next.Password = providers[index].Password
	}
	if providers[index].IsDefault {
		next.IsDefault = true
	}
	if err := s.validateProvider(next, next.Password != ""); err != nil {
		return UpstreamProviderConfig{}, err
	}
	providers[index] = next
	normalizeUpstreamProviderDefaults(providers, next.Slug)
	if err := s.saveProviders(ctx, providers); err != nil {
		return UpstreamProviderConfig{}, err
	}
	return redactUpstreamProvider(next), nil
}

func (s *UpstreamProviderService) DeleteProvider(ctx context.Context, slug string) error {
	slug = strings.TrimSpace(slug)
	providers, err := s.loadProviders(ctx)
	if err != nil {
		return err
	}
	out := providers[:0]
	deleted := false
	for _, provider := range providers {
		if provider.Slug == slug {
			deleted = true
			continue
		}
		out = append(out, provider)
	}
	if !deleted {
		return infraerrors.NotFound("UPSTREAM_PROVIDER_NOT_FOUND", "upstream provider not found")
	}
	return s.saveProviders(ctx, out)
}

func (s *UpstreamProviderService) SetDefaultProvider(ctx context.Context, slug string) (UpstreamProviderConfig, error) {
	slug = strings.TrimSpace(slug)
	providers, err := s.loadProviders(ctx)
	if err != nil {
		return UpstreamProviderConfig{}, err
	}
	index := -1
	for i := range providers {
		providers[i].IsDefault = false
		if providers[i].Slug == slug {
			index = i
		}
	}
	if index < 0 {
		return UpstreamProviderConfig{}, infraerrors.NotFound("UPSTREAM_PROVIDER_NOT_FOUND", "upstream provider not found")
	}
	providers[index].IsDefault = true
	if err := s.saveProviders(ctx, providers); err != nil {
		return UpstreamProviderConfig{}, err
	}
	return redactUpstreamProvider(providers[index]), nil
}

func (s *UpstreamProviderService) GetDefaultProvider(ctx context.Context) (UpstreamProviderConfig, error) {
	providers, err := s.loadProviders(ctx)
	if err != nil {
		return UpstreamProviderConfig{}, err
	}
	for _, provider := range providers {
		if provider.IsDefault {
			return provider, nil
		}
	}
	return UpstreamProviderConfig{}, infraerrors.NotFound("UPSTREAM_PROVIDER_DEFAULT_NOT_CONFIGURED", "default upstream provider is not configured")
}

func (s *UpstreamProviderService) TestProvider(ctx context.Context, slug string) (UpstreamProviderTestResult, error) {
	provider, err := s.getStoredProvider(ctx, slug)
	if err != nil {
		return UpstreamProviderTestResult{}, err
	}
	return s.TestProviderConfig(ctx, provider)
}

func (s *UpstreamProviderService) TestProviderConfig(ctx context.Context, provider UpstreamProviderConfig) (UpstreamProviderTestResult, error) {
	provider = normalizeUpstreamProvider(provider)
	if err := s.validateProvider(provider, provider.Password != ""); err != nil {
		return UpstreamProviderTestResult{}, err
	}
	adapter, err := s.registry.Get(provider.Type)
	if err != nil {
		return UpstreamProviderTestResult{}, err
	}
	return adapter.Test(ctx, provider), nil
}

func (s *UpstreamProviderService) FetchProviderKeys(ctx context.Context, slug string) ([]UpstreamProviderKey, []string, error) {
	provider, err := s.getStoredProvider(ctx, slug)
	if err != nil {
		return nil, nil, err
	}
	adapter, err := s.registry.Get(provider.Type)
	if err != nil {
		return nil, nil, err
	}
	return adapter.FetchKeys(ctx, provider)
}

func (s *UpstreamProviderService) FetchProviderBalance(ctx context.Context, slug string) (UpstreamProviderBalance, error) {
	provider, err := s.getStoredProvider(ctx, slug)
	if err != nil {
		return UpstreamProviderBalance{}, err
	}
	adapter, err := s.registry.Get(provider.Type)
	if err != nil {
		return UpstreamProviderBalance{}, err
	}
	return adapter.FetchBalance(ctx, provider)
}

func (s *UpstreamProviderService) FetchProviderBalanceStatus(ctx context.Context, slug string) (UpstreamProviderBalanceStatus, error) {
	provider, err := s.getStoredProvider(ctx, slug)
	if err != nil {
		return UpstreamProviderBalanceStatus{}, err
	}
	adapter, err := s.registry.Get(provider.Type)
	if err != nil {
		return UpstreamProviderBalanceStatus{}, err
	}
	balance, err := adapter.FetchBalance(ctx, provider)
	if err != nil {
		return UpstreamProviderBalanceStatus{}, err
	}
	cost, err := adapter.FetchTodayCost(ctx, provider, time.Now())
	if err != nil {
		return UpstreamProviderBalanceStatus{}, err
	}
	status := UpstreamProviderBalanceStatus{
		ProviderSlug: balance.ProviderSlug,
		ProviderName: balance.ProviderName,
		ProviderType: balance.ProviderType,
		Balance:      balance.Balance,
		TodayCost:    cost.TodayCost,
	}
	if status.ProviderSlug == "" {
		status.ProviderSlug = cost.ProviderSlug
	}
	if status.ProviderName == "" {
		status.ProviderName = cost.ProviderName
	}
	if status.ProviderType == "" {
		status.ProviderType = cost.ProviderType
	}
	return status, nil
}

func (s *UpstreamProviderService) FetchProviderTodayCost(ctx context.Context, slug string, day time.Time) (UpstreamProviderCost, error) {
	provider, err := s.getStoredProvider(ctx, slug)
	if err != nil {
		return UpstreamProviderCost{}, err
	}
	adapter, err := s.registry.Get(provider.Type)
	if err != nil {
		return UpstreamProviderCost{}, err
	}
	return adapter.FetchTodayCost(ctx, provider, day)
}

func (s *UpstreamProviderService) FetchProviderGroups(ctx context.Context, slug string) ([]UpstreamProviderGroup, []string, error) {
	provider, err := s.getStoredProvider(ctx, slug)
	if err != nil {
		return nil, nil, err
	}
	adapter, err := s.registry.Get(provider.Type)
	if err != nil {
		return nil, nil, err
	}
	groupAdapter, ok := adapter.(interface {
		FetchGroups(context.Context, UpstreamProviderConfig) ([]UpstreamProviderGroup, []string, error)
	})
	if !ok {
		return nil, nil, infraerrors.BadRequest("UPSTREAM_PROVIDER_GROUPS_UNSUPPORTED", "upstream provider groups are unsupported")
	}
	return groupAdapter.FetchGroups(ctx, provider)
}

func (s *UpstreamProviderService) getStoredProvider(ctx context.Context, slug string) (UpstreamProviderConfig, error) {
	slug = strings.TrimSpace(slug)
	providers, err := s.loadProviders(ctx)
	if err != nil {
		return UpstreamProviderConfig{}, err
	}
	for _, provider := range providers {
		if provider.Slug == slug {
			return provider, nil
		}
	}
	return UpstreamProviderConfig{}, infraerrors.NotFound("UPSTREAM_PROVIDER_NOT_FOUND", "upstream provider not found")
}

func (s *UpstreamProviderService) loadProviders(ctx context.Context) ([]UpstreamProviderConfig, error) {
	if s == nil || s.settingRepo == nil {
		return nil, infraerrors.InternalServer("UPSTREAM_PROVIDER_STORE_UNAVAILABLE", "upstream provider store unavailable")
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyUpstreamProviderConfigs)
	if err != nil {
		if err == ErrSettingNotFound {
			return []UpstreamProviderConfig{}, nil
		}
		return nil, fmt.Errorf("load upstream provider configs: %w", err)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []UpstreamProviderConfig{}, nil
	}
	var providers []UpstreamProviderConfig
	if err := json.Unmarshal([]byte(raw), &providers); err != nil {
		return nil, infraerrors.InternalServer("UPSTREAM_PROVIDER_CONFIG_INVALID", "upstream provider config is invalid")
	}
	out := make([]UpstreamProviderConfig, 0, len(providers))
	for _, provider := range providers {
		out = append(out, normalizeUpstreamProvider(provider))
	}
	normalizeUpstreamProviderDefaults(out, "")
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].IsDefault != out[j].IsDefault {
			return out[i].IsDefault
		}
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].Slug < out[j].Slug
	})
	return out, nil
}

func (s *UpstreamProviderService) saveProviders(ctx context.Context, providers []UpstreamProviderConfig) error {
	out := make([]UpstreamProviderConfig, 0, len(providers))
	for _, provider := range providers {
		out = append(out, normalizeUpstreamProvider(provider))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].IsDefault != out[j].IsDefault {
			return out[i].IsDefault
		}
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].Slug < out[j].Slug
	})
	raw, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("marshal upstream provider configs: %w", err)
	}
	return s.settingRepo.Set(ctx, SettingKeyUpstreamProviderConfigs, string(raw))
}

func (s *UpstreamProviderService) validateProvider(provider UpstreamProviderConfig, passwordAlreadyConfigured bool) error {
	if !upstreamProviderSlugPattern.MatchString(provider.Slug) {
		return infraerrors.BadRequest("UPSTREAM_PROVIDER_INVALID_SLUG", "upstream provider slug must be 1-64 chars and contain only letters, numbers, _ or -")
	}
	if provider.Name == "" {
		return infraerrors.BadRequest("UPSTREAM_PROVIDER_NAME_REQUIRED", "upstream provider name is required")
	}
	if provider.BaseURL == "" {
		return infraerrors.BadRequest("UPSTREAM_PROVIDER_BASE_URL_REQUIRED", "upstream provider base_url is required")
	}
	if provider.APIKeysURL == "" {
		return infraerrors.BadRequest("UPSTREAM_PROVIDER_KEYS_URL_REQUIRED", "upstream provider api_keys_url is required")
	}
	if provider.AccountRateMultiplierScale < 0 {
		return infraerrors.BadRequest("UPSTREAM_PROVIDER_ACCOUNT_RATE_MULTIPLIER_SCALE_INVALID", "upstream provider account_rate_multiplier_scale must be greater than 0")
	}
	if _, err := s.registry.Get(provider.Type); err != nil {
		return err
	}
	switch provider.Type {
	case UpstreamProviderTypeNewAPI:
		if provider.LoginURL == "" {
			return infraerrors.BadRequest("UPSTREAM_PROVIDER_LOGIN_URL_REQUIRED", "newapi provider login_url is required")
		}
		if provider.GroupsURL == "" {
			return infraerrors.BadRequest("UPSTREAM_PROVIDER_GROUPS_URL_REQUIRED", "newapi provider groups_url is required")
		}
		if provider.Username == "" && provider.Email == "" {
			return infraerrors.BadRequest("UPSTREAM_PROVIDER_USERNAME_REQUIRED", "newapi provider username or email is required")
		}
		if provider.Password == "" && !passwordAlreadyConfigured {
			return infraerrors.BadRequest("UPSTREAM_PROVIDER_PASSWORD_REQUIRED", "newapi provider password is required")
		}
	}
	return nil
}

func normalizeUpstreamProvider(provider UpstreamProviderConfig) UpstreamProviderConfig {
	provider.Type = normalizeUpstreamProviderType(provider.Type)
	provider.Slug = strings.TrimSpace(provider.Slug)
	provider.Name = strings.TrimSpace(provider.Name)
	provider.BaseURL = strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	provider.LoginURL = strings.TrimSpace(provider.LoginURL)
	provider.APIKeysURL = strings.TrimSpace(provider.APIKeysURL)
	provider.GroupsURL = strings.TrimSpace(provider.GroupsURL)
	provider.AvailableGroupsURL = strings.TrimSpace(provider.AvailableGroupsURL)
	provider.BalanceURL = strings.TrimSpace(provider.BalanceURL)
	provider.UsageCostURL = strings.TrimSpace(provider.UsageCostURL)
	provider.Email = strings.TrimSpace(provider.Email)
	provider.Username = strings.TrimSpace(provider.Username)
	provider.AccountNamePrefix = strings.TrimSpace(provider.AccountNamePrefix)
	if provider.SortOrder < 0 {
		provider.SortOrder = 0
	}
	if provider.TempDisableMinutes < 0 {
		provider.TempDisableMinutes = 0
	}
	if provider.AccountRateMultiplierScale == 0 {
		provider.AccountRateMultiplierScale = 1
	}
	return provider
}

func normalizeUpstreamProviderDefaults(providers []UpstreamProviderConfig, preferredSlug string) {
	preferredSlug = strings.TrimSpace(preferredSlug)
	defaultSlug := ""
	for _, provider := range providers {
		if provider.IsDefault {
			defaultSlug = provider.Slug
		}
	}
	if preferredSlug != "" {
		for _, provider := range providers {
			if provider.Slug == preferredSlug && provider.IsDefault {
				defaultSlug = preferredSlug
				break
			}
		}
	}
	if defaultSlug == "" {
		return
	}
	for i := range providers {
		providers[i].IsDefault = providers[i].Slug == defaultSlug
	}
}

func normalizeUpstreamProviderType(providerType string) string {
	switch strings.ToLower(strings.TrimSpace(providerType)) {
	case UpstreamProviderTypeNewAPI:
		return UpstreamProviderTypeNewAPI
	default:
		return UpstreamProviderTypeSub2API
	}
}

func redactUpstreamProviders(providers []UpstreamProviderConfig) []UpstreamProviderConfig {
	out := make([]UpstreamProviderConfig, 0, len(providers))
	for _, provider := range providers {
		out = append(out, redactUpstreamProvider(provider))
	}
	return out
}

func redactUpstreamProvider(provider UpstreamProviderConfig) UpstreamProviderConfig {
	provider.PasswordConfigured = strings.TrimSpace(provider.Password) != "" || provider.PasswordConfigured
	provider.Password = ""
	return provider
}

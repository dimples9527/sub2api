package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrSupplierProviderNotFound = infraerrors.NotFound("SUPPLIER_PROVIDER_NOT_FOUND", "supplier provider not found")
	ErrSupplierProviderExists   = infraerrors.Conflict("SUPPLIER_PROVIDER_EXISTS", "supplier provider code already exists")
	ErrSupplierProviderInvalid  = infraerrors.BadRequest("SUPPLIER_PROVIDER_INVALID", "invalid supplier provider configuration")

	ErrSupplierProviderTypeNotFound = infraerrors.NotFound("SUPPLIER_PROVIDER_TYPE_NOT_FOUND", "supplier provider type not found")
	ErrSupplierProviderTypeExists   = infraerrors.Conflict("SUPPLIER_PROVIDER_TYPE_EXISTS", "supplier provider type code already exists")
	ErrSupplierProviderTypeInvalid  = infraerrors.BadRequest("SUPPLIER_PROVIDER_TYPE_INVALID", "invalid supplier provider type configuration")
)

var supplierProviderCodePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

type SupplierProvider struct {
	ID                         int64      `json:"id"`
	Code                       string     `json:"code"`
	Name                       string     `json:"name"`
	ProviderType               string     `json:"provider_type"`
	BaseURL                    string     `json:"base_url"`
	LoginURL                   string     `json:"login_url"`
	APIKeysURL                 string     `json:"api_keys_url"`
	GroupsURL                  string     `json:"groups_url"`
	AvailableGroupsURL         string     `json:"available_groups_url"`
	BalanceURL                 string     `json:"balance_url"`
	UsageCostURL               string     `json:"usage_cost_url"`
	AccountNamePrefix          string     `json:"account_name_prefix"`
	TempDisableMinutes         int        `json:"temp_disable_minutes"`
	AccountRateMultiplierScale float64    `json:"account_rate_multiplier_scale"`
	SortOrder                  int        `json:"sort_order"`
	Enabled                    bool       `json:"enabled"`
	IsDefault                  bool       `json:"is_default"`
	Email                      string     `json:"email"`
	Username                   string     `json:"username"`
	PasswordEncrypted          string     `json:"-"`
	CredentialConfigured       bool       `json:"credential_configured"`
	Status                     string     `json:"status"`
	RiskLevel                  string     `json:"risk_level"`
	ValidAccountCount          int        `json:"valid_account_count"`
	SchedulableAccountCount    int        `json:"schedulable_account_count"`
	RequestCount               int64      `json:"request_count"`
	SuccessRate                float64    `json:"success_rate"`
	PeriodCost                 float64    `json:"period_cost"`
	CurrentBalance             float64    `json:"current_balance"`
	TodayCost                  float64    `json:"today_cost"`
	EstimatedDays              *float64   `json:"estimated_days,omitempty"`
	RateRiskCount              int        `json:"rate_risk_count"`
	SyncStatus                 string     `json:"sync_status"`
	SyncMessage                string     `json:"sync_message"`
	LastSyncAt                 *time.Time `json:"last_sync_at,omitempty"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
}

type SupplierProviderListParams struct {
	Search   string
	Enabled  *bool
	Page     int
	PageSize int
}

type SupplierProviderUpsertParams struct {
	Code                       string  `json:"code"`
	Name                       string  `json:"name"`
	ProviderType               string  `json:"provider_type"`
	BaseURL                    string  `json:"base_url"`
	LoginURL                   string  `json:"login_url"`
	APIKeysURL                 string  `json:"api_keys_url"`
	GroupsURL                  string  `json:"groups_url"`
	AvailableGroupsURL         string  `json:"available_groups_url"`
	BalanceURL                 string  `json:"balance_url"`
	UsageCostURL               string  `json:"usage_cost_url"`
	Email                      string  `json:"email"`
	Username                   string  `json:"username"`
	Password                   string  `json:"password"`
	AccountNamePrefix          string  `json:"account_name_prefix"`
	TempDisableMinutes         int     `json:"temp_disable_minutes"`
	AccountRateMultiplierScale float64 `json:"account_rate_multiplier_scale"`
	SortOrder                  int     `json:"sort_order"`
	Enabled                    bool    `json:"enabled"`
	IsDefault                  bool    `json:"is_default"`
}

type SupplierProviderType struct {
	ID                 int64     `json:"id"`
	Code               string    `json:"code"`
	Name               string    `json:"name"`
	LoginURL           string    `json:"login_url"`
	APIKeysURL         string    `json:"api_keys_url"`
	GroupsURL          string    `json:"groups_url"`
	AvailableGroupsURL string    `json:"available_groups_url"`
	BalanceURL         string    `json:"balance_url"`
	UsageCostURL       string    `json:"usage_cost_url"`
	Enabled            bool      `json:"enabled"`
	SortOrder          int       `json:"sort_order"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type SupplierProviderTypeUpsertParams struct {
	Code               string `json:"code"`
	Name               string `json:"name"`
	LoginURL           string `json:"login_url"`
	APIKeysURL         string `json:"api_keys_url"`
	GroupsURL          string `json:"groups_url"`
	AvailableGroupsURL string `json:"available_groups_url"`
	BalanceURL         string `json:"balance_url"`
	UsageCostURL       string `json:"usage_cost_url"`
	Enabled            bool   `json:"enabled"`
	SortOrder          int    `json:"sort_order"`
}

type SupplierProviderSummary struct {
	TotalCount       int64 `json:"total_count"`
	EnabledCount     int   `json:"enabled_count"`
	HighRiskCount    int   `json:"high_risk_count"`
	LowBalanceCount  int   `json:"low_balance_count"`
	SyncFailureCount int   `json:"sync_failure_count"`
	RateRiskCount    int   `json:"rate_risk_count"`
}

type SupplierProviderListResult struct {
	Items    []*SupplierProvider     `json:"items"`
	Summary  SupplierProviderSummary `json:"summary"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
}

type SupplierProviderRepository interface {
	List(ctx context.Context, params SupplierProviderListParams) ([]*SupplierProvider, int64, error)
	Summary(ctx context.Context, params SupplierProviderListParams) (SupplierProviderSummary, error)
	GetByID(ctx context.Context, id int64) (*SupplierProvider, error)
	Create(ctx context.Context, provider *SupplierProvider) error
	Update(ctx context.Context, provider *SupplierProvider) error
	Delete(ctx context.Context, id int64) error
	SetDefault(ctx context.Context, id int64) (*SupplierProvider, error)
}

type SupplierProviderTypeRepository interface {
	List(ctx context.Context, enabledOnly bool) ([]*SupplierProviderType, error)
	GetByID(ctx context.Context, id int64) (*SupplierProviderType, error)
	GetByCode(ctx context.Context, code string) (*SupplierProviderType, error)
	Create(ctx context.Context, providerType *SupplierProviderType) error
	Update(ctx context.Context, providerType *SupplierProviderType) error
	Delete(ctx context.Context, id int64) error
}

type SupplierProviderService struct {
	repo       SupplierProviderRepository
	encryptor  SecretEncryptor
	typeRepo   SupplierProviderTypeRepository
	tokenCache SupplierProviderTokenCache
}

func NewSupplierProviderService(repo SupplierProviderRepository, encryptor SecretEncryptor, typeRepo ...SupplierProviderTypeRepository) *SupplierProviderService {
	service := &SupplierProviderService{repo: repo, encryptor: encryptor}
	if len(typeRepo) > 0 {
		service.typeRepo = typeRepo[0]
	}
	return service
}

func (s *SupplierProviderService) SetTokenCache(cache SupplierProviderTokenCache) {
	s.tokenCache = cache
}

type SupplierProviderTypeService struct {
	repo SupplierProviderTypeRepository
}

func NewSupplierProviderTypeService(repo SupplierProviderTypeRepository) *SupplierProviderTypeService {
	return &SupplierProviderTypeService{repo: repo}
}

func (s *SupplierProviderTypeService) List(ctx context.Context, enabledOnly bool) ([]*SupplierProviderType, error) {
	items, err := s.repo.List(ctx, enabledOnly)
	if err != nil {
		return nil, fmt.Errorf("list supplier provider types: %w", err)
	}
	return items, nil
}

func (s *SupplierProviderTypeService) Get(ctx context.Context, id int64) (*SupplierProviderType, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *SupplierProviderTypeService) Create(ctx context.Context, params SupplierProviderTypeUpsertParams) (*SupplierProviderType, error) {
	providerType, err := buildSupplierProviderType(params)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, providerType); err != nil {
		return nil, fmt.Errorf("create supplier provider type: %w", err)
	}
	return s.Get(ctx, providerType.ID)
}

func (s *SupplierProviderTypeService) Update(ctx context.Context, id int64, params SupplierProviderTypeUpsertParams) (*SupplierProviderType, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	providerType, err := buildSupplierProviderType(params)
	if err != nil {
		return nil, err
	}
	providerType.ID = id
	providerType.CreatedAt = existing.CreatedAt
	if err := s.repo.Update(ctx, providerType); err != nil {
		return nil, fmt.Errorf("update supplier provider type: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *SupplierProviderTypeService) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete supplier provider type: %w", err)
	}
	return nil
}

func (s *SupplierProviderService) List(ctx context.Context, params SupplierProviderListParams) (SupplierProviderListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 200 {
		params.PageSize = 100
	}
	items, total, err := s.repo.List(ctx, params)
	if err != nil {
		return SupplierProviderListResult{}, fmt.Errorf("list supplier providers: %w", err)
	}
	summary, err := s.repo.Summary(ctx, params)
	if err != nil {
		return SupplierProviderListResult{}, fmt.Errorf("summarize supplier providers: %w", err)
	}
	if summary.TotalCount == 0 && total > 0 {
		summary.TotalCount = total
	}
	result := SupplierProviderListResult{Items: items, Summary: summary, Total: total, Page: params.Page, PageSize: params.PageSize}
	for _, item := range items {
		redactSupplierProvider(item)
	}
	return result, nil
}

func (s *SupplierProviderService) Get(ctx context.Context, id int64) (*SupplierProvider, error) {
	provider, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	redactSupplierProvider(provider)
	return provider, nil
}

func (s *SupplierProviderService) Create(ctx context.Context, params SupplierProviderUpsertParams) (*SupplierProvider, error) {
	if err := s.applyTypeTemplate(ctx, &params); err != nil {
		return nil, err
	}
	provider, err := s.buildProvider(params)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.Password) != "" {
		provider.PasswordEncrypted = strings.TrimSpace(params.Password)
	}
	if err := s.repo.Create(ctx, provider); err != nil {
		return nil, fmt.Errorf("create supplier provider: %w", err)
	}
	return s.Get(ctx, provider.ID)
}

func (s *SupplierProviderService) Update(ctx context.Context, id int64, params SupplierProviderUpsertParams) (*SupplierProvider, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.applyTypeTemplate(ctx, &params); err != nil {
		return nil, err
	}
	provider, err := s.buildProvider(params)
	if err != nil {
		return nil, err
	}
	provider.ID = id
	provider.CreatedAt = existing.CreatedAt
	provider.PasswordEncrypted = existing.PasswordEncrypted
	if strings.TrimSpace(params.Password) != "" {
		provider.PasswordEncrypted = strings.TrimSpace(params.Password)
	}
	if s.authConfigurationChanged(existing, provider) {
		if err := s.deleteToken(ctx, id); err != nil {
			return nil, err
		}
	}
	if err := s.repo.Update(ctx, provider); err != nil {
		return nil, fmt.Errorf("update supplier provider: %w", err)
	}
	return s.Get(ctx, provider.ID)
}

func (s *SupplierProviderService) Delete(ctx context.Context, id int64) error {
	if err := s.deleteToken(ctx, id); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete supplier provider: %w", err)
	}
	return nil
}

func (s *SupplierProviderService) SetDefault(ctx context.Context, id int64) (*SupplierProvider, error) {
	provider, err := s.repo.SetDefault(ctx, id)
	if err != nil {
		return nil, err
	}
	redactSupplierProvider(provider)
	return provider, nil
}

func (s *SupplierProviderService) buildProvider(params SupplierProviderUpsertParams) (*SupplierProvider, error) {
	params.Code = strings.TrimSpace(params.Code)
	params.Name = strings.TrimSpace(params.Name)
	params.ProviderType = strings.TrimSpace(params.ProviderType)
	params.BaseURL = strings.TrimRight(strings.TrimSpace(params.BaseURL), "/")
	params.AvailableGroupsURL = params.GroupsURL
	if !supplierProviderCodePattern.MatchString(params.Code) || params.Name == "" || params.ProviderType == "" || !validSupplierURL(params.BaseURL, true) {
		return nil, ErrSupplierProviderInvalid
	}
	for _, value := range []string{params.LoginURL, params.APIKeysURL, params.GroupsURL, params.AvailableGroupsURL, params.BalanceURL, params.UsageCostURL} {
		if !validSupplierEndpointURL(value) {
			return nil, ErrSupplierProviderInvalid
		}
	}
	if params.TempDisableMinutes < 0 {
		return nil, ErrSupplierProviderInvalid
	}
	if params.AccountRateMultiplierScale <= 0 {
		params.AccountRateMultiplierScale = 1
	}
	email := strings.TrimSpace(params.Email)
	username := strings.TrimSpace(params.Username)
	if strings.EqualFold(params.ProviderType, "sub2api") {
		username = ""
	}
	return &SupplierProvider{Code: params.Code, Name: params.Name, ProviderType: params.ProviderType, BaseURL: params.BaseURL, LoginURL: strings.TrimSpace(params.LoginURL), APIKeysURL: strings.TrimSpace(params.APIKeysURL), GroupsURL: strings.TrimSpace(params.GroupsURL), AvailableGroupsURL: strings.TrimSpace(params.AvailableGroupsURL), BalanceURL: strings.TrimSpace(params.BalanceURL), UsageCostURL: strings.TrimSpace(params.UsageCostURL), Email: email, Username: username, AccountNamePrefix: strings.TrimSpace(params.AccountNamePrefix), TempDisableMinutes: params.TempDisableMinutes, AccountRateMultiplierScale: params.AccountRateMultiplierScale, SortOrder: params.SortOrder, Enabled: params.Enabled, IsDefault: params.IsDefault}, nil
}

func validSupplierURL(value string, required bool) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return !required
	}
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func validSupplierEndpointURL(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	if parsed.Scheme != "" || parsed.Host != "" {
		return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
	}
	return strings.HasPrefix(value, "/")
}

func buildSupplierProviderType(params SupplierProviderTypeUpsertParams) (*SupplierProviderType, error) {
	params.Code = strings.TrimSpace(params.Code)
	params.Name = strings.TrimSpace(params.Name)
	params.AvailableGroupsURL = params.GroupsURL
	if !supplierProviderCodePattern.MatchString(params.Code) || params.Name == "" {
		return nil, ErrSupplierProviderTypeInvalid
	}
	for _, value := range []string{params.LoginURL, params.APIKeysURL, params.GroupsURL, params.AvailableGroupsURL, params.BalanceURL, params.UsageCostURL} {
		if !validSupplierEndpointURL(value) {
			return nil, ErrSupplierProviderTypeInvalid
		}
	}
	return &SupplierProviderType{
		Code:               params.Code,
		Name:               params.Name,
		LoginURL:           strings.TrimSpace(params.LoginURL),
		APIKeysURL:         strings.TrimSpace(params.APIKeysURL),
		GroupsURL:          strings.TrimSpace(params.GroupsURL),
		AvailableGroupsURL: strings.TrimSpace(params.AvailableGroupsURL),
		BalanceURL:         strings.TrimSpace(params.BalanceURL),
		UsageCostURL:       strings.TrimSpace(params.UsageCostURL),
		Enabled:            params.Enabled,
		SortOrder:          params.SortOrder,
	}, nil
}

func (s *SupplierProviderService) applyTypeTemplate(ctx context.Context, params *SupplierProviderUpsertParams) error {
	if s.typeRepo == nil {
		return nil
	}
	providerTypeCode := strings.TrimSpace(params.ProviderType)
	if providerTypeCode == "" {
		return nil
	}
	template, err := s.typeRepo.GetByCode(ctx, providerTypeCode)
	if err != nil {
		if errors.Is(err, ErrSupplierProviderTypeNotFound) {
			return nil
		}
		return fmt.Errorf("get supplier provider type template: %w", err)
	}
	fillBlankSupplierURL(&params.LoginURL, template.LoginURL)
	fillBlankSupplierURL(&params.APIKeysURL, template.APIKeysURL)
	fillBlankSupplierURL(&params.GroupsURL, template.GroupsURL)
	fillBlankSupplierURL(&params.AvailableGroupsURL, template.AvailableGroupsURL)
	fillBlankSupplierURL(&params.BalanceURL, template.BalanceURL)
	fillBlankSupplierURL(&params.UsageCostURL, template.UsageCostURL)
	return nil
}

func fillBlankSupplierURL(target *string, fallback string) {
	if strings.TrimSpace(*target) == "" {
		*target = strings.TrimSpace(fallback)
	}
}

func (s *SupplierProviderService) authConfigurationChanged(existing, next *SupplierProvider) bool {
	if existing == nil || next == nil {
		return false
	}
	return strings.TrimSpace(existing.ProviderType) != strings.TrimSpace(next.ProviderType) ||
		strings.TrimSpace(existing.BaseURL) != strings.TrimSpace(next.BaseURL) ||
		strings.TrimSpace(existing.LoginURL) != strings.TrimSpace(next.LoginURL) ||
		strings.TrimSpace(existing.Email) != strings.TrimSpace(next.Email) ||
		strings.TrimSpace(existing.Username) != strings.TrimSpace(next.Username) ||
		strings.TrimSpace(existing.PasswordEncrypted) != strings.TrimSpace(next.PasswordEncrypted)
}

func (s *SupplierProviderService) deleteToken(ctx context.Context, providerID int64) error {
	if s.tokenCache == nil {
		return nil
	}
	if err := s.tokenCache.Delete(ctx, providerID); err != nil {
		return fmt.Errorf("delete supplier provider token: %w", err)
	}
	return nil
}

func redactSupplierProvider(provider *SupplierProvider) {
	provider.CredentialConfigured = provider.PasswordEncrypted != ""
	provider.PasswordEncrypted = ""
}

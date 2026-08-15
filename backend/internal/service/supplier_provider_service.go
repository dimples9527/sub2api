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
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

var (
	ErrSupplierProviderNotFound = infraerrors.NotFound("SUPPLIER_PROVIDER_NOT_FOUND", "supplier provider not found")
	ErrSupplierProviderExists   = infraerrors.Conflict("SUPPLIER_PROVIDER_EXISTS", "supplier provider code already exists")
	ErrSupplierProviderInvalid  = infraerrors.BadRequest("SUPPLIER_PROVIDER_INVALID", "invalid supplier provider configuration")
	ErrSupplierProviderDisabled = infraerrors.BadRequest("SUPPLIER_PROVIDER_DISABLED", "supplier provider is disabled")

	ErrSupplierProviderTypeNotFound = infraerrors.NotFound("SUPPLIER_PROVIDER_TYPE_NOT_FOUND", "supplier provider type not found")
	ErrSupplierProviderTypeExists   = infraerrors.Conflict("SUPPLIER_PROVIDER_TYPE_EXISTS", "supplier provider type code already exists")
	ErrSupplierProviderTypeInvalid  = infraerrors.BadRequest("SUPPLIER_PROVIDER_TYPE_INVALID", "invalid supplier provider type configuration")
)

var supplierProviderCodePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

const (
	SupplierNewAPIAuthModeAuto               = "auto"
	SupplierNewAPIAuthModeCookieSession      = "cookie_session"
	SupplierNewAPIAuthModeAccessTokenRefresh = "access_token_refresh"
)

type SupplierProvider struct {
	ID                         int64                       `json:"id"`
	Code                       string                      `json:"code"`
	Name                       string                      `json:"name"`
	ProviderType               string                      `json:"provider_type"`
	NewAPIAuthMode             string                      `json:"newapi_auth_mode"`
	BaseURL                    string                      `json:"base_url"`
	LoginURL                   string                      `json:"login_url"`
	APIKeysURL                 string                      `json:"api_keys_url"`
	GroupsURL                  string                      `json:"groups_url"`
	AvailableGroupsURL         string                      `json:"available_groups_url"`
	BalanceURL                 string                      `json:"balance_url"`
	UsageCostURL               string                      `json:"usage_cost_url"`
	MonitorURL                 string                      `json:"monitor_url"`
	AccountNamePrefix          string                      `json:"account_name_prefix"`
	TempDisableMinutes         int                         `json:"temp_disable_minutes"`
	AccountRateMultiplierScale float64                     `json:"account_rate_multiplier_scale"`
	SortOrder                  int                         `json:"sort_order"`
	Enabled                    bool                        `json:"enabled"`
	TurnstileEnabled           bool                        `json:"turnstile_enabled"`
	IsDefault                  bool                        `json:"is_default"`
	Email                      string                      `json:"email"`
	Username                   string                      `json:"username"`
	PasswordEncrypted          string                      `json:"-"`
	CredentialConfigured       bool                        `json:"credential_configured"`
	Status                     string                      `json:"status"`
	RiskLevel                  string                      `json:"risk_level"`
	ValidAccountCount          int                         `json:"valid_account_count"`
	SchedulableAccountCount    int                         `json:"schedulable_account_count"`
	RequestCount               int64                       `json:"request_count"`
	SuccessRate                float64                     `json:"success_rate"`
	PeriodCost                 float64                     `json:"period_cost"`
	CurrentBalance             float64                     `json:"current_balance"`
	TodayCost                  float64                     `json:"today_cost"`
	EstimatedDays              *float64                    `json:"estimated_days,omitempty"`
	RateRiskCount              int                         `json:"rate_risk_count"`
	SyncStatus                 string                      `json:"sync_status"`
	SyncMessage                string                      `json:"sync_message"`
	LastSyncAt                 *time.Time                  `json:"last_sync_at,omitempty"`
	AuthSummary                SupplierProviderAuthSummary `json:"auth_summary"`
	CreatedAt                  time.Time                   `json:"created_at"`
	UpdatedAt                  time.Time                   `json:"updated_at"`
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
	NewAPIAuthMode             string  `json:"newapi_auth_mode"`
	BaseURL                    string  `json:"base_url"`
	LoginURL                   string  `json:"login_url"`
	APIKeysURL                 string  `json:"api_keys_url"`
	GroupsURL                  string  `json:"groups_url"`
	AvailableGroupsURL         string  `json:"available_groups_url"`
	BalanceURL                 string  `json:"balance_url"`
	UsageCostURL               string  `json:"usage_cost_url"`
	MonitorURL                 string  `json:"monitor_url"`
	Email                      string  `json:"email"`
	Username                   string  `json:"username"`
	Password                   string  `json:"password"`
	AccountNamePrefix          string  `json:"account_name_prefix"`
	TempDisableMinutes         int     `json:"temp_disable_minutes"`
	AccountRateMultiplierScale float64 `json:"account_rate_multiplier_scale"`
	SortOrder                  int     `json:"sort_order"`
	Enabled                    bool    `json:"enabled"`
	TurnstileEnabled           bool    `json:"turnstile_enabled"`
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
	MonitorURL         string    `json:"monitor_url"`
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
	MonitorURL         string `json:"monitor_url"`
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

// SupplierProviderCostTrendPoint 表示某一天的上游成本与本地成本对比。
type SupplierProviderCostTrendPoint struct {
	Date         string  `json:"date"`
	UpstreamCost float64 `json:"upstream_cost"`
	LocalCost    float64 `json:"local_cost"`
}

// SupplierProviderCostBreakdown 表示指定日期范围内单个供应商的成本拆分。
type SupplierProviderCostBreakdown struct {
	ProviderID   int64   `json:"provider_id"`
	ProviderName string  `json:"provider_name"`
	ProviderType string  `json:"provider_type"`
	UpstreamCost float64 `json:"upstream_cost"`
	LocalCost    float64 `json:"local_cost"`
}

// SupplierProviderCostTrendResult 是供应商组合成本趋势响应。
type SupplierProviderCostTrendResult struct {
	Days       int                              `json:"days"`
	StartDate  string                           `json:"start_date,omitempty"`
	EndDate    string                           `json:"end_date,omitempty"`
	ProviderID int64                            `json:"provider_id,omitempty"`
	Points     []SupplierProviderCostTrendPoint `json:"points"`
	Breakdown  []SupplierProviderCostBreakdown  `json:"breakdown"`
}

type SupplierProviderRepository interface {
	List(ctx context.Context, params SupplierProviderListParams) ([]*SupplierProvider, int64, error)
	Summary(ctx context.Context, params SupplierProviderListParams) (SupplierProviderSummary, error)
	ListCostTrends(ctx context.Context, start, end time.Time, providerID int64) ([]SupplierProviderCostTrendPoint, error)
	ListCostBreakdowns(ctx context.Context, start, end time.Time, providerID int64) ([]SupplierProviderCostBreakdown, error)
	GetByID(ctx context.Context, id int64) (*SupplierProvider, error)
	Create(ctx context.Context, provider *SupplierProvider) error
	Update(ctx context.Context, provider *SupplierProvider) error
	DisableAfterAuthFailure(ctx context.Context, providerID int64, message string, syncedAt time.Time) error
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

func (s *SupplierProviderService) ListCostTrends(ctx context.Context, days int, providerID int64) (SupplierProviderCostTrendResult, error) {
	if days < 1 {
		days = 14
	}
	if days > 90 {
		days = 90
	}

	today := timezone.Today()
	start := today.AddDate(0, 0, -(days - 1))
	return s.listCostTrendsBetween(ctx, start, today, providerID)
}

// ListCostTrendsByDateRange 按闭区间 [startDate, endDate] 返回成本趋势，日期格式为 YYYY-MM-DD。
func (s *SupplierProviderService) ListCostTrendsByDateRange(ctx context.Context, startDate, endDate string, providerID int64) (SupplierProviderCostTrendResult, error) {
	loc := timezone.Location()
	if loc == nil {
		loc = time.Local
	}

	start, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(startDate), loc)
	if err != nil {
		return SupplierProviderCostTrendResult{}, infraerrors.BadRequest("INVALID_COST_TREND_START_DATE", "start_date must be YYYY-MM-DD")
	}
	end, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(endDate), loc)
	if err != nil {
		return SupplierProviderCostTrendResult{}, infraerrors.BadRequest("INVALID_COST_TREND_END_DATE", "end_date must be YYYY-MM-DD")
	}
	if end.Before(start) {
		return SupplierProviderCostTrendResult{}, infraerrors.BadRequest("INVALID_COST_TREND_RANGE", "end_date must be on or after start_date")
	}

	today := timezone.Today()
	if end.After(today) {
		end = today
	}
	if start.After(end) {
		start = end
	}

	// 与 days 上限保持一致，最长 90 天。
	maxSpan := 89
	if int(end.Sub(start).Hours()/24) > maxSpan {
		start = end.AddDate(0, 0, -maxSpan)
	}

	return s.listCostTrendsBetween(ctx, start, end, providerID)
}

func (s *SupplierProviderService) listCostTrendsBetween(ctx context.Context, start, endInclusive time.Time, providerID int64) (SupplierProviderCostTrendResult, error) {
	if providerID < 0 {
		providerID = 0
	}

	loc := timezone.Location()
	if loc == nil {
		loc = time.Local
	}

	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	endInclusive = time.Date(endInclusive.Year(), endInclusive.Month(), endInclusive.Day(), 0, 0, 0, 0, loc)
	endExclusive := endInclusive.AddDate(0, 0, 1)
	days := int(endInclusive.Sub(start).Hours()/24) + 1
	if days < 1 {
		days = 1
	}

	cacheKey := supplierCostTrendCacheKey(start, endInclusive, providerID)
	if cached, ok := getSupplierCostTrendCache(cacheKey); ok {
		return cached, nil
	}

	rawPoints, err := s.repo.ListCostTrends(ctx, start, endExclusive, providerID)
	if err != nil {
		return SupplierProviderCostTrendResult{}, fmt.Errorf("list supplier provider cost trends: %w", err)
	}
	rawBreakdown, err := s.repo.ListCostBreakdowns(ctx, start, endExclusive, providerID)
	if err != nil {
		return SupplierProviderCostTrendResult{}, fmt.Errorf("list supplier provider cost breakdowns: %w", err)
	}
	if rawBreakdown == nil {
		rawBreakdown = []SupplierProviderCostBreakdown{}
	}

	byDate := make(map[string]SupplierProviderCostTrendPoint, len(rawPoints))
	for _, point := range rawPoints {
		byDate[point.Date] = point
	}

	points := make([]SupplierProviderCostTrendPoint, 0, days)
	for cursor := start; cursor.Before(endExclusive); cursor = cursor.AddDate(0, 0, 1) {
		date := cursor.In(loc).Format("2006-01-02")
		if point, ok := byDate[date]; ok {
			points = append(points, point)
			continue
		}
		points = append(points, SupplierProviderCostTrendPoint{Date: date})
	}

	result := SupplierProviderCostTrendResult{
		Days:       days,
		StartDate:  start.In(loc).Format("2006-01-02"),
		EndDate:    endInclusive.In(loc).Format("2006-01-02"),
		ProviderID: providerID,
		Points:     points,
		Breakdown:  rawBreakdown,
	}
	setSupplierCostTrendCache(cacheKey, result)
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
	requestedNewAPIAuthMode := params.NewAPIAuthMode
	provider, err := s.buildProvider(params)
	if err != nil {
		return nil, err
	}
	logger.LegacyPrintf("supplier_provider_service", "upsert provider action=create provider_code=%s provider_type=%s requested_newapi_auth_mode=%s normalized_newapi_auth_mode=%s enabled=%t", provider.Code, provider.ProviderType, requestedNewAPIAuthMode, provider.NewAPIAuthMode, provider.Enabled)
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
	requestedNewAPIAuthMode := params.NewAPIAuthMode
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
	logger.LegacyPrintf("supplier_provider_service", "upsert provider action=update provider_id=%d provider_code=%s provider_type=%s previous_newapi_auth_mode=%s requested_newapi_auth_mode=%s normalized_newapi_auth_mode=%s enabled=%t", provider.ID, provider.Code, provider.ProviderType, existing.NewAPIAuthMode, requestedNewAPIAuthMode, provider.NewAPIAuthMode, provider.Enabled)
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
	params.NewAPIAuthMode = strings.ToLower(strings.TrimSpace(params.NewAPIAuthMode))
	params.BaseURL = strings.TrimRight(strings.TrimSpace(params.BaseURL), "/")
	params.AvailableGroupsURL = params.GroupsURL
	if !supplierProviderCodePattern.MatchString(params.Code) || params.Name == "" || params.ProviderType == "" || !validSupplierURL(params.BaseURL, true) {
		return nil, ErrSupplierProviderInvalid
	}
	for _, value := range []string{params.LoginURL, params.APIKeysURL, params.GroupsURL, params.AvailableGroupsURL, params.BalanceURL, params.UsageCostURL, params.MonitorURL} {
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
	if strings.EqualFold(params.ProviderType, SupplierProviderTypeNewAPI) {
		switch params.NewAPIAuthMode {
		case "", SupplierNewAPIAuthModeAuto:
			params.NewAPIAuthMode = SupplierNewAPIAuthModeAuto
		case SupplierNewAPIAuthModeCookieSession, SupplierNewAPIAuthModeAccessTokenRefresh:
		default:
			return nil, ErrSupplierProviderInvalid
		}
	} else {
		params.NewAPIAuthMode = SupplierNewAPIAuthModeAuto
	}
	if strings.EqualFold(params.ProviderType, "sub2api") {
		username = ""
	}
	return &SupplierProvider{Code: params.Code, Name: params.Name, ProviderType: params.ProviderType, NewAPIAuthMode: params.NewAPIAuthMode, BaseURL: params.BaseURL, LoginURL: strings.TrimSpace(params.LoginURL), APIKeysURL: strings.TrimSpace(params.APIKeysURL), GroupsURL: strings.TrimSpace(params.GroupsURL), AvailableGroupsURL: strings.TrimSpace(params.AvailableGroupsURL), BalanceURL: strings.TrimSpace(params.BalanceURL), UsageCostURL: strings.TrimSpace(params.UsageCostURL), MonitorURL: strings.TrimSpace(params.MonitorURL), Email: email, Username: username, AccountNamePrefix: strings.TrimSpace(params.AccountNamePrefix), TempDisableMinutes: params.TempDisableMinutes, AccountRateMultiplierScale: params.AccountRateMultiplierScale, SortOrder: params.SortOrder, Enabled: params.Enabled, TurnstileEnabled: params.TurnstileEnabled, IsDefault: params.IsDefault}, nil
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
	for _, value := range []string{params.LoginURL, params.APIKeysURL, params.GroupsURL, params.AvailableGroupsURL, params.BalanceURL, params.UsageCostURL, params.MonitorURL} {
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
		MonitorURL:         strings.TrimSpace(params.MonitorURL),
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
	fillBlankSupplierURL(&params.MonitorURL, template.MonitorURL)
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
		normalizeSupplierNewAPIAuthModeForCompare(existing.NewAPIAuthMode) != normalizeSupplierNewAPIAuthModeForCompare(next.NewAPIAuthMode) ||
		strings.TrimSpace(existing.BaseURL) != strings.TrimSpace(next.BaseURL) ||
		strings.TrimSpace(existing.LoginURL) != strings.TrimSpace(next.LoginURL) ||
		strings.TrimSpace(existing.Email) != strings.TrimSpace(next.Email) ||
		strings.TrimSpace(existing.Username) != strings.TrimSpace(next.Username) ||
		strings.TrimSpace(existing.PasswordEncrypted) != strings.TrimSpace(next.PasswordEncrypted)
}

func normalizeSupplierNewAPIAuthModeForCompare(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		return SupplierNewAPIAuthModeAuto
	}
	return mode
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

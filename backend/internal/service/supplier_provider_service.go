package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"sort"
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
	RechargeURL                string                      `json:"recharge_url"`
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
	RechargeURL                string  `json:"recharge_url"`
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
	RechargeURL        string    `json:"recharge_url"`
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
	RechargeURL        string `json:"recharge_url"`
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
	// RawUpstreamCost 为上游成本接口返回的原始值（未做偏差覆盖），可能为 nil（历史数据未记录）。
	RawUpstreamCost *float64 `json:"raw_upstream_cost,omitempty"`
	// Warning 表示该日因上游与本地成本偏差过大而采用本地成本时记录的提示。
	Warning string `json:"warning,omitempty"`
}

// SupplierProviderCostBreakdown 表示指定日期范围内单个供应商的成本拆分。
type SupplierProviderCostBreakdown struct {
	ProviderID      int64   `json:"provider_id"`
	ProviderName    string  `json:"provider_name"`
	ProviderType    string  `json:"provider_type"`
	UpstreamCost    float64 `json:"upstream_cost"`
	LocalCost       float64 `json:"local_cost"`
	RawUpstreamCost float64 `json:"raw_upstream_cost,omitempty"`
	CostWarning     string  `json:"cost_warning,omitempty"`
}

// SupplierProviderBalanceCostDay 表示供应商在某一天的余额差额成本。
// BalanceCost 为当天第一条余额快照减去当天最后条余额快照得到的成本估算（仅大于 0 时有效）。
type SupplierProviderBalanceCostDay struct {
	Date        string  // YYYY-MM-DD（本地时区）
	ProviderID  int64   // 供应商 ID
	BalanceCost float64 // 第一条余额 - 最后条余额，仅在大于 0 时返回
}

// supplierCostDeviation 计算上游成本与本地成本的相对偏差，公式与前端保持一致：
// |上游-本地| / max(上游, 本地)，两者都非正时视为无偏差。
func supplierCostDeviation(upstream, local float64) float64 {
	if upstream <= 0 && local <= 0 {
		return 0
	}
	max := math.Max(upstream, local)
	if max <= 0 {
		return 0
	}
	return math.Abs(upstream-local) / max
}

// supplierCostDeviationWarning 生成成本偏差覆盖时的中文提示。
func supplierCostDeviationWarning(upstream, local float64) string {
	pct := int(math.Round(supplierCostDeviation(upstream, local) * 100))
	return fmt.Sprintf("上游成本 %.2f 与本地成本 %.2f 偏差 %d%%，已按本地成本展示", upstream, local, pct)
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

// SupplierProviderBalanceSummaryDay 表示某个统计日的供应商余额/成本合计。
type SupplierProviderBalanceSummaryDay struct {
	Date    string  `json:"date"`
	Balance float64 `json:"balance"`
	Cost    float64 `json:"cost"`
}

// SupplierProviderBalanceHistory 表示历史统计区间内的累计口径。
type SupplierProviderBalanceHistory struct {
	FirstDate    string  `json:"first_date"`
	Days         int     `json:"days"`
	TotalBalance float64 `json:"total_balance"`
	TotalCost    float64 `json:"total_cost"`
}

// SupplierProviderBalanceSummary 是供应商组合余额/成本汇总，用于页面统计卡。
type SupplierProviderBalanceSummary struct {
	LatestDate string                            `json:"latest_date"`
	Today      SupplierProviderBalanceSummaryDay `json:"today"`
	Previous   SupplierProviderBalanceSummaryDay `json:"previous"`
	History    SupplierProviderBalanceHistory    `json:"history"`
}

type SupplierProviderRepository interface {
	List(ctx context.Context, params SupplierProviderListParams) ([]*SupplierProvider, int64, error)
	Summary(ctx context.Context, params SupplierProviderListParams) (SupplierProviderSummary, error)
	ListCostTrends(ctx context.Context, start, end time.Time, providerID int64) ([]SupplierProviderCostTrendPoint, error)
	ListCostBreakdowns(ctx context.Context, start, end time.Time, providerID int64) ([]SupplierProviderCostBreakdown, error)
	ListBalanceSummaryDays(ctx context.Context) ([]SupplierProviderBalanceSummaryDay, error)
	// ListBalanceCosts returns balance difference costs per day per provider for date range.
	ListBalanceCosts(ctx context.Context, start, end time.Time, providerID int64) ([]SupplierProviderBalanceCostDay, error)

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
	repo               SupplierProviderRepository
	encryptor          SecretEncryptor
	typeRepo           SupplierProviderTypeRepository
	tokenCache         SupplierProviderTokenCache
	thresholdProvider  SupplierCostDeviationThresholdProvider
	costSourceResolver SupplierCostSourceResolver
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

// SetCostDeviationThresholdProvider 注入成本偏差覆盖阈值提供方，未注入时使用默认阈值。
func (s *SupplierProviderService) SetCostDeviationThresholdProvider(provider SupplierCostDeviationThresholdProvider) {
	if s != nil {
		s.thresholdProvider = provider
	}
}

func (s *SupplierProviderService) costDeviationThreshold(ctx context.Context) float64 {
	if s != nil && s.thresholdProvider != nil {
		return s.thresholdProvider.SupplierCostDeviationThreshold(ctx)
	}
	return DefaultSupplierCostDeviationThreshold
}

// SetCostSourceResolver 注入成本来源解析器；未注入时趋势展示按智能模式 + 全局阈值处理。
func (s *SupplierProviderService) SetCostSourceResolver(resolver SupplierCostSourceResolver) {
	if s != nil {
		s.costSourceResolver = resolver
	}
}

// costSourceFor 解析供应商成本来源；未注入解析器或解析失败时回退智能模式 + 全局阈值。
func (s *SupplierProviderService) costSourceFor(ctx context.Context, providerID int64) SupplierCostSourceResolution {
	if s != nil && s.costSourceResolver != nil {
		if resolution, err := s.costSourceResolver.ResolveCostSource(ctx, providerID); err == nil {
			return resolution
		}
	}
	return SupplierCostSourceResolution{Source: SupplierCostSourceAuto, Threshold: s.costDeviationThreshold(ctx)}
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

	// 偏差覆盖阈值；历史数据在展示时也会按该阈值做兜底改写。
	threshold := s.costDeviationThreshold(ctx)

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

	// 用当天首条余额 - 当天末条余额的成本，作为 today_cost 缺失(<=0)时的保底估算，
	// 同时补上只有余额快照、没有 daily_stats 的日期。
	balanceCosts, err := s.repo.ListBalanceCosts(ctx, start, endExclusive, providerID)
	if err != nil {
		return SupplierProviderCostTrendResult{}, fmt.Errorf("list balance costs: %w", err)
	}
	dateBalanceCost := make(map[string]float64, len(balanceCosts))
	for _, bc := range balanceCosts {
		dateBalanceCost[bc.Date] += bc.BalanceCost
	}
	for date, balanceCost := range dateBalanceCost {
		point, ok := byDate[date]
		if !ok {
			point = SupplierProviderCostTrendPoint{Date: date}
		}
		if point.UpstreamCost <= 0 {
			point.UpstreamCost = balanceCost
			byDate[date] = point
		}
	}

	// 按供应商拆分的成本同样用余额差额做保底估算。
	for i := range rawBreakdown {
		if rawBreakdown[i].UpstreamCost <= 0 {
			sum := 0.0
			for _, bc := range balanceCosts {
				if bc.ProviderID == rawBreakdown[i].ProviderID {
					sum += bc.BalanceCost
				}
			}
			rawBreakdown[i].UpstreamCost = sum
		}
	}

	// 展示时按供应商成本来源决定兜底改写：
	//   upstream 模式保持上游值；calculated 模式固定展示本地计算成本；
	//   auto 模式在上游与本地差距超过阈值时按本地成本展示并记录警告，
	//   优先以原始上游值为基准（历史数据未记录时退化为已写入的 today_cost）。
	for i := range rawBreakdown {
		b := &rawBreakdown[i]
		source := s.costSourceFor(ctx, b.ProviderID)
		if source.Source == SupplierCostSourceUpstream {
			continue
		}
		if source.Source == SupplierCostSourceCalculated {
			if b.LocalCost > 0 {
				b.UpstreamCost = b.LocalCost
			}
			continue
		}
		if b.LocalCost <= 0 {
			continue
		}
		base := b.UpstreamCost
		if b.RawUpstreamCost > 0 {
			base = b.RawUpstreamCost
		}
		if base > 0 && supplierCostDeviation(base, b.LocalCost) > source.Threshold {
			b.CostWarning = supplierCostDeviationWarning(base, b.LocalCost)
			b.UpstreamCost = b.LocalCost
		}
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

	// 汇总点位按查询维度应用成本来源：指定供应商时用该供应商的解析结果，
	// 未指定供应商（跨供应商汇总）时回退智能模式 + 全局阈值。
	pointSource := SupplierCostSourceResolution{Source: SupplierCostSourceAuto, Threshold: threshold}
	if providerID > 0 {
		pointSource = s.costSourceFor(ctx, providerID)
	}
	for i := range points {
		p := &points[i]
		if pointSource.Source == SupplierCostSourceUpstream {
			continue
		}
		if pointSource.Source == SupplierCostSourceCalculated {
			if p.LocalCost > 0 {
				p.UpstreamCost = p.LocalCost
			}
			continue
		}
		if p.LocalCost <= 0 {
			continue
		}
		base := p.UpstreamCost
		if p.RawUpstreamCost != nil && *p.RawUpstreamCost > 0 {
			base = *p.RawUpstreamCost
		}
		if base > 0 && supplierCostDeviation(base, p.LocalCost) > pointSource.Threshold {
			p.Warning = supplierCostDeviationWarning(base, p.LocalCost)
			p.UpstreamCost = p.LocalCost
		}
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

// GetBalanceSummary 汇总供应商每日快照的余额与成本，用于页面统计卡。
// today 为最近一个统计日，previous 为最近一个非当日统计日，history 为累计口径。
func (s *SupplierProviderService) GetBalanceSummary(ctx context.Context) (SupplierProviderBalanceSummary, error) {
	days, err := s.repo.ListBalanceSummaryDays(ctx)
	if err != nil {
		return SupplierProviderBalanceSummary{}, fmt.Errorf("list supplier provider balance summary days: %w", err)
	}
	summary := SupplierProviderBalanceSummary{
		History: SupplierProviderBalanceHistory{},
	}
	if len(days) == 0 {
		return summary, nil
	}
	sort.Slice(days, func(i, j int) bool {
		return days[i].Date < days[j].Date
	})
	summary.LatestDate = days[len(days)-1].Date
	summary.Today = days[len(days)-1]
	summary.History.FirstDate = days[0].Date
	summary.History.Days = len(days)
	for _, day := range days {
		summary.History.TotalBalance += day.Balance
		summary.History.TotalCost += day.Cost
	}
	if len(days) >= 2 {
		summary.Previous = days[len(days)-2]
	}
	return summary, nil
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
	for _, value := range []string{params.LoginURL, params.APIKeysURL, params.GroupsURL, params.AvailableGroupsURL, params.BalanceURL, params.UsageCostURL, params.RechargeURL, params.MonitorURL} {
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
	return &SupplierProvider{Code: params.Code, Name: params.Name, ProviderType: params.ProviderType, NewAPIAuthMode: params.NewAPIAuthMode, BaseURL: params.BaseURL, LoginURL: strings.TrimSpace(params.LoginURL), APIKeysURL: strings.TrimSpace(params.APIKeysURL), GroupsURL: strings.TrimSpace(params.GroupsURL), AvailableGroupsURL: strings.TrimSpace(params.AvailableGroupsURL), BalanceURL: strings.TrimSpace(params.BalanceURL), UsageCostURL: strings.TrimSpace(params.UsageCostURL), RechargeURL: strings.TrimSpace(params.RechargeURL), MonitorURL: strings.TrimSpace(params.MonitorURL), Email: email, Username: username, AccountNamePrefix: strings.TrimSpace(params.AccountNamePrefix), TempDisableMinutes: params.TempDisableMinutes, AccountRateMultiplierScale: params.AccountRateMultiplierScale, SortOrder: params.SortOrder, Enabled: params.Enabled, TurnstileEnabled: params.TurnstileEnabled, IsDefault: params.IsDefault}, nil
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
	for _, value := range []string{params.LoginURL, params.APIKeysURL, params.GroupsURL, params.AvailableGroupsURL, params.BalanceURL, params.UsageCostURL, params.RechargeURL, params.MonitorURL} {
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
		RechargeURL:        strings.TrimSpace(params.RechargeURL),
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
	fillBlankSupplierURL(&params.RechargeURL, template.RechargeURL)
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

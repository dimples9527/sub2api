package admin

import (
	"context"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// 供应商运维驾驶舱只读查询服务端口。
type supplierDashboardService interface {
	GetAccounts(ctx context.Context, q service.SupplierDashboardAccountsQuery) (service.SupplierDashboardAccountsResponse, error)
	GetRates(ctx context.Context, q service.SupplierDashboardRatesQuery) (service.SupplierDashboardRatesResponse, error)
	GetProviders(ctx context.Context, q service.SupplierDashboardProvidersQuery) (service.SupplierDashboardProvidersResponse, error)
	GetAccountTraffic(ctx context.Context, q service.SupplierDashboardTrafficQuery) (service.SupplierDashboardTrafficResponse, error)
	GetAccountProfitRanking(ctx context.Context, q service.SupplierDashboardProfitQuery) (service.SupplierDashboardProfitResponse, error)
	GetAccountHealthTimeline(ctx context.Context, q service.SupplierDashboardAccountHealthQuery) (service.SupplierDashboardAccountHealthResponse, error)
}

// SupplierDashboardHandler 提供供应商运维驾驶舱只读接口。
type SupplierDashboardHandler struct {
	service supplierDashboardService
}

// NewSupplierDashboardHandler 创建供应商运维驾驶舱 Handler。
func NewSupplierDashboardHandler(svc *service.SupplierDashboardService) *SupplierDashboardHandler {
	return newSupplierDashboardHandlerWithService(svc)
}

func newSupplierDashboardHandlerWithService(svc supplierDashboardService) *SupplierDashboardHandler {
	return &SupplierDashboardHandler{service: svc}
}

// GetAccounts 查询异常账号列表。
func (h *SupplierDashboardHandler) GetAccounts(c *gin.Context) {
	query, ok := parseSupplierDashboardAccountsQuery(c)
	if !ok {
		return
	}
	result, err := h.service.GetAccounts(c.Request.Context(), query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// GetRates 查询 Provider + Group 倍率分析。
func (h *SupplierDashboardHandler) GetRates(c *gin.Context) {
	query, ok := parseSupplierDashboardRatesQuery(c)
	if !ok {
		return
	}
	result, err := h.service.GetRates(c.Request.Context(), query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// GetProviders 查询供应商运行概览。
func (h *SupplierDashboardHandler) GetProviders(c *gin.Context) {
	query, ok := parseSupplierDashboardProvidersQuery(c)
	if !ok {
		return
	}
	result, err := h.service.GetProviders(c.Request.Context(), query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// GetAccountTraffic 查询账号小时级请求量 / Token 流量趋势。
func (h *SupplierDashboardHandler) GetAccountTraffic(c *gin.Context) {
	query, ok := parseSupplierDashboardTrafficQuery(c)
	if !ok {
		return
	}
	result, err := h.service.GetAccountTraffic(c.Request.Context(), query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// GetAccountProfitRanking 查询账号盈利排行。
func (h *SupplierDashboardHandler) GetAccountProfitRanking(c *gin.Context) {
	query, ok := parseSupplierDashboardProfitQuery(c)
	if !ok {
		return
	}
	result, err := h.service.GetAccountProfitRanking(c.Request.Context(), query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// GetAccountHealthTimeline 查询账号健康状态时间线（支持分钟级/小时级时间桶）。
func (h *SupplierDashboardHandler) GetAccountHealthTimeline(c *gin.Context) {
	query, ok := parseSupplierDashboardAccountHealthQuery(c)
	if !ok {
		return
	}
	result, err := h.service.GetAccountHealthTimeline(c.Request.Context(), query)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func parseSupplierDashboardAccountsQuery(c *gin.Context) (service.SupplierDashboardAccountsQuery, bool) {
	rangeValue, ok := parseSupplierDashboardRange(c)
	if !ok {
		return service.SupplierDashboardAccountsQuery{}, false
	}
	riskType, ok := parseSupplierDashboardRiskType(c)
	if !ok {
		return service.SupplierDashboardAccountsQuery{}, false
	}
	page, pageSize, ok := parseSupplierDashboardPagination(c)
	if !ok {
		return service.SupplierDashboardAccountsQuery{}, false
	}
	return service.SupplierDashboardAccountsQuery{
		Range:        rangeValue,
		RiskType:     riskType,
		ProviderSlug: strings.TrimSpace(c.Query("provider_slug")),
		GroupKey:     strings.TrimSpace(c.Query("group_key")),
		Page:         page,
		PageSize:     pageSize,
	}, true
}

func parseSupplierDashboardRatesQuery(c *gin.Context) (service.SupplierDashboardRatesQuery, bool) {
	rangeValue, ok := parseSupplierDashboardRange(c)
	if !ok {
		return service.SupplierDashboardRatesQuery{}, false
	}
	view, ok := parseSupplierDashboardRateView(c)
	if !ok {
		return service.SupplierDashboardRatesQuery{}, false
	}
	comparisonStatus, ok := parseSupplierDashboardComparisonStatus(c)
	if !ok {
		return service.SupplierDashboardRatesQuery{}, false
	}
	page, pageSize, ok := parseSupplierDashboardPagination(c)
	if !ok {
		return service.SupplierDashboardRatesQuery{}, false
	}
	return service.SupplierDashboardRatesQuery{
		Range:            rangeValue,
		View:             view,
		ComparisonStatus: comparisonStatus,
		ProviderSlug:     strings.TrimSpace(c.Query("provider_slug")),
		GroupKey:         strings.TrimSpace(c.Query("group_key")),
		Page:             page,
		PageSize:         pageSize,
	}, true
}

func parseSupplierDashboardProvidersQuery(c *gin.Context) (service.SupplierDashboardProvidersQuery, bool) {
	rangeValue, ok := parseSupplierDashboardRange(c)
	if !ok {
		return service.SupplierDashboardProvidersQuery{}, false
	}
	status, ok := parseSupplierDashboardProviderStatus(c)
	if !ok {
		return service.SupplierDashboardProvidersQuery{}, false
	}
	page, pageSize, ok := parseSupplierDashboardPagination(c)
	if !ok {
		return service.SupplierDashboardProvidersQuery{}, false
	}
	return service.SupplierDashboardProvidersQuery{
		Range:    rangeValue,
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	}, true
}

func parseSupplierDashboardTrafficQuery(c *gin.Context) (service.SupplierDashboardTrafficQuery, bool) {
	rangeValue, ok := parseSupplierDashboardRangeDefault(c, service.SupplierDashboardRange30Days)
	if !ok {
		return service.SupplierDashboardTrafficQuery{}, false
	}
	return service.SupplierDashboardTrafficQuery{
		Range:        rangeValue,
		ProviderSlug: strings.TrimSpace(c.Query("provider_slug")),
		GroupKey:     strings.TrimSpace(c.Query("group_key")),
	}, true
}

func parseSupplierDashboardProfitQuery(c *gin.Context) (service.SupplierDashboardProfitQuery, bool) {
	rangeValue, ok := parseSupplierDashboardRangeDefault(c, service.SupplierDashboardRange30Days)
	if !ok {
		return service.SupplierDashboardProfitQuery{}, false
	}
	limit, ok := parseSupplierDashboardPositiveInt(c, "limit", 20)
	if !ok {
		return service.SupplierDashboardProfitQuery{}, false
	}
	if limit > 100 {
		response.BadRequest(c, "limit must be between 1 and 100")
		return service.SupplierDashboardProfitQuery{}, false
	}
	return service.SupplierDashboardProfitQuery{
		Range:        rangeValue,
		ProviderSlug: strings.TrimSpace(c.Query("provider_slug")),
		GroupKey:     strings.TrimSpace(c.Query("group_key")),
		Limit:        limit,
	}, true
}

func parseSupplierDashboardAccountHealthQuery(c *gin.Context) (service.SupplierDashboardAccountHealthQuery, bool) {
	rangeValue, ok := parseSupplierDashboardAccountHealthRange(c)
	if !ok {
		return service.SupplierDashboardAccountHealthQuery{}, false
	}
	limit, ok := parseSupplierDashboardPositiveInt(c, "limit", 30)
	if !ok {
		return service.SupplierDashboardAccountHealthQuery{}, false
	}
	if limit > 100 {
		response.BadRequest(c, "limit must be between 1 and 100")
		return service.SupplierDashboardAccountHealthQuery{}, false
	}
	buckets, ok := parseSupplierDashboardPositiveInt(c, "buckets", 72)
	if !ok {
		return service.SupplierDashboardAccountHealthQuery{}, false
	}
	if buckets > 720 {
		response.BadRequest(c, "buckets must be between 1 and 720")
		return service.SupplierDashboardAccountHealthQuery{}, false
	}
	bucketHours, ok := parseSupplierDashboardPositiveInt(c, "bucket_hours", 1)
	if !ok {
		return service.SupplierDashboardAccountHealthQuery{}, false
	}
	if bucketHours > 24 {
		response.BadRequest(c, "bucket_hours must be between 1 and 24")
		return service.SupplierDashboardAccountHealthQuery{}, false
	}
	bucketMinutes, ok := parseSupplierDashboardOptionalPositiveInt(c, "bucket_minutes", 0)
	if !ok {
		return service.SupplierDashboardAccountHealthQuery{}, false
	}
	if bucketMinutes > 1440 {
		response.BadRequest(c, "bucket_minutes must be between 1 and 1440")
		return service.SupplierDashboardAccountHealthQuery{}, false
	}
	return service.SupplierDashboardAccountHealthQuery{
		Range:         rangeValue,
		ProviderSlug:  strings.TrimSpace(c.Query("provider_slug")),
		GroupKey:      strings.TrimSpace(c.Query("group_key")),
		Limit:         limit,
		Buckets:       buckets,
		BucketHours:   bucketHours,
		BucketMinutes: bucketMinutes,
	}, true
}

// parseSupplierDashboardAccountHealthRange 解析健康时间线区间，额外支持 1h / 6h 分钟级短区间。
func parseSupplierDashboardAccountHealthRange(c *gin.Context) (service.SupplierDashboardRange, bool) {
	raw := strings.TrimSpace(c.DefaultQuery("range", string(service.SupplierDashboardRange30Days)))
	rangeValue := service.SupplierDashboardRange(raw)
	switch rangeValue {
	case service.SupplierDashboardRange1Hour,
		service.SupplierDashboardRange6Hours,
		service.SupplierDashboardRange24Hours,
		service.SupplierDashboardRange7Days,
		service.SupplierDashboardRange30Days:
		return rangeValue, true
	default:
		response.BadRequest(c, "range must be 1h, 6h, 24h, 7d or 30d")
		return "", false
	}
}

// parseSupplierDashboardOptionalPositiveInt 解析可选正整数参数，未传时返回 defaultValue（0 表示未设置）。
func parseSupplierDashboardOptionalPositiveInt(c *gin.Context, key string, defaultValue int) (int, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return defaultValue, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		response.BadRequest(c, key+" must be a positive integer")
		return 0, false
	}
	return value, true
}

func parseSupplierDashboardRange(c *gin.Context) (service.SupplierDashboardRange, bool) {
	return parseSupplierDashboardRangeDefault(c, service.SupplierDashboardRange24Hours)
}

func parseSupplierDashboardRangeDefault(c *gin.Context, defaultValue service.SupplierDashboardRange) (service.SupplierDashboardRange, bool) {
	raw := strings.TrimSpace(c.DefaultQuery("range", string(defaultValue)))
	rangeValue := service.SupplierDashboardRange(raw)
	if rangeValue != service.SupplierDashboardRange24Hours &&
		rangeValue != service.SupplierDashboardRange7Days &&
		rangeValue != service.SupplierDashboardRange30Days {
		response.BadRequest(c, "range must be 24h, 7d or 30d")
		return "", false
	}
	return rangeValue, true
}

func parseSupplierDashboardRiskType(c *gin.Context) (service.SupplierDashboardRiskType, bool) {
	raw := strings.TrimSpace(c.DefaultQuery("risk_type", string(service.SupplierDashboardRiskTypeAll)))
	riskType := service.SupplierDashboardRiskType(raw)
	switch riskType {
	case service.SupplierDashboardRiskTypeAll,
		service.SupplierDashboardRiskTypeCritical,
		service.SupplierDashboardRiskTypeTraffic,
		service.SupplierDashboardRiskTypeRateUp,
		service.SupplierDashboardRiskTypeNotLowest,
		service.SupplierDashboardRiskTypeBalance,
		service.SupplierDashboardRiskTypeSync,
		service.SupplierDashboardRiskTypeTask:
		return riskType, true
	default:
		response.BadRequest(c, "invalid risk_type")
		return "", false
	}
}

func parseSupplierDashboardRateView(c *gin.Context) (service.SupplierDashboardRateView, bool) {
	raw := strings.TrimSpace(c.DefaultQuery("view", string(service.SupplierDashboardRateViewRisk)))
	view := service.SupplierDashboardRateView(raw)
	switch view {
	case service.SupplierDashboardRateViewRisk,
		service.SupplierDashboardRateViewChanged,
		service.SupplierDashboardRateViewAll:
		return view, true
	default:
		response.BadRequest(c, "invalid view")
		return "", false
	}
}

func parseSupplierDashboardComparisonStatus(c *gin.Context) (service.SupplierDashboardComparisonStatus, bool) {
	raw := strings.TrimSpace(c.Query("comparison_status"))
	if raw == "" {
		return "", true
	}
	status := service.SupplierDashboardComparisonStatus(raw)
	switch status {
	case service.SupplierDashboardComparisonStatusLowest,
		service.SupplierDashboardComparisonStatusTiedLowest,
		service.SupplierDashboardComparisonStatusNotLowest,
		service.SupplierDashboardComparisonStatusMissingGroup,
		service.SupplierDashboardComparisonStatusInsufficientAccounts,
		service.SupplierDashboardComparisonStatusUnknown:
		return status, true
	default:
		response.BadRequest(c, "invalid comparison_status")
		return "", false
	}
}

func parseSupplierDashboardProviderStatus(c *gin.Context) (service.SupplierDashboardProviderStatus, bool) {
	raw := strings.TrimSpace(c.Query("status"))
	if raw == "" {
		return "", true
	}
	status := service.SupplierDashboardProviderStatus(raw)
	switch status {
	case service.SupplierDashboardProviderStatusHealthy,
		service.SupplierDashboardProviderStatusWarning,
		service.SupplierDashboardProviderStatusHighRisk,
		service.SupplierDashboardProviderStatusDisabled,
		service.SupplierDashboardProviderStatusUnknown:
		return status, true
	default:
		response.BadRequest(c, "invalid status")
		return "", false
	}
}

func parseSupplierDashboardPagination(c *gin.Context) (int, int, bool) {
	page, ok := parseSupplierDashboardPositiveInt(c, "page", 1)
	if !ok {
		return 0, 0, false
	}
	pageSize, ok := parseSupplierDashboardPositiveInt(c, "page_size", 20)
	if !ok {
		return 0, 0, false
	}
	if pageSize > 100 {
		response.BadRequest(c, "page_size must be between 1 and 100")
		return 0, 0, false
	}
	return page, pageSize, true
}

func parseSupplierDashboardPositiveInt(c *gin.Context, key string, defaultValue int) (int, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return defaultValue, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		response.BadRequest(c, key+" must be a positive integer")
		return 0, false
	}
	return value, true
}

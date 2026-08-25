package admin

import (
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SupplierProviderCostReviewHandler struct {
	service *service.SupplierProviderCostReviewService
}

func NewSupplierProviderCostReviewHandler(svc *service.SupplierProviderCostReviewService) *SupplierProviderCostReviewHandler {
	return &SupplierProviderCostReviewHandler{service: svc}
}

func (h *SupplierProviderCostReviewHandler) List(c *gin.Context) {
	providerID := int64(0)
	if raw := strings.TrimSpace(c.Query("provider_id")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			response.ErrorFrom(c, infraerrors.BadRequest("INVALID_PROVIDER_ID", "供应商 ID 无效"))
			return
		}
		providerID = parsed
	}
	start, end, ok := parseCostReviewDateRange(c.Query("start_date"), c.Query("end_date"))
	if !ok {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_DATE_RANGE", "日期格式必须为 YYYY-MM-DD，且开始日期不能晚于结束日期"))
		return
	}
	page, pageSize := response.ParsePagination(c)
	result, err := h.service.List(c.Request.Context(), service.SupplierProviderCostReviewListParams{ProviderID: providerID, Keyword: strings.TrimSpace(c.Query("keyword")), StartDate: start, EndDate: end, Status: strings.TrimSpace(c.Query("status")), Page: page, PageSize: pageSize})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierProviderCostReviewHandler) History(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_REVIEW_ID", "核对记录 ID 无效"))
		return
	}
	result, err := h.service.History(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type supplierProviderCostReviewApproveRequest struct {
	DecisionType string   `json:"decision_type"`
	ManualCost   *float64 `json:"manual_cost"`
	Version      int64    `json:"version"`
}

type supplierProviderCostReviewBulkApproveRequest struct {
	Items        []service.SupplierProviderCostReviewApproveItem `json:"items"`
	DecisionType string                                          `json:"decision_type"`
	ManualCost   *float64                                        `json:"manual_cost"`
}

func (h *SupplierProviderCostReviewHandler) Approve(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_REVIEW_ID", "核对记录 ID 无效"))
		return
	}
	var req supplierProviderCostReviewApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	input := service.SupplierProviderCostReviewApproveInput{DecisionType: strings.TrimSpace(req.DecisionType), ManualCost: req.ManualCost, Version: req.Version}
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		input.OperatorID = subject.UserID
	}
	result, err := h.service.Approve(c.Request.Context(), id, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierProviderCostReviewHandler) BulkApprove(c *gin.Context) {
	var req supplierProviderCostReviewBulkApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	input := service.SupplierProviderCostReviewBulkApproveInput{
		Items:        req.Items,
		DecisionType: strings.TrimSpace(req.DecisionType),
		ManualCost:   req.ManualCost,
	}
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		input.OperatorID = subject.UserID
	}
	items, err := h.service.ApproveMany(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items, "count": len(items)})
}

func parseCostReviewDateRange(startValue, endValue string) (string, string, bool) {
	startValue, endValue = strings.TrimSpace(startValue), strings.TrimSpace(endValue)
	if startValue == "" && endValue == "" {
		return "", "", true
	}
	if startValue == "" || endValue == "" {
		return "", "", false
	}
	if _, err := time.Parse("2006-01-02", startValue); err != nil {
		return "", "", false
	}
	if _, err := time.Parse("2006-01-02", endValue); err != nil || startValue > endValue {
		return "", "", false
	}
	return startValue, endValue, true
}

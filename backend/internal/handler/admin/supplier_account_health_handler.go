package admin

import (
	"context"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SupplierAccountHealthTrendServicePort interface {
	ListAccounts(ctx context.Context, params service.SupplierAccountHealthAccountListParams) (service.SupplierAccountHealthAccountListResult, error)
	GetTrend(ctx context.Context, accountID int64, rangeValue string) (service.SupplierAccountHealthTrendResult, error)
}

type SupplierAccountHealthHandler struct {
	service SupplierAccountHealthTrendServicePort
}

func NewSupplierAccountHealthHandler(svc SupplierAccountHealthTrendServicePort) *SupplierAccountHealthHandler {
	return &SupplierAccountHealthHandler{service: svc}
}

func (h *SupplierAccountHealthHandler) ListAccounts(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > supplierProviderMaxPageSize {
		pageSize = supplierProviderMaxPageSize
	}
	providerID := parseOptionalInt64(c.Query("provider_id"))
	result, err := h.service.ListAccounts(c.Request.Context(), service.SupplierAccountHealthAccountListParams{
		ProviderID:   providerID,
		Platform:     strings.TrimSpace(c.Query("platform")),
		Search:       strings.TrimSpace(c.Query("search")),
		HealthStatus: strings.TrimSpace(c.Query("health_status")),
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierAccountHealthHandler) GetTrend(c *gin.Context) {
	accountID, err := strconv.ParseInt(strings.TrimSpace(c.Query("account_id")), 10, 64)
	if err != nil || accountID <= 0 {
		response.ErrorFrom(c, badRequest("账号 ID 必须为正整数"))
		return
	}
	rangeValue := strings.TrimSpace(c.Query("range"))
	if rangeValue == "" {
		rangeValue = service.SupplierAccountHealthRange24h
	}
	result, err := h.service.GetTrend(c.Request.Context(), accountID, rangeValue)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

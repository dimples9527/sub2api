package admin

import (
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SupplierBalanceAlertHandler struct {
	service *service.SupplierBalanceAlertService
}

func NewSupplierBalanceAlertHandler(svc *service.SupplierBalanceAlertService) *SupplierBalanceAlertHandler {
	return &SupplierBalanceAlertHandler{service: svc}
}

func (h *SupplierBalanceAlertHandler) ListConfigs(c *gin.Context) {
	providerID, ok := parseSupplierNotificationOptionalID(c, "provider_id")
	if !ok {
		return
	}
	items, err := h.service.ListConfigs(c.Request.Context(), providerID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *SupplierBalanceAlertHandler) UpdateConfig(c *gin.Context) {
	providerID, ok := parseSupplierBalanceAlertPathID(c, "provider_id")
	if !ok {
		return
	}
	var input service.SupplierBalanceAlertConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	config, err := h.service.UpdateConfig(c.Request.Context(), providerID, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, config)
}

func (h *SupplierBalanceAlertHandler) Scan(c *gin.Context) {
	result, err := h.service.RunNow(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierBalanceAlertHandler) ListEvents(c *gin.Context) {
	providerID, ok := parseSupplierNotificationOptionalID(c, "provider_id")
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	result, err := h.service.ListEvents(c.Request.Context(), service.SupplierBalanceAlertEventListParams{
		ProviderID: providerID,
		EventType:  strings.TrimSpace(c.Query("event_type")),
		Status:     strings.TrimSpace(c.Query("status")),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, result.Items, result.Total, result.Page, result.PageSize)
}

func parseSupplierBalanceAlertPathID(c *gin.Context, name string) (int64, bool) {
	raw := strings.TrimSpace(c.Param(name))
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_SUPPLIER_ID", "供应商 ID 无效"))
		return 0, false
	}
	return id, true
}

func parseSupplierNotificationOptionalID(c *gin.Context, name string) (int64, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return 0, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_ID", "ID 无效"))
		return 0, false
	}
	return id, true
}

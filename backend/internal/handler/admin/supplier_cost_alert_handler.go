package admin

import (
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type SupplierCostAlertHandler struct {
	service *service.SupplierCostAlertService
}

func NewSupplierCostAlertHandler(svc *service.SupplierCostAlertService) *SupplierCostAlertHandler {
	return &SupplierCostAlertHandler{service: svc}
}

type updateSupplierCostAlertSettingsRequest struct {
	Amount string `json:"amount"`
}

// GET /admin/supplier-management/cost-alert-settings
func (h *SupplierCostAlertHandler) GetSettings(c *gin.Context) {
	item, err := h.service.GetSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

// PUT /admin/supplier-management/cost-alert-settings
func (h *SupplierCostAlertHandler) UpdateSettings(c *gin.Context) {
	var req updateSupplierCostAlertSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	amount, err := decimal.NewFromString(strings.TrimSpace(req.Amount))
	if err != nil {
		response.ErrorFrom(c, service.ErrSupplierCostAlertInvalid)
		return
	}
	item, err := h.service.UpdateSettings(c.Request.Context(), amount)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

// GET /admin/supplier-management/cost-alert-overrides
func (h *SupplierCostAlertHandler) ListOverrides(c *gin.Context) {
	items, err := h.service.ListOverrides(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *SupplierCostAlertHandler) upsertOverride(c *gin.Context, id int64) {
	var input service.SupplierCostAlertOverrideInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	if id > 0 && input.ProviderID == 0 {
		input.ProviderID = id
	}
	item, err := h.service.UpsertOverride(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

// POST /admin/supplier-management/cost-alert-overrides
func (h *SupplierCostAlertHandler) CreateOverride(c *gin.Context) {
	h.upsertOverride(c, 0)
}

// PUT /admin/supplier-management/cost-alert-overrides/:id
func (h *SupplierCostAlertHandler) UpdateOverride(c *gin.Context) {
	id, ok := parseSupplierNotificationPathID(c, "id")
	if !ok {
		return
	}
	h.upsertOverride(c, id)
}

// DELETE /admin/supplier-management/cost-alert-overrides/:id
func (h *SupplierCostAlertHandler) DeleteOverride(c *gin.Context) {
	id, ok := parseSupplierNotificationPathID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteOverride(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "覆盖配置已删除"})
}

// GET /admin/supplier-management/cost-alert-events
func (h *SupplierCostAlertHandler) ListEvents(c *gin.Context) {
	providerID, ok := parseSupplierNotificationOptionalID(c, "provider_id")
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	result, err := h.service.ListEvents(c.Request.Context(), service.SupplierCostAlertEventListParams{
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

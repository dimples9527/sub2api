package admin

import (
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// SupplierCostSourceConfigHandler 管理供应商成本来源全局配置与供应商覆盖配置。
type SupplierCostSourceConfigHandler struct {
	service *service.SupplierCostSourceConfigService
}

func NewSupplierCostSourceConfigHandler(svc *service.SupplierCostSourceConfigService) *SupplierCostSourceConfigHandler {
	return &SupplierCostSourceConfigHandler{service: svc}
}

type supplierCostSourceSettingsRequest struct {
	CostSource string `json:"cost_source"`
}

// GetSettings 获取全局默认成本来源。
// GET /api/v1/admin/supplier-management/cost-source/settings
func (h *SupplierCostSourceConfigHandler) GetSettings(c *gin.Context) {
	settings, err := h.service.GetSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

// UpdateSettings 更新全局默认成本来源。
// PUT /api/v1/admin/supplier-management/cost-source/settings
func (h *SupplierCostSourceConfigHandler) UpdateSettings(c *gin.Context) {
	var req supplierCostSourceSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	settings, err := h.service.UpdateGlobalCostSource(c.Request.Context(), strings.TrimSpace(req.CostSource))
	if err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_COST_SOURCE", err.Error()))
		return
	}
	response.Success(c, settings)
}

// ListOverrides 列出供应商成本来源覆盖配置。
// GET /api/v1/admin/supplier-management/cost-source/overrides
func (h *SupplierCostSourceConfigHandler) ListOverrides(c *gin.Context) {
	items, err := h.service.ListOverrides(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *SupplierCostSourceConfigHandler) upsertOverride(c *gin.Context, id int64) {
	var input service.SupplierCostSourceOverrideInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	if id > 0 && input.ProviderID == 0 {
		input.ProviderID = id
	}
	input.CostSource = strings.TrimSpace(input.CostSource)
	item, err := h.service.UpsertOverride(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_COST_SOURCE_OVERRIDE", err.Error()))
		return
	}
	response.Success(c, item)
}

// CreateOverride 新增供应商成本来源覆盖配置。
// POST /api/v1/admin/supplier-management/cost-source/overrides
func (h *SupplierCostSourceConfigHandler) CreateOverride(c *gin.Context) {
	h.upsertOverride(c, 0)
}

// UpdateOverride 更新供应商成本来源覆盖配置。
// PUT /api/v1/admin/supplier-management/cost-source/overrides/:id
func (h *SupplierCostSourceConfigHandler) UpdateOverride(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_COST_SOURCE_OVERRIDE_ID", "覆盖配置 ID 无效"))
		return
	}
	h.upsertOverride(c, id)
}

// DeleteOverride 删除供应商成本来源覆盖配置。
// DELETE /api/v1/admin/supplier-management/cost-source/overrides/:id
func (h *SupplierCostSourceConfigHandler) DeleteOverride(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_COST_SOURCE_OVERRIDE_ID", "覆盖配置 ID 无效"))
		return
	}
	if err := h.service.DeleteOverride(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "成本来源覆盖配置已删除"})
}

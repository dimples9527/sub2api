package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// SupplierCostDeviationSettingsHandler 管理供应商成本偏差覆盖阈值配置。
type SupplierCostDeviationSettingsHandler struct {
	service *service.SupplierCostDeviationSettingsService
}

func NewSupplierCostDeviationSettingsHandler(svc *service.SupplierCostDeviationSettingsService) *SupplierCostDeviationSettingsHandler {
	return &SupplierCostDeviationSettingsHandler{service: svc}
}

// GetSettings 获取供应商成本偏差覆盖阈值配置。
// GET /api/v1/admin/supplier-management/cost-deviation-settings
func (h *SupplierCostDeviationSettingsHandler) GetSettings(c *gin.Context) {
	settings, err := h.service.GetSupplierCostDeviationSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

type updateSupplierCostDeviationSettingsRequest struct {
	Threshold float64 `json:"threshold"`
}

// UpdateSettings 更新供应商成本偏差覆盖阈值配置。
// PUT /api/v1/admin/supplier-management/cost-deviation-settings
func (h *SupplierCostDeviationSettingsHandler) UpdateSettings(c *gin.Context) {
	var req updateSupplierCostDeviationSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	updated, err := h.service.UpdateSupplierCostDeviationSettings(c.Request.Context(), req.Threshold)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, updated)
}

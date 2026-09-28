package admin

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// emptyHolidayPromoDTO is the stable shape returned when no promo is configured.
func emptyHolidayPromoDTO() *dto.PaymentHolidayPromoDTO {
	return &dto.PaymentHolidayPromoDTO{Tiers: []dto.PaymentHolidayPromoTierDTO{}}
}

// GetRechargeHolidayPromo 获取节假日阶梯充值优惠配置
// GET /api/v1/admin/settings/recharge-holiday-promo
func (h *SettingHandler) GetRechargeHolidayPromo(c *gin.Context) {
	if h.paymentConfigService == nil {
		response.Success(c, emptyHolidayPromoDTO())
		return
	}
	promo, err := h.paymentConfigService.GetRechargeHolidayPromo(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	promoDTO := dto.PaymentHolidayPromoFromService(promo)
	if promoDTO == nil {
		promoDTO = emptyHolidayPromoDTO()
	}
	response.Success(c, promoDTO)
}

// UpdateRechargeHolidayPromo 保存节假日阶梯充值优惠配置
// PUT /api/v1/admin/settings/recharge-holiday-promo
func (h *SettingHandler) UpdateRechargeHolidayPromo(c *gin.Context) {
	if h.paymentConfigService == nil {
		response.Error(c, http.StatusServiceUnavailable, "payment config service unavailable")
		return
	}
	var req dto.PaymentHolidayPromoDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.paymentConfigService.UpdateRechargeHolidayPromo(c.Request.Context(), req.ToService()); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	promo, err := h.paymentConfigService.GetRechargeHolidayPromo(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	promoDTO := dto.PaymentHolidayPromoFromService(promo)
	if promoDTO == nil {
		promoDTO = emptyHolidayPromoDTO()
	}
	response.Success(c, promoDTO)
}

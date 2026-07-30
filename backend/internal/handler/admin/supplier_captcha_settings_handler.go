package admin

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// GetSupplierCaptchaSettings 获取供应商上游打码全局配置。
// GET /api/v1/admin/supplier-management/captcha-settings
func (h *SettingHandler) GetSupplierCaptchaSettings(c *gin.Context) {
	settings, err := h.settingService.GetSupplierCaptchaSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

type updateSupplierCaptchaSettingsRequest struct {
	Provider    string `json:"provider"`
	APIKey      string `json:"api_key"`
	Endpoint    string `json:"endpoint"`
	ClearAPIKey bool   `json:"clear_api_key"`
}

// UpdateSupplierCaptchaSettings 更新供应商上游打码全局配置。
// PUT /api/v1/admin/supplier-management/captcha-settings
func (h *SettingHandler) UpdateSupplierCaptchaSettings(c *gin.Context) {
	var req updateSupplierCaptchaSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	updated, err := h.settingService.UpdateSupplierCaptchaSettings(c.Request.Context(), &service.UpdateSupplierCaptchaSettingsInput{
		Provider:    strings.TrimSpace(req.Provider),
		APIKey:      strings.TrimSpace(req.APIKey),
		Endpoint:    strings.TrimSpace(req.Endpoint),
		ClearAPIKey: req.ClearAPIKey,
	})
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, updated)
}

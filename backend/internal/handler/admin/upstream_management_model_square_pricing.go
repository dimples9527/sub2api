package admin

import (
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// GetModelSquareModelPricing 获取模型广场使用的官方参考价格。
// 价格单位保持为 USD / Token，由前端转换为 USD / 1M Tokens 展示。
func (h *UpstreamManagementHandler) GetModelSquareModelPricing(c *gin.Context) {
	model := strings.TrimSpace(c.Query("model"))
	if model == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("MISSING_PARAMETER", "model 参数不能为空").
			WithMetadata(map[string]string{"param": "model"}))
		return
	}
	if h.billingService == nil {
		response.InternalError(c, "模型广场价格服务不可用")
		return
	}

	pricing, err := h.billingService.GetModelPricing(model)
	if err != nil {
		response.Success(c, gin.H{"found": false})
		return
	}
	response.Success(c, modelSquareModelPricingResponse(pricing))
}

func modelSquareModelPricingResponse(pricing *service.ModelPricing) gin.H {
	return gin.H{
		"found":                      true,
		"input_price":                pricing.InputPricePerToken,
		"output_price":               pricing.OutputPricePerToken,
		"cache_write_price":          pricing.CacheCreationPricePerToken,
		"cache_write_1h_price":       pricing.CacheCreation1hPrice,
		"cache_read_price":           pricing.CacheReadPricePerToken,
		"input_price_priority":       pricing.InputPricePerTokenPriority,
		"output_price_priority":      pricing.OutputPricePerTokenPriority,
		"cache_write_price_priority": pricing.CacheCreationPricePerTokenPriority,
		"cache_read_price_priority":  pricing.CacheReadPricePerTokenPriority,
		"image_input_price":          pricing.ImageInputPricePerToken,
		"image_output_price":         pricing.ImageOutputPricePerToken,
	}
}

package admin

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type UpstreamProviderHandler struct {
	service *service.UpstreamProviderService
}

func NewUpstreamProviderHandler(service *service.UpstreamProviderService) *UpstreamProviderHandler {
	return &UpstreamProviderHandler{service: service}
}

func (h *UpstreamProviderHandler) List(c *gin.Context) {
	providers, err := h.service.ListProviders(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, providers)
}

func (h *UpstreamProviderHandler) Create(c *gin.Context) {
	var req service.UpstreamProviderConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	provider, err := h.service.CreateProvider(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, provider)
}

func (h *UpstreamProviderHandler) Update(c *gin.Context) {
	var req service.UpstreamProviderConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	provider, err := h.service.UpdateProvider(c.Request.Context(), strings.TrimSpace(c.Param("slug")), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, provider)
}

func (h *UpstreamProviderHandler) Delete(c *gin.Context) {
	if err := h.service.DeleteProvider(c.Request.Context(), strings.TrimSpace(c.Param("slug"))); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "provider deleted"})
}

func (h *UpstreamProviderHandler) SetDefault(c *gin.Context) {
	provider, err := h.service.SetDefaultProvider(c.Request.Context(), strings.TrimSpace(c.Param("slug")))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, provider)
}

func (h *UpstreamProviderHandler) TestSaved(c *gin.Context) {
	result, err := h.service.TestProvider(c.Request.Context(), strings.TrimSpace(c.Param("slug")))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamProviderHandler) TestConfig(c *gin.Context) {
	var req service.UpstreamProviderConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.service.TestProviderConfig(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamProviderHandler) Keys(c *gin.Context) {
	keys, warnings, err := h.service.FetchProviderKeys(c.Request.Context(), strings.TrimSpace(c.Param("slug")))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"items":    keys,
		"warnings": warnings,
	})
}

func (h *UpstreamProviderHandler) Balance(c *gin.Context) {
	balance, err := h.service.FetchProviderBalanceStatus(c.Request.Context(), strings.TrimSpace(c.Param("slug")))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, balance)
}

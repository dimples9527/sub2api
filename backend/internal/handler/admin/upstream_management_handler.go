package admin

import (
	"context"
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type upstreamManagementService interface {
	CompareGroups(ctx context.Context) (service.UpstreamGroupCompareResult, error)
	ApplyRateFixes(ctx context.Context) (service.UpstreamGroupCompareResult, error)
	SaveGroupMapping(ctx context.Context, input service.UpstreamGroupMappingInput) (service.UpstreamGroupCompareResult, error)
}

type upstreamModelSquareService interface {
	FetchDefaultModelSquare(ctx context.Context) (json.RawMessage, service.UpstreamProviderConfig, error)
}

type UpstreamManagementHandler struct {
	service upstreamManagementService
}

func NewUpstreamManagementHandler(service *service.UpstreamManagementService) *UpstreamManagementHandler {
	return &UpstreamManagementHandler{service: service}
}

func newUpstreamManagementHandlerWithService(service upstreamManagementService) *UpstreamManagementHandler {
	return &UpstreamManagementHandler{service: service}
}

func (h *UpstreamManagementHandler) CompareGroups(c *gin.Context) {
	result, err := h.service.CompareGroups(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamManagementHandler) ApplyRateFixes(c *gin.Context) {
	result, err := h.service.ApplyRateFixes(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamManagementHandler) SaveGroupMapping(c *gin.Context) {
	var input service.UpstreamGroupMappingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.service.SaveGroupMapping(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamManagementHandler) ModelSquare(c *gin.Context) {
	modelSquareService, ok := h.service.(upstreamModelSquareService)
	if !ok {
		response.InternalError(c, "upstream model square service is unavailable")
		return
	}
	payload, provider, err := modelSquareService.FetchDefaultModelSquare(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"provider_slug": provider.Slug,
		"provider_name": provider.Name,
		"provider_type": provider.Type,
		"payload":       payload,
	})
}

package admin

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type upstreamDashboardService interface {
	Get(ctx context.Context, rangeValue service.UpstreamDashboardRange) (service.UpstreamDashboardResponse, error)
}

type UpstreamDashboardHandler struct {
	service upstreamDashboardService
}

func NewUpstreamDashboardHandler(service *service.UpstreamDashboardService) *UpstreamDashboardHandler {
	return &UpstreamDashboardHandler{service: service}
}

func newUpstreamDashboardHandlerWithService(service upstreamDashboardService) *UpstreamDashboardHandler {
	return &UpstreamDashboardHandler{service: service}
}

func (h *UpstreamDashboardHandler) Get(c *gin.Context) {
	rangeValue := service.UpstreamDashboardRange(c.DefaultQuery("range", string(service.UpstreamDashboardRange24Hours)))
	if rangeValue != service.UpstreamDashboardRange24Hours && rangeValue != service.UpstreamDashboardRange7Days {
		response.BadRequest(c, "range must be 24h or 7d")
		return
	}
	result, err := h.service.Get(c.Request.Context(), rangeValue)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

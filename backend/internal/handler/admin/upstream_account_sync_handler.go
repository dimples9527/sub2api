package admin

import (
	"context"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type upstreamAccountSyncService interface {
	Preview(ctx context.Context) (service.UpstreamAccountSyncResult, error)
	Sync(ctx context.Context, req service.UpstreamAccountSyncRequest) (service.UpstreamAccountSyncResult, error)
	ListRecords(ctx context.Context) ([]service.UpstreamAccountSyncRecord, error)
}

type UpstreamAccountSyncHandler struct {
	service upstreamAccountSyncService
}

func NewUpstreamAccountSyncHandler(service *service.UpstreamAccountSyncService) *UpstreamAccountSyncHandler {
	return &UpstreamAccountSyncHandler{service: service}
}

func newUpstreamAccountSyncHandlerWithService(service upstreamAccountSyncService) *UpstreamAccountSyncHandler {
	return &UpstreamAccountSyncHandler{service: service}
}

func (h *UpstreamAccountSyncHandler) Preview(c *gin.Context) {
	result, err := h.service.Preview(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamAccountSyncHandler) Sync(c *gin.Context) {
	req := service.UpstreamAccountSyncRequest{
		CreateMissing:  true,
		UpdateExisting: true,
		ApplyRateGuard: true,
	}
	if c.Request.Body != nil && c.Request.Body != http.NoBody && c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "Invalid request: "+err.Error())
			return
		}
	}
	result, err := h.service.Sync(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamAccountSyncHandler) Records(c *gin.Context) {
	records, err := h.service.ListRecords(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, records)
}

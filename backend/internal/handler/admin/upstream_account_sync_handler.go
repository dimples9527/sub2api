package admin

import (
	"context"
	"net/http"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type upstreamAccountSyncService interface {
	Preview(ctx context.Context) (service.UpstreamAccountSyncResult, error)
	Sync(ctx context.Context, req service.UpstreamAccountSyncRequest) (service.UpstreamAccountSyncResult, error)
	ListRecords(ctx context.Context) ([]service.UpstreamAccountSyncRecord, error)
	GetRateGuardConfig(ctx context.Context) (service.UpstreamAccountRateGuardConfig, error)
	UpdateRateGuardConfig(ctx context.Context, input service.UpstreamAccountRateGuardConfig) (service.UpstreamAccountRateGuardConfig, error)
}

type upstreamAccountRateGuardScheduler interface {
	RunNow(ctx context.Context) (service.UpstreamAccountRateGuardConfig, error)
	ListPollLogs() []service.UpstreamAccountRateGuardPollLog
}

type UpstreamAccountSyncHandler struct {
	service   upstreamAccountSyncService
	scheduler upstreamAccountRateGuardScheduler
}

func NewUpstreamAccountSyncHandler(service *service.UpstreamAccountSyncService, scheduler *service.UpstreamAccountRateGuardScheduler) *UpstreamAccountSyncHandler {
	return &UpstreamAccountSyncHandler{service: service, scheduler: scheduler}
}

func newUpstreamAccountSyncHandlerWithService(service upstreamAccountSyncService) *UpstreamAccountSyncHandler {
	return &UpstreamAccountSyncHandler{service: service}
}

func newUpstreamAccountSyncHandlerWithDeps(service upstreamAccountSyncService, scheduler upstreamAccountRateGuardScheduler) *UpstreamAccountSyncHandler {
	return &UpstreamAccountSyncHandler{service: service, scheduler: scheduler}
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

func (h *UpstreamAccountSyncHandler) GetRateGuardConfig(c *gin.Context) {
	result, err := h.service.GetRateGuardConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamAccountSyncHandler) UpdateRateGuardConfig(c *gin.Context) {
	var input service.UpstreamAccountRateGuardConfig
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.service.UpdateRateGuardConfig(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamAccountSyncHandler) RunRateGuardNow(c *gin.Context) {
	if h.scheduler == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("UPSTREAM_ACCOUNT_RATE_GUARD_UNAVAILABLE", "upstream account rate guard scheduler is unavailable"))
		return
	}
	result, err := h.scheduler.RunNow(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamAccountSyncHandler) RateGuardPollLogs(c *gin.Context) {
	if h.scheduler == nil {
		response.Success(c, []service.UpstreamAccountRateGuardPollLog{})
		return
	}
	response.Success(c, h.scheduler.ListPollLogs())
}

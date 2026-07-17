package admin

import (
	"context"
	"net/http"
	"strconv"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type upstreamAccountSyncService interface {
	Preview(ctx context.Context) (service.UpstreamAccountSyncResult, error)
	Sync(ctx context.Context, req service.UpstreamAccountSyncRequest) (service.UpstreamAccountSyncResult, error)
	ListRecords(ctx context.Context) ([]service.UpstreamAccountSyncRecord, error)
	MarkRecordHandled(ctx context.Context, key string) ([]service.UpstreamAccountSyncRecord, error)
	GetRateGuardConfig(ctx context.Context) (service.UpstreamAccountRateGuardConfig, error)
	UpdateRateGuardConfig(ctx context.Context, input service.UpstreamAccountRateGuardConfig) (service.UpstreamAccountRateGuardConfig, error)
}

type upstreamAccountRateGuardScheduler interface {
	RunNow(ctx context.Context) (service.UpstreamAccountRateGuardConfig, error)
	ListPollLogs() []service.UpstreamAccountRateGuardPollLog
}

type upstreamBalanceConsumptionService interface {
	GetOverview(ctx context.Context, days int) (service.UpstreamBalanceConsumptionOverview, error)
	GetConfig(ctx context.Context) (service.UpstreamBalanceSamplerConfig, error)
	UpdateConfig(ctx context.Context, input service.UpstreamBalanceSamplerConfig) (service.UpstreamBalanceSamplerConfig, error)
	AddRecharge(ctx context.Context, input service.UpstreamBalanceRechargeInput) (service.UpstreamBalanceRecharge, error)
}

type upstreamBalanceSamplerScheduler interface {
	RunNow(ctx context.Context) (service.UpstreamBalanceSamplerConfig, error)
	ListPollLogs() []service.UpstreamBalanceSamplerPollLog
}

type upstreamAccountHealthGuardService interface {
	GetConfig(ctx context.Context) (service.UpstreamAccountHealthGuardConfig, error)
	UpdateConfig(ctx context.Context, input service.UpstreamAccountHealthGuardConfig) (service.UpstreamAccountHealthGuardConfig, error)
	ListRecords(ctx context.Context) ([]service.UpstreamAccountHealthGuardRunRecord, error)
}

type upstreamAccountHealthGuardScheduler interface {
	RunNow(ctx context.Context) (service.UpstreamAccountHealthGuardRunResponse, error)
	ListPollLogs() []service.UpstreamAccountHealthGuardPollLog
}

type UpstreamAccountSyncHandler struct {
	service          upstreamAccountSyncService
	scheduler        upstreamAccountRateGuardScheduler
	balance          upstreamBalanceConsumptionService
	balanceScheduler upstreamBalanceSamplerScheduler
	healthGuard      upstreamAccountHealthGuardService
	healthScheduler  upstreamAccountHealthGuardScheduler
}

func NewUpstreamAccountSyncHandler(
	service *service.UpstreamAccountSyncService,
	scheduler *service.UpstreamAccountRateGuardScheduler,
	balance *service.UpstreamBalanceConsumptionService,
	balanceScheduler *service.UpstreamBalanceSamplerScheduler,
	healthGuard *service.UpstreamAccountHealthGuardService,
	healthScheduler *service.UpstreamAccountHealthGuardScheduler,
) *UpstreamAccountSyncHandler {
	return &UpstreamAccountSyncHandler{
		service:          service,
		scheduler:        scheduler,
		balance:          balance,
		balanceScheduler: balanceScheduler,
		healthGuard:      healthGuard,
		healthScheduler:  healthScheduler,
	}
}

func newUpstreamAccountSyncHandlerWithService(service upstreamAccountSyncService) *UpstreamAccountSyncHandler {
	return &UpstreamAccountSyncHandler{service: service}
}

func newUpstreamAccountSyncHandlerWithDeps(service upstreamAccountSyncService, scheduler upstreamAccountRateGuardScheduler) *UpstreamAccountSyncHandler {
	balance, _ := service.(upstreamBalanceConsumptionService)
	return &UpstreamAccountSyncHandler{service: service, scheduler: scheduler, balance: balance}
}

func newUpstreamAccountSyncHandlerWithAllDeps(service upstreamAccountSyncService, scheduler upstreamAccountRateGuardScheduler, balance upstreamBalanceConsumptionService, balanceScheduler upstreamBalanceSamplerScheduler) *UpstreamAccountSyncHandler {
	if balance == nil {
		balance, _ = service.(upstreamBalanceConsumptionService)
	}
	return &UpstreamAccountSyncHandler{service: service, scheduler: scheduler, balance: balance, balanceScheduler: balanceScheduler}
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

func (h *UpstreamAccountSyncHandler) MarkRecordHandled(c *gin.Context) {
	records, err := h.service.MarkRecordHandled(c.Request.Context(), c.Param("key"))
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

func (h *UpstreamAccountSyncHandler) BalanceConsumptionOverview(c *gin.Context) {
	if h.balance == nil {
		response.InternalError(c, "upstream balance consumption service is unavailable")
		return
	}
	days := 30
	if raw := c.Query("days"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			days = parsed
		}
	}
	result, err := h.balance.GetOverview(c.Request.Context(), days)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamAccountSyncHandler) GetBalanceSamplerConfig(c *gin.Context) {
	if h.balance == nil {
		response.InternalError(c, "upstream balance consumption service is unavailable")
		return
	}
	result, err := h.balance.GetConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamAccountSyncHandler) UpdateBalanceSamplerConfig(c *gin.Context) {
	if h.balance == nil {
		response.InternalError(c, "upstream balance consumption service is unavailable")
		return
	}
	var input service.UpstreamBalanceSamplerConfig
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.balance.UpdateConfig(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamAccountSyncHandler) AddBalanceRecharge(c *gin.Context) {
	if h.balance == nil {
		response.InternalError(c, "upstream balance consumption service is unavailable")
		return
	}
	var input service.UpstreamBalanceRechargeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.balance.AddRecharge(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamAccountSyncHandler) RunBalanceSampleNow(c *gin.Context) {
	if h.balanceScheduler == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("UPSTREAM_BALANCE_SAMPLER_UNAVAILABLE", "upstream balance sampler scheduler is unavailable"))
		return
	}
	result, err := h.balanceScheduler.RunNow(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamAccountSyncHandler) BalanceSamplerPollLogs(c *gin.Context) {
	if h.balanceScheduler == nil {
		response.Success(c, []service.UpstreamBalanceSamplerPollLog{})
		return
	}
	response.Success(c, h.balanceScheduler.ListPollLogs())
}

func (h *UpstreamAccountSyncHandler) GetHealthGuardConfig(c *gin.Context) {
	if h.healthGuard == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("UPSTREAM_ACCOUNT_HEALTH_GUARD_UNAVAILABLE", "upstream account health guard service is unavailable"))
		return
	}
	result, err := h.healthGuard.GetConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamAccountSyncHandler) UpdateHealthGuardConfig(c *gin.Context) {
	if h.healthGuard == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("UPSTREAM_ACCOUNT_HEALTH_GUARD_UNAVAILABLE", "upstream account health guard service is unavailable"))
		return
	}
	var input service.UpstreamAccountHealthGuardConfig
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.healthGuard.UpdateConfig(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamAccountSyncHandler) RunHealthGuardNow(c *gin.Context) {
	if h.healthScheduler == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("UPSTREAM_ACCOUNT_HEALTH_GUARD_UNAVAILABLE", "upstream account health guard scheduler is unavailable"))
		return
	}
	result, err := h.healthScheduler.RunNow(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamAccountSyncHandler) HealthGuardRecords(c *gin.Context) {
	if h.healthGuard == nil {
		response.Success(c, []service.UpstreamAccountHealthGuardRunRecord{})
		return
	}
	result, err := h.healthGuard.ListRecords(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamAccountSyncHandler) HealthGuardPollLogs(c *gin.Context) {
	if h.healthScheduler == nil {
		response.Success(c, []service.UpstreamAccountHealthGuardPollLog{})
		return
	}
	response.Success(c, h.healthScheduler.ListPollLogs())
}

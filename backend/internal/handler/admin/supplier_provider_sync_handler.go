package admin

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SupplierProviderSyncServicePort interface {
	SyncAccounts(ctx context.Context, providerID int64, trigger string) (service.SupplierProviderSyncResult, error)
	SyncGroups(ctx context.Context, providerID int64, trigger string) (service.SupplierProviderSyncResult, error)
	SyncBalance(ctx context.Context, providerID int64, trigger string) (service.SupplierProviderSyncResult, error)
	SyncCost(ctx context.Context, providerID int64, day time.Time, trigger string) (service.SupplierProviderSyncResult, error)
	BackfillCosts(ctx context.Context, startDate, endDate string, providerID int64, trigger string) (service.SupplierProviderCostBackfillResult, error)
	SyncAll(ctx context.Context, providerID int64, trigger string) (service.SupplierProviderSyncResult, error)
	TestEndpoint(ctx context.Context, providerID int64, scope string) (service.SupplierProviderEndpointTestResult, error)
	RefreshToken(ctx context.Context, providerID int64) (service.SupplierProviderAuthToken, error)
}

type SupplierProviderDataRepositoryPort interface {
	ListAccounts(ctx context.Context, params service.SupplierProviderDataListParams) (service.SupplierProviderAccountListResult, error)
	IsUniqueMatchedLocalAccount(ctx context.Context, localAccountID int64) (bool, error)
	GetLocalAccountEffectivePlatform(ctx context.Context, localAccountID int64) (string, error)
	SetLocalAccountPlatformOverride(ctx context.Context, localAccountID int64, platform string) error
	ClearLocalAccountPlatformOverride(ctx context.Context, localAccountID int64) error
	ListGroups(ctx context.Context, params service.SupplierProviderDataListParams) (service.SupplierProviderGroupListResult, error)
	ListGroupHealthTrends(ctx context.Context, params service.SupplierProviderGroupHealthTrendParams) ([]service.SupplierProviderGroupHealthTrend, error)
	ListLocalGroupHealthTrends(ctx context.Context, params service.SupplierProviderGroupHealthTrendParams) ([]service.SupplierProviderGroupHealthTrend, error)
	ListMonitorTargets(ctx context.Context, params service.SupplierProviderMonitorTargetListParams) (service.SupplierProviderMonitorTargetListResult, error)
	BindMonitorTarget(ctx context.Context, monitorTargetID, localAccountID int64) error
	UnbindMonitorTarget(ctx context.Context, monitorTargetID int64) error
	ListMappingsByLocalGroup(ctx context.Context, localGroupIDs []int64) ([]service.SupplierProviderGroup, error)
	UpdateGroupMapping(ctx context.Context, groupID int64, localGroupID *int64) error
	DeleteGroup(ctx context.Context, groupID int64) error
	DeleteAccount(ctx context.Context, accountID int64) error
}

type SupplierProviderGroupMatcherPort interface {
	AutoMatch(ctx context.Context, providerID int64) (service.SupplierGroupAutoMatchResult, error)
	UpdateMapping(ctx context.Context, groupID int64, localGroupID *int64) error
	SetIgnored(ctx context.Context, groupID int64, ignored bool) (service.SupplierGroupAutoMatchResult, error)
	ResolveNameChange(ctx context.Context, groupID int64, action string) error
}

type SupplierGroupGuardPort interface {
	SetManualGuard(ctx context.Context, groupID int64, selected bool) error
	SetRateGuardEnabled(ctx context.Context, groupID int64, enabled bool) error
}

// SupplierCustomPlatformResolver 仅负责校验启用的自定义平台，保持处理器对完整服务接口的最小依赖。
type SupplierCustomPlatformResolver interface {
	ResolveEnabled(ctx context.Context, code string) (*service.CustomPlatform, error)
}

type SupplierProviderSyncHandler struct {
	syncService            SupplierProviderSyncServicePort
	dataRepo               SupplierProviderDataRepositoryPort
	groupMatcher           SupplierProviderGroupMatcherPort
	groupGuard             SupplierGroupGuardPort
	customPlatformResolver SupplierCustomPlatformResolver
	groupPlatformOverride  service.MonitorGroupPlatformOverrideService
}

func (h *SupplierProviderSyncHandler) SetGroupGuard(guard SupplierGroupGuardPort) {
	if h != nil {
		h.groupGuard = guard
	}
}

func (h *SupplierProviderSyncHandler) SetGroupMatcher(matcher SupplierProviderGroupMatcherPort) {
	if h != nil {
		h.groupMatcher = matcher
	}
}

func (h *SupplierProviderSyncHandler) SetCustomPlatformResolver(resolver SupplierCustomPlatformResolver) {
	if h != nil {
		h.customPlatformResolver = resolver
	}
}

func (h *SupplierProviderSyncHandler) SetGroupPlatformOverrideService(overrideService service.MonitorGroupPlatformOverrideService) {
	if h != nil {
		h.groupPlatformOverride = overrideService
	}
}

func NewSupplierProviderSyncHandler(syncService SupplierProviderSyncServicePort, dataRepo SupplierProviderDataRepositoryPort) *SupplierProviderSyncHandler {
	return &SupplierProviderSyncHandler{syncService: syncService, dataRepo: dataRepo}
}

func (h *SupplierProviderSyncHandler) SyncAccounts(c *gin.Context) {
	h.sync(c, func(ctx context.Context, id int64) (service.SupplierProviderSyncResult, error) {
		return h.syncService.SyncAccounts(ctx, id, service.SupplierSyncTriggerManual)
	})
}

func (h *SupplierProviderSyncHandler) SyncGroups(c *gin.Context) {
	h.sync(c, func(ctx context.Context, id int64) (service.SupplierProviderSyncResult, error) {
		return h.syncService.SyncGroups(ctx, id, service.SupplierSyncTriggerManual)
	})
}

func (h *SupplierProviderSyncHandler) SyncBalance(c *gin.Context) {
	h.sync(c, func(ctx context.Context, id int64) (service.SupplierProviderSyncResult, error) {
		return h.syncService.SyncBalance(ctx, id, service.SupplierSyncTriggerManual)
	})
}

func (h *SupplierProviderSyncHandler) SyncCost(c *gin.Context) {
	day, ok := supplierSyncCostDay(c.Query("date"))
	if !ok {
		response.ErrorFrom(c, badRequest("成本日期无效，需为 YYYY-MM-DD"))
		return
	}
	h.sync(c, func(ctx context.Context, id int64) (service.SupplierProviderSyncResult, error) {
		return h.syncService.SyncCost(ctx, id, day, service.SupplierSyncTriggerManual)
	})
}

func (h *SupplierProviderSyncHandler) BackfillCosts(c *gin.Context) {
	var body struct {
		StartDate  string `json:"start_date"`
		EndDate    string `json:"end_date"`
		ProviderID int64  `json:"provider_id"`
	}
	// 同时支持 JSON body 与 query，便于前端简单调用。
	_ = c.ShouldBindJSON(&body)
	if strings.TrimSpace(body.StartDate) == "" {
		body.StartDate = strings.TrimSpace(c.Query("start_date"))
	}
	if strings.TrimSpace(body.EndDate) == "" {
		body.EndDate = strings.TrimSpace(c.Query("end_date"))
	}
	if body.ProviderID == 0 {
		if raw := strings.TrimSpace(c.Query("provider_id")); raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || parsed < 0 {
				response.ErrorFrom(c, infraerrors.BadRequest("INVALID_COST_BACKFILL_PROVIDER", "provider_id must be a non-negative integer"))
				return
			}
			body.ProviderID = parsed
		}
	}
	if strings.TrimSpace(body.StartDate) == "" || strings.TrimSpace(body.EndDate) == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_COST_BACKFILL_RANGE", "start_date and end_date are both required"))
		return
	}

	result, err := h.syncService.BackfillCosts(c.Request.Context(), body.StartDate, body.EndDate, body.ProviderID, service.SupplierSyncTriggerManual)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierProviderSyncHandler) SyncAll(c *gin.Context) {
	h.sync(c, func(ctx context.Context, id int64) (service.SupplierProviderSyncResult, error) {
		return h.syncService.SyncAll(ctx, id, service.SupplierSyncTriggerManual)
	})
}

func (h *SupplierProviderSyncHandler) SyncAccountsStream(c *gin.Context) {
	h.syncStream(c, func(ctx context.Context, id int64) (service.SupplierProviderSyncResult, error) {
		return h.syncService.SyncAccounts(ctx, id, service.SupplierSyncTriggerManual)
	})
}

func (h *SupplierProviderSyncHandler) SyncGroupsStream(c *gin.Context) {
	h.syncStream(c, func(ctx context.Context, id int64) (service.SupplierProviderSyncResult, error) {
		return h.syncService.SyncGroups(ctx, id, service.SupplierSyncTriggerManual)
	})
}

func (h *SupplierProviderSyncHandler) SyncBalanceStream(c *gin.Context) {
	h.syncStream(c, func(ctx context.Context, id int64) (service.SupplierProviderSyncResult, error) {
		return h.syncService.SyncBalance(ctx, id, service.SupplierSyncTriggerManual)
	})
}

func (h *SupplierProviderSyncHandler) SyncCostStream(c *gin.Context) {
	h.syncStream(c, func(ctx context.Context, id int64) (service.SupplierProviderSyncResult, error) {
		return h.syncService.SyncCost(ctx, id, time.Now(), service.SupplierSyncTriggerManual)
	})
}

func (h *SupplierProviderSyncHandler) SyncAllStream(c *gin.Context) {
	h.syncStream(c, func(ctx context.Context, id int64) (service.SupplierProviderSyncResult, error) {
		return h.syncService.SyncAll(ctx, id, service.SupplierSyncTriggerManual)
	})
}

func (h *SupplierProviderSyncHandler) TestEndpoint(c *gin.Context) {
	id, ok := parseSupplierProviderID(c)
	if !ok {
		return
	}
	scope := strings.TrimSpace(c.Param("scope"))
	if !supplierProviderTestScopeAllowed(scope) {
		response.ErrorFrom(c, badRequest("不支持的测试接口"))
		return
	}
	result, err := h.syncService.TestEndpoint(c.Request.Context(), id, scope)
	if err != nil {
		if message := strings.TrimSpace(result.Error); message != "" {
			err = infraerrors.InternalServer("SUPPLIER_PROVIDER_ENDPOINT_TEST_FAILED", message)
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierProviderSyncHandler) sync(c *gin.Context, fn func(context.Context, int64) (service.SupplierProviderSyncResult, error)) {
	id, ok := parseSupplierProviderID(c)
	if !ok {
		return
	}
	result, err := fn(c.Request.Context(), id)
	if err != nil {
		if message := strings.TrimSpace(result.Message); message != "" {
			err = infraerrors.InternalServer("SUPPLIER_PROVIDER_SYNC_FAILED", message)
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierProviderSyncHandler) syncStream(c *gin.Context, fn func(context.Context, int64) (service.SupplierProviderSyncResult, error)) {
	id, ok := parseSupplierProviderID(c)
	if !ok {
		return
	}
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		response.ErrorFrom(c, badRequest("当前响应不支持同步进度流"))
		return
	}

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	streamCtx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	stream := newSupplierSyncProgressStream(c.Writer, flusher, streamCtx.Done(), cancel)
	ctx := service.WithSupplierSyncProgressObserver(streamCtx, stream.write)
	result, err := fn(ctx, id)
	if err != nil {
		terminalErr := err
		if message := strings.TrimSpace(result.Message); message != "" {
			terminalErr = errors.New(message)
		} else if errors.Is(err, service.ErrSupplierProviderDisabled) {
			terminalErr = errors.New("供应商已停用，请先启用后再同步")
		}
		service.SupplierSyncProgressFail(ctx, service.SupplierSyncProgressStageError, terminalErr)
		return
	}
	if result.Status == service.SupplierSyncStatusSuccess {
		service.SupplierSyncProgressOK(ctx, service.SupplierSyncProgressStageDone, result.Message)
		return
	}
	service.SupplierSyncProgressFail(ctx, service.SupplierSyncProgressStageDone, errors.New(result.Message))
}

type supplierSyncProgressStream struct {
	writer      io.Writer
	flusher     http.Flusher
	requestDone <-chan struct{}
	cancel      context.CancelFunc
	mu          sync.Mutex
	closed      bool
}

func newSupplierSyncProgressStream(writer io.Writer, flusher http.Flusher, requestDone <-chan struct{}, cancel context.CancelFunc) *supplierSyncProgressStream {
	return &supplierSyncProgressStream{
		writer:      writer,
		flusher:     flusher,
		requestDone: requestDone,
		cancel:      cancel,
	}
}

func (s *supplierSyncProgressStream) closeLocked() {
	if s.closed {
		return
	}
	s.closed = true
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *supplierSyncProgressStream) write(event service.SupplierSyncProgressEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	if s.requestDone != nil {
		select {
		case <-s.requestDone:
			s.closeLocked()
			return
		default:
		}
	}
	payload, err := json.Marshal(event)
	if err != nil {
		s.closeLocked()
		return
	}
	if _, err := io.WriteString(s.writer, "data: "); err != nil {
		s.closeLocked()
		return
	}
	if _, err := s.writer.Write(payload); err != nil {
		s.closeLocked()
		return
	}
	if _, err := io.WriteString(s.writer, "\n\n"); err != nil {
		s.closeLocked()
		return
	}
	s.flusher.Flush()
}

func (h *SupplierProviderSyncHandler) ListAccounts(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > supplierProviderMaxPageSize {
		pageSize = supplierProviderMaxPageSize
	}
	result, err := h.dataRepo.ListAccounts(c.Request.Context(), service.SupplierProviderDataListParams{
		ProviderID: parseOptionalInt64(c.Query("provider_id")),
		GroupID:    parseOptionalInt64(c.Query("group_id")),
		Active:     parseSupplierProviderEnabled(c.Query("active")),
		Status:     strings.TrimSpace(c.Query("status")),
		Search:     strings.TrimSpace(c.Query("search")),
		Platform:   strings.TrimSpace(c.Query("platform")),
		SortBy:     strings.TrimSpace(c.Query("sort_by")),
		SortOrder:  strings.TrimSpace(c.Query("sort_order")),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierProviderSyncHandler) ListMonitorTargets(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > supplierProviderMaxPageSize {
		pageSize = supplierProviderMaxPageSize
	}
	result, err := h.dataRepo.ListMonitorTargets(c.Request.Context(), service.SupplierProviderMonitorTargetListParams{
		ProviderID: parseOptionalInt64(c.Query("provider_id")),
		Active:     parseSupplierProviderEnabled(c.Query("active")),
		Search:     strings.TrimSpace(c.Query("search")),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierProviderSyncHandler) BindMonitorTarget(c *gin.Context) {
	monitorTargetID, ok := parseSupplierMonitorTargetID(c)
	if !ok {
		return
	}
	var input struct {
		LocalAccountID int64 `json:"local_account_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.LocalAccountID <= 0 {
		response.ErrorFrom(c, badRequest("本地账号 ID 无效"))
		return
	}
	if err := h.dataRepo.BindMonitorTarget(c.Request.Context(), monitorTargetID, input.LocalAccountID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"monitor_target_id": monitorTargetID, "local_account_id": input.LocalAccountID})
}

func (h *SupplierProviderSyncHandler) UnbindMonitorTarget(c *gin.Context) {
	monitorTargetID, ok := parseSupplierMonitorTargetID(c)
	if !ok {
		return
	}
	if err := h.dataRepo.UnbindMonitorTarget(c.Request.Context(), monitorTargetID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"monitor_target_id": monitorTargetID})
}

type supplierLocalAccountPlatformOverrideRequest struct {
	Platform string `json:"platform"`
}

type supplierLocalGroupPlatformOverrideRequest struct {
	Platform string `json:"platform"`
}

type supplierLocalAccountHealthGuardModel struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

func (h *SupplierProviderSyncHandler) ListLocalAccountHealthGuardModels(c *gin.Context) {
	localAccountID, ok := parseSupplierLocalAccountID(c)
	if !ok {
		return
	}
	matched, err := h.dataRepo.IsUniqueMatchedLocalAccount(c.Request.Context(), localAccountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !matched {
		response.ErrorFrom(c, badRequest("本地账号不属于供应商模块的唯一匹配账号"))
		return
	}
	platform, err := h.dataRepo.GetLocalAccountEffectivePlatform(c.Request.Context(), localAccountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	models, supported := supplierLocalAccountHealthGuardModels(platform)
	if !supported && h.customPlatformResolver != nil {
		_, resolveErr := h.customPlatformResolver.ResolveEnabled(c.Request.Context(), strings.ToLower(strings.TrimSpace(platform)))
		supported = resolveErr == nil
	}
	if !supported {
		response.ErrorFrom(c, badRequest("业务平台无效"))
		return
	}
	if models == nil {
		models = make([]supplierLocalAccountHealthGuardModel, 0)
	}
	response.Success(c, models)
}

func supplierLocalAccountHealthGuardModels(platform string) ([]supplierLocalAccountHealthGuardModel, bool) {
	models := make([]supplierLocalAccountHealthGuardModel, 0)
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case service.PlatformOpenAI:
		for _, model := range openai.DefaultModels {
			models = append(models, supplierLocalAccountHealthGuardModel{ID: model.ID, DisplayName: model.DisplayName})
		}
	case service.PlatformAnthropic:
		for _, model := range claude.DefaultModels {
			models = append(models, supplierLocalAccountHealthGuardModel{ID: model.ID, DisplayName: model.DisplayName})
		}
	case service.PlatformGemini:
		for _, model := range geminicli.DefaultModels {
			models = append(models, supplierLocalAccountHealthGuardModel{ID: model.ID, DisplayName: model.DisplayName})
		}
	case service.PlatformAntigravity:
		for _, model := range antigravity.DefaultModels() {
			models = append(models, supplierLocalAccountHealthGuardModel{ID: model.ID, DisplayName: model.DisplayName})
		}
	case service.PlatformGrok:
		for _, model := range xai.DefaultModels() {
			models = append(models, supplierLocalAccountHealthGuardModel{ID: model.ID, DisplayName: model.DisplayName})
		}
	default:
		return nil, false
	}
	return models, true
}

func (h *SupplierProviderSyncHandler) SetLocalAccountPlatformOverride(c *gin.Context) {
	localAccountID, ok := parseSupplierLocalAccountID(c)
	if !ok {
		return
	}
	var req supplierLocalAccountPlatformOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, badRequest("业务平台参数无效"))
		return
	}
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	if !h.isAllowedBusinessPlatform(c.Request.Context(), platform) {
		response.ErrorFrom(c, badRequest("业务平台无效"))
		return
	}
	matched, err := h.dataRepo.IsUniqueMatchedLocalAccount(c.Request.Context(), localAccountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !matched {
		response.ErrorFrom(c, badRequest("本地账号不属于供应商模块的唯一匹配账号"))
		return
	}
	if err := h.dataRepo.SetLocalAccountPlatformOverride(c.Request.Context(), localAccountID, platform); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"local_account_id": localAccountID, "platform_override": platform})
}

func (h *SupplierProviderSyncHandler) isAllowedBusinessPlatform(ctx context.Context, platform string) bool {
	if service.IsAllowedQuotaPlatform(platform) {
		return true
	}
	if h == nil || h.customPlatformResolver == nil {
		return false
	}
	_, err := h.customPlatformResolver.ResolveEnabled(ctx, platform)
	return err == nil
}

func (h *SupplierProviderSyncHandler) ClearLocalAccountPlatformOverride(c *gin.Context) {
	localAccountID, ok := parseSupplierLocalAccountID(c)
	if !ok {
		return
	}
	matched, err := h.dataRepo.IsUniqueMatchedLocalAccount(c.Request.Context(), localAccountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !matched {
		response.ErrorFrom(c, badRequest("本地账号不属于供应商模块的唯一匹配账号"))
		return
	}
	if err := h.dataRepo.ClearLocalAccountPlatformOverride(c.Request.Context(), localAccountID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"local_account_id": localAccountID, "platform_override": ""})
}

func parseSupplierLocalAccountID(c *gin.Context) (int64, bool) {
	rawID := strings.TrimSpace(c.Param("local_account_id"))
	if rawID == "" {
		rawID = strings.TrimSpace(c.Param("id"))
	}
	localAccountID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || localAccountID <= 0 {
		response.ErrorFrom(c, badRequest("本地账号 ID 无效"))
		return 0, false
	}
	return localAccountID, true
}

func (h *SupplierProviderSyncHandler) SetLocalGroupPlatformOverride(c *gin.Context) {
	localGroupID, ok := parseSupplierLocalGroupID(c)
	if !ok {
		return
	}
	var req supplierLocalGroupPlatformOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, badRequest("业务平台参数无效"))
		return
	}
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	if !h.isAllowedBusinessPlatform(c.Request.Context(), platform) {
		response.ErrorFrom(c, badRequest("业务平台无效"))
		return
	}
	if !h.ensureSupplierLocalGroupMapped(c, localGroupID) {
		return
	}
	if h.groupPlatformOverride == nil {
		response.ErrorFrom(c, infraerrors.InternalServer("SUPPLIER_GROUP_PLATFORM_OVERRIDE_UNAVAILABLE", "supplier group platform override service unavailable"))
		return
	}
	if err := h.groupPlatformOverride.Set(c.Request.Context(), localGroupID, platform); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"local_group_id": localGroupID, "platform_override": platform})
}

func (h *SupplierProviderSyncHandler) ClearLocalGroupPlatformOverride(c *gin.Context) {
	localGroupID, ok := parseSupplierLocalGroupID(c)
	if !ok {
		return
	}
	if !h.ensureSupplierLocalGroupMapped(c, localGroupID) {
		return
	}
	if h.groupPlatformOverride == nil {
		response.ErrorFrom(c, infraerrors.InternalServer("SUPPLIER_GROUP_PLATFORM_OVERRIDE_UNAVAILABLE", "supplier group platform override service unavailable"))
		return
	}
	if err := h.groupPlatformOverride.Clear(c.Request.Context(), localGroupID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"local_group_id": localGroupID, "platform_override": ""})
}

func (h *SupplierProviderSyncHandler) ensureSupplierLocalGroupMapped(c *gin.Context, localGroupID int64) bool {
	mappings, err := h.dataRepo.ListMappingsByLocalGroup(c.Request.Context(), []int64{localGroupID})
	if err != nil {
		response.ErrorFrom(c, err)
		return false
	}
	if len(mappings) == 0 {
		response.ErrorFrom(c, badRequest("本地分组不属于供应商管理模块的匹配分组"))
		return false
	}
	return true
}

func parseSupplierLocalGroupID(c *gin.Context) (int64, bool) {
	rawID := strings.TrimSpace(c.Param("local_group_id"))
	if rawID == "" {
		rawID = strings.TrimSpace(c.Param("id"))
	}
	localGroupID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || localGroupID <= 0 {
		response.ErrorFrom(c, badRequest("本地分组 ID 无效"))
		return 0, false
	}
	return localGroupID, true
}

func (h *SupplierProviderSyncHandler) ListGroups(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > supplierProviderMaxPageSize {
		pageSize = supplierProviderMaxPageSize
	}
	result, err := h.dataRepo.ListGroups(c.Request.Context(), service.SupplierProviderDataListParams{
		ProviderID:  parseOptionalInt64(c.Query("provider_id")),
		Active:      parseSupplierProviderEnabled(c.Query("active")),
		KeyStatus:   strings.TrimSpace(c.Query("key_status")),
		Search:      strings.TrimSpace(c.Query("search")),
		Platform:    strings.TrimSpace(c.Query("platform")),
		MatchStatus: strings.TrimSpace(c.Query("match_status")),
		RateStatus:  strings.TrimSpace(c.Query("rate_status")),
		SortBy:      strings.TrimSpace(c.Query("sort_by")),
		SortOrder:   strings.TrimSpace(c.Query("sort_order")),
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierProviderSyncHandler) ListGroupHealthTrends(c *gin.Context) {
	groupIDs, ok := parseSupplierProviderGroupHealthTrendIDs(c.Query("group_ids"))
	if !ok {
		response.ErrorFrom(c, badRequest("分组 ID 参数无效"))
		return
	}
	period := strings.TrimSpace(c.Query("period"))
	if period != "" && period != "90m" {
		response.ErrorFrom(c, badRequest("趋势时间范围无效"))
		return
	}

	result, err := h.dataRepo.ListGroupHealthTrends(c.Request.Context(), service.SupplierProviderGroupHealthTrendParams{
		GroupIDs:    groupIDs,
		Period:      90 * time.Minute,
		BucketCount: 18,
		Now:         time.Now().UTC(),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// ListLocalGroupHealthTrends 返回按本地分组归属的账号健康守护趋势。
func (h *SupplierProviderSyncHandler) ListLocalGroupHealthTrends(ctx context.Context, params service.SupplierProviderGroupHealthTrendParams) ([]service.SupplierProviderGroupHealthTrend, error) {
	return h.dataRepo.ListLocalGroupHealthTrends(ctx, params)
}

// ListMappingsByLocalGroup 返回本地分组关联的供应商分组映射。
func (h *SupplierProviderSyncHandler) ListMappingsByLocalGroup(ctx context.Context, localGroupIDs []int64) ([]service.SupplierProviderGroup, error) {
	return h.dataRepo.ListMappingsByLocalGroup(ctx, localGroupIDs)
}

func parseSupplierProviderGroupHealthTrendIDs(raw string) ([]int64, bool) {
	seen := make(map[int64]struct{})
	parts := strings.Split(raw, ",")
	groupIDs := make([]int64, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || value <= 0 {
			return nil, false
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		groupIDs = append(groupIDs, value)
		if len(groupIDs) > 200 {
			return nil, false
		}
	}
	return groupIDs, len(groupIDs) > 0
}
func (h *SupplierProviderSyncHandler) UpdateGroupMapping(c *gin.Context) {
	groupID, ok := parseSupplierGroupID(c)
	if !ok {
		return
	}

	var payload map[string]json.RawMessage
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.ErrorFrom(c, badRequest("映射参数格式无效"))
		return
	}
	rawLocalGroupID, ok := payload["local_group_id"]
	if !ok {
		response.ErrorFrom(c, badRequest("缺少本地分组 ID"))
		return
	}

	var localGroupID *int64
	if string(rawLocalGroupID) != "null" {
		var value int64
		if err := json.Unmarshal(rawLocalGroupID, &value); err != nil || value <= 0 {
			response.ErrorFrom(c, badRequest("本地分组 ID 无效"))
			return
		}
		localGroupID = &value
	}

	var err error
	if h.groupMatcher != nil {
		err = h.groupMatcher.UpdateMapping(c.Request.Context(), groupID, localGroupID)
	} else {
		err = h.dataRepo.UpdateGroupMapping(c.Request.Context(), groupID, localGroupID)
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"group_id": groupID, "local_group_id": localGroupID})
}

func (h *SupplierProviderSyncHandler) DeleteGroup(c *gin.Context) {
	groupID, ok := parseSupplierGroupID(c)
	if !ok {
		return
	}
	if err := h.dataRepo.DeleteGroup(c.Request.Context(), groupID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"group_id": groupID})
}

func (h *SupplierProviderSyncHandler) DeleteAccount(c *gin.Context) {
	accountID, ok := parseSupplierAccountID(c)
	if !ok {
		return
	}
	if err := h.dataRepo.DeleteAccount(c.Request.Context(), accountID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"account_id": accountID})
}

func (h *SupplierProviderSyncHandler) AutoMatchGroups(c *gin.Context) {
	if h.groupMatcher == nil {
		response.ErrorFrom(c, infraerrors.InternalServer("SUPPLIER_GROUP_MATCHER_UNAVAILABLE", "supplier group matcher unavailable"))
		return
	}
	result, err := h.groupMatcher.AutoMatch(c.Request.Context(), parseOptionalInt64(c.Query("provider_id")))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierProviderSyncHandler) UpdateAutoMatchPolicy(c *gin.Context) {
	groupID, ok := parseSupplierGroupID(c)
	if !ok {
		return
	}
	var input struct {
		Ignored *bool `json:"ignored"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Ignored == nil {
		response.ErrorFrom(c, badRequest("忽略自动匹配参数无效"))
		return
	}
	result, err := h.groupMatcher.SetIgnored(c.Request.Context(), groupID, *input.Ignored)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierProviderSyncHandler) ResolveGroupNameChange(c *gin.Context) {
	groupID, ok := parseSupplierGroupID(c)
	if !ok {
		return
	}
	var input struct {
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ErrorFrom(c, badRequest("名称变化处理参数无效"))
		return
	}
	input.Action = strings.TrimSpace(input.Action)
	if input.Action != service.NameChangeActionKeepLocal && input.Action != service.NameChangeActionSyncLocal {
		response.ErrorFrom(c, badRequest("不支持的名称变化处理方式"))
		return
	}
	if err := h.groupMatcher.ResolveNameChange(c.Request.Context(), groupID, input.Action); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"group_id": groupID, "action": input.Action})
}

func (h *SupplierProviderSyncHandler) UpdateGroupRateGuard(c *gin.Context) {
	groupID, ok := parseSupplierGroupID(c)
	if !ok {
		return
	}
	var input struct {
		Selected *bool `json:"selected"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Selected == nil {
		response.ErrorFrom(c, badRequest("倍率守护参数无效"))
		return
	}
	if h.groupGuard == nil {
		response.ErrorFrom(c, infraerrors.InternalServer("SUPPLIER_GROUP_GUARD_UNAVAILABLE", "supplier group guard unavailable"))
		return
	}
	if err := h.groupGuard.SetManualGuard(c.Request.Context(), groupID, *input.Selected); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"group_id": groupID, "selected": *input.Selected})
}

func parseSupplierGroupID(c *gin.Context) (int64, bool) {
	groupID, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || groupID <= 0 {
		response.ErrorFrom(c, badRequest("供应商分组 ID 无效"))
		return 0, false
	}
	return groupID, true
}

func parseSupplierAccountID(c *gin.Context) (int64, bool) {
	accountID, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || accountID <= 0 {
		response.ErrorFrom(c, badRequest("供应商账号 ID 无效"))
		return 0, false
	}
	return accountID, true
}

func parseSupplierMonitorTargetID(c *gin.Context) (int64, bool) {
	monitorTargetID, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || monitorTargetID <= 0 {
		response.ErrorFrom(c, badRequest("监控项 ID 无效"))
		return 0, false
	}
	return monitorTargetID, true
}

func parseOptionalInt64(raw string) int64 {
	value, _ := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if value < 0 {
		return 0
	}
	return value
}

// supplierSyncCostDay 解析成本同步的归属日期；空值返回今天。
func supplierSyncCostDay(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Now(), true
	}
	day, err := time.ParseInLocation("2006-01-02", raw, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return day, true
}

func badRequest(message string) error {
	return infraerrors.BadRequest("VALIDATION_ERROR", message)
}

func supplierProviderTestScopeAllowed(scope string) bool {
	switch scope {
	case service.SupplierSyncScopeAccounts, service.SupplierSyncScopeGroups, service.SupplierSyncScopeBalance, service.SupplierSyncScopeCost, service.SupplierSyncScopeMonitor:
		return true
	default:
		return false
	}
}

func (h *SupplierProviderSyncHandler) UpdateGroupRateGuardEnabled(c *gin.Context) {
	groupID, ok := parseSupplierGroupID(c)
	if !ok {
		return
	}
	var input struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Enabled == nil {
		response.ErrorFrom(c, badRequest("倍率守护参与参数无效"))
		return
	}
	if h.groupGuard == nil {
		response.ErrorFrom(c, infraerrors.InternalServer("SUPPLIER_GROUP_GUARD_UNAVAILABLE", "supplier group guard unavailable"))
		return
	}
	if err := h.groupGuard.SetRateGuardEnabled(c.Request.Context(), groupID, *input.Enabled); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"group_id": groupID, "enabled": *input.Enabled})
}

func (h *SupplierProviderSyncHandler) RefreshToken(c *gin.Context) {
	id, ok := parseSupplierProviderID(c)
	if !ok {
		return
	}
	token, err := h.syncService.RefreshToken(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	message := "Token 刷新成功"
	if token.AccessToken == "" && token.CookieHeader != "" {
		message = "登录会话已更新"
	}
	response.Success(c, gin.H{
		"provider_id": id,
		"expires_at":  token.ExpiresAt,
		"message":     message,
	})
}

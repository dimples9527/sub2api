package admin

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SupplierProviderSyncServicePort interface {
	SyncAccounts(ctx context.Context, providerID int64, trigger string) (service.SupplierProviderSyncResult, error)
	SyncGroups(ctx context.Context, providerID int64, trigger string) (service.SupplierProviderSyncResult, error)
	SyncBalance(ctx context.Context, providerID int64, trigger string) (service.SupplierProviderSyncResult, error)
	SyncCost(ctx context.Context, providerID int64, day time.Time, trigger string) (service.SupplierProviderSyncResult, error)
	SyncAll(ctx context.Context, providerID int64, trigger string) (service.SupplierProviderSyncResult, error)
	TestEndpoint(ctx context.Context, providerID int64, scope string) (service.SupplierProviderEndpointTestResult, error)
}

type SupplierProviderDataRepositoryPort interface {
	ListAccounts(ctx context.Context, params service.SupplierProviderDataListParams) (service.SupplierProviderAccountListResult, error)
	ListGroups(ctx context.Context, params service.SupplierProviderDataListParams) (service.SupplierProviderGroupListResult, error)
	UpdateGroupMapping(ctx context.Context, groupID int64, localGroupID *int64) error
}

type SupplierProviderGroupMatcherPort interface {
	AutoMatch(ctx context.Context, providerID int64) (service.SupplierGroupAutoMatchResult, error)
	UpdateMapping(ctx context.Context, groupID int64, localGroupID *int64) error
	SetIgnored(ctx context.Context, groupID int64, ignored bool) (service.SupplierGroupAutoMatchResult, error)
	ResolveNameChange(ctx context.Context, groupID int64, action string) error
}

type SupplierGroupGuardPort interface {
	SetManualGuard(ctx context.Context, groupID int64, selected bool) error
}

type SupplierProviderSyncHandler struct {
	syncService  SupplierProviderSyncServicePort
	dataRepo     SupplierProviderDataRepositoryPort
	groupMatcher SupplierProviderGroupMatcherPort
	groupGuard   SupplierGroupGuardPort
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
	h.sync(c, func(ctx context.Context, id int64) (service.SupplierProviderSyncResult, error) {
		return h.syncService.SyncCost(ctx, id, time.Now(), service.SupplierSyncTriggerManual)
	})
}

func (h *SupplierProviderSyncHandler) SyncAll(c *gin.Context) {
	h.sync(c, func(ctx context.Context, id int64) (service.SupplierProviderSyncResult, error) {
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
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierProviderSyncHandler) ListAccounts(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > supplierProviderMaxPageSize {
		pageSize = supplierProviderMaxPageSize
	}
	result, err := h.dataRepo.ListAccounts(c.Request.Context(), service.SupplierProviderDataListParams{
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

func (h *SupplierProviderSyncHandler) ListGroups(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > supplierProviderMaxPageSize {
		pageSize = supplierProviderMaxPageSize
	}
	result, err := h.dataRepo.ListGroups(c.Request.Context(), service.SupplierProviderDataListParams{
		ProviderID:  parseOptionalInt64(c.Query("provider_id")),
		Active:      parseSupplierProviderEnabled(c.Query("active")),
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

func parseOptionalInt64(raw string) int64 {
	value, _ := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if value < 0 {
		return 0
	}
	return value
}

func badRequest(message string) error {
	return infraerrors.BadRequest("VALIDATION_ERROR", message)
}

func supplierProviderTestScopeAllowed(scope string) bool {
	switch scope {
	case service.SupplierSyncScopeAccounts, service.SupplierSyncScopeGroups, service.SupplierSyncScopeBalance, service.SupplierSyncScopeCost:
		return true
	default:
		return false
	}
}

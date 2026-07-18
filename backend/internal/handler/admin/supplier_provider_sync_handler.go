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

type SupplierProviderSyncHandler struct {
	syncService SupplierProviderSyncServicePort
	dataRepo    SupplierProviderDataRepositoryPort
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

func (h *SupplierProviderSyncHandler) UpdateGroupMapping(c *gin.Context) {
	groupID, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || groupID <= 0 {
		response.ErrorFrom(c, badRequest("供应商分组 ID 无效"))
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

	if err := h.dataRepo.UpdateGroupMapping(c.Request.Context(), groupID, localGroupID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"group_id": groupID, "local_group_id": localGroupID})
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

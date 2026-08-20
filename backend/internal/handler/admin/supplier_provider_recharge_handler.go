package admin

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const supplierProviderRechargeTimezone = "Asia/Shanghai"

type SupplierProviderRechargeHandler struct {
	service *service.SupplierProviderRechargeService
}

func NewSupplierProviderRechargeHandler(svc *service.SupplierProviderRechargeService) *SupplierProviderRechargeHandler {
	return &SupplierProviderRechargeHandler{service: svc}
}

type supplierProviderRechargeSyncRequest struct {
	ProviderID int64 `json:"provider_id"`
	FullSync   bool  `json:"full_sync"`
}

func (h *SupplierProviderRechargeHandler) List(c *gin.Context) {
	providerID, ok := parseSupplierProviderRechargeOptionalID(c.Query("provider_id"))
	if !ok {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_PROVIDER_ID", "供应商 ID 无效"))
		return
	}
	start, end, ok := parseSupplierProviderRechargeDateRange(c.Query("start_date"), c.Query("end_date"))
	if !ok {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_DATE_RANGE", "日期格式必须为 YYYY-MM-DD，且开始日期不能晚于结束日期"))
		return
	}
	page, pageSize := response.ParsePagination(c)
	result, err := h.service.List(c.Request.Context(), service.SupplierProviderRechargeListParams{
		ProviderID: providerID,
		Start:      start,
		End:        end,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierProviderRechargeHandler) Sync(c *gin.Context) {
	var request supplierProviderRechargeSyncRequest
	if err := c.ShouldBindJSON(&request); err != nil && !errors.Is(err, io.EOF) {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	if rawProviderID := strings.TrimSpace(c.Query("provider_id")); rawProviderID != "" {
		providerID, ok := parseSupplierProviderRechargeOptionalID(rawProviderID)
		if !ok {
			response.ErrorFrom(c, infraerrors.BadRequest("INVALID_PROVIDER_ID", "供应商 ID 无效"))
			return
		}
		request.ProviderID = providerID
	}
	if request.ProviderID > 0 {
		result, err := h.service.Sync(c.Request.Context(), service.SupplierProviderRechargeSyncParams{
			ProviderID: request.ProviderID,
			FullSync:   request.FullSync,
		})
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, result)
		return
	}
	result, err := h.service.SyncAll(c.Request.Context(), request.FullSync)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func parseSupplierProviderRechargeOptionalID(raw string) (int64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	return id, err == nil && id > 0
}

func parseSupplierProviderRechargeDateRange(startValue, endValue string) (time.Time, time.Time, bool) {
	startValue = strings.TrimSpace(startValue)
	endValue = strings.TrimSpace(endValue)
	if startValue == "" && endValue == "" {
		return time.Time{}, time.Time{}, true
	}
	if startValue == "" || endValue == "" {
		return time.Time{}, time.Time{}, false
	}
	location, err := time.LoadLocation(supplierProviderRechargeTimezone)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	start, err := time.ParseInLocation("2006-01-02", startValue, location)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	endDate, err := time.ParseInLocation("2006-01-02", endValue, location)
	if err != nil || endDate.Before(start) {
		return time.Time{}, time.Time{}, false
	}
	return start, endDate.AddDate(0, 0, 1), true
}

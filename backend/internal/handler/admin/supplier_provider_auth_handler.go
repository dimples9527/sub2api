package admin

import (
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SupplierProviderAuthHandler struct {
	service *service.SupplierProviderAuthAuditService
}

func NewSupplierProviderAuthHandler(authService *service.SupplierProviderAuthAuditService) *SupplierProviderAuthHandler {
	return &SupplierProviderAuthHandler{service: authService}
}

func (h *SupplierProviderAuthHandler) GetStatus(c *gin.Context) {
	id, ok := parseSupplierProviderID(c)
	if !ok {
		return
	}
	result, err := h.service.GetStatus(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierProviderAuthHandler) ListHistory(c *gin.Context) {
	id, ok := parseSupplierProviderID(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	if pageSize > 100 {
		pageSize = 100
	}
	eventType := service.SupplierProviderAuthEventType(strings.TrimSpace(c.Query("event_type")))
	if eventType != "" && !isSupplierProviderAuthEventTypeForHandler(eventType) {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_SUPPLIER_PROVIDER_AUTH_EVENT_TYPE", "invalid supplier provider auth event type"))
		return
	}
	result, err := h.service.ListHistory(c.Request.Context(), id, service.SupplierProviderAuthHistoryParams{
		Page: page, PageSize: pageSize, EventType: eventType,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func isSupplierProviderAuthEventTypeForHandler(eventType service.SupplierProviderAuthEventType) bool {
	switch eventType {
	case service.SupplierProviderAuthEventCacheHit,
		service.SupplierProviderAuthEventCacheMiss,
		service.SupplierProviderAuthEventLoginSuccess,
		service.SupplierProviderAuthEventLoginFailed,
		service.SupplierProviderAuthEventRefreshSuccess,
		service.SupplierProviderAuthEventRefreshFailed,
		service.SupplierProviderAuthEventCacheInvalidated,
		service.SupplierProviderAuthEventCacheError:
		return true
	default:
		return false
	}
}

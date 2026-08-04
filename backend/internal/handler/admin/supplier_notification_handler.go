package admin

import (
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SupplierNotificationHandler struct {
	service *service.SupplierNotificationService
}

func NewSupplierNotificationHandler(svc *service.SupplierNotificationService) *SupplierNotificationHandler {
	return &SupplierNotificationHandler{service: svc}
}

func (h *SupplierNotificationHandler) ListChannels(c *gin.Context) {
	items, err := h.service.ListChannels(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *SupplierNotificationHandler) CreateChannel(c *gin.Context) {
	var input service.SupplierNotificationChannelInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	item, err := h.service.SaveChannel(c.Request.Context(), 0, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, item)
}

func (h *SupplierNotificationHandler) UpdateChannel(c *gin.Context) {
	id, ok := parseSupplierNotificationPathID(c, "id")
	if !ok {
		return
	}
	var input service.SupplierNotificationChannelInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	item, err := h.service.SaveChannel(c.Request.Context(), id, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *SupplierNotificationHandler) DeleteChannel(c *gin.Context) {
	id, ok := parseSupplierNotificationPathID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteChannel(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "通知渠道已删除"})
}

func (h *SupplierNotificationHandler) TestChannel(c *gin.Context) {
	id, ok := parseSupplierNotificationPathID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.TestChannel(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierNotificationHandler) ListSubscriptions(c *gin.Context) {
	channelID, ok := parseSupplierNotificationOptionalID(c, "channel_id")
	if !ok {
		return
	}
	items, err := h.service.ListSubscriptions(c.Request.Context(), channelID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *SupplierNotificationHandler) CreateSubscription(c *gin.Context) {
	var input service.SupplierNotificationSubscriptionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	item, err := h.service.SaveSubscription(c.Request.Context(), 0, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, item)
}

func (h *SupplierNotificationHandler) UpdateSubscription(c *gin.Context) {
	id, ok := parseSupplierNotificationPathID(c, "id")
	if !ok {
		return
	}
	var input service.SupplierNotificationSubscriptionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	item, err := h.service.SaveSubscription(c.Request.Context(), id, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *SupplierNotificationHandler) DeleteSubscription(c *gin.Context) {
	id, ok := parseSupplierNotificationPathID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteSubscription(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "通知订阅已删除"})
}

func (h *SupplierNotificationHandler) ListDeliveries(c *gin.Context) {
	channelID, ok := parseSupplierNotificationOptionalID(c, "channel_id")
	if !ok {
		return
	}
	providerID, ok := parseSupplierNotificationOptionalID(c, "provider_id")
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	result, err := h.service.ListDeliveries(c.Request.Context(), service.SupplierNotificationDeliveryListParams{
		ChannelID:  channelID,
		ProviderID: providerID,
		EventType:  strings.TrimSpace(c.Query("event_type")),
		Status:     strings.TrimSpace(c.Query("status")),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, result.Items, result.Total, result.Page, result.PageSize)
}

func (h *SupplierNotificationHandler) GetDelivery(c *gin.Context) {
	id, ok := parseSupplierNotificationPathID(c, "id")
	if !ok {
		return
	}
	item, err := h.service.GetDelivery(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *SupplierNotificationHandler) ListDeliveryAttempts(c *gin.Context) {
	id, ok := parseSupplierNotificationPathID(c, "id")
	if !ok {
		return
	}
	items, err := h.service.ListAttempts(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func parseSupplierNotificationPathID(c *gin.Context, name string) (int64, bool) {
	raw := strings.TrimSpace(c.Param(name))
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_NOTIFICATION_ID", "通知资源 ID 无效"))
		return 0, false
	}
	return id, true
}

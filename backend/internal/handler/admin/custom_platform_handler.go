package admin

import (
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// CustomPlatformHandler 负责自定义平台的管理接口。
type CustomPlatformHandler struct {
	service service.CustomPlatformService
}

func NewCustomPlatformHandler(service service.CustomPlatformService) *CustomPlatformHandler {
	return &CustomPlatformHandler{service: service}
}

func (h *CustomPlatformHandler) List(c *gin.Context) {
	enabledOnly := parseCustomPlatformEnabled(c.Query("enabled_only"))
	items, err := h.service.List(c.Request.Context(), enabledOnly != nil && *enabledOnly)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *CustomPlatformHandler) Get(c *gin.Context) {
	id, ok := parseCustomPlatformID(c)
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *CustomPlatformHandler) Create(c *gin.Context) {
	var req service.CustomPlatformUpsertParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	item, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, item)
}

func (h *CustomPlatformHandler) Update(c *gin.Context) {
	id, ok := parseCustomPlatformID(c)
	if !ok {
		return
	}
	var req service.CustomPlatformUpsertParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	item, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *CustomPlatformHandler) Delete(c *gin.Context) {
	id, ok := parseCustomPlatformID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "删除成功"})
}

func parseCustomPlatformID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_CUSTOM_PLATFORM_ID", "invalid custom platform id"))
		return 0, false
	}
	return id, true
}

func parseCustomPlatformEnabled(raw string) *bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes":
		value := true
		return &value
	case "false", "0", "no":
		value := false
		return &value
	default:
		return nil
	}
}

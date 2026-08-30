package admin

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type modelMonitorGroupHealthService interface {
	Get(context.Context, service.ModelMonitorGroupHealthQuery) ([]service.ModelMonitorGroupHealth, error)
}

// ModelMonitorGroupHealthHandler 返回模型监控分组健康趋势。
type ModelMonitorGroupHealthHandler struct {
	service modelMonitorGroupHealthService
}

// NewModelMonitorGroupHealthHandler 创建模型监控分组健康趋势处理器。
func NewModelMonitorGroupHealthHandler(svc *service.ModelMonitorGroupHealthService) *ModelMonitorGroupHealthHandler {
	return newModelMonitorGroupHealthHandlerWithService(svc)
}

func newModelMonitorGroupHealthHandlerWithService(svc modelMonitorGroupHealthService) *ModelMonitorGroupHealthHandler {
	return &ModelMonitorGroupHealthHandler{service: svc}
}

// Get 查询分组健康趋势。
func (h *ModelMonitorGroupHealthHandler) Get(c *gin.Context) {
	if h == nil || h.service == nil {
		response.InternalError(c, "model monitor group health service is not initialized")
		return
	}

	groupIDs, err := parseModelMonitorGroupHealthGroupIDs(c.Query("group_ids"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.service.Get(c.Request.Context(), service.ModelMonitorGroupHealthQuery{
		Range:    strings.ToLower(strings.TrimSpace(c.Query("range"))),
		GroupIDs: groupIDs,
		Platform: strings.ToLower(strings.TrimSpace(c.Query("platform"))),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func parseModelMonitorGroupHealthGroupIDs(raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	if len(parts) > service.ModelMonitorGroupHealthMaxGroupIDs {
		return nil, fmt.Errorf("too many group ids, maximum is %d", service.ModelMonitorGroupHealthMaxGroupIDs)
	}

	ids := make([]int64, 0, len(parts))
	seen := make(map[int64]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("group_ids contains an empty group id")
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("invalid group id: %s", part)
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
}

package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// LLMMonitorGroup 是模型监控页面使用的分组展示数据。
type LLMMonitorGroup struct {
	ID                    int64   `json:"id"`
	Name                  string  `json:"name"`
	Platform              string  `json:"platform"`
	ActualPlatform        string  `json:"actual_platform,omitempty"`
	EffectivePlatform     string  `json:"effective_platform"`
	EffectivePlatformName string  `json:"effective_platform_name"`
	RateMultiplier        float64 `json:"rate_multiplier"`
	ShowInMonitor         bool    `json:"show_in_monitor"`
}

// GetLLMMonitorGroups 返回模型监控页面需要的启用分组数据。
// GET /api/llm-monitor/groups
func (h *GroupHandler) GetLLMMonitorGroups(c *gin.Context) {
	groups, err := h.adminService.GetAllGroups(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	outGroups, ok := h.buildLLMMonitorGroups(c, groups, false)
	if !ok {
		return
	}
	response.Success(c, outGroups)
}

// ListLLMMonitorPlatformOverrides 返回全部分组及其模型监控专用平台配置。
func (h *GroupHandler) ListLLMMonitorPlatformOverrides(c *gin.Context) {
	groups, err := h.adminService.GetAllGroupsIncludingInactive(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	outGroups, ok := h.buildLLMMonitorGroups(c, groups, true)
	if !ok {
		return
	}
	response.Success(c, outGroups)
}

// SetLLMMonitorGroupVisibility 设置分组是否显示在模型监控页面。
func (h *GroupHandler) SetLLMMonitorGroupVisibility(c *gin.Context) {
	groupID, ok := parsePositiveIDParam(c, "group_id")
	if !ok {
		return
	}
	var req struct {
		ShowInMonitor *bool `json:"show_in_monitor" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ShowInMonitor == nil {
		response.BadRequest(c, "show_in_monitor is required")
		return
	}
	if h.monitorGroupPlatformOverrideService == nil {
		response.ErrorFrom(c, fmt.Errorf("monitor group platform override service is not initialized"))
		return
	}
	if err := h.monitorGroupPlatformOverrideService.SetShowInMonitor(c.Request.Context(), groupID, *req.ShowInMonitor); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "monitor group visibility updated"})
}

// SetLLMMonitorPlatformOverride 设置模型监控页面使用的分组实际平台。
func (h *GroupHandler) SetLLMMonitorPlatformOverride(c *gin.Context) {
	groupID, ok := parsePositiveIDParam(c, "group_id")
	if !ok {
		return
	}
	var req struct {
		ActualPlatform string `json:"actual_platform" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "actual_platform is required")
		return
	}
	if h.monitorGroupPlatformOverrideService == nil {
		response.ErrorFrom(c, fmt.Errorf("monitor group platform override service is not initialized"))
		return
	}
	if err := h.monitorGroupPlatformOverrideService.Set(c.Request.Context(), groupID, req.ActualPlatform); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "platform override updated"})
}

// ClearLLMMonitorPlatformOverride 清除模型监控页面使用的分组实际平台。
func (h *GroupHandler) ClearLLMMonitorPlatformOverride(c *gin.Context) {
	groupID, ok := parsePositiveIDParam(c, "group_id")
	if !ok {
		return
	}
	if h.monitorGroupPlatformOverrideService == nil {
		response.ErrorFrom(c, fmt.Errorf("monitor group platform override service is not initialized"))
		return
	}
	if err := h.monitorGroupPlatformOverrideService.Clear(c.Request.Context(), groupID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "platform override cleared"})
}

func (h *GroupHandler) buildLLMMonitorGroups(c *gin.Context, groups []service.Group, includeHidden bool) ([]LLMMonitorGroup, bool) {
	groupIDs := make([]int64, 0, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.ID)
	}
	overrides := map[int64]service.MonitorGroupPlatformOverride{}
	if h.monitorGroupPlatformOverrideService == nil {
		response.ErrorFrom(c, fmt.Errorf("monitor group platform override service is not initialized"))
		return nil, false
	}
	loaded, err := h.monitorGroupPlatformOverrideService.ListByGroupIDs(c.Request.Context(), groupIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return nil, false
	}
	overrides = loaded
	outGroups := make([]LLMMonitorGroup, 0, len(groups))
	for _, group := range groups {
		settings, hasSettings := overrides[group.ID]
		showInMonitor := true
		if hasSettings {
			showInMonitor = settings.ShowInMonitor
		}
		if !includeHidden && !showInMonitor {
			continue
		}
		effective := group.Platform
		actual := strings.TrimSpace(settings.ActualPlatform)
		if actual != "" {
			effective = actual
		}
		outGroups = append(outGroups, LLMMonitorGroup{
			ID:                    group.ID,
			Name:                  group.Name,
			Platform:              group.Platform,
			ActualPlatform:        actual,
			EffectivePlatform:     effective,
			EffectivePlatformName: h.resolveLLMMonitorPlatformName(c.Request.Context(), effective),
			RateMultiplier:        group.RateMultiplier,
			ShowInMonitor:         showInMonitor,
		})
	}
	return outGroups, true
}

func (h *GroupHandler) resolveLLMMonitorPlatformName(ctx context.Context, platform string) string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform == "" {
		return "API"
	}
	if h != nil && h.customPlatformService != nil && !service.IsCorePlatform(platform) {
		if item, err := h.customPlatformService.ResolveEnabled(ctx, platform); err == nil && item != nil {
			return item.Name
		}
	}
	label := service.PlatformLabel(platform)
	if label != "" {
		return label
	}
	return platform
}

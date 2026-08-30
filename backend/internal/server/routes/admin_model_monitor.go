package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

func registerModelMonitorRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	modelMonitor := admin.Group("/model-monitor")
	{
		modelMonitor.GET("/platform-overrides", h.Admin.Group.ListLLMMonitorPlatformOverrides)
		modelMonitor.PUT("/platform-overrides/:group_id", h.Admin.Group.SetLLMMonitorPlatformOverride)
		modelMonitor.PUT("/visibility/:group_id", h.Admin.Group.SetLLMMonitorGroupVisibility)
		modelMonitor.DELETE("/platform-overrides/:group_id", h.Admin.Group.ClearLLMMonitorPlatformOverride)
		modelMonitor.GET("/group-health", h.Admin.ModelMonitorGroupHealth.Get)
	}
}

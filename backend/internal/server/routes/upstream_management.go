package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

func registerUpstreamManagementRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	upstream := admin.Group("/upstream-management")
	{
		providers := upstream.Group("/providers")
		{
			providers.GET("", h.Admin.UpstreamProvider.List)
			providers.POST("", h.Admin.UpstreamProvider.Create)
			providers.POST("/test", h.Admin.UpstreamProvider.TestConfig)
			providers.PUT("/:slug", h.Admin.UpstreamProvider.Update)
			providers.DELETE("/:slug", h.Admin.UpstreamProvider.Delete)
			providers.POST("/:slug/default", h.Admin.UpstreamProvider.SetDefault)
			providers.POST("/:slug/test", h.Admin.UpstreamProvider.TestSaved)
			providers.GET("/:slug/keys", h.Admin.UpstreamProvider.Keys)
		}
		groups := upstream.Group("/groups")
		{
			groups.GET("", h.Admin.UpstreamManagement.CompareGroups)
			groups.POST("/rate-fixes", h.Admin.UpstreamManagement.ApplyRateFixes)
		}
	}
}

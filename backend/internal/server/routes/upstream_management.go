package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

func registerUpstreamManagementRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	upstream := admin.Group("/upstream-management")
	{
		upstream.GET("/model-square", h.Admin.UpstreamManagement.ModelSquare)
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
			groups.PUT("/mappings", h.Admin.UpstreamManagement.SaveGroupMapping)
			groups.POST("/local-groups", h.Admin.UpstreamManagement.CreateLocalGroupFromUpstream)
			groups.GET("/rate-fix-config", h.Admin.UpstreamManagement.GetRateFixConfig)
			groups.PUT("/rate-fix-config", h.Admin.UpstreamManagement.UpdateRateFixConfig)
			groups.POST("/rate-fixes", h.Admin.UpstreamManagement.ApplyRateFixes)
		}
		accounts := upstream.Group("/accounts")
		{
			accounts.GET("/sync-preview", h.Admin.UpstreamAccountSync.Preview)
			accounts.POST("/sync", h.Admin.UpstreamAccountSync.Sync)
			accounts.GET("/sync-records", h.Admin.UpstreamAccountSync.Records)
			accounts.GET("/rate-guard-config", h.Admin.UpstreamAccountSync.GetRateGuardConfig)
			accounts.PUT("/rate-guard-config", h.Admin.UpstreamAccountSync.UpdateRateGuardConfig)
		}
	}
}

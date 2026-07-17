package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

func registerUpstreamManagementRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	upstream := admin.Group("/upstream-management")
	{
		upstream.GET("/dashboard", h.Admin.UpstreamDashboard.Get)
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
			providers.GET("/:slug/balance", h.Admin.UpstreamProvider.Balance)
		}
		groups := upstream.Group("/groups")
		{
			groups.GET("", h.Admin.UpstreamManagement.CompareGroups)
			groups.PUT("/mappings", h.Admin.UpstreamManagement.SaveGroupMapping)
			groups.POST("/local-groups", h.Admin.UpstreamManagement.CreateLocalGroupFromUpstream)
			groups.GET("/rate-fix-config", h.Admin.UpstreamManagement.GetRateFixConfig)
			groups.PUT("/rate-fix-config", h.Admin.UpstreamManagement.UpdateRateFixConfig)
			groups.POST("/rate-fixes", h.Admin.UpstreamManagement.ApplyRateFixes)
			groups.POST("/rate-fix-records/:key/handled", h.Admin.UpstreamManagement.MarkRateFixRecordHandled)
		}
		accounts := upstream.Group("/accounts")
		{
			accounts.GET("/sync-preview", h.Admin.UpstreamAccountSync.Preview)
			accounts.POST("/sync", h.Admin.UpstreamAccountSync.Sync)
			accounts.GET("/sync-records", h.Admin.UpstreamAccountSync.Records)
			accounts.POST("/sync-records/:key/handled", h.Admin.UpstreamAccountSync.MarkRecordHandled)
			accounts.GET("/rate-guard-config", h.Admin.UpstreamAccountSync.GetRateGuardConfig)
			accounts.PUT("/rate-guard-config", h.Admin.UpstreamAccountSync.UpdateRateGuardConfig)
			accounts.POST("/rate-guard-runs", h.Admin.UpstreamAccountSync.RunRateGuardNow)
			accounts.GET("/rate-guard-poll-logs", h.Admin.UpstreamAccountSync.RateGuardPollLogs)
			accounts.GET("/balance-consumption", h.Admin.UpstreamAccountSync.BalanceConsumptionOverview)
			accounts.GET("/balance-consumption/config", h.Admin.UpstreamAccountSync.GetBalanceSamplerConfig)
			accounts.PUT("/balance-consumption/config", h.Admin.UpstreamAccountSync.UpdateBalanceSamplerConfig)
			accounts.POST("/balance-consumption/recharges", h.Admin.UpstreamAccountSync.AddBalanceRecharge)
			accounts.POST("/balance-consumption/samples", h.Admin.UpstreamAccountSync.RunBalanceSampleNow)
			accounts.GET("/balance-consumption/poll-logs", h.Admin.UpstreamAccountSync.BalanceSamplerPollLogs)
		}
		providers.GET("/balance-consumption", h.Admin.UpstreamAccountSync.BalanceConsumptionOverview)
		providers.GET("/balance-consumption/config", h.Admin.UpstreamAccountSync.GetBalanceSamplerConfig)
		providers.PUT("/balance-consumption/config", h.Admin.UpstreamAccountSync.UpdateBalanceSamplerConfig)
		providers.POST("/balance-consumption/recharges", h.Admin.UpstreamAccountSync.AddBalanceRecharge)
		providers.POST("/balance-consumption/samples", h.Admin.UpstreamAccountSync.RunBalanceSampleNow)
		providers.GET("/balance-consumption/poll-logs", h.Admin.UpstreamAccountSync.BalanceSamplerPollLogs)
		providers.GET("/health-guard/config", h.Admin.UpstreamAccountSync.GetHealthGuardConfig)
		providers.PUT("/health-guard/config", h.Admin.UpstreamAccountSync.UpdateHealthGuardConfig)
		providers.POST("/health-guard/runs", h.Admin.UpstreamAccountSync.RunHealthGuardNow)
		providers.GET("/health-guard/records", h.Admin.UpstreamAccountSync.HealthGuardRecords)
		providers.GET("/health-guard/poll-logs", h.Admin.UpstreamAccountSync.HealthGuardPollLogs)
	}
}

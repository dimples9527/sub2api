package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

func registerSupplierManagementRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	supplier := admin.Group("/supplier-management")
	{
		providerTypes := supplier.Group("/provider-types")
		{
			providerTypes.GET("", h.Admin.SupplierProviderType.List)
			providerTypes.GET("/:id", h.Admin.SupplierProviderType.Get)
			providerTypes.POST("", h.Admin.SupplierProviderType.Create)
			providerTypes.PUT("/:id", h.Admin.SupplierProviderType.Update)
			providerTypes.DELETE("/:id", h.Admin.SupplierProviderType.Delete)
		}

		providers := supplier.Group("/providers")
		{
			providers.GET("", h.Admin.SupplierProvider.List)
			providers.GET("/cost-trends", h.Admin.SupplierProvider.ListCostTrends)
			providers.GET("/:id", h.Admin.SupplierProvider.Get)
			providers.POST("", h.Admin.SupplierProvider.Create)
			providers.PUT("/:id", h.Admin.SupplierProvider.Update)
			providers.DELETE("/:id", h.Admin.SupplierProvider.Delete)
			providers.PUT("/:id/default", h.Admin.SupplierProvider.SetDefault)
			providers.POST("/:id/sync/accounts", h.Admin.SupplierProviderSync.SyncAccounts)
			providers.POST("/:id/sync/groups", h.Admin.SupplierProviderSync.SyncGroups)
			providers.POST("/:id/sync/balance", h.Admin.SupplierProviderSync.SyncBalance)
			providers.POST("/:id/sync/cost", h.Admin.SupplierProviderSync.SyncCost)
			providers.POST("/:id/sync/all", h.Admin.SupplierProviderSync.SyncAll)
			providers.POST("/:id/test/:scope", h.Admin.SupplierProviderSync.TestEndpoint)
		}

		supplier.GET("/captcha-settings", h.Admin.Setting.GetSupplierCaptchaSettings)
		supplier.PUT("/captcha-settings", h.Admin.Setting.UpdateSupplierCaptchaSettings)

		supplier.GET("/accounts", h.Admin.SupplierProviderSync.ListAccounts)
		supplier.GET("/accounts/:local_account_id/health-guard-models", h.Admin.SupplierProviderSync.ListLocalAccountHealthGuardModels)
		supplier.PUT("/accounts/:local_account_id/platform-override", h.Admin.SupplierProviderSync.SetLocalAccountPlatformOverride)
		supplier.DELETE("/accounts/:local_account_id/platform-override", h.Admin.SupplierProviderSync.ClearLocalAccountPlatformOverride)
		supplier.POST("/accounts/batch-test", h.Admin.Account.SupplierBatchTest)
		supplier.GET("/accounts/batch-test/:job_id", h.Admin.Account.GetSupplierBatchTest)
		supplier.POST("/accounts/batch-test/:job_id/cancel", h.Admin.Account.CancelSupplierBatchTest)
		supplier.GET("/groups", h.Admin.SupplierProviderSync.ListGroups)
		supplier.GET("/groups/health-trends", h.Admin.SupplierProviderSync.ListGroupHealthTrends)
		supplier.POST("/groups/auto-match", h.Admin.SupplierProviderSync.AutoMatchGroups)
		supplier.PUT("/groups/:id/mapping", h.Admin.SupplierProviderSync.UpdateGroupMapping)
		supplier.PUT("/groups/:id/auto-match-policy", h.Admin.SupplierProviderSync.UpdateAutoMatchPolicy)
		supplier.PUT("/groups/:id/rate-guard", h.Admin.SupplierProviderSync.UpdateGroupRateGuard)
		supplier.PUT("/groups/:id/rate-guard-ignore", h.Admin.SupplierProviderSync.UpdateGroupRateGuardIgnored)
		supplier.POST("/groups/:id/name-change/resolve", h.Admin.SupplierProviderSync.ResolveGroupNameChange)

		automation := supplier.Group("/automation")
		{
			automation.GET("/tasks", h.Admin.SupplierAutomation.ListTasks)
			automation.PUT("/tasks/:task_code", h.Admin.SupplierAutomation.UpdateTask)
			automation.POST("/tasks/:task_code/run", h.Admin.SupplierAutomation.RunTask)
			automation.GET("/runs", h.Admin.SupplierAutomation.ListRuns)
			automation.GET("/rate-guard-change-logs", h.Admin.SupplierAutomation.ListRateGuardChangeLogs)
			automation.GET("/account-rate-guard-unbind-logs", h.Admin.SupplierAutomation.ListAccountRateGuardUnbindLogs)
			automation.POST("/rate-guard-change-logs/:id/handled", h.Admin.SupplierAutomation.MarkRateGuardChangeLogHandled)
			automation.POST("/account-rate-guard-unbind-logs/:id/handled", h.Admin.SupplierAutomation.MarkAccountRateGuardUnbindLogHandled)
		}
	}
}

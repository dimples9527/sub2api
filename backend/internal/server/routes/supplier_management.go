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
			providers.GET("/balance-summary", h.Admin.SupplierProvider.BalanceSummary)
			providers.POST("/cost-trends/backfill", h.Admin.SupplierProviderSync.BackfillCosts)
			providers.GET("/:id", h.Admin.SupplierProvider.Get)
			providers.GET("/:id/auth-status", h.Admin.SupplierProviderAuth.GetStatus)
			providers.GET("/:id/auth-history", h.Admin.SupplierProviderAuth.ListHistory)
			providers.POST("", h.Admin.SupplierProvider.Create)
			providers.PUT("/:id", h.Admin.SupplierProvider.Update)
			providers.DELETE("/:id", h.Admin.SupplierProvider.Delete)
			providers.PUT("/:id/default", h.Admin.SupplierProvider.SetDefault)
			providers.POST("/:id/sync/accounts", h.Admin.SupplierProviderSync.SyncAccounts)
			providers.POST("/:id/sync/accounts/stream", h.Admin.SupplierProviderSync.SyncAccountsStream)
			providers.POST("/:id/sync/groups", h.Admin.SupplierProviderSync.SyncGroups)
			providers.POST("/:id/sync/groups/stream", h.Admin.SupplierProviderSync.SyncGroupsStream)
			providers.POST("/:id/sync/balance", h.Admin.SupplierProviderSync.SyncBalance)
			providers.POST("/:id/sync/balance/stream", h.Admin.SupplierProviderSync.SyncBalanceStream)
			providers.POST("/:id/sync/cost", h.Admin.SupplierProviderSync.SyncCost)
			providers.POST("/:id/sync/cost/stream", h.Admin.SupplierProviderSync.SyncCostStream)
			providers.POST("/:id/sync/all", h.Admin.SupplierProviderSync.SyncAll)
			providers.POST("/:id/sync/all/stream", h.Admin.SupplierProviderSync.SyncAllStream)
			providers.POST("/:id/refresh-token", h.Admin.SupplierProviderSync.RefreshToken)
			providers.POST("/:id/test/:scope", h.Admin.SupplierProviderSync.TestEndpoint)
		}

		supplier.GET("/captcha-settings", h.Admin.Setting.GetSupplierCaptchaSettings)
		supplier.PUT("/captcha-settings", h.Admin.Setting.UpdateSupplierCaptchaSettings)
		supplier.GET("/cost-deviation-settings", h.Admin.SupplierCostDeviationSettings.GetSettings)
		supplier.PUT("/cost-deviation-settings", h.Admin.SupplierCostDeviationSettings.UpdateSettings)

		costSource := supplier.Group("/cost-source")
		{
			costSource.GET("/settings", h.Admin.SupplierCostSourceConfig.GetSettings)
			costSource.PUT("/settings", h.Admin.SupplierCostSourceConfig.UpdateSettings)
			costSource.GET("/overrides", h.Admin.SupplierCostSourceConfig.ListOverrides)
			costSource.POST("/overrides", h.Admin.SupplierCostSourceConfig.CreateOverride)
			costSource.PUT("/overrides/:id", h.Admin.SupplierCostSourceConfig.UpdateOverride)
			costSource.DELETE("/overrides/:id", h.Admin.SupplierCostSourceConfig.DeleteOverride)
		}
		supplier.GET("/recharges", h.Admin.SupplierProviderRecharge.List)
		supplier.POST("/recharges/sync", h.Admin.SupplierProviderRecharge.Sync)
		supplier.GET("/cost-reviews", h.Admin.SupplierProviderCostReview.List)
		supplier.GET("/cost-reviews/:id/history", h.Admin.SupplierProviderCostReview.History)
		supplier.POST("/cost-reviews/bulk-approve", h.Admin.SupplierProviderCostReview.BulkApprove)
		supplier.POST("/cost-reviews/:id/approve", h.Admin.SupplierProviderCostReview.Approve)

		costAlert := supplier.Group("/cost-alert")
		{
			costAlert.GET("/settings", h.Admin.SupplierCostAlert.GetSettings)
			costAlert.PUT("/settings", h.Admin.SupplierCostAlert.UpdateSettings)
			costAlert.GET("/overrides", h.Admin.SupplierCostAlert.ListOverrides)
			costAlert.POST("/overrides", h.Admin.SupplierCostAlert.CreateOverride)
			costAlert.PUT("/overrides/:id", h.Admin.SupplierCostAlert.UpdateOverride)
			costAlert.DELETE("/overrides/:id", h.Admin.SupplierCostAlert.DeleteOverride)
			costAlert.GET("/events", h.Admin.SupplierCostAlert.ListEvents)
		}

		balanceAlert := supplier.Group("/balance-alert")
		{
			balanceAlert.GET("/configs", h.Admin.SupplierBalanceAlert.ListConfigs)
			balanceAlert.PUT("/configs/:provider_id", h.Admin.SupplierBalanceAlert.UpdateConfig)
			balanceAlert.POST("/scan", h.Admin.SupplierBalanceAlert.Scan)
			balanceAlert.GET("/events", h.Admin.SupplierBalanceAlert.ListEvents)
			balanceAlert.DELETE("/events/:id", h.Admin.SupplierBalanceAlert.DeleteEvent)
		}

		notificationChannels := supplier.Group("/notification-channels")
		{
			notificationChannels.GET("", h.Admin.SupplierNotification.ListChannels)
			notificationChannels.POST("", h.Admin.SupplierNotification.CreateChannel)
			notificationChannels.PUT("/:id", h.Admin.SupplierNotification.UpdateChannel)
			notificationChannels.DELETE("/:id", h.Admin.SupplierNotification.DeleteChannel)
			notificationChannels.POST("/:id/test", h.Admin.SupplierNotification.TestChannel)
		}

		notificationSubscriptions := supplier.Group("/notification-subscriptions")
		{
			notificationSubscriptions.GET("", h.Admin.SupplierNotification.ListSubscriptions)
			notificationSubscriptions.POST("", h.Admin.SupplierNotification.CreateSubscription)
			notificationSubscriptions.PUT("/:id", h.Admin.SupplierNotification.UpdateSubscription)
			notificationSubscriptions.DELETE("/:id", h.Admin.SupplierNotification.DeleteSubscription)
		}

		notificationDeliveries := supplier.Group("/notification-deliveries")
		{
			notificationDeliveries.GET("", h.Admin.SupplierNotification.ListDeliveries)
			notificationDeliveries.GET("/:id", h.Admin.SupplierNotification.GetDelivery)
			notificationDeliveries.GET("/:id/attempts", h.Admin.SupplierNotification.ListDeliveryAttempts)
		}

		supplier.GET("/accounts", h.Admin.SupplierProviderSync.ListAccounts)
		accountHealth := supplier.Group("/account-health")
		{
			accountHealth.GET("/accounts", h.Admin.SupplierAccountHealth.ListAccounts)
			accountHealth.GET("/summary", h.Admin.SupplierAccountHealth.GetSummary)
			accountHealth.GET("/trend", h.Admin.SupplierAccountHealth.GetTrend)
			accountHealth.GET("/trends", h.Admin.SupplierAccountHealth.GetTrends)
		}
		supplier.DELETE("/accounts/:id", h.Admin.SupplierProviderSync.DeleteAccount)
		supplier.GET("/monitor-targets", h.Admin.SupplierProviderSync.ListMonitorTargets)
		supplier.GET("/monitor-targets/local-accounts", h.Admin.SupplierProviderSync.ListBindableLocalAccounts)
		supplier.PUT("/monitor-targets/:id/binding", h.Admin.SupplierProviderSync.BindMonitorTarget)
		supplier.POST("/monitor-targets/auto-match", h.Admin.SupplierProviderSync.AutoMatchMonitorTargets)
		supplier.DELETE("/monitor-targets/:id/binding", h.Admin.SupplierProviderSync.UnbindMonitorTarget)
		supplier.GET("/accounts/:local_account_id/health-guard-models", h.Admin.SupplierProviderSync.ListLocalAccountHealthGuardModels)
		supplier.PUT("/accounts/:local_account_id/platform-override", h.Admin.SupplierProviderSync.SetLocalAccountPlatformOverride)
		supplier.DELETE("/accounts/:id/platform-override", h.Admin.SupplierProviderSync.ClearLocalAccountPlatformOverride)
		supplier.POST("/accounts/batch-test", h.Admin.Account.SupplierBatchTest)
		supplier.GET("/accounts/batch-test/:job_id", h.Admin.Account.GetSupplierBatchTest)
		supplier.POST("/accounts/batch-test/:job_id/cancel", h.Admin.Account.CancelSupplierBatchTest)
		supplier.GET("/groups", h.Admin.SupplierProviderSync.ListGroups)
		supplier.GET("/groups/health-trends", h.Admin.SupplierProviderSync.ListGroupHealthTrends)
		supplier.PUT("/local-groups/:id/platform-override", h.Admin.SupplierProviderSync.SetLocalGroupPlatformOverride)
		supplier.DELETE("/local-groups/:id/platform-override", h.Admin.SupplierProviderSync.ClearLocalGroupPlatformOverride)
		supplier.POST("/groups/auto-match", h.Admin.SupplierProviderSync.AutoMatchGroups)
		supplier.DELETE("/groups/:id", h.Admin.SupplierProviderSync.DeleteGroup)
		supplier.PUT("/groups/:id/mapping", h.Admin.SupplierProviderSync.UpdateGroupMapping)
		supplier.PUT("/groups/:id/auto-match-policy", h.Admin.SupplierProviderSync.UpdateAutoMatchPolicy)
		supplier.PUT("/groups/:id/rate-guard", h.Admin.SupplierProviderSync.UpdateGroupRateGuard)
		supplier.PUT("/groups/:id/rate-guard-enabled", h.Admin.SupplierProviderSync.UpdateGroupRateGuardEnabled)
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

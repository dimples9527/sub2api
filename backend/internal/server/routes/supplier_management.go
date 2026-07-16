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

		supplier.GET("/accounts", h.Admin.SupplierProviderSync.ListAccounts)
		supplier.GET("/groups", h.Admin.SupplierProviderSync.ListGroups)

		automation := supplier.Group("/automation")
		{
			automation.GET("/tasks", h.Admin.SupplierAutomation.ListTasks)
			automation.PUT("/tasks/:task_code", h.Admin.SupplierAutomation.UpdateTask)
			automation.POST("/tasks/:task_code/run", h.Admin.SupplierAutomation.RunTask)
			automation.GET("/runs", h.Admin.SupplierAutomation.ListRuns)
		}
	}
}

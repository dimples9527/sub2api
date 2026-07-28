package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

// registerSupplierDashboardRoutes 注册供应商运维驾驶舱只读路由。
// 路径挂在 /upstream-management/dashboard 下，但不修改 upstream_management.go。
func registerSupplierDashboardRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	dashboard := admin.Group("/upstream-management/dashboard")
	{
		dashboard.GET("/accounts", h.Admin.SupplierDashboard.GetAccounts)
		dashboard.GET("/rates", h.Admin.SupplierDashboard.GetRates)
		dashboard.GET("/providers", h.Admin.SupplierDashboard.GetProviders)
	}
}

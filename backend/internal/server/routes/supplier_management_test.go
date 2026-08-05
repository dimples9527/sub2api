package routes

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterSupplierManagementRoutesAllowsSupplierAccountCleanupRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	require.NotPanics(t, func() {
		router := gin.New()
		registerSupplierManagementRoutes(router.Group("/api/v1/admin"), &handler.Handlers{
			Admin: &handler.AdminHandlers{},
		})
	})
}

func TestRegisterSupplierManagementRoutesIncludesBalanceAlertEventDeletionRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	registerSupplierManagementRoutes(router.Group("/api/v1/admin"), &handler.Handlers{
		Admin: &handler.AdminHandlers{},
	})

	for _, route := range router.Routes() {
		if route.Method == "DELETE" && route.Path == "/api/v1/admin/supplier-management/balance-alert/events/:id" {
			return
		}
	}
	t.Fatalf("未注册供应商余额预警事件删除路由")
}

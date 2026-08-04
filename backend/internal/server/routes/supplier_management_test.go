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

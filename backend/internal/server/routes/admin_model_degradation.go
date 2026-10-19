package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

// registerModelDegradationAdminRoutes 降智检测（模型审计）的管理端路由。
//
// 模块自有的接入点：handler 复用用户侧 ModelDegradationHandler 上的 Admin* 方法，
// 不接入全局 Setting 体系，也不需要新的 Wire provider —— 这里只做最小路由挂载。
// 「可见性」配置只约束普通用户读取检测历史，管理员不受影响。
func registerModelDegradationAdminRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	visibility := admin.Group("/model-degradation/history-visibility")
	{
		visibility.GET("", h.ModelDegradation.AdminGetHistoryVisibility)
		visibility.PUT("/groups/:group_id", h.ModelDegradation.AdminSetHistoryGroupVisibility)
		visibility.PUT("/hosts", h.ModelDegradation.AdminSetHistoryHostVisibility)
	}
}

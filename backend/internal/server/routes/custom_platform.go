package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
)

func registerCustomPlatformRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	platforms := admin.Group("/custom-platforms")
	{
		platforms.GET("", h.Admin.CustomPlatform.List)
		platforms.GET("/:id", h.Admin.CustomPlatform.Get)
		platforms.POST("", h.Admin.CustomPlatform.Create)
		platforms.PUT("/:id", h.Admin.CustomPlatform.Update)
		platforms.DELETE("/:id", h.Admin.CustomPlatform.Delete)
	}
}

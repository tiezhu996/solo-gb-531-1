package router

import (
	"hazop-safeguard-coverage/backend/internal/constants"
	"hazop-safeguard-coverage/backend/internal/handler"
	"hazop-safeguard-coverage/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterIndependenceConflictRoutes(
	api *gin.RouterGroup,
	h *handler.IndependenceConflictHandler,
) {
	group := api.Group("/coverage-evaluations/:id/independence-conflicts")
	group.GET("", middleware.RequirePermission(constants.PermissionRead), h.List)
	group.POST("/:conflictId/resolve", middleware.RequirePermission(constants.PermissionConfirm), h.Resolve)
}

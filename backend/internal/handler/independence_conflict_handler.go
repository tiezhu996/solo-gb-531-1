package handler

import (
	"github.com/gin-gonic/gin"
	"hazop-safeguard-coverage/backend/internal/dto"
	"hazop-safeguard-coverage/backend/internal/service"
	"hazop-safeguard-coverage/backend/internal/util"
	"net/http"
)

type IndependenceConflictHandler struct {
	service service.IndependenceConflictService
}

func NewIndependenceConflictHandler(value service.IndependenceConflictService) *IndependenceConflictHandler {
	return &IndependenceConflictHandler{service: value}
}

func (h *IndependenceConflictHandler) List(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	result, err := h.service.List(c.Request.Context(), id)
	respond(c, http.StatusOK, result, err)
}

func (h *IndependenceConflictHandler) Resolve(c *gin.Context) {
	id, err := util.ParseUintParam(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	conflictID, err := util.ParseUintParam(c, "conflictId")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.ResolveIndependenceConflictRequest
	if !bindJSON(c, &request) {
		return
	}
	result, err := h.service.Resolve(c.Request.Context(), id, conflictID, request, mustActor(c))
	respond(c, http.StatusOK, result, err)
}

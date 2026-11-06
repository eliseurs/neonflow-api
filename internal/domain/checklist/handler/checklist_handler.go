package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/checklist"
	"neonflow-api/internal/infra/api"
)

type ChecklistHandler struct {
	svc checklist.Service
	log checklist.Logger
}

func NewChecklistHandler(svc checklist.Service, log checklist.Logger) *ChecklistHandler {
	return &ChecklistHandler{svc: svc, log: log}
}

// List godoc: GET /api/checklist-items
func (h *ChecklistHandler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context())
	if err != nil {
		h.fail(c, "list checklist items failed", err)
		return
	}
	api.Respond(c, http.StatusOK, items)
}

// Create godoc: POST /api/checklist-items
func (h *ChecklistHandler) Create(c *gin.Context) {
	var req checklist.ItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	it, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		h.fail(c, "create checklist item failed", err)
		return
	}
	api.Respond(c, http.StatusCreated, it)
}

// Update godoc: PUT /api/checklist-items/:id
func (h *ChecklistHandler) Update(c *gin.Context) {
	var req checklist.ItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	it, err := h.svc.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		h.fail(c, "update checklist item failed", err)
		return
	}
	api.Respond(c, http.StatusOK, it)
}

func (h *ChecklistHandler) fail(c *gin.Context, msg string, err error) {
	switch {
	case errors.Is(err, checklist.ErrItemNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, checklist.ErrKeyTaken), errors.Is(err, checklist.ErrKeyReserved):
		api.RespondError(c, http.StatusConflict, err.Error())
	case errors.Is(err, checklist.ErrLabelRequired), errors.Is(err, checklist.ErrInvalidLabel):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	default:
		h.log.Error(msg, "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}

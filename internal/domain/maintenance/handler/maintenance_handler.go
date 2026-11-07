package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/maintenance"
	"neonflow-api/internal/infra/api"
)

type MaintenanceHandler struct {
	svc maintenance.Service
	log maintenance.Logger
}

func NewMaintenanceHandler(svc maintenance.Service, log maintenance.Logger) *MaintenanceHandler {
	return &MaintenanceHandler{svc: svc, log: log}
}

// List godoc: GET /api/vehicles/:id/maintenance
func (h *MaintenanceHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), c.Param("id"))
	h.respond(c, http.StatusOK, list, err, "list maintenance failed")
}

// Create godoc: POST /api/vehicles/:id/maintenance
func (h *MaintenanceHandler) Create(c *gin.Context) {
	var req maintenance.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	rec, err := h.svc.Create(c.Request.Context(), c.Param("id"), c.GetString("userID"), req)
	h.respond(c, http.StatusCreated, rec, err, "create maintenance failed")
}

// Update godoc: PUT /api/maintenance/:id
func (h *MaintenanceHandler) Update(c *gin.Context) {
	var req maintenance.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	rec, err := h.svc.Update(c.Request.Context(), c.Param("id"), req)
	h.respond(c, http.StatusOK, rec, err, "update maintenance failed")
}

// Delete godoc: DELETE /api/maintenance/:id
func (h *MaintenanceHandler) Delete(c *gin.Context) {
	err := h.svc.Delete(c.Request.Context(), c.Param("id"))
	h.respond(c, http.StatusOK, gin.H{"deleted": true}, err, "delete maintenance failed")
}

func (h *MaintenanceHandler) respond(c *gin.Context, status int, data any, err error, msg string) {
	switch {
	case err == nil:
		api.Respond(c, status, data)
	case errors.Is(err, maintenance.ErrNotFound), errors.Is(err, maintenance.ErrVehicleNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, maintenance.ErrInvalidDate), errors.Is(err, maintenance.ErrFutureDate), errors.Is(err, maintenance.ErrInvalidKind),
		errors.Is(err, maintenance.ErrDescriptionRequired), errors.Is(err, maintenance.ErrDescriptionTooLong),
		errors.Is(err, maintenance.ErrInvalidCost), errors.Is(err, maintenance.ErrInvalidOdometer):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	default:
		h.log.Error(msg, "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}

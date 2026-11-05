package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/fleet"
	"neonflow-api/internal/infra/api"
)

type FleetHandler struct {
	svc fleet.Service
	log fleet.Logger
}

func NewFleetHandler(svc fleet.Service, log fleet.Logger) *FleetHandler {
	return &FleetHandler{svc: svc, log: log}
}

// List godoc: GET /api/vehicles
func (h *FleetHandler) List(c *gin.Context) {
	vehicles, err := h.svc.List(c.Request.Context())
	if err != nil {
		h.fail(c, "list vehicles failed", err)
		return
	}
	api.Respond(c, http.StatusOK, vehicles)
}

// Get godoc: GET /api/vehicles/:id
func (h *FleetHandler) Get(c *gin.Context) {
	v, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.fail(c, "get vehicle failed", err)
		return
	}
	api.Respond(c, http.StatusOK, v)
}

// Create godoc: POST /api/vehicles
func (h *FleetHandler) Create(c *gin.Context) {
	var req fleet.VehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	v, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		h.fail(c, "create vehicle failed", err)
		return
	}
	api.Respond(c, http.StatusCreated, v)
}

// Update godoc: PUT /api/vehicles/:id
func (h *FleetHandler) Update(c *gin.Context) {
	var req fleet.VehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	v, err := h.svc.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		h.fail(c, "update vehicle failed", err)
		return
	}
	api.Respond(c, http.StatusOK, v)
}

func (h *FleetHandler) fail(c *gin.Context, msg string, err error) {
	switch {
	case errors.Is(err, fleet.ErrVehicleNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, fleet.ErrPlateTaken):
		api.RespondError(c, http.StatusConflict, err.Error())
	case isValidation(err):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	default:
		h.log.Error(msg, "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}

func isValidation(err error) bool {
	for _, target := range []error{
		fleet.ErrInvalidPlate, fleet.ErrNameRequired, fleet.ErrInvalidCategory, fleet.ErrOwnerRequired,
		fleet.ErrInvalidSeats, fleet.ErrInvalidGear, fleet.ErrInvalidDailyRate,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/booking"
	"neonflow-api/internal/infra/api"
)

type BookingHandler struct {
	svc booking.Service
	log booking.Logger
}

func NewBookingHandler(svc booking.Service, log booking.Logger) *BookingHandler {
	return &BookingHandler{svc: svc, log: log}
}

// List godoc: GET /api/bookings?vehicle_id=
func (h *BookingHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), c.Query("vehicle_id"))
	if err != nil {
		h.fail(c, "list bookings failed", err)
		return
	}
	api.Respond(c, http.StatusOK, list)
}

// Get godoc: GET /api/bookings/:id
func (h *BookingHandler) Get(c *gin.Context) {
	b, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.fail(c, "get booking failed", err)
		return
	}
	api.Respond(c, http.StatusOK, b)
}

// Events godoc: GET /api/bookings/:id/events
func (h *BookingHandler) Events(c *gin.Context) {
	events, err := h.svc.Events(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.fail(c, "booking events failed", err)
		return
	}
	api.Respond(c, http.StatusOK, events)
}

// Create godoc: POST /api/bookings
func (h *BookingHandler) Create(c *gin.Context) {
	var req booking.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	b, err := h.svc.Create(c.Request.Context(), req, c.GetString("userID"))
	if err != nil {
		h.fail(c, "create booking failed", err)
		return
	}
	api.Respond(c, http.StatusCreated, b)
}

// SetChecks godoc: PATCH /api/bookings/:id/checks  {"cnh": true, ...}
func (h *BookingHandler) SetChecks(c *gin.Context) {
	var checks map[string]bool
	if err := c.ShouldBindJSON(&checks); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	b, err := h.svc.SetChecks(c.Request.Context(), c.Param("id"), checks)
	if err != nil {
		h.fail(c, "set checks failed", err)
		return
	}
	api.Respond(c, http.StatusOK, b)
}

// Advance godoc: POST /api/bookings/:id/advance  {"at": "AAAA-MM-DDTHH:MM"} (opcional: hora real da entrega ou da devolução)
func (h *BookingHandler) Advance(c *gin.Context) {
	var req booking.AdvanceRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
			return
		}
	}
	b, err := h.svc.Advance(c.Request.Context(), c.Param("id"), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "advance booking failed", err)
		return
	}
	api.Respond(c, http.StatusOK, b)
}

// Cancel godoc: POST /api/bookings/:id/cancel
func (h *BookingHandler) Cancel(c *gin.Context) {
	b, err := h.svc.Cancel(c.Request.Context(), c.Param("id"), c.GetString("userID"))
	if err != nil {
		h.fail(c, "cancel booking failed", err)
		return
	}
	api.Respond(c, http.StatusOK, b)
}

// Available godoc: GET /api/availability?start_at=AAAA-MM-DDTHH:MM&end_at=AAAA-MM-DDTHH:MM
// (start_date e end_date, só com a data, continuam aceitos e viram 09:00).
func (h *BookingHandler) Available(c *gin.Context) {
	start := c.DefaultQuery("start_at", c.Query("start_date"))
	end := c.DefaultQuery("end_at", c.Query("end_date"))
	vehicles, err := h.svc.Available(c.Request.Context(), start, end)
	if err != nil {
		h.fail(c, "availability failed", err)
		return
	}
	api.Respond(c, http.StatusOK, vehicles)
}

func (h *BookingHandler) fail(c *gin.Context, msg string, err error) {
	status := statusFor(err)
	if status == http.StatusInternalServerError {
		h.log.Error(msg, "error", err)
		api.RespondError(c, status, "erro interno")
		return
	}
	api.RespondError(c, status, err.Error())
}

func statusFor(err error) int {
	switch {
	case errors.Is(err, booking.ErrBookingNotFound), errors.Is(err, booking.ErrVehicleNotFound):
		return http.StatusNotFound
	case errors.Is(err, booking.ErrVehicleRequired), errors.Is(err, booking.ErrClientNameRequired),
		errors.Is(err, booking.ErrInvalidDate), errors.Is(err, booking.ErrInvalidPeriod),
		errors.Is(err, booking.ErrInvalidCheck), errors.Is(err, booking.ErrFutureTime),
		errors.Is(err, booking.ErrReturnBeforeDelivery):
		return http.StatusBadRequest
	case errors.Is(err, booking.ErrVehicleInMaintenance), errors.Is(err, booking.ErrPeriodUnavailable),
		errors.Is(err, booking.ErrChecksPending), errors.Is(err, booking.ErrStageFinal),
		errors.Is(err, booking.ErrNotCancelable), errors.Is(err, booking.ErrStageChanged),
		errors.Is(err, booking.ErrAwaitingChecklist), errors.Is(err, booking.ErrVehicleInUse):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/booking"
	"neonflow-api/internal/domain/siterequest"
	"neonflow-api/internal/domain/transfer"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/infra/email"
)

type SiteRequestHandler struct {
	svc siterequest.Service
	log siterequest.Logger
}

func NewSiteRequestHandler(svc siterequest.Service, log siterequest.Logger) *SiteRequestHandler {
	return &SiteRequestHandler{svc: svc, log: log}
}

// Quote godoc: GET /api/public/fleet/:id/quote?start_at=&end_at= (sem login)
func (h *SiteRequestHandler) Quote(c *gin.Context) {
	q, err := h.svc.Quote(c.Request.Context(), c.Param("id"), c.Query("start_at"), c.Query("end_at"))
	h.reply(c, http.StatusOK, q, err, "public quote")
}

// Guest godoc: POST /api/public/requests (sem login; o pedido segue para a central depois do clique no e-mail)
func (h *SiteRequestHandler) Guest(c *gin.Context) {
	var req siterequest.GuestRequest
	if !bind(c, &req) {
		return
	}
	err := h.svc.Guest(c.Request.Context(), req)
	h.reply(c, http.StatusAccepted, gin.H{"message": "enviamos um link para o seu e-mail; o pedido vai para a central quando você confirmar"}, err, "guest site request")
}

// Cars godoc: GET /api/client/cars?start_at=&end_at=
func (h *SiteRequestHandler) Cars(c *gin.Context) {
	cars, err := h.svc.Cars(c.Request.Context(), c.Query("start_at"), c.Query("end_at"))
	h.reply(c, http.StatusOK, cars, err, "client cars")
}

// Create godoc: POST /api/client/requests
func (h *SiteRequestHandler) Create(c *gin.Context) {
	var req siterequest.CreateRequest
	if !bind(c, &req) {
		return
	}
	r, err := h.svc.Create(c.Request.Context(), c.GetString("userID"), req)
	h.reply(c, http.StatusCreated, r, err, "create site request")
}

// Mine godoc: GET /api/client/requests
func (h *SiteRequestHandler) Mine(c *gin.Context) {
	list, err := h.svc.Mine(c.Request.Context(), c.GetString("userID"))
	h.reply(c, http.StatusOK, list, err, "my site requests")
}

// MineDetail godoc: GET /api/client/requests/:id
func (h *SiteRequestHandler) MineDetail(c *gin.Context) {
	d, err := h.svc.MineDetail(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	h.reply(c, http.StatusOK, d, err, "my site request")
}

// Resubmit godoc: PUT /api/client/requests/:id
func (h *SiteRequestHandler) Resubmit(c *gin.Context) {
	var req siterequest.CreateRequest
	if !bind(c, &req) {
		return
	}
	r, err := h.svc.Resubmit(c.Request.Context(), c.GetString("userID"), c.Param("id"), req)
	h.reply(c, http.StatusOK, r, err, "resubmit site request")
}

// CancelMine godoc: POST /api/client/requests/:id/cancel
func (h *SiteRequestHandler) CancelMine(c *gin.Context) {
	r, err := h.svc.CancelMine(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	h.reply(c, http.StatusOK, r, err, "cancel site request")
}

// List godoc: GET /api/requests?status=
func (h *SiteRequestHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), c.Query("status"))
	h.reply(c, http.StatusOK, list, err, "list site requests")
}

// Get godoc: GET /api/requests/:id
func (h *SiteRequestHandler) Get(c *gin.Context) {
	d, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	h.reply(c, http.StatusOK, d, err, "get site request")
}

// Confirm godoc: POST /api/requests/:id/confirm
func (h *SiteRequestHandler) Confirm(c *gin.Context) {
	var req siterequest.ConfirmRequest
	if c.Request.ContentLength > 0 && !bind(c, &req) {
		return
	}
	r, err := h.svc.Confirm(c.Request.Context(), c.GetString("userID"), c.Param("id"), req)
	h.reply(c, http.StatusOK, r, err, "confirm site request")
}

// Adjust godoc: POST /api/requests/:id/adjust
func (h *SiteRequestHandler) Adjust(c *gin.Context) {
	var req siterequest.NoteRequest
	if !bind(c, &req) {
		return
	}
	r, err := h.svc.Adjust(c.Request.Context(), c.GetString("userID"), c.Param("id"), req)
	h.reply(c, http.StatusOK, r, err, "adjust site request")
}

// Refuse godoc: POST /api/requests/:id/refuse
func (h *SiteRequestHandler) Refuse(c *gin.Context) {
	var req siterequest.NoteRequest
	if !bind(c, &req) {
		return
	}
	r, err := h.svc.Refuse(c.Request.Context(), c.GetString("userID"), c.Param("id"), req)
	h.reply(c, http.StatusOK, r, err, "refuse site request")
}

func bind(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return false
	}
	return true
}

var (
	badRequest = []error{
		siterequest.ErrInvalidKind, siterequest.ErrInvalidWhatsApp, siterequest.ErrPastDate, siterequest.ErrRouteRequired,
		siterequest.ErrInvalidPassengers, siterequest.ErrKindChanged, siterequest.ErrNoteRequired, siterequest.ErrPriceRequired,
		siterequest.ErrNameRequired, siterequest.ErrInvalidEmail,
		booking.ErrVehicleRequired, booking.ErrInvalidKmPlan, booking.ErrInvalidDate, booking.ErrInvalidPeriod,
		booking.ErrBelowMinDays, booking.ErrKmPlanUnavailable, transfer.ErrInvalidPrice,
	}
	notFound = []error{siterequest.ErrNotFound, booking.ErrVehicleNotFound}
	conflict = []error{
		siterequest.ErrStatusChanged, siterequest.ErrPriceChanged, booking.ErrPeriodUnavailable, booking.ErrVehicleInMaintenance,
		siterequest.ErrEmailInUse,
	}
)

func (h *SiteRequestHandler) reply(c *gin.Context, status int, data any, err error, op string) {
	switch {
	case err == nil:
		api.Respond(c, status, data)
	case errors.Is(err, siterequest.ErrNoAccount):
		api.RespondError(c, http.StatusForbidden, err.Error())
	case errors.Is(err, email.ErrNotConfigured):
		api.RespondError(c, http.StatusServiceUnavailable, err.Error())
	case errors.Is(err, siterequest.ErrSendFailed):
		api.RespondError(c, http.StatusBadGateway, err.Error())
	case anyIs(err, badRequest):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	case anyIs(err, notFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case anyIs(err, conflict):
		api.RespondError(c, http.StatusConflict, err.Error())
	default:
		h.log.Error(op+" failed", "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}

func anyIs(err error, targets []error) bool {
	for _, t := range targets {
		if errors.Is(err, t) {
			return true
		}
	}
	return false
}

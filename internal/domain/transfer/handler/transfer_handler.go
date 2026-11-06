package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/transfer"
	"neonflow-api/internal/infra/api"
)

type TransferHandler struct {
	svc transfer.Service
	log transfer.Logger
}

func NewTransferHandler(svc transfer.Service, log transfer.Logger) *TransferHandler {
	return &TransferHandler{svc: svc, log: log}
}

// List godoc: GET /api/transfers
func (h *TransferHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context())
	h.respond(c, http.StatusOK, list, err, "list transfers failed")
}

// Get godoc: GET /api/transfers/:id
func (h *TransferHandler) Get(c *gin.Context) {
	t, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	h.respond(c, http.StatusOK, t, err, "get transfer failed")
}

// Events godoc: GET /api/transfers/:id/events
func (h *TransferHandler) Events(c *gin.Context) {
	events, err := h.svc.Events(c.Request.Context(), c.Param("id"))
	h.respond(c, http.StatusOK, events, err, "transfer events failed")
}

// Create godoc: POST /api/transfers
func (h *TransferHandler) Create(c *gin.Context) {
	var req transfer.CreateRequest
	if !bind(c, &req) {
		return
	}
	t, err := h.svc.Create(c.Request.Context(), req, c.GetString("userID"))
	h.respond(c, http.StatusCreated, t, err, "create transfer failed")
}

// Drivers godoc: GET /api/transfers/:id/drivers
func (h *TransferHandler) Drivers(c *gin.Context) {
	list, err := h.svc.Drivers(c.Request.Context(), c.Param("id"))
	h.respond(c, http.StatusOK, list, err, "transfer drivers failed")
}

// Notify godoc: POST /api/transfers/:id/notify {driver_partner_id}
func (h *TransferHandler) Notify(c *gin.Context) {
	var req transfer.NotifyRequest
	if !bind(c, &req) {
		return
	}
	t, err := h.svc.Notify(c.Request.Context(), c.Param("id"), c.GetString("userID"), req)
	h.respond(c, http.StatusOK, t, err, "notify transfer failed")
}

// Offer godoc: POST /api/transfers/:id/offer {offer_cents}
func (h *TransferHandler) Offer(c *gin.Context) {
	var req transfer.OfferRequest
	if !bind(c, &req) {
		return
	}
	t, err := h.svc.Offer(c.Request.Context(), c.Param("id"), c.GetString("userID"), req)
	h.respond(c, http.StatusOK, t, err, "offer transfer failed")
}

// Accept godoc: POST /api/transfers/:id/accept {driver_partner_id, offer_cents?, vehicle_id?}
func (h *TransferHandler) Accept(c *gin.Context) {
	var req transfer.AcceptRequest
	if !bind(c, &req) {
		return
	}
	t, err := h.svc.Accept(c.Request.Context(), c.Param("id"), c.GetString("userID"), req)
	h.respond(c, http.StatusOK, t, err, "accept transfer failed")
}

// Release godoc: POST /api/transfers/:id/release
func (h *TransferHandler) Release(c *gin.Context) {
	t, err := h.svc.Release(c.Request.Context(), c.Param("id"), c.GetString("userID"))
	h.respond(c, http.StatusOK, t, err, "release transfer failed")
}

// SetPaid godoc: PATCH /api/transfers/:id/paid {paid}
func (h *TransferHandler) SetPaid(c *gin.Context) {
	var req struct {
		Paid bool `json:"paid"`
	}
	if !bind(c, &req) {
		return
	}
	t, err := h.svc.SetPaid(c.Request.Context(), c.Param("id"), c.GetString("userID"), req.Paid)
	h.respond(c, http.StatusOK, t, err, "set transfer paid failed")
}

// Start godoc: POST /api/transfers/:id/start
func (h *TransferHandler) Start(c *gin.Context) {
	t, err := h.svc.Start(c.Request.Context(), c.Param("id"), c.GetString("userID"))
	h.respond(c, http.StatusOK, t, err, "start transfer failed")
}

// Complete godoc: POST /api/transfers/:id/complete {extra_wait_cents}
func (h *TransferHandler) Complete(c *gin.Context) {
	var req transfer.CompleteRequest
	if !bind(c, &req) {
		return
	}
	t, err := h.svc.Complete(c.Request.Context(), c.Param("id"), c.GetString("userID"), req)
	h.respond(c, http.StatusOK, t, err, "complete transfer failed")
}

// Cancel godoc: POST /api/transfers/:id/cancel {flight_cancelled}
func (h *TransferHandler) Cancel(c *gin.Context) {
	var req transfer.CancelRequest
	if !bind(c, &req) {
		return
	}
	t, err := h.svc.Cancel(c.Request.Context(), c.Param("id"), c.GetString("userID"), req)
	h.respond(c, http.StatusOK, t, err, "cancel transfer failed")
}

// bind aceita corpo vazio (ações sem parâmetros obrigatórios) e recusa JSON inválido.
func bind(c *gin.Context, dst any) bool {
	if c.Request.ContentLength == 0 {
		return true
	}
	if err := c.ShouldBindJSON(dst); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return false
	}
	return true
}

func (h *TransferHandler) respond(c *gin.Context, status int, data any, err error, msg string) {
	switch {
	case err == nil:
		api.Respond(c, status, data)
	case errors.Is(err, transfer.ErrNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, transfer.ErrClientRequired), errors.Is(err, transfer.ErrInvalidPickup),
		errors.Is(err, transfer.ErrRouteRequired), errors.Is(err, transfer.ErrInvalidPassengers),
		errors.Is(err, transfer.ErrInvalidPrice), errors.Is(err, transfer.ErrReferrerNotFound),
		errors.Is(err, transfer.ErrCustomerNotFound),
		errors.Is(err, transfer.ErrInvalidOffer), errors.Is(err, transfer.ErrOfferDecreased),
		errors.Is(err, transfer.ErrVehicleNotFound), errors.Is(err, transfer.ErrDriverNotFound),
		errors.Is(err, transfer.ErrInvalidExtraWait):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, transfer.ErrWrongStage), errors.Is(err, transfer.ErrNoOffer), errors.Is(err, transfer.ErrNotPaid):
		api.RespondError(c, http.StatusConflict, err.Error())
	default:
		h.log.Error(msg, "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}

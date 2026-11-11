package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/terms"
	"neonflow-api/internal/infra/api"
)

type TermsHandler struct {
	svc terms.Service
	log terms.Logger
}

func NewTermsHandler(svc terms.Service, log terms.Logger) *TermsHandler {
	return &TermsHandler{svc: svc, log: log}
}

// Versions godoc: GET /api/terms
func (h *TermsHandler) Versions(c *gin.Context) {
	list, err := h.svc.Versions(c.Request.Context())
	h.respond(c, http.StatusOK, list, err, "list terms failed")
}

// Publish godoc: POST /api/terms {body} (só admin)
func (h *TermsHandler) Publish(c *gin.Context) {
	var req struct {
		Body string `json:"body"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	v, err := h.svc.Publish(c.Request.Context(), req.Body, c.GetString("userID"))
	h.respond(c, http.StatusCreated, v, err, "publish terms failed")
}

// CreateLink godoc: POST /api/bookings/:id/terms-link
func (h *TermsHandler) CreateLink(c *gin.Context) {
	l, err := h.svc.CreateLink(c.Request.Context(), c.Param("id"), c.GetString("userID"))
	h.respond(c, http.StatusCreated, l, err, "create terms link failed")
}

// BookingAcceptance godoc: GET /api/bookings/:id/terms (nulo = ainda não aceitou)
func (h *TermsHandler) BookingAcceptance(c *gin.Context) {
	a, err := h.svc.BookingAcceptance(c.Request.Context(), c.Param("id"))
	h.respond(c, http.StatusOK, a, err, "booking terms failed")
}

// Public godoc: GET /api/public/terms/:token (sem login)
func (h *TermsHandler) Public(c *gin.Context) {
	p, err := h.svc.Public(c.Request.Context(), c.Param("token"))
	h.respond(c, http.StatusOK, p, err, "public terms failed")
}

// Accept godoc: POST /api/public/terms/:token/accept {typed_name, version} (sem login)
func (h *TermsHandler) Accept(c *gin.Context) {
	var req terms.AcceptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	a, err := h.svc.Accept(c.Request.Context(), c.Param("token"), req)
	h.respond(c, http.StatusCreated, a, err, "accept terms failed")
}

// respond nunca registra o código do link nem o nome digitado no log.
func (h *TermsHandler) respond(c *gin.Context, status int, data any, err error, msg string) {
	switch {
	case err == nil:
		api.Respond(c, status, data)
	case errors.Is(err, terms.ErrBookingNotFound), errors.Is(err, terms.ErrLinkNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, terms.ErrBodyRequired), errors.Is(err, terms.ErrBodyTooLong), errors.Is(err, terms.ErrInvalidName),
		errors.Is(err, terms.ErrConfirmRequired):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, terms.ErrNoTerms), errors.Is(err, terms.ErrWrongStage), errors.Is(err, terms.ErrVersionChanged):
		api.RespondError(c, http.StatusConflict, err.Error())
	default:
		h.log.Error(msg, "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}

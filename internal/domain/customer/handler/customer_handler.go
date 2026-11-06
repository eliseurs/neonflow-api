package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/customer"
	"neonflow-api/internal/infra/api"
)

type CustomerHandler struct {
	svc customer.Service
	log customer.Logger
}

func NewCustomerHandler(svc customer.Service, log customer.Logger) *CustomerHandler {
	return &CustomerHandler{svc: svc, log: log}
}

// List godoc: GET /api/customers?q=
func (h *CustomerHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), c.Query("q"))
	h.respond(c, http.StatusOK, list, err, "list customers failed")
}

// Get godoc: GET /api/customers/:id
func (h *CustomerHandler) Get(c *gin.Context) {
	d, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	h.respond(c, http.StatusOK, d, err, "get customer failed")
}

// Create godoc: POST /api/customers
func (h *CustomerHandler) Create(c *gin.Context) {
	var req customer.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	out, err := h.svc.Create(c.Request.Context(), req)
	h.respond(c, http.StatusCreated, out, err, "create customer failed")
}

// Update godoc: PUT /api/customers/:id
func (h *CustomerHandler) Update(c *gin.Context) {
	var req customer.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	out, err := h.svc.Update(c.Request.Context(), c.Param("id"), req)
	h.respond(c, http.StatusOK, out, err, "update customer failed")
}

// Anonymize godoc: POST /api/customers/:id/anonymize (só admin)
func (h *CustomerHandler) Anonymize(c *gin.Context) {
	out, err := h.svc.Anonymize(c.Request.Context(), c.Param("id"))
	h.respond(c, http.StatusOK, out, err, "anonymize customer failed")
}

// respond nunca registra dados do cliente no log: só a mensagem e o erro técnico.
func (h *CustomerHandler) respond(c *gin.Context, status int, data any, err error, msg string) {
	switch {
	case err == nil:
		api.Respond(c, status, data)
	case errors.Is(err, customer.ErrNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, customer.ErrNameRequired), errors.Is(err, customer.ErrInvalidWhatsApp),
		errors.Is(err, customer.ErrInvalidEmail), errors.Is(err, customer.ErrInvalidLanguage):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, customer.ErrWhatsAppTaken), errors.Is(err, customer.ErrAnonymized):
		api.RespondError(c, http.StatusConflict, err.Error())
	default:
		h.log.Error(msg, "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}

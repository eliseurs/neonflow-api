package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/partner"
	"neonflow-api/internal/infra/api"
)

type PartnerHandler struct {
	svc partner.Service
	log partner.Logger
}

func NewPartnerHandler(svc partner.Service, log partner.Logger) *PartnerHandler {
	return &PartnerHandler{svc: svc, log: log}
}

// List godoc: GET /api/partners?kind=proprietario|motorista|indicador
func (h *PartnerHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), c.Query("kind"))
	if err != nil {
		h.fail(c, "list partners failed", err)
		return
	}
	api.Respond(c, http.StatusOK, list)
}

// Get godoc: GET /api/partners/:id
func (h *PartnerHandler) Get(c *gin.Context) {
	p, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.fail(c, "get partner failed", err)
		return
	}
	api.Respond(c, http.StatusOK, p)
}

// ByCode godoc: GET /api/partners/code/:code
func (h *PartnerHandler) ByCode(c *gin.Context) {
	p, err := h.svc.ByCode(c.Request.Context(), c.Param("code"))
	if err != nil {
		h.fail(c, "partner by code failed", err)
		return
	}
	api.Respond(c, http.StatusOK, p)
}

// Create godoc: POST /api/partners
func (h *PartnerHandler) Create(c *gin.Context) {
	var req partner.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	p, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		h.fail(c, "create partner failed", err)
		return
	}
	api.Respond(c, http.StatusCreated, p)
}

// Update godoc: PUT /api/partners/:id
func (h *PartnerHandler) Update(c *gin.Context) {
	var req partner.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	p, err := h.svc.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		h.fail(c, "update partner failed", err)
		return
	}
	api.Respond(c, http.StatusOK, p)
}

func (h *PartnerHandler) fail(c *gin.Context, msg string, err error) {
	switch {
	case errors.Is(err, partner.ErrPartnerNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, partner.ErrNameRequired), errors.Is(err, partner.ErrInvalidKind),
		errors.Is(err, partner.ErrInvalidEmail), errors.Is(err, partner.ErrInvalidPercent):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, partner.ErrCodeExhausted):
		api.RespondError(c, http.StatusConflict, err.Error())
	default:
		h.log.Error(msg, "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}

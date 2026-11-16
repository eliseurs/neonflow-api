package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/partnerarea"
	"neonflow-api/internal/domain/transfer"
	"neonflow-api/internal/infra/api"
)

type PartnerAreaHandler struct {
	svc partnerarea.Service
	log partnerarea.Logger
}

func NewPartnerAreaHandler(svc partnerarea.Service, log partnerarea.Logger) *PartnerAreaHandler {
	return &PartnerAreaHandler{svc: svc, log: log}
}

// Area godoc: GET /api/partner/area — o parceiro vem da conta logada, nunca da requisição.
func (h *PartnerAreaHandler) Area(c *gin.Context) {
	a, err := h.svc.Area(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "partner area", err)
		return
	}
	api.Respond(c, http.StatusOK, a)
}

// Accept godoc: POST /api/partner/rides/:id/accept
func (h *PartnerAreaHandler) Accept(c *gin.Context) {
	var req partnerarea.AcceptRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
			return
		}
	}
	if err := h.svc.Accept(c.Request.Context(), c.GetString("userID"), c.Param("id"), req); err != nil {
		h.fail(c, "accept ride", err)
		return
	}
	api.Respond(c, http.StatusOK, gin.H{"accepted": true})
}

// Release godoc: POST /api/partner/rides/:id/release
func (h *PartnerAreaHandler) Release(c *gin.Context) {
	if err := h.svc.Release(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		h.fail(c, "release ride", err)
		return
	}
	api.Respond(c, http.StatusOK, gin.H{"released": true})
}

func (h *PartnerAreaHandler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, partnerarea.ErrNoPartner), errors.Is(err, partnerarea.ErrNotDriver):
		api.RespondError(c, http.StatusForbidden, err.Error())
	case errors.Is(err, partnerarea.ErrNotYourVehicle):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, partnerarea.ErrNotYourRide), errors.Is(err, transfer.ErrNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, transfer.ErrWrongStage), errors.Is(err, transfer.ErrNoOffer):
		// Outro motorista aceitou antes, ou a central mudou ou tirou o valor da corrida.
		api.RespondError(c, http.StatusConflict, "esta corrida não está mais disponível")
	case errors.Is(err, transfer.ErrDriverNotFound):
		api.RespondError(c, http.StatusForbidden, err.Error())
	default:
		h.log.Error(op+" failed", "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}

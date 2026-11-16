package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/partnerarea"
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
	if errors.Is(err, partnerarea.ErrNoPartner) {
		api.RespondError(c, http.StatusForbidden, err.Error())
		return
	}
	if err != nil {
		h.log.Error("partner area failed", "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
		return
	}
	api.Respond(c, http.StatusOK, a)
}

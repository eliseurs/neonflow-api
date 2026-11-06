package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/payout"
	"neonflow-api/internal/infra/api"
)

type PayoutHandler struct {
	svc payout.Service
	log payout.Logger
}

func NewPayoutHandler(svc payout.Service, log payout.Logger) *PayoutHandler {
	return &PayoutHandler{svc: svc, log: log}
}

// List godoc: GET /api/payouts?kind=comissao|repasse&status=pendente|paga&partner_id=
func (h *PayoutHandler) List(c *gin.Context) {
	f := payout.Filter{Kind: c.Query("kind"), Status: c.Query("status"), PartnerID: c.Query("partner_id")}
	list, err := h.svc.List(c.Request.Context(), f)
	if err != nil {
		h.fail(c, "list payouts failed", err)
		return
	}
	api.Respond(c, http.StatusOK, list)
}

// Pay godoc: POST /api/payouts/:id/pay
func (h *PayoutHandler) Pay(c *gin.Context) {
	out, err := h.svc.MarkPaid(c.Request.Context(), c.Param("id"), c.GetString("userID"))
	if err != nil {
		h.fail(c, "pay payout failed", err)
		return
	}
	api.Respond(c, http.StatusOK, out)
}

func (h *PayoutHandler) fail(c *gin.Context, msg string, err error) {
	switch {
	case errors.Is(err, payout.ErrNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, payout.ErrInvalidFilter):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, payout.ErrAlreadyPaid), errors.Is(err, payout.ErrPercentUndefined):
		api.RespondError(c, http.StatusConflict, err.Error())
	default:
		h.log.Error(msg, "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}

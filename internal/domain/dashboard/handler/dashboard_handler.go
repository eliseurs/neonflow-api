package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/dashboard"
	"neonflow-api/internal/infra/api"
)

type DashboardHandler struct {
	svc dashboard.Service
	log dashboard.Logger
}

func NewDashboardHandler(svc dashboard.Service, log dashboard.Logger) *DashboardHandler {
	return &DashboardHandler{svc: svc, log: log}
}

// Summary godoc: GET /api/dashboard?month=AAAA-MM
func (h *DashboardHandler) Summary(c *gin.Context) {
	s, err := h.svc.Summary(c.Request.Context(), c.Query("month"))
	switch {
	case err == nil:
		api.Respond(c, http.StatusOK, s)
	case errors.Is(err, dashboard.ErrInvalidMonth):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	default:
		h.log.Error("dashboard summary failed", "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}

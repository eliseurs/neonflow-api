package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/audit"
	"neonflow-api/internal/infra/api"
)

type AuditHandler struct {
	svc audit.Service
	log audit.Logger
}

func NewAuditHandler(svc audit.Service, log audit.Logger) *AuditHandler {
	return &AuditHandler{svc: svc, log: log}
}

// List godoc: GET /api/audit?user_id=&denied=true&limit=
func (h *AuditHandler) List(c *gin.Context) {
	f := audit.Filter{UserID: c.Query("user_id"), Denied: c.Query("denied") == "true"}
	if l := c.Query("limit"); l != "" {
		n, err := strconv.Atoi(l)
		if err != nil {
			api.RespondError(c, http.StatusBadRequest, audit.ErrInvalidLimit.Error())
			return
		}
		f.Limit = n
	}
	list, err := h.svc.List(c.Request.Context(), f)
	if errors.Is(err, audit.ErrInvalidLimit) {
		api.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		h.log.Error("list audit failed", "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
		return
	}
	api.Respond(c, http.StatusOK, list)
}

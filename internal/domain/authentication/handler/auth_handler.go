package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/infra/api"
)

type AuthHandler struct {
	svc authentication.Service
	log authentication.Logger
}

func NewAuthHandler(svc authentication.Service, log authentication.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, log: log}
}

// Login godoc: POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req authentication.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}

	res, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, authentication.ErrInvalidCredentials) {
			api.RespondError(c, http.StatusUnauthorized, authentication.ErrInvalidCredentials.Error())
			return
		}
		h.log.Error("login failed", "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
		return
	}
	api.Respond(c, http.StatusOK, res)
}

// Me godoc: GET /api/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		api.RespondError(c, http.StatusUnauthorized, "não autenticado")
		return
	}

	user, err := h.svc.Me(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, authentication.ErrUserNotFound) {
			api.RespondError(c, http.StatusNotFound, authentication.ErrUserNotFound.Error())
			return
		}
		h.log.Error("me failed", "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
		return
	}
	api.Respond(c, http.StatusOK, user)
}

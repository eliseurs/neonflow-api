package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/account"
	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/infra/email"
)

// sentMessage é a resposta do cadastro e da nova senha, a mesma com ou sem conta para o e-mail.
const sentMessage = "se o e-mail puder ser usado, enviamos um link para ele"

type AccountHandler struct {
	svc account.Service
	log account.Logger
}

func NewAccountHandler(svc account.Service, log account.Logger) *AccountHandler {
	return &AccountHandler{svc: svc, log: log}
}

// Signup godoc: POST /api/public/accounts/signup
func (h *AccountHandler) Signup(c *gin.Context) {
	var req account.SignupRequest
	if !bind(c, &req) {
		return
	}
	if err := h.svc.Signup(c.Request.Context(), req); err != nil {
		h.fail(c, "signup", err)
		return
	}
	api.Respond(c, http.StatusAccepted, gin.H{"message": sentMessage})
}

// Confirm godoc: POST /api/public/accounts/confirm
func (h *AccountHandler) Confirm(c *gin.Context) {
	var req account.TokenRequest
	if !bind(c, &req) {
		return
	}
	if err := h.svc.ConfirmEmail(c.Request.Context(), req); err != nil {
		h.fail(c, "confirm email", err)
		return
	}
	api.Respond(c, http.StatusOK, gin.H{"confirmed": true})
}

// ForgotPassword godoc: POST /api/public/accounts/forgot-password
func (h *AccountHandler) ForgotPassword(c *gin.Context) {
	var req account.EmailRequest
	if !bind(c, &req) {
		return
	}
	if err := h.svc.ForgotPassword(c.Request.Context(), req); err != nil {
		h.fail(c, "forgot password", err)
		return
	}
	api.Respond(c, http.StatusAccepted, gin.H{"message": sentMessage})
}

// ResetPassword godoc: POST /api/public/accounts/reset-password
func (h *AccountHandler) ResetPassword(c *gin.Context) {
	var req account.PasswordRequest
	if !bind(c, &req) {
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), req); err != nil {
		h.fail(c, "reset password", err)
		return
	}
	api.Respond(c, http.StatusOK, gin.H{"password_changed": true})
}

// AcceptInvite godoc: POST /api/public/accounts/accept-invite
func (h *AccountHandler) AcceptInvite(c *gin.Context) {
	var req account.PasswordRequest
	if !bind(c, &req) {
		return
	}
	if err := h.svc.AcceptInvite(c.Request.Context(), req); err != nil {
		h.fail(c, "accept invite", err)
		return
	}
	api.Respond(c, http.StatusCreated, gin.H{"account_created": true})
}

// InvitePartner godoc: POST /api/partners/:id/invite
func (h *AccountHandler) InvitePartner(c *gin.Context) {
	inv, err := h.svc.InvitePartner(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.fail(c, "invite partner", err)
		return
	}
	api.Respond(c, http.StatusCreated, inv)
}

func bind(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return false
	}
	return true
}

func (h *AccountHandler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, authentication.ErrNameRequired), errors.Is(err, authentication.ErrInvalidEmail),
		errors.Is(err, authentication.ErrWeakPassword), errors.Is(err, account.ErrInvalidToken):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, account.ErrPartnerNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, account.ErrPartnerInactive), errors.Is(err, account.ErrPartnerHasAccount),
		errors.Is(err, account.ErrPartnerWithoutEmail), errors.Is(err, authentication.ErrEmailTaken):
		api.RespondError(c, http.StatusConflict, err.Error())
	case errors.Is(err, email.ErrNotConfigured):
		api.RespondError(c, http.StatusServiceUnavailable, err.Error())
	case errors.Is(err, account.ErrSendFailed):
		api.RespondError(c, http.StatusBadGateway, err.Error())
	default:
		h.log.Error(op+" failed", "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}

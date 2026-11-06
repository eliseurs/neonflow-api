package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/user"
	"neonflow-api/internal/infra/api"
)

type UserHandler struct {
	svc user.Service
	log user.Logger
}

func NewUserHandler(svc user.Service, log user.Logger) *UserHandler {
	return &UserHandler{svc: svc, log: log}
}

// List godoc: GET /api/users
func (h *UserHandler) List(c *gin.Context) {
	users, err := h.svc.List(c.Request.Context())
	if err != nil {
		h.fail(c, "list users failed", err)
		return
	}
	api.Respond(c, http.StatusOK, users)
}

// Create godoc: POST /api/users
func (h *UserHandler) Create(c *gin.Context) {
	var req user.CreateRequest
	if !bind(c, &req) {
		return
	}
	u, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		h.fail(c, "create user failed", err)
		return
	}
	api.Respond(c, http.StatusCreated, u)
}

// Update godoc: PUT /api/users/:id
func (h *UserHandler) Update(c *gin.Context) {
	var req user.UpdateRequest
	if !bind(c, &req) {
		return
	}
	u, err := h.svc.Update(c.Request.Context(), c.GetString("userID"), c.Param("id"), req)
	if err != nil {
		h.fail(c, "update user failed", err)
		return
	}
	api.Respond(c, http.StatusOK, u)
}

// ResetPassword godoc: POST /api/users/:id/password
func (h *UserHandler) ResetPassword(c *gin.Context) {
	var req user.PasswordRequest
	if !bind(c, &req) {
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), c.Param("id"), req); err != nil {
		h.fail(c, "reset password failed", err)
		return
	}
	api.Respond(c, http.StatusOK, gin.H{"message": "senha redefinida"})
}

// ChangeOwnPassword godoc: POST /api/auth/password
func (h *UserHandler) ChangeOwnPassword(c *gin.Context) {
	var req user.ChangeOwnPasswordRequest
	if !bind(c, &req) {
		return
	}
	if err := h.svc.ChangeOwnPassword(c.Request.Context(), c.GetString("userID"), req); err != nil {
		h.fail(c, "change password failed", err)
		return
	}
	api.Respond(c, http.StatusOK, gin.H{"message": "senha alterada"})
}

func bind(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return false
	}
	return true
}

func (h *UserHandler) fail(c *gin.Context, msg string, err error) {
	switch {
	case errors.Is(err, authentication.ErrUserNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, authentication.ErrEmailTaken), errors.Is(err, user.ErrSelfLockout):
		api.RespondError(c, http.StatusConflict, err.Error())
	case errors.Is(err, authentication.ErrNameRequired), errors.Is(err, authentication.ErrInvalidEmail),
		errors.Is(err, authentication.ErrWeakPassword), errors.Is(err, authentication.ErrInvalidRole),
		errors.Is(err, user.ErrWrongPassword):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	default:
		h.log.Error(msg, "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}

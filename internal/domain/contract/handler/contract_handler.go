package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/contract"
	"neonflow-api/internal/infra/api"
)

type ContractHandler struct {
	svc contract.Service
	log contract.Logger
}

func NewContractHandler(svc contract.Service, log contract.Logger) *ContractHandler {
	return &ContractHandler{svc: svc, log: log}
}

// Documents godoc: GET /api/contracts
func (h *ContractHandler) Documents(c *gin.Context) {
	list, err := h.svc.Documents(c.Request.Context())
	h.respond(c, http.StatusOK, list, err, "list contracts failed")
}

// Versions godoc: GET /api/contracts/:code/versions
func (h *ContractHandler) Versions(c *gin.Context) {
	list, err := h.svc.Versions(c.Request.Context(), c.Param("code"))
	h.respond(c, http.StatusOK, list, err, "list contract versions failed")
}

// CreateVersion godoc: POST /api/contracts/:code/versions {label, body} (só admin)
func (h *ContractHandler) CreateVersion(c *gin.Context) {
	var req contract.VersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	v, err := h.svc.CreateVersion(c.Request.Context(), c.Param("code"), req, actor(c))
	h.respond(c, http.StatusCreated, v, err, "create contract version failed")
}

// Version godoc: GET /api/contract-versions/:id (texto, hash, eventos e comentários)
func (h *ContractHandler) Version(c *gin.Context) {
	v, err := h.svc.Version(c.Request.Context(), c.Param("id"))
	h.respond(c, http.StatusOK, v, err, "find contract version failed")
}

// AddComment godoc: POST /api/contract-versions/:id/comments {clause, kind, body, proposed_text}
func (h *ContractHandler) AddComment(c *gin.Context) {
	var req contract.CommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	cm, err := h.svc.AddComment(c.Request.Context(), c.Param("id"), req, actor(c))
	h.respond(c, http.StatusCreated, cm, err, "add contract comment failed")
}

// AddEvent godoc: POST /api/contract-versions/:id/events {action, note, sha256, confirm}
func (h *ContractHandler) AddEvent(c *gin.Context) {
	var req contract.EventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}
	e, err := h.svc.AddEvent(c.Request.Context(), c.Param("id"), req, actor(c))
	h.respond(c, http.StatusCreated, e, err, "add contract event failed")
}

// actor vem só do JWT: ninguém registra decisão em nome de outra pessoa.
func actor(c *gin.Context) contract.Actor {
	return contract.Actor{UserID: c.GetString("userID"), Role: c.GetString("role")}
}

// respond não registra texto de contrato nem comentário no log.
func (h *ContractHandler) respond(c *gin.Context, status int, data any, err error, msg string) {
	switch {
	case err == nil:
		api.Respond(c, status, data)
	case errors.Is(err, contract.ErrDocumentNotFound), errors.Is(err, contract.ErrVersionNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, contract.ErrForbiddenAction), errors.Is(err, contract.ErrForbiddenComment):
		api.RespondError(c, http.StatusForbidden, err.Error())
	case errors.Is(err, contract.ErrWrongStatus), errors.Is(err, contract.ErrHashMismatch):
		api.RespondError(c, http.StatusConflict, err.Error())
	case errors.Is(err, contract.ErrBodyRequired), errors.Is(err, contract.ErrBodyTooLong),
		errors.Is(err, contract.ErrLabelTooLong), errors.Is(err, contract.ErrInvalidClause),
		errors.Is(err, contract.ErrInvalidKind), errors.Is(err, contract.ErrInvalidComment),
		errors.Is(err, contract.ErrInvalidAction), errors.Is(err, contract.ErrNoteRequired),
		errors.Is(err, contract.ErrNoteTooLong), errors.Is(err, contract.ErrConfirmRequired):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	default:
		h.log.Error(msg, "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}

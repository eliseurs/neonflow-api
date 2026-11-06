package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/domain/photo"
	"neonflow-api/internal/infra/api"
	"neonflow-api/internal/infra/storage"
)

type PhotoHandler struct {
	svc photo.Service
	log photo.Logger
}

func NewPhotoHandler(svc photo.Service, log photo.Logger) *PhotoHandler {
	return &PhotoHandler{svc: svc, log: log}
}

// ListVehicle godoc: GET /api/vehicles/:id/photos
func (h *PhotoHandler) ListVehicle(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), photo.OwnerVehicle, c.Param("id"), "")
	h.respond(c, http.StatusOK, list, err, "list vehicle photos failed")
}

// UploadVehicle godoc: POST /api/vehicles/:id/photos (multipart, campo "file")
func (h *PhotoHandler) UploadVehicle(c *gin.Context) {
	h.upload(c, photo.OwnerVehicle, "")
}

// ListBooking godoc: GET /api/bookings/:id/photos?stage=entrega|devolucao
func (h *PhotoHandler) ListBooking(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), photo.OwnerBooking, c.Param("id"), c.Query("stage"))
	h.respond(c, http.StatusOK, list, err, "list booking photos failed")
}

// UploadBooking godoc: POST /api/bookings/:id/photos?stage=entrega|devolucao (multipart, campo "file")
func (h *PhotoHandler) UploadBooking(c *gin.Context) {
	h.upload(c, photo.OwnerBooking, c.Query("stage"))
}

// Delete godoc: DELETE /api/photos/:id
func (h *PhotoHandler) Delete(c *gin.Context) {
	err := h.svc.Delete(c.Request.Context(), c.Param("id"))
	h.respond(c, http.StatusOK, gin.H{"deleted": true}, err, "delete photo failed")
}

// Cover godoc: POST /api/photos/:id/cover
func (h *PhotoHandler) Cover(c *gin.Context) {
	p, err := h.svc.SetCover(c.Request.Context(), c.Param("id"))
	h.respond(c, http.StatusOK, p, err, "cover photo failed")
}

func (h *PhotoHandler) upload(c *gin.Context, ownerType, stage string) {
	// Limita o corpo antes de ler: a foto mais a folga do multipart.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, photo.MaxSize+1<<20)
	fh, err := c.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			api.RespondError(c, http.StatusRequestEntityTooLarge, photo.ErrTooLarge.Error())
			return
		}
		api.RespondError(c, http.StatusBadRequest, photo.ErrEmpty.Error())
		return
	}
	f, err := fh.Open()
	if err != nil {
		api.RespondError(c, http.StatusBadRequest, photo.ErrEmpty.Error())
		return
	}
	defer f.Close() //nolint:errcheck
	p, err := h.svc.Upload(c.Request.Context(), ownerType, c.Param("id"), stage, c.GetString("userID"), f, fh.Size)
	h.respond(c, http.StatusCreated, p, err, "upload photo failed")
}

func (h *PhotoHandler) respond(c *gin.Context, status int, data any, err error, msg string) {
	switch {
	case err == nil:
		api.Respond(c, status, data)
	case errors.Is(err, photo.ErrNotFound), errors.Is(err, photo.ErrOwnerNotFound):
		api.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, photo.ErrInvalidType), errors.Is(err, photo.ErrEmpty), errors.Is(err, photo.ErrInvalidStage):
		api.RespondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, photo.ErrTooLarge):
		api.RespondError(c, http.StatusRequestEntityTooLarge, err.Error())
	case errors.Is(err, photo.ErrWrongStage):
		api.RespondError(c, http.StatusConflict, err.Error())
	case errors.Is(err, storage.ErrNotConfigured):
		api.RespondError(c, http.StatusServiceUnavailable, err.Error())
	default:
		h.log.Error(msg, "error", err)
		api.RespondError(c, http.StatusInternalServerError, "erro interno")
	}
}

package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/photo"
	"neonflow-api/internal/infra/storage"
)

type fakeService struct {
	err     error
	owner   string
	stage   string
	userID  string
	content []byte
}

func (f *fakeService) Upload(_ context.Context, owner, _, stage, userID string, body io.Reader, _ int64) (photo.Photo, error) {
	f.owner, f.stage, f.userID = owner, stage, userID
	f.content, _ = io.ReadAll(body)
	return photo.Photo{ID: "p1"}, f.err
}
func (f *fakeService) List(_ context.Context, owner, _, stage string) ([]photo.Photo, error) {
	f.owner, f.stage = owner, stage
	return []photo.Photo{{ID: "p1"}}, f.err
}
func (f *fakeService) Delete(context.Context, string) error { return f.err }
func (f *fakeService) SetCover(context.Context, string) (photo.Photo, error) {
	return photo.Photo{ID: "p1"}, f.err
}

type nopLog struct{}

func (nopLog) Error(string, ...any) {}

func router(svc photo.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewPhotoHandler(svc, nopLog{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u1") })
	r.GET("/vehicles/:id/photos", h.ListVehicle)
	r.POST("/vehicles/:id/photos", h.UploadVehicle)
	r.GET("/bookings/:id/photos", h.ListBooking)
	r.POST("/bookings/:id/photos", h.UploadBooking)
	r.DELETE("/photos/:id", h.Delete)
	r.POST("/photos/:id/cover", h.Cover)
	return r
}

func multipartBody(t *testing.T, field string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile(field, "foto.jpg")
	require.NoError(t, err)
	_, err = fw.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return &buf, w.FormDataContentType()
}

func do(r *gin.Engine, method, path string, body io.Reader, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestPhotoHandler(t *testing.T) {
	svc := &fakeService{}
	r := router(svc)

	body, ct := multipartBody(t, "file", []byte("conteudo"))
	w := do(r, http.MethodPost, "/vehicles/v1/photos", body, ct)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, photo.OwnerVehicle, svc.owner)
	assert.Equal(t, "u1", svc.userID)
	assert.Equal(t, []byte("conteudo"), svc.content)

	body, ct = multipartBody(t, "file", []byte("x"))
	assert.Equal(t, http.StatusCreated, do(r, http.MethodPost, "/bookings/b1/photos?stage=devolucao", body, ct).Code)
	assert.Equal(t, photo.StageReturn, svc.stage)

	body, ct = multipartBody(t, "outro", []byte("x"))
	assert.Equal(t, http.StatusBadRequest, do(r, http.MethodPost, "/vehicles/v1/photos", body, ct).Code, "sem o campo file")
	big, ct := multipartBody(t, "file", bytes.Repeat([]byte("x"), photo.MaxSize+2<<20))
	assert.Equal(t, http.StatusRequestEntityTooLarge, do(r, http.MethodPost, "/vehicles/v1/photos", big, ct).Code)

	assert.Equal(t, http.StatusOK, do(r, http.MethodGet, "/vehicles/v1/photos", nil, "").Code)
	assert.Equal(t, http.StatusOK, do(r, http.MethodGet, "/bookings/b1/photos?stage=entrega", nil, "").Code)
	assert.Equal(t, photo.StageDelivery, svc.stage)
	assert.Equal(t, http.StatusOK, do(r, http.MethodDelete, "/photos/p1", nil, "").Code)
	assert.Equal(t, http.StatusOK, do(r, http.MethodPost, "/photos/p1/cover", nil, "").Code)
}

func TestPhotoHandler_Errors(t *testing.T) {
	cases := []struct {
		err    error
		status int
	}{
		{photo.ErrNotFound, http.StatusNotFound},
		{photo.ErrOwnerNotFound, http.StatusNotFound},
		{photo.ErrInvalidType, http.StatusBadRequest},
		{photo.ErrEmpty, http.StatusBadRequest},
		{photo.ErrInvalidStage, http.StatusBadRequest},
		{photo.ErrTooLarge, http.StatusRequestEntityTooLarge},
		{photo.ErrWrongStage, http.StatusConflict},
		{storage.ErrNotConfigured, http.StatusServiceUnavailable},
		{errors.New("db"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		r := router(&fakeService{err: tc.err})
		body, ct := multipartBody(t, "file", []byte("x"))
		assert.Equal(t, tc.status, do(r, http.MethodPost, "/vehicles/v1/photos", body, ct).Code, tc.err.Error())
		assert.Equal(t, tc.status, do(r, http.MethodGet, "/bookings/b1/photos", nil, "").Code, tc.err.Error())
		assert.Equal(t, tc.status, do(r, http.MethodDelete, "/photos/p1", nil, "").Code, tc.err.Error())
		assert.Equal(t, tc.status, do(r, http.MethodPost, "/photos/p1/cover", nil, "").Code, tc.err.Error())
	}
}

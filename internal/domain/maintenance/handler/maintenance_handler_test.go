package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"neonflow-api/internal/domain/maintenance"
)

type fakeService struct {
	err  error
	user string
}

func (f *fakeService) List(context.Context, string) ([]maintenance.Record, error) {
	return []maintenance.Record{{ID: "m1"}}, f.err
}
func (f *fakeService) Create(_ context.Context, _, user string, _ maintenance.Request) (maintenance.Record, error) {
	f.user = user
	return maintenance.Record{ID: "m1"}, f.err
}
func (f *fakeService) Update(_ context.Context, id string, _ maintenance.Request) (maintenance.Record, error) {
	return maintenance.Record{ID: id}, f.err
}
func (f *fakeService) Delete(context.Context, string) error { return f.err }

type nopLog struct{}

func (nopLog) Error(string, ...any) {}

func do(svc maintenance.Service, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewMaintenanceHandler(svc, nopLog{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u1") })
	r.GET("/vehicles/:id/maintenance", h.List)
	r.POST("/vehicles/:id/maintenance", h.Create)
	r.PUT("/maintenance/:id", h.Update)
	r.DELETE("/maintenance/:id", h.Delete)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

var routes = []struct {
	method, path, body string
	status             int
}{
	{http.MethodGet, "/vehicles/v1/maintenance", "", http.StatusOK},
	{http.MethodPost, "/vehicles/v1/maintenance", `{"kind":"outros"}`, http.StatusCreated},
	{http.MethodPut, "/maintenance/m1", `{"kind":"outros"}`, http.StatusOK},
	{http.MethodDelete, "/maintenance/m1", "", http.StatusOK},
}

func TestMaintenanceHandler(t *testing.T) {
	svc := &fakeService{}
	for _, r := range routes {
		assert.Equal(t, r.status, do(svc, r.method, r.path, r.body).Code, r.path)
	}
	assert.Equal(t, "u1", svc.user)
	assert.Equal(t, http.StatusBadRequest, do(&fakeService{}, http.MethodPost, "/vehicles/v1/maintenance", `{`).Code)
	assert.Equal(t, http.StatusBadRequest, do(&fakeService{}, http.MethodPut, "/maintenance/m1", `{`).Code)

	cases := []struct {
		err    error
		status int
	}{
		{maintenance.ErrNotFound, http.StatusNotFound},
		{maintenance.ErrVehicleNotFound, http.StatusNotFound},
		{maintenance.ErrInvalidDate, http.StatusBadRequest},
		{maintenance.ErrFutureDate, http.StatusBadRequest},
		{maintenance.ErrInvalidKind, http.StatusBadRequest},
		{maintenance.ErrDescriptionRequired, http.StatusBadRequest},
		{maintenance.ErrDescriptionTooLong, http.StatusBadRequest},
		{maintenance.ErrInvalidCost, http.StatusBadRequest},
		{maintenance.ErrInvalidOdometer, http.StatusBadRequest},
		{errors.New("db"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		for _, r := range routes {
			assert.Equal(t, tc.status, do(&fakeService{err: tc.err}, r.method, r.path, r.body).Code, tc.err.Error()+" "+r.path)
		}
	}
}

package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"neonflow-api/internal/domain/dashboard"
)

type fakeService struct {
	month string
	err   error
}

func (f *fakeService) Summary(_ context.Context, month string) (dashboard.Summary, error) {
	f.month = month
	return dashboard.Summary{Month: "2026-11", FleetSize: 2}, f.err
}

type nopLog struct{}

func (nopLog) Error(string, ...any) {}

func do(svc dashboard.Service, path string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/dashboard", NewDashboardHandler(svc, nopLog{}).Summary)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestDashboardHandler(t *testing.T) {
	svc := &fakeService{}
	w := do(svc, "/dashboard?month=2026-11")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "2026-11", svc.month)
	assert.Contains(t, w.Body.String(), `"fleet_size":2`)
	assert.Equal(t, http.StatusBadRequest, do(&fakeService{err: dashboard.ErrInvalidMonth}, "/dashboard").Code)
	assert.Equal(t, http.StatusInternalServerError, do(&fakeService{err: errors.New("db")}, "/dashboard").Code)
}

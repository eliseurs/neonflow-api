package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"neonflow-api/internal/domain/audit"
)

type fakeService struct {
	got audit.Filter
	err error
}

func (f *fakeService) List(_ context.Context, filter audit.Filter) ([]audit.Entry, error) {
	f.got = filter
	return []audit.Entry{{ID: "a1"}}, f.err
}

type nopLog struct{}

func (nopLog) Error(string, ...any) {}

func get(svc audit.Service, path string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/audit", NewAuditHandler(svc, nopLog{}).List)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestList(t *testing.T) {
	svc := &fakeService{}
	w := get(svc, "/audit?user_id=u1&denied=true&limit=20")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"a1"`)
	assert.Equal(t, audit.Filter{UserID: "u1", Denied: true, Limit: 20}, svc.got)

	assert.Equal(t, http.StatusBadRequest, get(&fakeService{}, "/audit?limit=x").Code)
	assert.Equal(t, http.StatusBadRequest, get(&fakeService{err: audit.ErrInvalidLimit}, "/audit").Code)
	assert.Equal(t, http.StatusInternalServerError, get(&fakeService{err: errors.New("db")}, "/audit").Code)
}

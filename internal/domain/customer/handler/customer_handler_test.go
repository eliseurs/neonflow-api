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

	"neonflow-api/internal/domain/customer"
)

type fakeService struct{ err error }

func (f fakeService) List(_ context.Context, q string) ([]customer.Customer, error) {
	return []customer.Customer{{ID: "c1", Name: q}}, f.err
}
func (f fakeService) Get(_ context.Context, id string) (customer.Detail, error) {
	return customer.Detail{Customer: customer.Customer{ID: id}}, f.err
}
func (f fakeService) Create(_ context.Context, r customer.Request) (customer.Customer, error) {
	return customer.Customer{ID: "new", Name: r.Name}, f.err
}
func (f fakeService) Update(_ context.Context, id string, r customer.Request) (customer.Customer, error) {
	return customer.Customer{ID: id, Name: r.Name}, f.err
}
func (f fakeService) Anonymize(_ context.Context, id string) (customer.Customer, error) {
	return customer.Customer{ID: id}, f.err
}

type spyLog struct{ args []any }

func (s *spyLog) Error(_ string, args ...any) { s.args = append(s.args, args...) }

func do(svc customer.Service, log customer.Logger, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewCustomerHandler(svc, log)
	r := gin.New()
	r.GET("/customers", h.List)
	r.GET("/customers/:id", h.Get)
	r.POST("/customers", h.Create)
	r.PUT("/customers/:id", h.Update)
	r.POST("/customers/:id/anonymize", h.Anonymize)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

var routes = []struct{ method, path, body string }{
	{http.MethodGet, "/customers?q=ana", ""},
	{http.MethodGet, "/customers/c1", ""},
	{http.MethodPost, "/customers", `{"name":"Ana"}`},
	{http.MethodPut, "/customers/c1", `{"name":"Ana"}`},
	{http.MethodPost, "/customers/c1/anonymize", ""},
}

func TestCustomerHandler(t *testing.T) {
	for _, r := range routes {
		want := http.StatusOK
		if r.method == http.MethodPost && r.path == "/customers" {
			want = http.StatusCreated
		}
		assert.Equal(t, want, do(fakeService{}, &spyLog{}, r.method, r.path, r.body).Code, r.path)
	}
	assert.Equal(t, http.StatusBadRequest, do(fakeService{}, &spyLog{}, http.MethodPost, "/customers", `{`).Code)
	assert.Equal(t, http.StatusBadRequest, do(fakeService{}, &spyLog{}, http.MethodPut, "/customers/c1", `{`).Code)

	cases := []struct {
		err    error
		status int
	}{
		{customer.ErrNotFound, http.StatusNotFound},
		{customer.ErrNameRequired, http.StatusBadRequest},
		{customer.ErrInvalidWhatsApp, http.StatusBadRequest},
		{customer.ErrInvalidEmail, http.StatusBadRequest},
		{customer.ErrInvalidLanguage, http.StatusBadRequest},
		{customer.ErrWhatsAppTaken, http.StatusConflict},
		{customer.ErrAnonymized, http.StatusConflict},
		{errors.New("db"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		for _, r := range routes {
			assert.Equal(t, tc.status, do(fakeService{err: tc.err}, &spyLog{}, r.method, r.path, r.body).Code, tc.err.Error()+" "+r.path)
		}
	}
}

func TestCustomerHandler_NoPIIInLog(t *testing.T) {
	log := &spyLog{}
	do(fakeService{err: errors.New("db")}, log, http.MethodPost, "/customers", `{"name":"Maria Sigilosa","whatsapp":"5554999990000"}`)
	for _, a := range log.args {
		s, _ := a.(string)
		assert.NotContains(t, s, "Maria Sigilosa")
		assert.NotContains(t, s, "5554999990000")
	}
}

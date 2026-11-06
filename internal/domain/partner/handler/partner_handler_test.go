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

	"neonflow-api/internal/domain/partner"
)

type fakeService struct{ err error }

func (f fakeService) List(_ context.Context, kind string) ([]partner.Partner, error) {
	return []partner.Partner{{ID: "p1", Name: kind}}, f.err
}
func (f fakeService) Get(_ context.Context, id string) (partner.Partner, error) {
	return partner.Partner{ID: id}, f.err
}
func (f fakeService) ByCode(_ context.Context, code string) (partner.Partner, error) {
	return partner.Partner{ReferralCode: code}, f.err
}
func (f fakeService) Create(_ context.Context, r partner.Request) (partner.Partner, error) {
	return partner.Partner{ID: "new", Name: r.Name}, f.err
}
func (f fakeService) Update(_ context.Context, id string, r partner.Request) (partner.Partner, error) {
	return partner.Partner{ID: id, Name: r.Name}, f.err
}

type nopLog struct{}

func (nopLog) Error(string, ...any) {}

func do(svc partner.Service, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewPartnerHandler(svc, nopLog{})
	r := gin.New()
	r.GET("/partners", h.List)
	r.GET("/partners/code/:code", h.ByCode)
	r.GET("/partners/:id", h.Get)
	r.POST("/partners", h.Create)
	r.PUT("/partners/:id", h.Update)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestPartnerHandler_OK(t *testing.T) {
	cases := []struct {
		method, path, body, want string
		status                   int
	}{
		{http.MethodGet, "/partners?kind=motorista", "", `"name":"motorista"`, http.StatusOK},
		{http.MethodGet, "/partners/p9", "", `"id":"p9"`, http.StatusOK},
		{http.MethodGet, "/partners/code/ABC234", "", `"referral_code":"ABC234"`, http.StatusOK},
		{http.MethodPost, "/partners", `{"name":"Hotel"}`, `"name":"Hotel"`, http.StatusCreated},
		{http.MethodPut, "/partners/p1", `{"name":"Novo"}`, `"name":"Novo"`, http.StatusOK},
	}
	for _, tc := range cases {
		w := do(fakeService{}, tc.method, tc.path, tc.body)
		assert.Equal(t, tc.status, w.Code, tc.path)
		assert.Contains(t, w.Body.String(), tc.want)
	}
	assert.Equal(t, http.StatusBadRequest, do(fakeService{}, http.MethodPost, "/partners", `{`).Code)
	assert.Equal(t, http.StatusBadRequest, do(fakeService{}, http.MethodPut, "/partners/p1", `{"kinds":"x"}`).Code)
}

func TestPartnerHandler_Errors(t *testing.T) {
	cases := []struct {
		err    error
		status int
	}{
		{partner.ErrPartnerNotFound, http.StatusNotFound},
		{partner.ErrNameRequired, http.StatusBadRequest},
		{partner.ErrInvalidKind, http.StatusBadRequest},
		{partner.ErrInvalidEmail, http.StatusBadRequest},
		{partner.ErrInvalidPercent, http.StatusBadRequest},
		{partner.ErrCodeExhausted, http.StatusConflict},
		{errors.New("db"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		for _, c := range [][3]string{
			{http.MethodGet, "/partners", ""}, {http.MethodGet, "/partners/p1", ""}, {http.MethodGet, "/partners/code/X", ""},
			{http.MethodPost, "/partners", `{}`}, {http.MethodPut, "/partners/p1", `{}`},
		} {
			assert.Equal(t, tc.status, do(fakeService{err: tc.err}, c[0], c[1], c[2]).Code, tc.err.Error()+" "+c[1])
		}
	}
}

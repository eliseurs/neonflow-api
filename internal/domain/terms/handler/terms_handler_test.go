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

	"neonflow-api/internal/domain/terms"
)

type fakeService struct {
	err  error
	user string
}

func (f *fakeService) Versions(context.Context) ([]terms.Version, error) { return nil, f.err }
func (f *fakeService) Publish(_ context.Context, body, user string) (terms.Version, error) {
	f.user = user
	return terms.Version{Body: body}, f.err
}
func (f *fakeService) CreateLink(context.Context, string, string) (terms.Link, error) {
	return terms.Link{Token: "t"}, f.err
}
func (f *fakeService) BookingAcceptance(context.Context, string) (*terms.Acceptance, error) {
	return nil, f.err
}
func (f *fakeService) Public(context.Context, string) (terms.Public, error) {
	return terms.Public{}, f.err
}
func (f *fakeService) Accept(context.Context, string, terms.AcceptRequest) (terms.Acceptance, error) {
	return terms.Acceptance{}, f.err
}

type spyLog struct{ args []any }

func (s *spyLog) Error(_ string, args ...any) { s.args = append(s.args, args...) }

func do(svc terms.Service, log terms.Logger, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewTermsHandler(svc, log)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u1") })
	r.GET("/terms", h.Versions)
	r.POST("/terms", h.Publish)
	r.POST("/bookings/:id/terms-link", h.CreateLink)
	r.GET("/bookings/:id/terms", h.BookingAcceptance)
	r.GET("/public/terms/:token", h.Public)
	r.POST("/public/terms/:token/accept", h.Accept)
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
	{http.MethodGet, "/terms", "", http.StatusOK},
	{http.MethodPost, "/terms", `{"body":"x"}`, http.StatusCreated},
	{http.MethodPost, "/bookings/b1/terms-link", "", http.StatusCreated},
	{http.MethodGet, "/bookings/b1/terms", "", http.StatusOK},
	{http.MethodGet, "/public/terms/tok", "", http.StatusOK},
	{http.MethodPost, "/public/terms/tok/accept", `{"typed_name":"Ana Teste","version":1}`, http.StatusCreated},
}

func TestTermsHandler(t *testing.T) {
	for _, r := range routes {
		assert.Equal(t, r.status, do(&fakeService{}, &spyLog{}, r.method, r.path, r.body).Code, r.path)
	}
	svc := &fakeService{}
	do(svc, &spyLog{}, http.MethodPost, "/terms", `{"body":"x"}`)
	assert.Equal(t, "u1", svc.user)
	assert.Equal(t, http.StatusBadRequest, do(&fakeService{}, &spyLog{}, http.MethodPost, "/terms", `{`).Code)
	assert.Equal(t, http.StatusBadRequest, do(&fakeService{}, &spyLog{}, http.MethodPost, "/public/terms/tok/accept", `{`).Code)

	cases := []struct {
		err    error
		status int
	}{
		{terms.ErrBookingNotFound, http.StatusNotFound},
		{terms.ErrLinkNotFound, http.StatusNotFound},
		{terms.ErrBodyRequired, http.StatusBadRequest},
		{terms.ErrBodyTooLong, http.StatusBadRequest},
		{terms.ErrInvalidName, http.StatusBadRequest},
		{terms.ErrNoTerms, http.StatusConflict},
		{terms.ErrWrongStage, http.StatusConflict},
		{terms.ErrVersionChanged, http.StatusConflict},
		{errors.New("db"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		for _, r := range routes {
			assert.Equal(t, tc.status, do(&fakeService{err: tc.err}, &spyLog{}, r.method, r.path, r.body).Code, tc.err.Error()+" "+r.path)
		}
	}
}

func TestTermsHandler_NoSecretsInLog(t *testing.T) {
	log := &spyLog{}
	do(&fakeService{err: errors.New("db")}, log, http.MethodPost, "/public/terms/CODIGO-SECRETO/accept", `{"typed_name":"Maria Sigilosa","version":1}`)
	for _, a := range log.args {
		s, _ := a.(string)
		assert.NotContains(t, s, "CODIGO-SECRETO")
		assert.NotContains(t, s, "Maria Sigilosa")
	}
}

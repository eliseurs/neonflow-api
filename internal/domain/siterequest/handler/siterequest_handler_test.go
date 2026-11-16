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

	"neonflow-api/internal/domain/booking"
	"neonflow-api/internal/domain/siterequest"
	"neonflow-api/internal/infra/email"
)

type fakeService struct {
	err  error
	user string
	conf siterequest.ConfirmRequest
}

func (f *fakeService) Quote(context.Context, string, string, string) (siterequest.PublicQuote, error) {
	return siterequest.PublicQuote{Available: true}, f.err
}
func (f *fakeService) Guest(context.Context, siterequest.GuestRequest) error { return f.err }
func (f *fakeService) Cars(context.Context, string, string) ([]siterequest.ClientCar, error) {
	return []siterequest.ClientCar{}, f.err
}
func (f *fakeService) Create(_ context.Context, user string, _ siterequest.CreateRequest) (siterequest.Request, error) {
	f.user = user
	return siterequest.Request{ID: "r1"}, f.err
}
func (f *fakeService) Mine(context.Context, string) ([]siterequest.Request, error) { return nil, f.err }
func (f *fakeService) MineDetail(context.Context, string, string) (siterequest.Detail, error) {
	return siterequest.Detail{}, f.err
}
func (f *fakeService) Resubmit(context.Context, string, string, siterequest.CreateRequest) (siterequest.Request, error) {
	return siterequest.Request{}, f.err
}
func (f *fakeService) CancelMine(context.Context, string, string) (siterequest.Request, error) {
	return siterequest.Request{}, f.err
}
func (f *fakeService) List(context.Context, string) ([]siterequest.Request, error) { return nil, f.err }
func (f *fakeService) Get(context.Context, string) (siterequest.Detail, error) {
	return siterequest.Detail{}, f.err
}
func (f *fakeService) Confirm(_ context.Context, _, _ string, req siterequest.ConfirmRequest) (siterequest.Request, error) {
	f.conf = req
	return siterequest.Request{}, f.err
}
func (f *fakeService) Adjust(context.Context, string, string, siterequest.NoteRequest) (siterequest.Request, error) {
	return siterequest.Request{}, f.err
}
func (f *fakeService) Refuse(context.Context, string, string, siterequest.NoteRequest) (siterequest.Request, error) {
	return siterequest.Request{}, f.err
}

type spyLog struct{ n int }

func (s *spyLog) Error(string, ...any) { s.n++ }

func do(svc siterequest.Service, log siterequest.Logger, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewSiteRequestHandler(svc, log)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u1") })
	r.GET("/public/fleet/:id/quote", h.Quote)
	r.POST("/public/requests", h.Guest)
	r.GET("/client/cars", h.Cars)
	r.GET("/client/requests", h.Mine)
	r.POST("/client/requests", h.Create)
	r.GET("/client/requests/:id", h.MineDetail)
	r.PUT("/client/requests/:id", h.Resubmit)
	r.POST("/client/requests/:id/cancel", h.CancelMine)
	r.GET("/requests", h.List)
	r.GET("/requests/:id", h.Get)
	r.POST("/requests/:id/confirm", h.Confirm)
	r.POST("/requests/:id/adjust", h.Adjust)
	r.POST("/requests/:id/refuse", h.Refuse)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandler_Routes(t *testing.T) {
	cases := []struct {
		method, path, body string
		status             int
	}{
		{http.MethodGet, "/public/fleet/v1/quote?start_at=a&end_at=b", "", http.StatusOK},
		{http.MethodPost, "/public/requests", `{"name":"Ana"}`, http.StatusAccepted},
		{http.MethodGet, "/client/cars?start_at=a&end_at=b", "", http.StatusOK},
		{http.MethodGet, "/client/requests", "", http.StatusOK},
		{http.MethodPost, "/client/requests", `{"kind":"aluguel"}`, http.StatusCreated},
		{http.MethodGet, "/client/requests/r1", "", http.StatusOK},
		{http.MethodPut, "/client/requests/r1", `{"kind":"aluguel"}`, http.StatusOK},
		{http.MethodPost, "/client/requests/r1/cancel", "", http.StatusOK},
		{http.MethodGet, "/requests?status=solicitado", "", http.StatusOK},
		{http.MethodGet, "/requests/r1", "", http.StatusOK},
		{http.MethodPost, "/requests/r1/confirm", "", http.StatusOK},
		{http.MethodPost, "/requests/r1/adjust", `{"note":"x"}`, http.StatusOK},
		{http.MethodPost, "/requests/r1/refuse", `{"note":"x"}`, http.StatusOK},
	}
	for _, tc := range cases {
		w := do(&fakeService{}, &spyLog{}, tc.method, tc.path, tc.body)
		assert.Equal(t, tc.status, w.Code, tc.path)
	}
	for _, path := range []string{"/client/requests", "/requests/r1/adjust", "/requests/r1/refuse", "/requests/r1/confirm"} {
		assert.Equal(t, http.StatusBadRequest, do(&fakeService{}, &spyLog{}, http.MethodPost, path, "{").Code, path)
	}
	assert.Equal(t, http.StatusBadRequest, do(&fakeService{}, &spyLog{}, http.MethodPut, "/client/requests/r1", "{").Code)

	svc := &fakeService{}
	do(svc, &spyLog{}, http.MethodPost, "/client/requests", `{}`)
	assert.Equal(t, "u1", svc.user, "o cliente vem do token")
	do(svc, &spyLog{}, http.MethodPost, "/requests/r1/confirm", `{"price_cents":35000}`)
	assert.Equal(t, int64(35000), svc.conf.PriceCents)
}

func TestHandler_Errors(t *testing.T) {
	cases := []struct {
		err    error
		status int
	}{
		{siterequest.ErrNoAccount, http.StatusForbidden},
		{siterequest.ErrInvalidWhatsApp, http.StatusBadRequest},
		{booking.ErrBelowMinDays, http.StatusBadRequest},
		{siterequest.ErrNotFound, http.StatusNotFound},
		{booking.ErrVehicleNotFound, http.StatusNotFound},
		{siterequest.ErrPriceChanged, http.StatusConflict},
		{siterequest.ErrEmailInUse, http.StatusConflict},
		{siterequest.ErrInvalidEmail, http.StatusBadRequest},
		{email.ErrNotConfigured, http.StatusServiceUnavailable},
		{siterequest.ErrSendFailed, http.StatusBadGateway},
		{booking.ErrPeriodUnavailable, http.StatusConflict},
	}
	for _, tc := range cases {
		w := do(&fakeService{err: tc.err}, &spyLog{}, http.MethodGet, "/client/requests", "")
		assert.Equal(t, tc.status, w.Code, tc.err.Error())
		assert.Contains(t, w.Body.String(), tc.err.Error())
	}
	log := &spyLog{}
	w := do(&fakeService{err: errors.New("db")}, log, http.MethodGet, "/requests", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, 1, log.n)
}

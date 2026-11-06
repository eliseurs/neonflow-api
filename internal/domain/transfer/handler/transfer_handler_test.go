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

	"neonflow-api/internal/domain/transfer"
)

type fakeService struct {
	err    error
	userID string
	paid   bool
}

func (f *fakeService) out(userID string) (transfer.Transfer, error) {
	f.userID = userID
	return transfer.Transfer{ID: "t1"}, f.err
}
func (f *fakeService) List(context.Context) ([]transfer.Transfer, error) {
	return []transfer.Transfer{{ID: "t1"}}, f.err
}
func (f *fakeService) Get(context.Context, string) (transfer.Transfer, error) { return f.out("") }
func (f *fakeService) Events(context.Context, string) ([]transfer.Event, error) {
	return []transfer.Event{{Action: "criado"}}, f.err
}
func (f *fakeService) Create(_ context.Context, _ transfer.CreateRequest, u string) (transfer.Transfer, error) {
	return f.out(u)
}
func (f *fakeService) Drivers(context.Context, string) ([]transfer.Driver, error) {
	return []transfer.Driver{{Name: "M"}}, f.err
}
func (f *fakeService) Notify(_ context.Context, _, u string, _ transfer.NotifyRequest) (transfer.Transfer, error) {
	return f.out(u)
}
func (f *fakeService) Offer(_ context.Context, _, u string, _ transfer.OfferRequest) (transfer.Transfer, error) {
	return f.out(u)
}
func (f *fakeService) Accept(_ context.Context, _, u string, _ transfer.AcceptRequest) (transfer.Transfer, error) {
	return f.out(u)
}
func (f *fakeService) Release(_ context.Context, _, u string) (transfer.Transfer, error) {
	return f.out(u)
}
func (f *fakeService) SetPaid(_ context.Context, _, u string, paid bool) (transfer.Transfer, error) {
	f.paid = paid
	return f.out(u)
}
func (f *fakeService) Start(_ context.Context, _, u string) (transfer.Transfer, error) {
	return f.out(u)
}
func (f *fakeService) Complete(_ context.Context, _, u string, _ transfer.CompleteRequest) (transfer.Transfer, error) {
	return f.out(u)
}
func (f *fakeService) Cancel(_ context.Context, _, u string, _ transfer.CancelRequest) (transfer.Transfer, error) {
	return f.out(u)
}

type nopLog struct{}

func (nopLog) Error(string, ...any) {}

func do(svc transfer.Service, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewTransferHandler(svc, nopLog{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u1") })
	r.GET("/transfers", h.List)
	r.POST("/transfers", h.Create)
	r.GET("/transfers/:id", h.Get)
	r.GET("/transfers/:id/events", h.Events)
	r.GET("/transfers/:id/drivers", h.Drivers)
	r.POST("/transfers/:id/offer", h.Offer)
	r.POST("/transfers/:id/notify", h.Notify)
	r.POST("/transfers/:id/accept", h.Accept)
	r.POST("/transfers/:id/release", h.Release)
	r.PATCH("/transfers/:id/paid", h.SetPaid)
	r.POST("/transfers/:id/start", h.Start)
	r.POST("/transfers/:id/complete", h.Complete)
	r.POST("/transfers/:id/cancel", h.Cancel)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

var routes = []struct{ method, path, body string }{
	{http.MethodGet, "/transfers", ""},
	{http.MethodPost, "/transfers", `{"client_name":"C"}`},
	{http.MethodGet, "/transfers/t1", ""},
	{http.MethodGet, "/transfers/t1/events", ""},
	{http.MethodGet, "/transfers/t1/drivers", ""},
	{http.MethodPost, "/transfers/t1/offer", `{"offer_cents":100}`},
	{http.MethodPost, "/transfers/t1/notify", `{"driver_partner_id":"p1"}`},
	{http.MethodPost, "/transfers/t1/accept", `{"driver_partner_id":"p1"}`},
	{http.MethodPost, "/transfers/t1/release", ""},
	{http.MethodPatch, "/transfers/t1/paid", `{"paid":true}`},
	{http.MethodPost, "/transfers/t1/start", ""},
	{http.MethodPost, "/transfers/t1/complete", `{"extra_wait_cents":0}`},
	{http.MethodPost, "/transfers/t1/cancel", ""},
}

func TestTransferHandler_OK(t *testing.T) {
	for _, r := range routes {
		svc := &fakeService{}
		w := do(svc, r.method, r.path, r.body)
		want := http.StatusOK
		if r.method == http.MethodPost && r.path == "/transfers" {
			want = http.StatusCreated
		}
		assert.Equal(t, want, w.Code, r.path)
	}
	svc := &fakeService{}
	do(svc, http.MethodPatch, "/transfers/t1/paid", `{"paid":true}`)
	assert.True(t, svc.paid)
	assert.Equal(t, "u1", svc.userID, "quem fez vem do token")
	for _, path := range []string{"/transfers", "/transfers/t1/offer", "/transfers/t1/notify", "/transfers/t1/accept", "/transfers/t1/complete", "/transfers/t1/cancel"} {
		assert.Equal(t, http.StatusBadRequest, do(&fakeService{}, http.MethodPost, path, `{`).Code, path)
	}
	assert.Equal(t, http.StatusBadRequest, do(&fakeService{}, http.MethodPatch, "/transfers/t1/paid", `{`).Code)
}

func TestTransferHandler_Errors(t *testing.T) {
	cases := []struct {
		err    error
		status int
	}{
		{transfer.ErrNotFound, http.StatusNotFound},
		{transfer.ErrClientRequired, http.StatusBadRequest},
		{transfer.ErrInvalidPickup, http.StatusBadRequest},
		{transfer.ErrRouteRequired, http.StatusBadRequest},
		{transfer.ErrInvalidPassengers, http.StatusBadRequest},
		{transfer.ErrInvalidPrice, http.StatusBadRequest},
		{transfer.ErrReferrerNotFound, http.StatusBadRequest},
		{transfer.ErrInvalidOffer, http.StatusBadRequest},
		{transfer.ErrOfferDecreased, http.StatusBadRequest},
		{transfer.ErrVehicleNotFound, http.StatusBadRequest},
		{transfer.ErrDriverNotFound, http.StatusBadRequest},
		{transfer.ErrInvalidExtraWait, http.StatusBadRequest},
		{transfer.ErrWrongStage, http.StatusConflict},
		{transfer.ErrNoOffer, http.StatusConflict},
		{transfer.ErrNotPaid, http.StatusConflict},
		{errors.New("db"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		for _, r := range routes {
			assert.Equal(t, tc.status, do(&fakeService{err: tc.err}, r.method, r.path, r.body).Code, tc.err.Error()+" "+r.path)
		}
	}
}

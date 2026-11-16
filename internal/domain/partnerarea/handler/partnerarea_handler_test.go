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

	"neonflow-api/internal/domain/partnerarea"
	"neonflow-api/internal/domain/transfer"
)

type fakeService struct {
	err     error
	user    string
	vehicle string
}

func (f *fakeService) Area(_ context.Context, user string) (partnerarea.Area, error) {
	f.user = user
	return partnerarea.Area{Partner: partnerarea.Partner{Name: "Pedro"}}, f.err
}

func (f *fakeService) Accept(_ context.Context, _, _ string, req partnerarea.AcceptRequest) error {
	f.vehicle = req.VehicleID
	return f.err
}
func (f *fakeService) Release(context.Context, string, string) error { return f.err }

type spyLog struct{ n int }

func (s *spyLog) Error(string, ...any) { s.n++ }

func doReq(svc partnerarea.Service, log partnerarea.Logger, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewPartnerAreaHandler(svc, log)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u1") })
	r.GET("/partner/area", h.Area)
	r.POST("/partner/rides/:id/accept", h.Accept)
	r.POST("/partner/rides/:id/release", h.Release)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func do(svc partnerarea.Service, log partnerarea.Logger) *httptest.ResponseRecorder {
	return doReq(svc, log, http.MethodGet, "/partner/area", "")
}

func TestArea(t *testing.T) {
	svc := &fakeService{}
	w := do(svc, &spyLog{})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"name":"Pedro"`)
	assert.Equal(t, "u1", svc.user, "o parceiro vem da conta logada")

	assert.Equal(t, http.StatusForbidden, do(&fakeService{err: partnerarea.ErrNoPartner}, &spyLog{}).Code)
	log := &spyLog{}
	assert.Equal(t, http.StatusInternalServerError, do(&fakeService{err: errors.New("db")}, log).Code)
	assert.Equal(t, 1, log.n)
}

func TestAcceptAndRelease(t *testing.T) {
	svc := &fakeService{}
	w := doReq(svc, &spyLog{}, http.MethodPost, "/partner/rides/t1/accept", `{"vehicle_id":"v1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "v1", svc.vehicle)
	assert.Equal(t, http.StatusOK, doReq(&fakeService{}, &spyLog{}, http.MethodPost, "/partner/rides/t1/accept", "").Code, "sem corpo, sem carro")
	assert.Equal(t, http.StatusBadRequest, doReq(&fakeService{}, &spyLog{}, http.MethodPost, "/partner/rides/t1/accept", "{").Code)
	assert.Equal(t, http.StatusOK, doReq(&fakeService{}, &spyLog{}, http.MethodPost, "/partner/rides/t1/release", "").Code)

	cases := []struct {
		err    error
		status int
	}{
		{partnerarea.ErrNotDriver, http.StatusForbidden},
		{partnerarea.ErrNotYourVehicle, http.StatusBadRequest},
		{partnerarea.ErrNotYourRide, http.StatusNotFound},
		{transfer.ErrNotFound, http.StatusNotFound},
		{transfer.ErrWrongStage, http.StatusConflict},
		{transfer.ErrNoOffer, http.StatusConflict},
		{transfer.ErrDriverNotFound, http.StatusForbidden},
	}
	for _, tc := range cases {
		w := doReq(&fakeService{err: tc.err}, &spyLog{}, http.MethodPost, "/partner/rides/t1/accept", "")
		assert.Equal(t, tc.status, w.Code, tc.err.Error())
	}
	w = doReq(&fakeService{err: transfer.ErrWrongStage}, &spyLog{}, http.MethodPost, "/partner/rides/t1/accept", "")
	assert.Contains(t, w.Body.String(), "não está mais disponível")
}

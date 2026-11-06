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
	"neonflow-api/internal/domain/fleet"
)

type fakeService struct {
	err  error
	user *string
}

func (f fakeService) List(_ context.Context, vehicleID string) ([]booking.Booking, error) {
	return []booking.Booking{{ID: "b1", VehicleID: vehicleID}}, f.err
}

func (f fakeService) Get(_ context.Context, id string) (booking.Booking, error) {
	return booking.Booking{ID: id}, f.err
}

func (f fakeService) Events(context.Context, string) ([]booking.Event, error) {
	return []booking.Event{{ToStage: "preparacao"}}, f.err
}

func (f fakeService) Create(_ context.Context, r booking.CreateRequest, userID string) (booking.Booking, error) {
	*f.user = userID
	return booking.Booking{ID: "new", ClientName: r.ClientName}, f.err
}

func (f fakeService) SetChecks(_ context.Context, id string, checks map[string]bool) (booking.Booking, error) {
	return booking.Booking{ID: id, Checks: checks}, f.err
}

func (f fakeService) Advance(_ context.Context, id, userID string, req booking.AdvanceRequest) (booking.Booking, error) {
	*f.user = userID
	return booking.Booking{ID: id, Stage: "confirmada", Place: req.At}, f.err
}

func (f fakeService) Cancel(_ context.Context, id, userID string) (booking.Booking, error) {
	*f.user = userID
	return booking.Booking{ID: id, Stage: "cancelada"}, f.err
}

func (f fakeService) Available(_ context.Context, start, end string) ([]fleet.Vehicle, error) {
	return []fleet.Vehicle{{ID: start + "_" + end}}, f.err
}

type nopLog struct{}

func (nopLog) Error(string, ...any) {}

func do(svc booking.Service, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewBookingHandler(svc, nopLog{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u1") })
	r.GET("/availability", h.Available)
	r.GET("/bookings", h.List)
	r.GET("/bookings/:id", h.Get)
	r.GET("/bookings/:id/events", h.Events)
	r.POST("/bookings", h.Create)
	r.PATCH("/bookings/:id/checks", h.SetChecks)
	r.POST("/bookings/:id/advance", h.Advance)
	r.POST("/bookings/:id/cancel", h.Cancel)

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestBookingHandler_OK(t *testing.T) {
	cases := []struct {
		method, path, body string
		status             int
		want               string
	}{
		{http.MethodGet, "/availability?start_date=2026-12-10&end_date=2026-12-12", "", http.StatusOK, `"id":"2026-12-10_2026-12-12"`},
		{http.MethodGet, "/availability?start_at=2026-12-10T14:00&end_at=2026-12-12T10:00&start_date=x", "", http.StatusOK, `"id":"2026-12-10T14:00_2026-12-12T10:00"`},
		{http.MethodGet, "/bookings?vehicle_id=v1", "", http.StatusOK, `"vehicle_id":"v1"`},
		{http.MethodGet, "/bookings/b9", "", http.StatusOK, `"id":"b9"`},
		{http.MethodGet, "/bookings/b9/events", "", http.StatusOK, `"to_stage":"preparacao"`},
		{http.MethodPost, "/bookings", `{"client_name":"Cliente"}`, http.StatusCreated, `"client_name":"Cliente"`},
		{http.MethodPatch, "/bookings/b1/checks", `{"cnh":true}`, http.StatusOK, `"cnh":true`},
		{http.MethodPost, "/bookings/b1/advance", "", http.StatusOK, `"stage":"confirmada"`},
		{http.MethodPost, "/bookings/b1/advance", `{"at":"2026-12-12T10:30"}`, http.StatusOK, `"place":"2026-12-12T10:30"`},
		{http.MethodPost, "/bookings/b1/cancel", "", http.StatusOK, `"stage":"cancelada"`},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			var user string
			w := do(fakeService{user: &user}, tc.method, tc.path, tc.body)
			assert.Equal(t, tc.status, w.Code)
			assert.Contains(t, w.Body.String(), tc.want)
			if tc.method == http.MethodPost {
				assert.Equal(t, "u1", user, "o usuário do token vai para o histórico")
			}
		})
	}
}

func TestBookingHandler_BadBody(t *testing.T) {
	var user string
	svc := fakeService{user: &user}
	assert.Equal(t, http.StatusBadRequest, do(svc, http.MethodPost, "/bookings", `{`).Code)
	assert.Equal(t, http.StatusBadRequest, do(svc, http.MethodPatch, "/bookings/b1/checks", `{"cnh":"sim"}`).Code)
	assert.Equal(t, http.StatusBadRequest, do(svc, http.MethodPost, "/bookings/b1/advance", `{"at":10}`).Code)
}

func TestBookingHandler_Errors(t *testing.T) {
	cases := []struct {
		err    error
		status int
	}{
		{booking.ErrBookingNotFound, http.StatusNotFound},
		{booking.ErrVehicleNotFound, http.StatusNotFound},
		{booking.ErrVehicleRequired, http.StatusBadRequest},
		{booking.ErrClientNameRequired, http.StatusBadRequest},
		{booking.ErrInvalidDate, http.StatusBadRequest},
		{booking.ErrInvalidPeriod, http.StatusBadRequest},
		{booking.ErrInvalidCheck, http.StatusBadRequest},
		{booking.ErrVehicleInMaintenance, http.StatusConflict},
		{booking.ErrPeriodUnavailable, http.StatusConflict},
		{booking.ErrChecksPending, http.StatusConflict},
		{booking.ErrStageFinal, http.StatusConflict},
		{booking.ErrNotCancelable, http.StatusConflict},
		{booking.ErrStageChanged, http.StatusConflict},
		{booking.ErrAwaitingChecklist, http.StatusConflict},
		{booking.ErrVehicleInUse, http.StatusConflict},
		{booking.ErrKmPlanUnavailable, http.StatusConflict},
		{booking.ErrInvalidKmPlan, http.StatusBadRequest},
		{booking.ErrKmRequired, http.StatusBadRequest},
		{booking.ErrInvalidKm, http.StatusBadRequest},
		{booking.ErrKmBeforeDelivery, http.StatusBadRequest},
		{booking.ErrReferrerNotFound, http.StatusBadRequest},
		{booking.ErrCustomerNotFound, http.StatusBadRequest},
		{booking.ErrPhotosRequired, http.StatusBadRequest},
		{booking.ErrFutureTime, http.StatusBadRequest},
		{booking.ErrReturnBeforeDelivery, http.StatusBadRequest},
		{errors.New("db"), http.StatusInternalServerError},
	}
	paths := []struct{ method, path, body string }{
		{http.MethodGet, "/availability", ""},
		{http.MethodGet, "/bookings", ""},
		{http.MethodGet, "/bookings/b1", ""},
		{http.MethodGet, "/bookings/b1/events", ""},
		{http.MethodPost, "/bookings", `{}`},
		{http.MethodPatch, "/bookings/b1/checks", `{"cnh":true}`},
		{http.MethodPost, "/bookings/b1/advance", ""},
		{http.MethodPost, "/bookings/b1/cancel", ""},
	}
	for _, tc := range cases {
		for _, p := range paths {
			var user string
			w := do(fakeService{err: tc.err, user: &user}, p.method, p.path, p.body)
			assert.Equal(t, tc.status, w.Code, "%v em %s %s", tc.err, p.method, p.path)
			if tc.status == http.StatusInternalServerError {
				assert.Contains(t, w.Body.String(), "erro interno")
			} else {
				assert.Contains(t, w.Body.String(), tc.err.Error())
			}
		}
	}
}

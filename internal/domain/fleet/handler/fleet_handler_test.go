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

	"neonflow-api/internal/domain/fleet"
)

type fakeService struct{ err error }

func (f fakeService) List(context.Context) ([]fleet.Vehicle, error) {
	return []fleet.Vehicle{{ID: "v1"}}, f.err
}

func (f fakeService) Get(_ context.Context, id string) (fleet.Vehicle, error) {
	return fleet.Vehicle{ID: id}, f.err
}

func (f fakeService) Create(_ context.Context, r fleet.VehicleRequest) (fleet.Vehicle, error) {
	return fleet.Vehicle{ID: "new", Plate: r.Plate}, f.err
}

func (f fakeService) Update(_ context.Context, id string, r fleet.VehicleRequest) (fleet.Vehicle, error) {
	return fleet.Vehicle{ID: id, Plate: r.Plate}, f.err
}

type nopLog struct{}

func (nopLog) Error(string, ...any) {}

func do(svc fleet.Service, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewFleetHandler(svc, nopLog{})
	r := gin.New()
	r.GET("/vehicles", h.List)
	r.GET("/vehicles/:id", h.Get)
	r.POST("/vehicles", h.Create)
	r.PUT("/vehicles/:id", h.Update)

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestFleetHandler(t *testing.T) {
	body := `{"plate":"TST1A23"}`
	cases := []struct {
		name   string
		err    error
		method string
		path   string
		body   string
		status int
		want   string
	}{
		{"list", nil, http.MethodGet, "/vehicles", "", http.StatusOK, `"id":"v1"`},
		{"get", nil, http.MethodGet, "/vehicles/v9", "", http.StatusOK, `"id":"v9"`},
		{"create", nil, http.MethodPost, "/vehicles", body, http.StatusCreated, `"plate":"TST1A23"`},
		{"update", nil, http.MethodPut, "/vehicles/v1", body, http.StatusOK, `"id":"v1"`},
		{"create json inválido", nil, http.MethodPost, "/vehicles", `{`, http.StatusBadRequest, "corpo da requisição inválido"},
		{"update json inválido", nil, http.MethodPut, "/vehicles/v1", `[`, http.StatusBadRequest, "corpo da requisição inválido"},
		{"não encontrado", fleet.ErrVehicleNotFound, http.MethodGet, "/vehicles/x", "", http.StatusNotFound, "veículo não encontrado"},
		{"placa repetida", fleet.ErrPlateTaken, http.MethodPost, "/vehicles", body, http.StatusConflict, "placa já cadastrada"},
		{"validação", fleet.ErrInvalidDailyRate, http.MethodPut, "/vehicles/v1", body, http.StatusBadRequest, "diária deve ser maior"},
		{"plano de km", fleet.ErrKmPlanIncomplete, http.MethodPost, "/vehicles", body, http.StatusBadRequest, "juntos"},
		{"erro interno", errors.New("db"), http.MethodGet, "/vehicles", "", http.StatusInternalServerError, "erro interno"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := do(fakeService{err: tc.err}, tc.method, tc.path, tc.body)
			assert.Equal(t, tc.status, w.Code)
			assert.Contains(t, w.Body.String(), tc.want)
		})
	}
}

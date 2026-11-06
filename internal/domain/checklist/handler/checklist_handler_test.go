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

	"neonflow-api/internal/domain/checklist"
)

type fakeService struct{ err error }

func (f fakeService) List(context.Context) ([]checklist.Item, error) {
	return []checklist.Item{{Key: "pneus"}}, f.err
}

func (f fakeService) Create(_ context.Context, r checklist.ItemRequest) (checklist.Item, error) {
	return checklist.Item{Key: "novo", Label: r.Label}, f.err
}

func (f fakeService) Update(_ context.Context, id string, r checklist.ItemRequest) (checklist.Item, error) {
	return checklist.Item{ID: id, Label: r.Label}, f.err
}

type nopLog struct{}

func (nopLog) Error(string, ...any) {}

func do(svc checklist.Service, method, path, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewChecklistHandler(svc, nopLog{})
	r := gin.New()
	r.GET("/checklist-items", h.List)
	r.POST("/checklist-items", h.Create)
	r.PUT("/checklist-items/:id", h.Update)

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestChecklistHandler_OK(t *testing.T) {
	w := do(fakeService{}, http.MethodGet, "/checklist-items", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"key":"pneus"`)

	w = do(fakeService{}, http.MethodPost, "/checklist-items", `{"label":"Calota"}`)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), `"label":"Calota"`)

	w = do(fakeService{}, http.MethodPut, "/checklist-items/i1", `{"label":"Pneus","active":false}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"i1"`)
}

func TestChecklistHandler_BadBody(t *testing.T) {
	assert.Equal(t, http.StatusBadRequest, do(fakeService{}, http.MethodPost, "/checklist-items", `{`).Code)
	assert.Equal(t, http.StatusBadRequest, do(fakeService{}, http.MethodPut, "/checklist-items/i1", `{"active":"sim"}`).Code)
}

func TestChecklistHandler_Errors(t *testing.T) {
	cases := []struct {
		err    error
		status int
	}{
		{checklist.ErrItemNotFound, http.StatusNotFound},
		{checklist.ErrKeyTaken, http.StatusConflict},
		{checklist.ErrKeyReserved, http.StatusConflict},
		{checklist.ErrLabelRequired, http.StatusBadRequest},
		{checklist.ErrInvalidLabel, http.StatusBadRequest},
		{errors.New("db"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		for _, call := range [][3]string{
			{http.MethodGet, "/checklist-items", ""},
			{http.MethodPost, "/checklist-items", `{"label":"x"}`},
			{http.MethodPut, "/checklist-items/i1", `{"label":"x"}`},
		} {
			w := do(fakeService{err: tc.err}, call[0], call[1], call[2])
			assert.Equal(t, tc.status, w.Code, tc.err.Error()+" "+call[0])
			if tc.status == http.StatusInternalServerError {
				assert.Contains(t, w.Body.String(), "erro interno")
			} else {
				assert.Contains(t, w.Body.String(), tc.err.Error())
			}
		}
	}
}

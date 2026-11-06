package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"neonflow-api/internal/domain/payout"
)

type fakeService struct {
	err    error
	filter payout.Filter
	paidBy string
}

func (f *fakeService) List(_ context.Context, filter payout.Filter) ([]payout.Payout, error) {
	f.filter = filter
	return []payout.Payout{{ID: "o1", Kind: filter.Kind}}, f.err
}
func (f *fakeService) MarkPaid(_ context.Context, id, userID string) (payout.Payout, error) {
	f.paidBy = userID
	return payout.Payout{ID: id, Status: payout.StatusPaid}, f.err
}

type nopLog struct{}

func (nopLog) Error(string, ...any) {}

func do(svc payout.Service, method, path string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	h := NewPayoutHandler(svc, nopLog{})
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "u1") })
	r.GET("/payouts", h.List)
	r.POST("/payouts/:id/pay", h.Pay)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func TestPayoutHandler(t *testing.T) {
	svc := &fakeService{}
	w := do(svc, http.MethodGet, "/payouts?kind=repasse&status=pendente&partner_id=p1")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, payout.Filter{Kind: "repasse", Status: "pendente", PartnerID: "p1"}, svc.filter)
	w = do(svc, http.MethodPost, "/payouts/o1/pay")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "u1", svc.paidBy, "quem marcou vem do token")

	cases := []struct {
		err    error
		status int
	}{
		{payout.ErrNotFound, http.StatusNotFound},
		{payout.ErrInvalidFilter, http.StatusBadRequest},
		{payout.ErrAlreadyPaid, http.StatusConflict},
		{payout.ErrPercentUndefined, http.StatusConflict},
		{errors.New("db"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.status, do(&fakeService{err: tc.err}, http.MethodGet, "/payouts").Code, tc.err.Error())
		assert.Equal(t, tc.status, do(&fakeService{err: tc.err}, http.MethodPost, "/payouts/o1/pay").Code, tc.err.Error())
	}
}

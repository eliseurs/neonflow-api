package transfer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/pkg/timezone"
)

func ptr[T any](v T) *T { return &v }

func TestCreateRequestValidate(t *testing.T) {
	n, err := CreateRequest{ClientName: " Cliente Teste ", PickupAt: "2026-12-10T08:30", Origin: " Aeroporto POA ",
		Destination: "Gramado", Flight: " la 3090 ", PriceCents: 45000}.Validate("u1")
	require.NoError(t, err)
	assert.Equal(t, "Cliente Teste", n.ClientName)
	assert.Equal(t, "LA3090", n.Flight)
	assert.Equal(t, 1, n.Passengers, "padrão: 1 passageiro")
	assert.True(t, n.PickupAt.Equal(time.Date(2026, 12, 10, 8, 30, 0, 0, timezone.Location)))
	assert.Equal(t, "u1", n.CreatedBy)

	ok := CreateRequest{ClientName: "C", PickupAt: "2026-12-10T08:30", Origin: "A", Destination: "B", PriceCents: 1}
	cases := []struct {
		name   string
		change func(r *CreateRequest)
		want   error
	}{
		{"cliente", func(r *CreateRequest) { r.ClientName = " " }, ErrClientRequired},
		{"data", func(r *CreateRequest) { r.PickupAt = "10/12/2026" }, ErrInvalidPickup},
		{"origem", func(r *CreateRequest) { r.Origin = "" }, ErrRouteRequired},
		{"destino", func(r *CreateRequest) { r.Destination = "" }, ErrRouteRequired},
		{"passageiros", func(r *CreateRequest) { r.Passengers = 61 }, ErrInvalidPassengers},
		{"passageiros negativos", func(r *CreateRequest) { r.Passengers = -1 }, ErrInvalidPassengers},
		{"preço", func(r *CreateRequest) { r.PriceCents = 0 }, ErrInvalidPrice},
		{"indicador", func(r *CreateRequest) { r.ReferredByPartnerID = "x" }, ErrReferrerNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := ok
			tc.change(&r)
			_, err := r.Validate("u1")
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestOfferValidate(t *testing.T) {
	assert.NoError(t, OfferRequest{OfferCents: 20000, Minutes: 30}.Validate(nil))
	assert.NoError(t, OfferRequest{OfferCents: 25000, Minutes: 30}.Validate(ptr(int64(20000))))
	assert.ErrorIs(t, OfferRequest{OfferCents: 0, Minutes: 30}.Validate(nil), ErrInvalidOffer)
	assert.ErrorIs(t, OfferRequest{OfferCents: 15000, Minutes: 30}.Validate(ptr(int64(20000))), ErrOfferDecreased)
	assert.ErrorIs(t, OfferRequest{OfferCents: 20000, Minutes: 0}.Validate(nil), ErrInvalidDeadline)
	assert.ErrorIs(t, OfferRequest{OfferCents: 20000, Minutes: MaxOfferMinutes + 1}.Validate(nil), ErrInvalidDeadline)
}

func TestCancelFee(t *testing.T) {
	pickup := time.Date(2026, 12, 10, 8, 0, 0, 0, timezone.Location)
	// RN-02.10: grátis até 24 horas antes; dentro, 50%; voo cancelado, nada.
	assert.Equal(t, int64(0), CancelFee(45000, pickup, pickup.Add(-24*time.Hour), false))
	assert.Equal(t, int64(22500), CancelFee(45000, pickup, pickup.Add(-23*time.Hour), false))
	assert.Equal(t, int64(22501), CancelFee(45001, pickup, pickup.Add(-time.Hour), false), "arredonda para cima")
	assert.Equal(t, int64(0), CancelFee(45000, pickup, pickup.Add(-time.Hour), true))
	assert.True(t, Cancellable(StageWaiting))
	assert.True(t, Cancellable(StageAssigned))
	assert.False(t, Cancellable(StageOnTrip))
	assert.False(t, Cancellable(StageDone))
}

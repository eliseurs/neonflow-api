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

func TestValidateOffer(t *testing.T) {
	assert.NoError(t, ValidateOffer(20000, nil))
	assert.NoError(t, ValidateOffer(25000, ptr(int64(20000))))
	assert.NoError(t, ValidateOffer(20000, ptr(int64(20000))))
	assert.ErrorIs(t, ValidateOffer(0, nil), ErrInvalidOffer)
	assert.ErrorIs(t, ValidateOffer(15000, ptr(int64(20000))), ErrOfferDecreased)
}

func TestAcceptRequestValidate(t *testing.T) {
	id := "00000000-0000-0000-0000-000000000001"
	r, err := AcceptRequest{DriverPartnerID: " " + id + " ", VehicleID: " " + id + " "}.Validate(ptr(int64(20000)))
	require.NoError(t, err)
	assert.Equal(t, id, r.DriverPartnerID)
	assert.Equal(t, id, r.VehicleID)
	_, err = AcceptRequest{DriverPartnerID: id, OfferCents: ptr(int64(30000))}.Validate(nil)
	require.NoError(t, err, "valor informado no aceite dispensa o anterior")

	_, err = AcceptRequest{DriverPartnerID: "x"}.Validate(ptr(int64(1)))
	assert.ErrorIs(t, err, ErrDriverNotFound)
	_, err = AcceptRequest{DriverPartnerID: id, VehicleID: "x"}.Validate(ptr(int64(1)))
	assert.ErrorIs(t, err, ErrVehicleNotFound)
	_, err = AcceptRequest{DriverPartnerID: id}.Validate(nil)
	assert.ErrorIs(t, err, ErrNoOffer)
	_, err = AcceptRequest{DriverPartnerID: id, OfferCents: ptr(int64(100))}.Validate(ptr(int64(200)))
	assert.ErrorIs(t, err, ErrOfferDecreased)
}

func TestFormatReais(t *testing.T) {
	assert.Equal(t, "R$ 350,00", FormatReais(35000))
	assert.Equal(t, "R$ 1.350,05", FormatReais(135005))
	assert.Equal(t, "R$ 1.234.567,89", FormatReais(123456789))
	assert.Equal(t, "R$ 0,07", FormatReais(7))
	assert.Equal(t, "-R$ 10,00", FormatReais(-1000))
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

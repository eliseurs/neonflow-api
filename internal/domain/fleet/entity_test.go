package fleet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validRequest() VehicleRequest {
	return VehicleRequest{Plate: " tst-1a23 ", Name: " Sedã Teste ", Category: "Economy", Owner: " Parceiro ",
		Seats: 5, Gear: "MANUAL", DailyRateCents: 10000}
}

func TestVehicleRequestValidate(t *testing.T) {
	got, err := validRequest().Validate()
	require.NoError(t, err)
	assert.Equal(t, "TST1A23", got.Plate)
	assert.Equal(t, "Sedã Teste", got.Name)
	assert.Equal(t, "economy", got.Category)
	assert.Equal(t, "Parceiro", got.Owner)
	assert.Equal(t, "manual", got.Gear)

	old := validRequest()
	old.Plate = "abc1234"
	_, err = old.Validate()
	assert.NoError(t, err, "placa no formato antigo")

	cases := []struct {
		name   string
		change func(*VehicleRequest)
		want   error
	}{
		{"placa", func(r *VehicleRequest) { r.Plate = "12345" }, ErrInvalidPlate},
		{"nome", func(r *VehicleRequest) { r.Name = " " }, ErrNameRequired},
		{"categoria", func(r *VehicleRequest) { r.Category = "luxo" }, ErrInvalidCategory},
		{"dono", func(r *VehicleRequest) { r.Owner = "" }, ErrOwnerRequired},
		{"lugares", func(r *VehicleRequest) { r.Seats = 0 }, ErrInvalidSeats},
		{"câmbio", func(r *VehicleRequest) { r.Gear = "cvt" }, ErrInvalidGear},
		{"diária", func(r *VehicleRequest) { r.DailyRateCents = 0 }, ErrInvalidDailyRate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := validRequest()
			tc.change(&r)
			_, err := r.Validate()
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

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

func ptr[T any](v T) *T { return &v }

func TestVehicleRequestValidate(t *testing.T) {
	// Caução e plano de km são opcionais (RN-06.4, RN-02.18).
	full := validRequest()
	full.DepositCents, full.KmLimitPerDay, full.ExtraKmCents = ptr(int64(100000)), ptr(200), ptr(int64(150))
	_, err := full.Validate()
	require.NoError(t, err)

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
		{"caução", func(r *VehicleRequest) { r.DepositCents = ptr(int64(0)) }, ErrInvalidDeposit},
		{"limite de km", func(r *VehicleRequest) { r.KmLimitPerDay, r.ExtraKmCents = ptr(0), ptr(int64(150)) }, ErrInvalidKmPlan},
		{"km excedente", func(r *VehicleRequest) { r.KmLimitPerDay, r.ExtraKmCents = ptr(200), ptr(int64(-1)) }, ErrInvalidKmPlan},
		{"km sem valor", func(r *VehicleRequest) { r.KmLimitPerDay = ptr(200) }, ErrKmPlanIncomplete},
		{"valor sem km", func(r *VehicleRequest) { r.ExtraKmCents = ptr(int64(150)) }, ErrKmPlanIncomplete},
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

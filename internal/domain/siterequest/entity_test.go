package siterequest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/booking"
)

var now = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

const carID = "11111111-1111-1111-1111-111111111111"

func TestValidate_Rental(t *testing.T) {
	n, err := CreateRequest{Kind: "aluguel", WhatsApp: "+55 (54) 99999-0001", VehicleID: carID, StartAt: "2026-12-10T10:00", EndAt: "2026-12-12T12:00",
		Place: " Hotel ", KmPlan: "LIVRE", ReferralCode: " htl234 "}.Validate(now)
	require.NoError(t, err)
	assert.Equal(t, "5554999990001", n.WhatsApp)
	assert.Equal(t, "Hotel", n.Place)
	assert.Equal(t, "livre", n.KmPlan)
	assert.Equal(t, "HTL234", n.ReferralCode)
	assert.Equal(t, 3, n.Period.Days())

	cases := map[string]struct {
		req  CreateRequest
		want error
	}{
		"tipo":       {CreateRequest{Kind: "passeio", WhatsApp: "5554999990001"}, ErrInvalidKind},
		"whatsapp":   {CreateRequest{Kind: "aluguel", WhatsApp: "123"}, ErrInvalidWhatsApp},
		"carro":      {CreateRequest{Kind: "aluguel", WhatsApp: "5554999990001", VehicleID: "x"}, booking.ErrVehicleRequired},
		"km":         {CreateRequest{Kind: "aluguel", WhatsApp: "5554999990001", VehicleID: carID, KmPlan: "meio"}, booking.ErrInvalidKmPlan},
		"período":    {CreateRequest{Kind: "aluguel", WhatsApp: "5554999990001", VehicleID: carID, StartAt: "2026-12-12T10:00", EndAt: "2026-12-10T10:00"}, booking.ErrInvalidPeriod},
		"no passado": {CreateRequest{Kind: "aluguel", WhatsApp: "5554999990001", VehicleID: carID, StartAt: "2026-11-01T10:00", EndAt: "2026-11-03T10:00"}, ErrPastDate},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := tc.req.Validate(now)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestValidate_Transfer(t *testing.T) {
	n, err := CreateRequest{Kind: "transfer", WhatsApp: "5554999990001", Origin: " POA ", Destination: "Gramado", PickupAt: "2026-12-10T08:30",
		Passengers: 3, Flight: " G3 1234 "}.Validate(now)
	require.NoError(t, err)
	assert.Equal(t, "POA", n.Origin)
	assert.Equal(t, "G3 1234", n.Flight)
	assert.Equal(t, 3, n.Passengers)

	base := CreateRequest{Kind: "transfer", WhatsApp: "5554999990001", Origin: "POA", Destination: "Gramado", PickupAt: "2026-12-10T08:30", Passengers: 2}
	noRoute, noPax, past, badDate := base, base, base, base
	noRoute.Destination = " "
	noPax.Passengers = 0
	past.PickupAt = "2026-11-10T08:30"
	badDate.PickupAt = "amanhã"
	for req, want := range map[*CreateRequest]error{&noRoute: ErrRouteRequired, &noPax: ErrInvalidPassengers, &past: ErrPastDate, &badDate: booking.ErrInvalidDate} {
		_, err := req.Validate(now)
		assert.ErrorIs(t, err, want)
	}
}

func TestQuoteFor(t *testing.T) {
	deposit, extra, limit := int64(35000), int64(150), 200
	terms := VehicleTerms{DailyRateCents: 25000, DepositCents: &deposit, KmLimitPerDay: &limit, ExtraKmCents: &extra, MinDays: 1}
	p := booking.Period{Start: now, End: now.Add(49 * time.Hour)}

	q, err := QuoteFor(terms, p, "")
	require.NoError(t, err)
	assert.Equal(t, 3, *q.Days, "blocos de 24 horas para cima (RN-02.24)")
	assert.Equal(t, int64(75000), *q.TotalCents)
	assert.Equal(t, 200, *q.KmLimitPerDay, "sem plano escolhido, vale o limite do carro")

	free, err := QuoteFor(terms, p, booking.KmPlanLivre)
	require.NoError(t, err)
	assert.Nil(t, free.KmLimitPerDay)
	assert.False(t, q.Same(free), "km diferente muda o orçamento")
	assert.True(t, q.Same(q))

	_, err = QuoteFor(VehicleTerms{DailyRateCents: 1, MinDays: 1}, p, booking.KmPlanLimitado)
	assert.ErrorIs(t, err, booking.ErrKmPlanUnavailable)
	_, err = QuoteFor(VehicleTerms{DailyRateCents: 1, MinDays: 5}, p, "")
	assert.ErrorIs(t, err, booking.ErrBelowMinDays)
	_, err = QuoteFor(VehicleTerms{DailyRateCents: 1, InMaintenance: true}, p, "")
	assert.ErrorIs(t, err, booking.ErrVehicleInMaintenance)
}

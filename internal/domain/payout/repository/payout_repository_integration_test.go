//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/booking"
	bookingRepo "neonflow-api/internal/domain/booking/repository"
	"neonflow-api/internal/domain/payout"
	"neonflow-api/internal/testutil"
)

func TestPayoutRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewPayoutRepository(database)
	bookings := bookingRepo.NewBookingRepository(database)
	ctx := context.Background()

	var userID, ownerID, vehicleID, ownVehicleID, hotelID, newID string
	insert := func(dest *string, query string, args ...any) {
		t.Helper()
		require.NoError(t, database.QueryRowContext(ctx, query, args...).Scan(dest))
	}
	insert(&userID, `INSERT INTO users (name, email, password_hash, role) VALUES ('Central Teste', 'central@neonflow.test', 'h', 'central') RETURNING id`)
	insert(&ownerID, `INSERT INTO partners (name, is_owner, referral_code, owner_share_bp) VALUES ('Dono Teste', true, 'OWN234', 7000) RETURNING id`)
	insert(&hotelID, `INSERT INTO partners (name, is_referrer, referral_code, commission_bp) VALUES ('Hotel Teste', true, 'HTL234', 1000) RETURNING id`)
	insert(&newID, `INSERT INTO partners (name, is_referrer, referral_code) VALUES ('Sem Percentual', true, 'NEW234') RETURNING id`)
	insert(&vehicleID, `INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents, owner_partner_id)
		VALUES ('TST1A23', 'Sedã Parceiro', 'economy', 'Dono Teste', 5, 'manual', 15000, $1) RETURNING id`, ownerID)
	insert(&ownVehicleID, `INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents)
		VALUES ('TST2B34', 'Sedã Próprio', 'economy', 'Neon Flow', 5, 'manual', 15000) RETURNING id`)

	// Leva a reserva até "encerrada" (o repository não confere as conferências; isso é do serviço).
	closeBooking := func(vehicle, start, end, referrer string, extras booking.Record) booking.Booking {
		t.Helper()
		p, err := booking.ParsePeriod(start, end)
		require.NoError(t, err)
		b, err := bookings.Create(ctx, booking.NewBooking{VehicleID: vehicle, ClientName: "Cliente Teste", Period: p,
			CreatedBy: userID, ReferredBy: referrer})
		require.NoError(t, err)
		steps := []struct {
			from, to string
			rec      booking.Record
		}{
			{booking.StagePreparacao, booking.StageConfirmada, booking.Record{}},
			{booking.StageConfirmada, booking.StageEmLocacao, booking.Record{}},
			{booking.StageEmLocacao, booking.StageDevolvida, extras},
			{booking.StageDevolvida, booking.StageEncerrada, booking.Record{}},
		}
		for _, s := range steps {
			_, err = bookings.Transition(ctx, b.ID, s.from, s.to, userID, s.rec)
			require.NoError(t, err)
		}
		return b
	}
	byKind := func(list []payout.Payout, bookingID string) map[string]payout.Payout {
		out := map[string]payout.Payout{}
		for _, p := range list {
			if p.BookingID != nil && *p.BookingID == bookingID {
				out[p.Kind] = p
			}
		}
		return out
	}

	// Carro de parceiro, indicado, com atraso e km: comissão de 10% sobre as 2 diárias (R$ 300) e
	// repasse de 70% sobre diárias + atraso + km (R$ 300 + 25 + 15 = R$ 340).
	late, lateCents, excessCents := 100, int64(2500), int64(1500)
	excess := 10
	b1 := closeBooking(vehicleID, "2026-12-10T10:00", "2026-12-12T10:00", hotelID,
		booking.Record{ExtraMinutes: &late, ExtraCents: &lateCents, ExcessKm: &excess, ExcessKmCents: &excessCents})
	all, err := repo.List(ctx, payout.Filter{})
	require.NoError(t, err)
	got := byKind(all, b1.ID)
	require.Len(t, got, 2)
	c := got[payout.KindCommission]
	assert.Equal(t, "Hotel Teste", c.PartnerName)
	assert.Equal(t, int64(30000), c.BaseCents, "comissão só sobre as diárias")
	assert.Equal(t, int64(3000), *c.AmountCents)
	o := got[payout.KindOwnerShare]
	assert.Equal(t, "Dono Teste", o.PartnerName)
	assert.Equal(t, "Sedã Parceiro", o.Description)
	assert.Equal(t, int64(34000), o.BaseCents, "repasse sobre diárias + atraso + km")
	assert.Equal(t, 70.0, *o.Percent)
	assert.Equal(t, int64(23800), *o.AmountCents)

	// Carro próprio sem indicação: nada a pagar a parceiros.
	b2 := closeBooking(ownVehicleID, "2026-12-13T10:00", "2026-12-14T10:00", "", booking.Record{})
	all, err = repo.List(ctx, payout.Filter{})
	require.NoError(t, err)
	assert.Empty(t, byKind(all, b2.ID))

	// Percentual que muda depois não altera o já gerado; pagar registra quem e quando.
	_, err = database.ExecContext(ctx, `UPDATE partners SET commission_bp = 2000 WHERE id = $1`, hotelID)
	require.NoError(t, err)
	paid, err := repo.MarkPaid(ctx, c.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, payout.StatusPaid, paid.Status)
	assert.Equal(t, int64(3000), *paid.AmountCents, "vale o percentual do encerramento")
	assert.Equal(t, "Central Teste", *paid.PaidByName)
	assert.WithinDuration(t, time.Now(), *paid.PaidAt, time.Minute)
	_, err = repo.MarkPaid(ctx, c.ID, userID)
	assert.ErrorIs(t, err, payout.ErrAlreadyPaid)

	// Indicador sem percentual: a definir; só paga depois de definir, com o percentual do momento.
	b3 := closeBooking(ownVehicleID, "2026-12-15T10:00", "2026-12-16T10:00", newID, booking.Record{})
	pending, err := repo.List(ctx, payout.Filter{Kind: payout.KindCommission, Status: payout.StatusPending, PartnerID: newID})
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, b3.ID, *pending[0].BookingID)
	assert.Nil(t, pending[0].AmountCents)
	_, err = repo.MarkPaid(ctx, pending[0].ID, userID)
	assert.ErrorIs(t, err, payout.ErrPercentUndefined)
	_, err = database.ExecContext(ctx, `UPDATE partners SET commission_bp = 500 WHERE id = $1`, newID)
	require.NoError(t, err)
	later, err := repo.MarkPaid(ctx, pending[0].ID, userID)
	require.NoError(t, err)
	assert.Equal(t, 5.0, *later.Percent)
	assert.Equal(t, int64(750), *later.AmountCents)

	repasses, err := repo.List(ctx, payout.Filter{Kind: payout.KindOwnerShare, Status: payout.StatusPending})
	require.NoError(t, err)
	require.Len(t, repasses, 1)
	paidList, err := repo.List(ctx, payout.Filter{Status: payout.StatusPaid})
	require.NoError(t, err)
	assert.Len(t, paidList, 2)

	_, err = repo.MarkPaid(ctx, "00000000-0000-0000-0000-000000000000", userID)
	assert.ErrorIs(t, err, payout.ErrNotFound)
	_, err = repo.MarkPaid(ctx, "nao-e-uuid", userID)
	assert.ErrorIs(t, err, payout.ErrNotFound)

	require.NoError(t, database.Close())
	_, err = repo.List(ctx, payout.Filter{})
	assert.Error(t, err)
	_, err = repo.MarkPaid(ctx, o.ID, userID)
	assert.Error(t, err)
}

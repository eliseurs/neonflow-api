//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/partnerarea"
	"neonflow-api/internal/testutil"
)

func TestPartnerAreaRepository_Integration(t *testing.T) {
	db := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewPartnerAreaRepository(db)
	ctx := context.Background()
	q := func(query string, args ...any) string {
		var id string
		require.NoError(t, db.QueryRowContext(ctx, query, args...).Scan(&id))
		return id
	}

	pedro := q(`INSERT INTO partners (name, is_owner, is_driver, referral_code) VALUES ('Pedro', true, true, 'PEDRO2') RETURNING id`)
	outro := q(`INSERT INTO partners (name, is_owner, referral_code) VALUES ('Outro', true, 'OUTRO2') RETURNING id`)
	user := q(`INSERT INTO users (name, email, password_hash, role, partner_id, email_verified_at) VALUES ('Pedro', 'pedro@neonflow.test', 'h', 'parceiro', $1, now()) RETURNING id`, pedro)
	carPedro := q(`INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents, owner_partner_id) VALUES ('AAA1A11', 'Jetta Pedro', 'economy', 'Pedro', 5, 'manual', 20000, $1) RETURNING id`, pedro)
	carOutro := q(`INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents, owner_partner_id) VALUES ('BBB2B22', 'C180 Outro', 'premium', 'Outro', 5, 'automatico', 50000, $1) RETURNING id`, outro)

	start := time.Now().Add(48 * time.Hour)
	booking := func(car, client string, referrer any) string {
		return q(`INSERT INTO bookings (vehicle_id, client_name, start_at, end_at, days, daily_rate_cents, total_cents, expires_at, referred_by_partner_id)
			VALUES ($1, $2, $3, $4, 2, 20000, 40000, now() + interval '1 day', $5) RETURNING id`, car, client, start, start.Add(48*time.Hour), referrer)
	}
	bPedro := booking(carPedro, "Cliente Secreto", nil)
	booking(carOutro, "Outro Cliente", pedro) // carro do outro, indicado pelo Pedro
	_, err := db.ExecContext(ctx, `INSERT INTO partner_payouts (booking_id, partner_id, kind, base_cents, percent_bp, amount_cents) VALUES ($1, $2, 'repasse', 40000, 7000, 28000)`, bPedro, pedro)
	require.NoError(t, err)

	ride := q(`INSERT INTO transfers (client_name, pickup_at, origin, destination, passengers, price_cents, stage, driver_partner_id, driver_cents, notes)
		VALUES ('Passageira Ana', $1, 'POA', 'Gramado', 2, 50000, 'com_motorista', $2, 30000, 'WhatsApp 5554999990000') RETURNING id`, start, pedro)
	q(`INSERT INTO transfers (client_name, pickup_at, origin, destination, passengers, price_cents) VALUES ('De ninguém', $1, 'CXJ', 'Canela', 1, 30000) RETURNING id`, start)

	p, err := repo.Partner(ctx, user)
	require.NoError(t, err)
	assert.Equal(t, pedro, p.ID)
	assert.Equal(t, []string{"proprietario", "motorista"}, p.Kinds)
	assert.Equal(t, "PEDRO2", p.ReferralCode)

	vehicles, err := repo.Vehicles(ctx, pedro)
	require.NoError(t, err)
	require.Len(t, vehicles, 1, "só os carros do próprio parceiro")
	assert.Equal(t, "Jetta Pedro", vehicles[0].Name)

	bookings, err := repo.Bookings(ctx, pedro)
	require.NoError(t, err)
	require.Len(t, bookings, 1)
	require.NotNil(t, bookings[0].Payout)
	assert.Equal(t, int64(28000), *bookings[0].Payout.AmountCents)
	assert.Equal(t, 7000, *bookings[0].Payout.PercentBP)
	raw, err := json.Marshal(bookings)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "Cliente Secreto", "a reserva não leva o cliente (RN-07.9)")

	rides, err := repo.Rides(ctx, pedro)
	require.NoError(t, err)
	require.Len(t, rides, 1)
	assert.Equal(t, ride, rides[0].ID)
	assert.Equal(t, "Passageira Ana", rides[0].ClientName)
	assert.Nil(t, rides[0].Payout, "pagamento ainda não gerado")
	raw, err = json.Marshal(rides)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "5554999990000", "observações e contato ficam com a central")

	refs, err := repo.Referrals(ctx, pedro)
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "aluguel", refs[0].Kind)
	raw, err = json.Marshal(refs)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "Outro Cliente")

	// Parceiro desativado, conta de outro papel ou id inválido: sem área.
	_, err = db.ExecContext(ctx, `UPDATE partners SET active = false WHERE id = $1`, pedro)
	require.NoError(t, err)
	_, err = repo.Partner(ctx, user)
	assert.ErrorIs(t, err, partnerarea.ErrNoPartner)
	_, err = repo.Partner(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, partnerarea.ErrNoPartner)

	require.NoError(t, db.Close())
	_, err = repo.Partner(ctx, user)
	assert.Error(t, err)
	_, err = repo.Vehicles(ctx, pedro)
	assert.Error(t, err)
}

//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/partnerarea"
	"neonflow-api/internal/domain/partnerarea/service"
	"neonflow-api/internal/domain/transfer"
	transferRepo "neonflow-api/internal/domain/transfer/repository"
	transferService "neonflow-api/internal/domain/transfer/service"
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

// Oferta aberta (PEN-74): dois motoristas aceitam ao mesmo tempo e só um leva; quem levou pode desistir.
func TestOpenOffer_Integration(t *testing.T) {
	db := testutil.MigratedDB(t, "../../../../sql/migrations")
	ctx := context.Background()
	q := func(query string, args ...any) string {
		var id string
		require.NoError(t, db.QueryRowContext(ctx, query, args...).Scan(&id))
		return id
	}
	repo := NewPartnerAreaRepository(db)
	svc := service.NewPartnerAreaService(repo, transferService.NewTransferService(transferRepo.NewTransferRepository(db)))

	type driver struct{ partner, user string }
	drivers := make([]driver, 2)
	for i, name := range []string{"Mario", "Luiz"} {
		p := q(`INSERT INTO partners (name, is_driver, referral_code) VALUES ($1, true, $2) RETURNING id`, name, "DRV"+strings.ToUpper(name[:3])+"2")
		u := q(`INSERT INTO users (name, email, password_hash, role, partner_id, email_verified_at) VALUES ($1, $2, 'h', 'parceiro', $3, now()) RETURNING id`, name, name+"@neonflow.test", p)
		drivers[i] = driver{p, u}
	}
	car := q(`INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents, owner_partner_id) VALUES ('CCC3C33', 'Spin Mario', 'economy', 'Mario', 7, 'manual', 15000, $1) RETURNING id`, drivers[0].partner)
	other := q(`INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents) VALUES ('DDD4D44', 'Da central', 'economy', 'Central', 5, 'manual', 15000) RETURNING id`)
	pickup := time.Now().Add(48 * time.Hour)
	ride := q(`INSERT INTO transfers (client_name, pickup_at, origin, destination, passengers, price_cents, offer_cents) VALUES ('Ana', $1, 'POA', 'Gramado', 2, 50000, 30000) RETURNING id`, pickup)
	q(`INSERT INTO transfers (client_name, pickup_at, origin, destination, passengers, price_cents) VALUES ('Sem valor', $1, 'CXJ', 'Canela', 1, 30000) RETURNING id`, pickup)

	offers, err := repo.Offers(ctx)
	require.NoError(t, err)
	require.Len(t, offers, 1, "só a corrida com o valor da central entra na oferta")
	assert.Equal(t, int64(30000), offers[0].OfferCents)
	assert.Equal(t, "Ana", offers[0].ClientName)

	owns, err := repo.OwnsVehicle(ctx, drivers[0].partner, car)
	require.NoError(t, err)
	assert.True(t, owns)
	owns, err = repo.OwnsVehicle(ctx, drivers[0].partner, other)
	require.NoError(t, err)
	assert.False(t, owns)
	owns, err = repo.OwnsVehicle(ctx, drivers[0].partner, "nao-e-uuid")
	require.NoError(t, err)
	assert.False(t, owns)
	assert.ErrorIs(t, svc.Accept(ctx, drivers[0].user, ride, partnerarea.AcceptRequest{VehicleID: other}), partnerarea.ErrNotYourVehicle)

	// Os dois ao mesmo tempo.
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i, d := range drivers {
		wg.Add(1)
		go func(d driver, vehicle string) {
			defer wg.Done()
			errs <- svc.Accept(ctx, d.user, ride, partnerarea.AcceptRequest{VehicleID: vehicle})
		}(d, map[int]string{0: car, 1: ""}[i])
	}
	wg.Wait()
	close(errs)
	var ok, lost int
	for err := range errs {
		if err == nil {
			ok++
		} else {
			assert.ErrorIs(t, err, transfer.ErrWrongStage)
			lost++
		}
	}
	assert.Equal(t, 1, ok, "só um leva")
	assert.Equal(t, 1, lost)

	var winner string
	var driverCents int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT driver_partner_id, driver_cents FROM transfers WHERE id = $1`, ride).Scan(&winner, &driverCents))
	assert.Equal(t, int64(30000), driverCents, "recebe o valor oferecido")
	offers, err = repo.Offers(ctx)
	require.NoError(t, err)
	assert.Empty(t, offers, "aceita, sai da oferta")

	loser := drivers[0]
	if winner == drivers[0].partner {
		loser = drivers[1]
	}
	assert.ErrorIs(t, svc.Release(ctx, loser.user, ride), partnerarea.ErrNotYourRide, "quem não levou não desiste")
	winnerUser := drivers[0].user
	if winner == drivers[1].partner {
		winnerUser = drivers[1].user
	}
	require.NoError(t, svc.Release(ctx, winnerUser, ride))
	offers, err = repo.Offers(ctx)
	require.NoError(t, err)
	require.Len(t, offers, 1, "desistência devolve a corrida à oferta com o mesmo valor")
	assert.Equal(t, int64(30000), offers[0].OfferCents)

	var actions []string
	rows, err := db.QueryContext(ctx, `SELECT action FROM transfer_events WHERE transfer_id = $1 ORDER BY created_at, id`, ride)
	require.NoError(t, err)
	for rows.Next() {
		var a string
		require.NoError(t, rows.Scan(&a))
		actions = append(actions, a)
	}
	require.NoError(t, rows.Close())
	assert.Equal(t, []string{"aceito", "desistiu"}, actions[len(actions)-2:], "o histórico da central mostra o aceite e a desistência")
}

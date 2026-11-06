//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/payout"
	payoutRepo "neonflow-api/internal/domain/payout/repository"
	"neonflow-api/internal/domain/transfer"
	"neonflow-api/internal/pkg/timezone"
	"neonflow-api/internal/testutil"
)

func TestTransferRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewTransferRepository(database)
	payouts := payoutRepo.NewPayoutRepository(database)
	ctx := context.Background()

	var userID, driverID, otherID, inactiveID, hotelID string
	insert := func(dest *string, query string) {
		t.Helper()
		require.NoError(t, database.QueryRowContext(ctx, query).Scan(dest))
	}
	insert(&userID, `INSERT INTO users (name, email, password_hash, role) VALUES ('Central Teste', 'central@neonflow.test', 'h', 'central') RETURNING id`)
	insert(&driverID, `INSERT INTO partners (name, is_driver, is_owner, referral_code) VALUES ('Motorista Teste', true, true, 'DRV234') RETURNING id`)
	insert(&otherID, `INSERT INTO partners (name, is_referrer, referral_code) VALUES ('Só Indicador', true, 'REF234') RETURNING id`)
	insert(&inactiveID, `INSERT INTO partners (name, is_driver, referral_code, active) VALUES ('Inativo', true, 'OFF234', false) RETURNING id`)
	insert(&hotelID, `INSERT INTO partners (name, is_referrer, referral_code, commission_bp) VALUES ('Hotel Teste', true, 'HTL234', 1000) RETURNING id`)

	pickup := time.Date(2026, 12, 10, 8, 30, 0, 0, timezone.Location)
	tr, err := repo.Create(ctx, transfer.NewTransfer{ClientName: "Cliente Teste", PickupAt: pickup, Origin: "Aeroporto POA",
		Destination: "Gramado", Flight: "LA3090", Passengers: 2, PriceCents: 45000, ReferredBy: hotelID, CreatedBy: userID})
	require.NoError(t, err)
	assert.Equal(t, transfer.StageWaiting, tr.Stage)
	assert.True(t, pickup.Equal(tr.PickupAt))
	assert.Equal(t, "America/Sao_Paulo", tr.PickupAt.Location().String())
	assert.Equal(t, hotelID, *tr.ReferredByPartnerID)

	// Quem pode dirigir: motoristas e proprietários ativos (o motorista nem sempre é o dono do carro).
	var ownerID, vehicleID string
	insert(&ownerID, `INSERT INTO partners (name, is_owner, referral_code, whatsapp) VALUES ('Dono Teste', true, 'OWN234', '5554999990000') RETURNING id`)
	require.NoError(t, database.QueryRowContext(ctx, `INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents, owner_partner_id)
		VALUES ('TST1A23', 'Sedã Teste', 'premium', 'Dono Teste', 5, 'automatico', 10000, $1) RETURNING id`, ownerID).Scan(&vehicleID))
	drivers, err := repo.Drivers(ctx, tr.ID)
	require.NoError(t, err)
	require.Len(t, drivers, 2, "motorista ativo e proprietário; indicador e inativo ficam fora")
	assert.Equal(t, "Motorista Teste", drivers[0].Name, "motoristas primeiro")
	assert.Equal(t, []string{"motorista", "proprietario"}, drivers[0].Kinds)
	assert.Equal(t, "Dono Teste", drivers[1].Name)
	require.Len(t, drivers[1].Vehicles, 1)
	assert.Equal(t, "Sedã Teste", drivers[1].Vehicles[0].Name)
	assert.Empty(t, drivers[1].SameDay)

	// Sem valor, não há aceite; o valor só aumenta (RN-02.22).
	_, err = repo.Accept(ctx, tr.ID, userID, transfer.AcceptRequest{DriverPartnerID: driverID})
	assert.ErrorIs(t, err, transfer.ErrNoOffer)
	offered, err := repo.Offer(ctx, tr.ID, userID, 20000)
	require.NoError(t, err)
	assert.Equal(t, int64(20000), *offered.OfferCents)
	_, err = repo.Offer(ctx, tr.ID, userID, 15000)
	assert.ErrorIs(t, err, transfer.ErrOfferDecreased)
	_, err = repo.Offer(ctx, tr.ID, userID, 25000)
	require.NoError(t, err)

	// Avisos pelo WhatsApp ficam no histórico; só quem pode dirigir.
	_, err = repo.Notify(ctx, tr.ID, userID, ownerID)
	require.NoError(t, err)
	for _, bad := range []string{otherID, "nao-e-uuid"} {
		_, err = repo.Notify(ctx, tr.ID, userID, bad)
		assert.ErrorIs(t, err, transfer.ErrDriverNotFound, bad)
	}

	for _, bad := range []string{otherID, inactiveID, "00000000-0000-0000-0000-000000000000"} {
		_, err = repo.Accept(ctx, tr.ID, userID, transfer.AcceptRequest{DriverPartnerID: bad})
		assert.ErrorIs(t, err, transfer.ErrDriverNotFound, bad)
	}
	_, err = repo.Accept(ctx, tr.ID, userID, transfer.AcceptRequest{DriverPartnerID: driverID, VehicleID: "00000000-0000-0000-0000-000000000000"})
	assert.ErrorIs(t, err, transfer.ErrVehicleNotFound)

	// O motorista aceita dirigindo o carro de outro dono.
	accepted, err := repo.Accept(ctx, tr.ID, userID, transfer.AcceptRequest{DriverPartnerID: driverID, VehicleID: vehicleID})
	require.NoError(t, err)
	assert.Equal(t, transfer.StageAssigned, accepted.Stage)
	assert.Equal(t, "Motorista Teste", *accepted.DriverName)
	assert.Equal(t, "Sedã Teste", *accepted.VehicleName)
	assert.Equal(t, int64(25000), *accepted.DriverCents)
	assert.Nil(t, accepted.OfferCents)
	_, err = repo.Accept(ctx, tr.ID, userID, transfer.AcceptRequest{DriverPartnerID: ownerID})
	assert.ErrorIs(t, err, transfer.ErrWrongStage, "o segundo aceite não leva")

	// Outra corrida no mesmo dia aparece para o motorista.
	sameDay, err := repo.Create(ctx, transfer.NewTransfer{ClientName: "Mesmo Dia", PickupAt: pickup.Add(4 * time.Hour), Origin: "A",
		Destination: "B", Passengers: 1, PriceCents: 10000, CreatedBy: userID})
	require.NoError(t, err)
	drivers, err = repo.Drivers(ctx, sameDay.ID)
	require.NoError(t, err)
	require.Len(t, drivers[0].SameDay, 1)
	assert.True(t, pickup.Equal(drivers[0].SameDay[0]))

	// Desistência: volta para aguardando, sem motorista nem carro, com o mesmo valor (RN-02.31).
	released, err := repo.Release(ctx, tr.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, transfer.StageWaiting, released.Stage)
	assert.Nil(t, released.DriverPartnerID)
	assert.Nil(t, released.VehicleID)
	assert.Equal(t, int64(25000), *released.OfferCents)
	_, err = repo.Release(ctx, tr.ID, userID)
	assert.ErrorIs(t, err, transfer.ErrWrongStage)
	_, err = repo.Start(ctx, tr.ID, userID)
	assert.ErrorIs(t, err, transfer.ErrWrongStage)
	// Aceite com valor maior informado na hora.
	higher := int64(25000)
	_, err = repo.Accept(ctx, tr.ID, userID, transfer.AcceptRequest{DriverPartnerID: driverID, OfferCents: &higher})
	require.NoError(t, err)

	// Pagamento, início e conclusão; concluir exige pagamento.
	paid, err := repo.SetPaid(ctx, tr.ID, userID, true)
	require.NoError(t, err)
	require.NotNil(t, paid.PaidAt)
	unpaid, err := repo.SetPaid(ctx, tr.ID, userID, false)
	require.NoError(t, err)
	assert.Nil(t, unpaid.PaidAt)
	_, err = repo.Start(ctx, tr.ID, userID)
	require.NoError(t, err)
	_, err = repo.Complete(ctx, tr.ID, userID, 5000)
	assert.ErrorIs(t, err, transfer.ErrNotPaid)
	_, err = repo.Cancel(ctx, tr.ID, userID, 0, false)
	assert.ErrorIs(t, err, transfer.ErrWrongStage, "em andamento não cancela")
	_, err = repo.SetPaid(ctx, tr.ID, userID, true)
	require.NoError(t, err)
	done, err := repo.Complete(ctx, tr.ID, userID, 5000)
	require.NoError(t, err)
	assert.Equal(t, transfer.StageDone, done.Stage)
	assert.Equal(t, int64(5000), done.ExtraWaitCents)
	_, err = repo.SetPaid(ctx, tr.ID, userID, false)
	assert.ErrorIs(t, err, transfer.ErrWrongStage)

	// Motorista: oferta + espera (R$ 300); indicador: 10% do preço (R$ 45).
	list, err := payouts.List(ctx, payout.Filter{})
	require.NoError(t, err)
	byKind := map[string]payout.Payout{}
	for _, p := range list {
		byKind[p.Kind] = p
	}
	require.Len(t, byKind, 2)
	d := byKind[payout.KindDriver]
	assert.Equal(t, tr.ID, *d.TransferID)
	assert.Nil(t, d.BookingID)
	assert.Equal(t, "Aeroporto POA → Gramado", d.Description)
	assert.Equal(t, int64(30000), *d.AmountCents)
	assert.Nil(t, d.Percent)
	assert.Equal(t, int64(4500), *byKind[payout.KindCommission].AmountCents)
	paidDriver, err := payouts.MarkPaid(ctx, d.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, int64(30000), *paidDriver.AmountCents, "o motorista mantém o valor")

	events, err := repo.Events(ctx, tr.ID)
	require.NoError(t, err)
	actions := []string{}
	for _, e := range events {
		actions = append(actions, e.Action)
	}
	assert.Equal(t, []string{"criado", "valor", "valor_aumentado", "avisado", "aceito", "desistiu", "aceito",
		"pago", "pagamento_desfeito", "iniciado", "pago", "concluido"}, actions)
	assert.Equal(t, "Central Teste", *events[0].UserName)
	assert.Equal(t, "R$ 200,00", events[1].Detail)
	assert.Equal(t, "R$ 200,00 → R$ 250,00", events[2].Detail)
	assert.Equal(t, "Dono Teste", events[3].Detail)
	assert.Equal(t, "Motorista Teste · R$ 250,00", events[4].Detail)

	// Cancelamento com multa e voo cancelado; lista; não encontrados.
	other, err := repo.Create(ctx, transfer.NewTransfer{ClientName: "Outro", PickupAt: pickup.Add(48 * time.Hour), Origin: "A",
		Destination: "B", Passengers: 1, PriceCents: 10000, CreatedBy: userID})
	require.NoError(t, err)
	cancelled, err := repo.Cancel(ctx, other.ID, userID, 5000, true)
	require.NoError(t, err)
	assert.Equal(t, transfer.StageCancelled, cancelled.Stage)
	assert.Equal(t, int64(5000), cancelled.CancelFeeCents)
	assert.True(t, cancelled.FlightCancelled)
	all, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, all, 3)
	assert.Equal(t, other.ID, all[0].ID, "mais recentes primeiro")

	_, err = repo.Create(ctx, transfer.NewTransfer{ClientName: "X", PickupAt: pickup, Origin: "A", Destination: "B",
		Passengers: 1, PriceCents: 1, ReferredBy: "00000000-0000-0000-0000-000000000000"})
	assert.ErrorIs(t, err, transfer.ErrReferrerNotFound)
	for _, id := range []string{"00000000-0000-0000-0000-000000000000", "nao-e-uuid"} {
		_, err = repo.FindByID(ctx, id)
		assert.ErrorIs(t, err, transfer.ErrNotFound)
		_, err = repo.Events(ctx, id)
		assert.ErrorIs(t, err, transfer.ErrNotFound)
		_, err = repo.Start(ctx, id, userID)
		assert.ErrorIs(t, err, transfer.ErrNotFound)
		_, err = repo.Drivers(ctx, id)
		assert.ErrorIs(t, err, transfer.ErrNotFound)
	}
	_, err = repo.Notify(ctx, other.ID, userID, driverID)
	assert.ErrorIs(t, err, transfer.ErrWrongStage, "cancelado")
	_, err = repo.Offer(ctx, other.ID, userID, 1)
	assert.ErrorIs(t, err, transfer.ErrWrongStage)

	require.NoError(t, database.Close())
	_, err = repo.List(ctx)
	assert.Error(t, err)
	_, err = repo.Create(ctx, transfer.NewTransfer{ClientName: "X", PickupAt: pickup, Origin: "A", Destination: "B", Passengers: 1, PriceCents: 1})
	assert.Error(t, err)
}

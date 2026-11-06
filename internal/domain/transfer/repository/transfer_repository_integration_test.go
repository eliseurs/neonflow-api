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

	// Sem oferta, não há aceite; oferta reaberta não diminui (RN-02.22).
	_, err = repo.Accept(ctx, tr.ID, userID, driverID)
	assert.ErrorIs(t, err, transfer.ErrNoOffer)
	expires := time.Now().Add(30 * time.Minute)
	offered, err := repo.Offer(ctx, tr.ID, userID, 20000, expires)
	require.NoError(t, err)
	assert.Equal(t, int64(20000), *offered.OfferCents)
	_, err = repo.Offer(ctx, tr.ID, userID, 15000, expires)
	assert.ErrorIs(t, err, transfer.ErrOfferDecreased)
	_, err = repo.Offer(ctx, tr.ID, userID, 25000, expires)
	require.NoError(t, err)

	// Só motorista ativo aceita; quem aceita leva o valor da oferta.
	for _, bad := range []string{otherID, inactiveID, "00000000-0000-0000-0000-000000000000", "nao-e-uuid"} {
		_, err = repo.Accept(ctx, tr.ID, userID, bad)
		assert.ErrorIs(t, err, transfer.ErrDriverNotFound, bad)
	}
	accepted, err := repo.Accept(ctx, tr.ID, userID, driverID)
	require.NoError(t, err)
	assert.Equal(t, transfer.StageAssigned, accepted.Stage)
	assert.Equal(t, "Motorista Teste", *accepted.DriverName)
	assert.Equal(t, int64(25000), *accepted.DriverCents)
	assert.Nil(t, accepted.OfferCents, "a oferta fecha")
	_, err = repo.Accept(ctx, tr.ID, userID, driverID)
	assert.ErrorIs(t, err, transfer.ErrWrongStage, "o segundo aceite não leva")

	// Desistência: volta para aguardando, sem motorista (RN-02.31); depois aceita de novo.
	released, err := repo.Release(ctx, tr.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, transfer.StageWaiting, released.Stage)
	assert.Nil(t, released.DriverPartnerID)
	_, err = repo.Release(ctx, tr.ID, userID)
	assert.ErrorIs(t, err, transfer.ErrWrongStage)
	_, err = repo.Start(ctx, tr.ID, userID)
	assert.ErrorIs(t, err, transfer.ErrWrongStage)
	_, err = repo.Offer(ctx, tr.ID, userID, 25000, expires)
	require.NoError(t, err)
	_, err = repo.Accept(ctx, tr.ID, userID, driverID)
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
	assert.Equal(t, []string{"criado", "oferta", "oferta_reaberta", "aceito", "desistiu", "oferta", "aceito",
		"pago", "pagamento_desfeito", "iniciado", "pago", "concluido"}, actions)
	assert.Equal(t, "Central Teste", *events[0].UserName)
	assert.Equal(t, "Motorista Teste", events[4].Detail)

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
	require.Len(t, all, 2)
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
	}

	require.NoError(t, database.Close())
	_, err = repo.List(ctx)
	assert.Error(t, err)
	_, err = repo.Create(ctx, transfer.NewTransfer{ClientName: "X", PickupAt: pickup, Origin: "A", Destination: "B", Passengers: 1, PriceCents: 1})
	assert.Error(t, err)
}

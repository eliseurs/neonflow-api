//go:build integration

package repository

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/booking"
	"neonflow-api/internal/testutil"
)

func seed(t *testing.T, db *sql.DB) (userID, vehicleID string) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('Central', 'central@neonflow.test', 'h', 'central') RETURNING id`).Scan(&userID))
	require.NoError(t, db.QueryRowContext(ctx,
		`INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents)
		 VALUES ('TST1A23', 'Sedã Teste', 'economy', 'Parceiro Teste', 5, 'manual', 10000) RETURNING id`).Scan(&vehicleID))
	return userID, vehicleID
}

func period(t *testing.T, start, end string) booking.Period {
	t.Helper()
	p, err := booking.ParsePeriod(start, end)
	require.NoError(t, err)
	return p
}

func TestBookingRepository_CreateAndLifecycle(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewBookingRepository(database)
	ctx := context.Background()
	userID, vehicleID := seed(t, database)

	b, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Cliente Teste",
		Period: period(t, "2026-12-10", "2026-12-13"), Place: "Gramado", CreatedBy: userID})
	require.NoError(t, err)
	assert.Equal(t, booking.StagePreparacao, b.Stage)
	assert.Equal(t, "2026-12-10", b.StartDate)
	assert.Equal(t, "2026-12-13", b.EndDate)
	assert.Equal(t, 3, b.Days)
	assert.Equal(t, int64(30000), b.TotalCents)
	assert.Equal(t, userID, *b.CreatedBy)
	assert.WithinDuration(t, b.CreatedAt.Add(booking.PaymentWindow), b.ExpiresAt, 0)

	// RN-02.6: mudar a diária do veículo não altera a reserva.
	_, err = database.ExecContext(ctx, `UPDATE vehicles SET daily_rate_cents = 99900 WHERE id = $1`, vehicleID)
	require.NoError(t, err)
	got, err := repo.FindByID(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(30000), got.TotalCents)

	// RN-02.2: sobreposição recusada, datas adjacentes permitidas.
	_, err = repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Outro",
		Period: period(t, "2026-12-12", "2026-12-15"), CreatedBy: userID})
	assert.ErrorIs(t, err, booking.ErrPeriodUnavailable)
	adj, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Outro",
		Period: period(t, "2026-12-13", "2026-12-14"), CreatedBy: userID})
	require.NoError(t, err)

	avail, err := repo.Available(ctx, period(t, "2026-12-11", "2026-12-12"))
	require.NoError(t, err)
	assert.Empty(t, avail)
	avail, err = repo.Available(ctx, period(t, "2026-12-14", "2026-12-20"))
	require.NoError(t, err)
	assert.Len(t, avail, 1)

	withChecks, err := repo.SetChecks(ctx, b.ID, booking.StagePreparacao, map[string]bool{"cnh": true, "payment": true})
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"cnh": true, "payment": true}, withChecks.Checks)
	_, err = repo.SetChecks(ctx, b.ID, booking.StageConfirmada, map[string]bool{"delivery": true})
	assert.ErrorIs(t, err, booking.ErrStageChanged)

	moved, err := repo.Transition(ctx, b.ID, booking.StagePreparacao, booking.StageConfirmada, userID, booking.Record{})
	require.NoError(t, err)
	assert.Equal(t, booking.StageConfirmada, moved.Stage)
	_, err = repo.Transition(ctx, b.ID, booking.StagePreparacao, booking.StageConfirmada, userID, booking.Record{})
	assert.ErrorIs(t, err, booking.ErrStageChanged, "segundo clique não avança de novo")

	// Cancelar libera o período (RN-02.5).
	_, err = repo.Transition(ctx, adj.ID, booking.StagePreparacao, booking.StageCancelada, userID, booking.Record{})
	require.NoError(t, err)
	_, err = repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Terceiro",
		Period: period(t, "2026-12-13", "2026-12-14"), CreatedBy: userID})
	require.NoError(t, err)

	events, err := repo.Events(ctx, b.ID)
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Nil(t, events[0].FromStage)
	assert.Equal(t, booking.StagePreparacao, events[0].ToStage)
	assert.Equal(t, booking.StagePreparacao, *events[1].FromStage)
	assert.Equal(t, userID, *events[1].UserID)

	all, err := repo.List(ctx, "")
	require.NoError(t, err)
	assert.Len(t, all, 3)
	byVehicle, err := repo.List(ctx, vehicleID)
	require.NoError(t, err)
	assert.Len(t, byVehicle, 3)
	none, err := repo.List(ctx, "00000000-0000-0000-0000-000000000000")
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestBookingRepository_Errors(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewBookingRepository(database)
	ctx := context.Background()
	userID, vehicleID := seed(t, database)
	p := period(t, "2026-12-10", "2026-12-11")

	_, err := repo.Create(ctx, booking.NewBooking{VehicleID: "00000000-0000-0000-0000-000000000000", ClientName: "X", Period: p, CreatedBy: userID})
	assert.ErrorIs(t, err, booking.ErrVehicleNotFound)
	_, err = repo.Create(ctx, booking.NewBooking{VehicleID: "nao-e-uuid", ClientName: "X", Period: p, CreatedBy: userID})
	assert.ErrorIs(t, err, booking.ErrVehicleNotFound)

	// RN-01.1: veículo em manutenção não aceita reserva e some da disponibilidade.
	_, err = database.ExecContext(ctx, `UPDATE vehicles SET in_maintenance = true WHERE id = $1`, vehicleID)
	require.NoError(t, err)
	_, err = repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "X", Period: p, CreatedBy: userID})
	assert.ErrorIs(t, err, booking.ErrVehicleInMaintenance)
	avail, err := repo.Available(ctx, p)
	require.NoError(t, err)
	assert.Empty(t, avail)

	_, err = repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, booking.ErrBookingNotFound)
	_, err = repo.FindByID(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, booking.ErrBookingNotFound)
	_, err = repo.Events(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, booking.ErrBookingNotFound)
	_, err = repo.Transition(ctx, "00000000-0000-0000-0000-000000000000", "preparacao", "confirmada", userID, booking.Record{})
	assert.ErrorIs(t, err, booking.ErrStageChanged)

	require.NoError(t, database.Close())
	_, err = repo.List(ctx, "")
	assert.Error(t, err)
	_, err = repo.FindByID(ctx, "x")
	assert.Error(t, err)
	_, err = repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "X", Period: p})
	assert.Error(t, err)
	_, err = repo.SetChecks(ctx, "x", "preparacao", map[string]bool{"cnh": true})
	assert.Error(t, err)
	_, err = repo.Available(ctx, p)
	assert.Error(t, err)
}

// RN-02.11: pré-reserva vencida e não paga é cancelada e libera o período; paga não expira.
func TestBookingRepository_Expiration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewBookingRepository(database)
	ctx := context.Background()
	userID, vehicleID := seed(t, database)
	p := period(t, "2026-12-10", "2026-12-12")
	later := period(t, "2026-12-20", "2026-12-22")

	unpaid, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Sem pagamento", Period: p, CreatedBy: userID})
	require.NoError(t, err)
	paid, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Pago", Period: later, CreatedBy: userID})
	require.NoError(t, err)
	_, err = repo.SetChecks(ctx, paid.ID, booking.StagePreparacao, map[string]bool{"payment": true})
	require.NoError(t, err)

	_, err = database.ExecContext(ctx, `UPDATE bookings SET expires_at = now() - interval '1 minute'`)
	require.NoError(t, err)

	again, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Novo", Period: p, CreatedBy: userID})
	require.NoError(t, err, "o período da pré-reserva vencida fica livre")
	assert.Equal(t, booking.StagePreparacao, again.Stage)

	got, err := repo.FindByID(ctx, unpaid.ID)
	require.NoError(t, err)
	assert.Equal(t, booking.StageCancelada, got.Stage)
	events, err := repo.Events(ctx, unpaid.ID)
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Nil(t, events[1].UserID, "expiração é automática")

	got, err = repo.FindByID(ctx, paid.ID)
	require.NoError(t, err)
	assert.Equal(t, booking.StagePreparacao, got.Stage)
}

// Cenário do plano: duas centrais reservam o mesmo período ao mesmo tempo; só uma consegue (PEN-24).
func TestBookingRepository_ConcurrentCreate(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewBookingRepository(database)
	ctx := context.Background()
	userID, vehicleID := seed(t, database)
	p := period(t, "2026-12-10", "2026-12-13")

	const attempts = 8
	var wg sync.WaitGroup
	errs := make(chan error, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Concorrente", Period: p, CreatedBy: userID})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	ok := 0
	for err := range errs {
		if err == nil {
			ok++
			continue
		}
		assert.ErrorIs(t, err, booking.ErrPeriodUnavailable)
	}
	assert.Equal(t, 1, ok)
}

func TestBookingRepository_ReturnChecklist(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewBookingRepository(database)
	ctx := context.Background()
	userID, vehicleID := seed(t, database)

	// Lista inicial do fundador (RN-01.6, q56 e q80), em ordem.
	keys, err := repo.ReturnChecklist(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"combustivel", "estetica", "oleo", "agua", "pastilhas", "pneus", "limpeza",
		"documentos", "fotos", "quilometragem"}, keys)
	_, err = database.ExecContext(ctx, `UPDATE checklist_items SET active = false WHERE key = 'fotos'`)
	require.NoError(t, err)
	keys, err = repo.ReturnChecklist(ctx)
	require.NoError(t, err)
	assert.NotContains(t, keys, "fotos")

	first, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Primeiro",
		Period: period(t, "2026-12-10", "2026-12-12"), CreatedBy: userID})
	require.NoError(t, err)
	second, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Segundo",
		Period: period(t, "2026-12-12", "2026-12-14"), CreatedBy: userID})
	require.NoError(t, err)

	for _, step := range [][2]string{
		{booking.StagePreparacao, booking.StageConfirmada},
		{booking.StageConfirmada, booking.StageEmLocacao},
		{booking.StageEmLocacao, booking.StageDevolvida},
	} {
		_, err := repo.Transition(ctx, first.ID, step[0], step[1], userID, booking.Record{})
		require.NoError(t, err)
	}
	_, err = repo.Transition(ctx, second.ID, booking.StagePreparacao, booking.StageConfirmada, userID, booking.Record{})
	require.NoError(t, err)

	// O carro ainda com o primeiro cliente não sai para o segundo.
	_, err = database.ExecContext(ctx, `UPDATE bookings SET stage = 'em_locacao' WHERE id = $1`, first.ID)
	require.NoError(t, err)
	_, err = repo.Transition(ctx, second.ID, booking.StageConfirmada, booking.StageEmLocacao, userID, booking.Record{})
	assert.ErrorIs(t, err, booking.ErrVehicleInUse)
	_, err = database.ExecContext(ctx, `UPDATE bookings SET stage = 'devolvida' WHERE id = $1`, first.ID)
	require.NoError(t, err)

	// RN-01.6 e RN-02.25: o carro só sai de novo com o checklist da devolução anterior aprovado.
	_, err = repo.Transition(ctx, second.ID, booking.StageConfirmada, booking.StageEmLocacao, userID, booking.Record{})
	assert.ErrorIs(t, err, booking.ErrAwaitingChecklist)
	still, err := repo.FindByID(ctx, second.ID)
	require.NoError(t, err)
	assert.Equal(t, booking.StageConfirmada, still.Stage)

	_, err = repo.Transition(ctx, first.ID, booking.StageDevolvida, booking.StageEncerrada, userID, booking.Record{})
	require.NoError(t, err)
	out, err := repo.Transition(ctx, second.ID, booking.StageConfirmada, booking.StageEmLocacao, userID, booking.Record{})
	require.NoError(t, err)
	assert.Equal(t, booking.StageEmLocacao, out.Stage)

	require.NoError(t, database.Close())
	_, err = repo.ReturnChecklist(ctx)
	assert.Error(t, err)
}

func TestBookingRepository_Hours(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewBookingRepository(database)
	ctx := context.Background()
	userID, vehicleID := seed(t, database)

	// RN-02.24: 25 horas são 2 diárias.
	first, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Primeiro",
		Period: period(t, "2026-12-20T10:00", "2026-12-21T11:00"), CreatedBy: userID})
	require.NoError(t, err)
	assert.Equal(t, 2, first.Days)
	assert.Equal(t, int64(20000), first.TotalCents)
	assert.Equal(t, 10, first.StartAt.Hour(), "devolvido no fuso de Gramado")
	assert.Equal(t, "America/Sao_Paulo", first.StartAt.Location().String())
	assert.Equal(t, "2026-12-20", first.StartDate)
	assert.Equal(t, "2026-12-21", first.EndDate)

	// RN-02.25: retirada na mesma hora da devolução anterior é permitida; uma hora antes, não.
	_, err = repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Sobreposto",
		Period: period(t, "2026-12-21T10:00", "2026-12-22T10:00"), CreatedBy: userID})
	assert.ErrorIs(t, err, booking.ErrPeriodUnavailable)
	same, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Mesmo dia",
		Period: period(t, "2026-12-21T11:00", "2026-12-22T09:00"), CreatedBy: userID})
	require.NoError(t, err)
	assert.Equal(t, 1, same.Days)

	// Disponibilidade por hora: livre antes das 10h do dia 20, ocupado às 10h30.
	avail, err := repo.Available(ctx, period(t, "2026-12-19T10:00", "2026-12-20T10:00"))
	require.NoError(t, err)
	assert.Len(t, avail, 1)
	avail, err = repo.Available(ctx, period(t, "2026-12-20T10:30", "2026-12-20T12:00"))
	require.NoError(t, err)
	assert.Empty(t, avail)

	list, err := repo.List(ctx, vehicleID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, first.ID, list[0].ID, "ordem pela retirada")
}

func TestBookingRepository_ActualTimes(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewBookingRepository(database)
	ctx := context.Background()
	userID, vehicleID := seed(t, database)

	b, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Cliente",
		Period: period(t, "2026-12-10T10:00", "2026-12-12T10:00"), CreatedBy: userID})
	require.NoError(t, err)
	assert.Nil(t, b.DeliveredAt)
	assert.Zero(t, b.ExtraDays)

	_, err = repo.Transition(ctx, b.ID, booking.StagePreparacao, booking.StageConfirmada, userID, booking.Record{})
	require.NoError(t, err)

	delivered := period(t, "2026-12-10T10:20", "2026-12-12T15:00")
	out, err := repo.Transition(ctx, b.ID, booking.StageConfirmada, booking.StageEmLocacao, userID,
		booking.Record{DeliveredAt: &delivered.Start})
	require.NoError(t, err)
	require.NotNil(t, out.DeliveredAt)
	assert.True(t, delivered.Start.Equal(*out.DeliveredAt))
	assert.Equal(t, "America/Sao_Paulo", out.DeliveredAt.Location().String())
	assert.Nil(t, out.ReturnedAt)

	days, cents := 1, int64(10000)
	back, err := repo.Transition(ctx, b.ID, booking.StageEmLocacao, booking.StageDevolvida, userID,
		booking.Record{ReturnedAt: &delivered.End, ExtraDays: &days, ExtraCents: &cents})
	require.NoError(t, err)
	assert.True(t, delivered.Start.Equal(*back.DeliveredAt), "campos nulos mantêm o valor")
	assert.True(t, delivered.End.Equal(*back.ReturnedAt))
	assert.Equal(t, 1, back.ExtraDays)
	assert.Equal(t, int64(10000), back.ExtraCents)
	assert.Equal(t, int64(20000), back.TotalCents, "o total combinado não muda")

	// O banco recusa devolução antes da entrega.
	_, err = database.ExecContext(ctx, `UPDATE bookings SET returned_at = delivered_at - interval '1 hour' WHERE id = $1`, b.ID)
	assert.Error(t, err)
}

func TestBookingRepository_DepositAndKm(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewBookingRepository(database)
	ctx := context.Background()
	userID, vehicleID := seed(t, database)

	// Sem limite cadastrado: km livre por padrão, e "limitado" é recusado.
	free, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Livre",
		Period: period(t, "2026-12-01T10:00", "2026-12-02T10:00"), CreatedBy: userID})
	require.NoError(t, err)
	assert.Nil(t, free.KmLimitPerDay)
	assert.Nil(t, free.DepositCents)
	_, err = repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Limitado",
		Period: period(t, "2026-12-03T10:00", "2026-12-04T10:00"), CreatedBy: userID, KmPlan: booking.KmPlanLimitado})
	assert.ErrorIs(t, err, booking.ErrKmPlanUnavailable)

	_, err = database.ExecContext(ctx, `UPDATE vehicles SET deposit_cents = 150000, km_limit_per_day = 200, extra_km_cents = 150 WHERE id = $1`, vehicleID)
	require.NoError(t, err)

	// Com limite: limitado por padrão, com caução e valores copiados do veículo (RN-06.4, RN-02.18).
	limited, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Padrão",
		Period: period(t, "2026-12-05T10:00", "2026-12-07T10:00"), CreatedBy: userID})
	require.NoError(t, err)
	assert.Equal(t, int64(150000), *limited.DepositCents)
	assert.Equal(t, 200, *limited.KmLimitPerDay)
	assert.Equal(t, int64(150), *limited.ExtraKmCents)

	chosenFree, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Escolheu livre",
		Period: period(t, "2026-12-08T10:00", "2026-12-09T10:00"), CreatedBy: userID, KmPlan: booking.KmPlanLivre})
	require.NoError(t, err)
	assert.Nil(t, chosenFree.KmLimitPerDay)
	assert.Equal(t, int64(150000), *chosenFree.DepositCents, "a caução vale para qualquer plano")

	// Mudar o veículo depois não muda a reserva (como a diária, RN-02.6).
	_, err = database.ExecContext(ctx, `UPDATE vehicles SET km_limit_per_day = 300 WHERE id = $1`, vehicleID)
	require.NoError(t, err)
	got, err := repo.FindByID(ctx, limited.ID)
	require.NoError(t, err)
	assert.Equal(t, 200, *got.KmLimitPerDay)

	// Odômetro e excedente gravados nas transições.
	_, err = repo.Transition(ctx, limited.ID, booking.StagePreparacao, booking.StageConfirmada, userID, booking.Record{})
	require.NoError(t, err)
	start, end, excess, cents := 10000, 10450, 50, int64(7500)
	_, err = repo.Transition(ctx, limited.ID, booking.StageConfirmada, booking.StageEmLocacao, userID, booking.Record{DeliveredKm: &start})
	require.NoError(t, err)
	back, err := repo.Transition(ctx, limited.ID, booking.StageEmLocacao, booking.StageDevolvida, userID,
		booking.Record{ReturnedKm: &end, ExcessKm: &excess, ExcessKmCents: &cents})
	require.NoError(t, err)
	assert.Equal(t, 10000, *back.DeliveredKm)
	assert.Equal(t, 10450, *back.ReturnedKm)
	assert.Equal(t, 50, back.ExcessKm)
	assert.Equal(t, int64(7500), back.ExcessKmCents)

	avail, err := repo.Available(ctx, period(t, "2026-12-20T10:00", "2026-12-21T10:00"))
	require.NoError(t, err)
	require.Len(t, avail, 1)
	assert.Equal(t, 300, *avail[0].KmLimitPerDay, "a disponibilidade traz o plano atual do veículo")
	assert.Equal(t, int64(150000), *avail[0].DepositCents)
}

func TestBookingRepository_Referrer(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewBookingRepository(database)
	ctx := context.Background()
	userID, vehicleID := seed(t, database)
	var partnerID string
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO partners (name, is_referrer, referral_code) VALUES ('Hotel Teste', true, 'HTL234') RETURNING id`).Scan(&partnerID))

	// RN-07.5: a reserva guarda o parceiro que indicou.
	b, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Indicado",
		Period: period(t, "2026-12-10T10:00", "2026-12-11T10:00"), CreatedBy: userID, ReferredBy: partnerID})
	require.NoError(t, err)
	require.NotNil(t, b.ReferredByPartnerID)
	assert.Equal(t, partnerID, *b.ReferredByPartnerID)

	direct, err := repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Direto",
		Period: period(t, "2026-12-12T10:00", "2026-12-13T10:00"), CreatedBy: userID})
	require.NoError(t, err)
	assert.Nil(t, direct.ReferredByPartnerID)

	_, err = repo.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Inexistente",
		Period: period(t, "2026-12-14T10:00", "2026-12-15T10:00"), CreatedBy: userID,
		ReferredBy: "00000000-0000-0000-0000-000000000000"})
	assert.ErrorIs(t, err, booking.ErrReferrerNotFound)
}

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/booking"
	"neonflow-api/internal/domain/fleet"
	"neonflow-api/internal/pkg/timezone"
)

type transition struct{ from, to, user string }

type fakeRepo struct {
	current    booking.Booking
	findErr    error
	created    booking.NewBooking
	checks     map[string]bool
	transition transition
	period     booking.Period
	items      []string
	itemsErr   error
	record     booking.Record
}

func (f *fakeRepo) ReturnChecklist(context.Context) ([]string, error) {
	return f.items, f.itemsErr
}

func (f *fakeRepo) List(context.Context, string) ([]booking.Booking, error) {
	return []booking.Booking{f.current}, nil
}

func (f *fakeRepo) FindByID(context.Context, string) (booking.Booking, error) {
	return f.current, f.findErr
}

func (f *fakeRepo) Events(context.Context, string) ([]booking.Event, error) {
	return []booking.Event{{ToStage: booking.StagePreparacao}}, nil
}

func (f *fakeRepo) Create(_ context.Context, n booking.NewBooking) (booking.Booking, error) {
	f.created = n
	return booking.Booking{ID: "b1", Stage: booking.StagePreparacao}, nil
}

func (f *fakeRepo) SetChecks(_ context.Context, id, stage string, checks map[string]bool) (booking.Booking, error) {
	f.checks = checks
	return booking.Booking{ID: id, Stage: stage, Checks: checks}, nil
}

func (f *fakeRepo) Transition(_ context.Context, id, from, to, user string, rec booking.Record) (booking.Booking, error) {
	f.transition = transition{from, to, user}
	f.record = rec
	return booking.Booking{ID: id, Stage: to}, nil
}

func (f *fakeRepo) Available(_ context.Context, p booking.Period) ([]fleet.Vehicle, error) {
	f.period = p
	return []fleet.Vehicle{{ID: "v1"}}, nil
}

func TestBookingService_Reads(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{current: booking.Booking{ID: "b1"}}
	svc := NewBookingService(repo)

	list, err := svc.List(ctx, "")
	require.NoError(t, err)
	assert.Len(t, list, 1)
	b, err := svc.Get(ctx, "b1")
	require.NoError(t, err)
	assert.Equal(t, "b1", b.ID)
	events, err := svc.Events(ctx, "b1")
	require.NoError(t, err)
	assert.Len(t, events, 1)

	vehicles, err := svc.Available(ctx, "2026-12-10", "2026-12-12")
	require.NoError(t, err)
	assert.Len(t, vehicles, 1)
	assert.Equal(t, 2, repo.period.Days())
	_, err = svc.Available(ctx, "2026-12-12", "2026-12-10")
	assert.ErrorIs(t, err, booking.ErrInvalidPeriod)
}

func TestBookingService_Create(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewBookingService(repo)

	b, err := svc.Create(context.Background(), booking.CreateRequest{VehicleID: "v1", ClientName: "Cliente",
		StartDate: "2026-12-10", EndDate: "2026-12-13"}, "u1")
	require.NoError(t, err)
	assert.Equal(t, booking.StagePreparacao, b.Stage)
	assert.Equal(t, "u1", repo.created.CreatedBy)
	assert.Equal(t, 3, repo.created.Period.Days())

	_, err = svc.Create(context.Background(), booking.CreateRequest{VehicleID: "v1"}, "u1")
	assert.ErrorIs(t, err, booking.ErrClientNameRequired)
}

func TestBookingService_SetChecks(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{current: booking.Booking{ID: "b1", Stage: booking.StagePreparacao}}
	svc := NewBookingService(repo)

	b, err := svc.SetChecks(ctx, "b1", map[string]bool{"cnh": true})
	require.NoError(t, err)
	assert.Equal(t, booking.StagePreparacao, b.Stage)
	assert.Equal(t, map[string]bool{"cnh": true}, repo.checks)

	_, err = svc.SetChecks(ctx, "b1", map[string]bool{"delivery": true})
	assert.ErrorIs(t, err, booking.ErrInvalidCheck)

	repo.findErr = booking.ErrBookingNotFound
	_, err = svc.SetChecks(ctx, "x", map[string]bool{"cnh": true})
	assert.ErrorIs(t, err, booking.ErrBookingNotFound)
}

func TestBookingService_Advance(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{current: booking.Booking{ID: "b1", Stage: booking.StageConfirmada, Checks: map[string]bool{"delivery": true}}}
	svc := NewBookingService(repo)

	b, err := svc.Advance(ctx, "b1", "u1", booking.AdvanceRequest{})
	require.NoError(t, err)
	assert.Equal(t, booking.StageEmLocacao, b.Stage)
	assert.Equal(t, transition{booking.StageConfirmada, booking.StageEmLocacao, "u1"}, repo.transition)

	repo.current.Checks = map[string]bool{}
	_, err = svc.Advance(ctx, "b1", "u1", booking.AdvanceRequest{})
	assert.ErrorIs(t, err, booking.ErrChecksPending)

	repo.findErr = booking.ErrBookingNotFound
	_, err = svc.Advance(ctx, "x", "u1", booking.AdvanceRequest{})
	assert.ErrorIs(t, err, booking.ErrBookingNotFound)
}

func TestBookingService_ActualTimes(t *testing.T) {
	ctx := context.Background()
	gramado := timezone.Location
	now := time.Date(2026, 12, 12, 15, 0, 0, 0, gramado)
	end := time.Date(2026, 12, 12, 10, 0, 0, 0, gramado)
	delivered := time.Date(2026, 12, 10, 9, 40, 0, 0, gramado)
	repo := &fakeRepo{current: booking.Booking{ID: "b1", Stage: booking.StageConfirmada, Checks: map[string]bool{"delivery": true},
		EndAt: end, DailyRateCents: 10000}}
	svc := NewBookingService(repo)
	svc.now = func() time.Time { return now }

	// Entrega sem hora informada: vale agora.
	_, err := svc.Advance(ctx, "b1", "u1", booking.AdvanceRequest{})
	require.NoError(t, err)
	require.NotNil(t, repo.record.DeliveredAt)
	assert.True(t, now.Equal(*repo.record.DeliveredAt))
	assert.Nil(t, repo.record.ReturnedAt)

	// Entrega com a hora real.
	_, err = svc.Advance(ctx, "b1", "u1", booking.AdvanceRequest{At: "2026-12-10T09:40"})
	require.NoError(t, err)
	assert.True(t, delivered.Equal(*repo.record.DeliveredAt))

	_, err = svc.Advance(ctx, "b1", "u1", booking.AdvanceRequest{At: "2026-12-12T16:00"})
	assert.ErrorIs(t, err, booking.ErrFutureTime)
	_, err = svc.Advance(ctx, "b1", "u1", booking.AdvanceRequest{At: "ontem"})
	assert.ErrorIs(t, err, booking.ErrInvalidDate)

	// Devolução às 15h, 5 horas depois do combinado: passou da tolerância, 1 diária extra (RN-02.19).
	repo.current = booking.Booking{ID: "b1", Stage: booking.StageEmLocacao, Checks: map[string]bool{"return": true},
		EndAt: end, DailyRateCents: 10000, DeliveredAt: &delivered}
	_, err = svc.Advance(ctx, "b1", "u1", booking.AdvanceRequest{})
	require.NoError(t, err)
	assert.True(t, now.Equal(*repo.record.ReturnedAt))
	assert.Equal(t, 1, *repo.record.ExtraDays)
	assert.Equal(t, int64(10000), *repo.record.ExtraCents)

	// Dentro da tolerância de 1 hora: sem cobrança.
	_, err = svc.Advance(ctx, "b1", "u1", booking.AdvanceRequest{At: "2026-12-12T11:00"})
	require.NoError(t, err)
	assert.Equal(t, 0, *repo.record.ExtraDays)
	assert.Equal(t, int64(0), *repo.record.ExtraCents)

	_, err = svc.Advance(ctx, "b1", "u1", booking.AdvanceRequest{At: "2026-12-10T09:00"})
	assert.ErrorIs(t, err, booking.ErrReturnBeforeDelivery)

	// Outras etapas não gravam hora.
	repo.current = booking.Booking{ID: "b1", Stage: booking.StagePreparacao,
		Checks: map[string]bool{"cnh": true, "terms": true, "deposit": true, "payment": true}}
	_, err = svc.Advance(ctx, "b1", "u1", booking.AdvanceRequest{At: "2026-12-12T11:00"})
	require.NoError(t, err)
	assert.Equal(t, booking.Record{}, repo.record)
}

func TestBookingService_Km(t *testing.T) {
	ctx := context.Background()
	gramado := timezone.Location
	now := time.Date(2026, 12, 12, 10, 30, 0, 0, gramado)
	end := time.Date(2026, 12, 12, 10, 0, 0, 0, gramado)
	limit, extraKm := 200, int64(150)
	repo := &fakeRepo{current: booking.Booking{ID: "b1", Stage: booking.StageConfirmada, Checks: map[string]bool{"delivery": true},
		EndAt: end, Days: 2, DailyRateCents: 10000, KmLimitPerDay: &limit, ExtraKmCents: &extraKm}}
	svc := NewBookingService(repo)
	svc.now = func() time.Time { return now }
	km := func(v int) booking.AdvanceRequest { return booking.AdvanceRequest{Km: &v} }

	// Plano limitado exige o odômetro (RN-02.18).
	_, err := svc.Advance(ctx, "b1", "u1", booking.AdvanceRequest{})
	assert.ErrorIs(t, err, booking.ErrKmRequired)
	_, err = svc.Advance(ctx, "b1", "u1", km(-1))
	assert.ErrorIs(t, err, booking.ErrInvalidKm)
	_, err = svc.Advance(ctx, "b1", "u1", km(10000))
	require.NoError(t, err)
	assert.Equal(t, 10000, *repo.record.DeliveredKm)

	delivered := 10000
	repo.current.Stage, repo.current.Checks, repo.current.DeliveredKm = booking.StageEmLocacao, map[string]bool{"return": true}, &delivered
	_, err = svc.Advance(ctx, "b1", "u1", km(9999))
	assert.ErrorIs(t, err, booking.ErrKmBeforeDelivery)

	// 2 diárias × 200 km = 400 km de franquia; rodou 450 → 50 km × R$ 1,50.
	_, err = svc.Advance(ctx, "b1", "u1", km(10450))
	require.NoError(t, err)
	assert.Equal(t, 10450, *repo.record.ReturnedKm)
	assert.Equal(t, 50, *repo.record.ExcessKm)
	assert.Equal(t, int64(7500), *repo.record.ExcessKmCents)

	// Entregue antes do odômetro existir: guarda a devolução, sem excedente.
	repo.current.DeliveredKm = nil
	_, err = svc.Advance(ctx, "b1", "u1", km(10450))
	require.NoError(t, err)
	assert.Nil(t, repo.record.ExcessKm)

	// Km livre: odômetro opcional e sem excedente.
	repo.current.KmLimitPerDay, repo.current.ExtraKmCents = nil, nil
	_, err = svc.Advance(ctx, "b1", "u1", booking.AdvanceRequest{})
	require.NoError(t, err)
	assert.Nil(t, repo.record.ReturnedKm)
	assert.Nil(t, repo.record.ExcessKm)
}

func TestBookingService_ReturnChecklist(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{
		current: booking.Booking{ID: "b1", Stage: booking.StageDevolvida, Checks: map[string]bool{"issues": true}},
		items:   []string{"combustivel", "pneus"},
	}
	svc := NewBookingService(repo)

	// RN-01.6: os itens do checklist são aceitos e exigidos em "devolvida".
	_, err := svc.SetChecks(ctx, "b1", map[string]bool{"pneus": true})
	require.NoError(t, err)
	_, err = svc.Advance(ctx, "b1", "u1", booking.AdvanceRequest{})
	assert.ErrorIs(t, err, booking.ErrChecksPending)

	repo.current.Checks = map[string]bool{"issues": true, "combustivel": true, "pneus": true}
	b, err := svc.Advance(ctx, "b1", "u1", booking.AdvanceRequest{})
	require.NoError(t, err)
	assert.Equal(t, booking.StageEncerrada, b.Stage)

	repo.itemsErr = errors.New("db")
	_, err = svc.SetChecks(ctx, "b1", map[string]bool{"pneus": true})
	assert.Error(t, err)
	_, err = svc.Advance(ctx, "b1", "u1", booking.AdvanceRequest{})
	assert.Error(t, err)
}

func TestBookingService_Cancel(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{current: booking.Booking{ID: "b1", Stage: booking.StageConfirmada}}
	svc := NewBookingService(repo)

	b, err := svc.Cancel(ctx, "b1", "u1")
	require.NoError(t, err)
	assert.Equal(t, booking.StageCancelada, b.Stage)
	assert.Equal(t, transition{booking.StageConfirmada, booking.StageCancelada, "u1"}, repo.transition)

	repo.current.Stage = booking.StageEmLocacao
	_, err = svc.Cancel(ctx, "b1", "u1")
	assert.ErrorIs(t, err, booking.ErrNotCancelable)

	repo.findErr = booking.ErrBookingNotFound
	_, err = svc.Cancel(ctx, "x", "u1")
	assert.ErrorIs(t, err, booking.ErrBookingNotFound)
}

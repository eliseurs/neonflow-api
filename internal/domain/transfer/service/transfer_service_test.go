package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/transfer"
	"neonflow-api/internal/pkg/timezone"
)

type fakeRepo struct {
	current  transfer.Transfer
	offered  int64
	notified string
	accept   transfer.AcceptRequest
	fee      int64
	flight   bool
	extra    int64
	calls    []string
}

func (f *fakeRepo) List(context.Context) ([]transfer.Transfer, error) {
	return []transfer.Transfer{f.current}, nil
}
func (f *fakeRepo) FindByID(context.Context, string) (transfer.Transfer, error) {
	if f.current.ID == "" {
		return transfer.Transfer{}, transfer.ErrNotFound
	}
	return f.current, nil
}
func (f *fakeRepo) Events(context.Context, string) ([]transfer.Event, error) {
	return []transfer.Event{{Action: "criado"}}, nil
}
func (f *fakeRepo) Create(_ context.Context, n transfer.NewTransfer) (transfer.Transfer, error) {
	return transfer.Transfer{ID: "t1", ClientName: n.ClientName}, nil
}
func (f *fakeRepo) Drivers(context.Context, string) ([]transfer.Driver, error) {
	return []transfer.Driver{{Name: "Motorista"}}, nil
}
func (f *fakeRepo) Offer(_ context.Context, _, _ string, cents int64) (transfer.Transfer, error) {
	f.offered = cents
	return f.current, nil
}
func (f *fakeRepo) Notify(_ context.Context, _, _, driver string) (transfer.Transfer, error) {
	f.notified = driver
	return f.current, nil
}
func (f *fakeRepo) Accept(_ context.Context, _, _ string, req transfer.AcceptRequest) (transfer.Transfer, error) {
	f.accept = req
	return f.current, nil
}
func (f *fakeRepo) Release(context.Context, string, string) (transfer.Transfer, error) {
	f.calls = append(f.calls, "release")
	return f.current, nil
}
func (f *fakeRepo) SetPaid(_ context.Context, _, _ string, paid bool) (transfer.Transfer, error) {
	f.calls = append(f.calls, map[bool]string{true: "paid", false: "unpaid"}[paid])
	return f.current, nil
}
func (f *fakeRepo) Start(context.Context, string, string) (transfer.Transfer, error) {
	f.calls = append(f.calls, "start")
	return f.current, nil
}
func (f *fakeRepo) Complete(_ context.Context, _, _ string, extra int64) (transfer.Transfer, error) {
	f.extra = extra
	return f.current, nil
}
func (f *fakeRepo) Cancel(_ context.Context, _, _ string, fee int64, flight bool) (transfer.Transfer, error) {
	f.fee, f.flight = fee, flight
	return f.current, nil
}

func TestTransferService(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 12, 9, 10, 0, 0, 0, timezone.Location)
	pickup := time.Date(2026, 12, 10, 8, 0, 0, 0, timezone.Location)
	repo := &fakeRepo{current: transfer.Transfer{ID: "t1", Stage: transfer.StageWaiting, PickupAt: pickup, PriceCents: 45000}}
	svc := NewTransferService(repo)
	svc.now = func() time.Time { return now }

	list, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	_, err = svc.Get(ctx, "t1")
	require.NoError(t, err)
	events, err := svc.Events(ctx, "t1")
	require.NoError(t, err)
	assert.Len(t, events, 1)

	created, err := svc.Create(ctx, transfer.CreateRequest{ClientName: "C", PickupAt: "2026-12-10T08:00", Origin: "A",
		Destination: "B", PriceCents: 45000}, "u1")
	require.NoError(t, err)
	assert.Equal(t, "t1", created.ID)
	_, err = svc.Create(ctx, transfer.CreateRequest{}, "u1")
	assert.ErrorIs(t, err, transfer.ErrClientRequired)

	drivers, err := svc.Drivers(ctx, "t1")
	require.NoError(t, err)
	assert.Len(t, drivers, 1)

	// Valor para o motorista; o repository confere se não diminui (RN-02.22).
	_, err = svc.Offer(ctx, "t1", "u1", transfer.OfferRequest{OfferCents: 20000})
	require.NoError(t, err)
	assert.Equal(t, int64(20000), repo.offered)
	_, err = svc.Offer(ctx, "t1", "u1", transfer.OfferRequest{})
	assert.ErrorIs(t, err, transfer.ErrInvalidOffer)

	_, err = svc.Notify(ctx, "t1", "u1", transfer.NotifyRequest{DriverPartnerID: " p1 "})
	require.NoError(t, err)
	assert.Equal(t, "p1", repo.notified)
	_, err = svc.Notify(ctx, "t1", "u1", transfer.NotifyRequest{})
	assert.ErrorIs(t, err, transfer.ErrDriverNotFound)

	_, err = svc.Accept(ctx, "t1", "u1", transfer.AcceptRequest{DriverPartnerID: "p1", VehicleID: "v1"})
	require.NoError(t, err)
	assert.Equal(t, "v1", repo.accept.VehicleID)

	_, err = svc.Release(ctx, "t1", "u1")
	require.NoError(t, err)
	_, err = svc.SetPaid(ctx, "t1", "u1", true)
	require.NoError(t, err)
	_, err = svc.Start(ctx, "t1", "u1")
	require.NoError(t, err)
	assert.Equal(t, []string{"release", "paid", "start"}, repo.calls)

	_, err = svc.Complete(ctx, "t1", "u1", transfer.CompleteRequest{ExtraWaitCents: 5000})
	require.NoError(t, err)
	assert.Equal(t, int64(5000), repo.extra)
	_, err = svc.Complete(ctx, "t1", "u1", transfer.CompleteRequest{ExtraWaitCents: -1})
	assert.ErrorIs(t, err, transfer.ErrInvalidExtraWait)

	// Cancelamento a 22 horas: 50% (RN-02.10); com voo cancelado, nada.
	_, err = svc.Cancel(ctx, "t1", "u1", transfer.CancelRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(22500), repo.fee)
	_, err = svc.Cancel(ctx, "t1", "u1", transfer.CancelRequest{FlightCancelled: true})
	require.NoError(t, err)
	assert.Equal(t, int64(0), repo.fee)
	assert.True(t, repo.flight)

	repo.current.Stage = transfer.StageOnTrip
	_, err = svc.Cancel(ctx, "t1", "u1", transfer.CancelRequest{})
	assert.ErrorIs(t, err, transfer.ErrWrongStage)

	repo.current = transfer.Transfer{}
	_, err = svc.Cancel(ctx, "x", "u1", transfer.CancelRequest{})
	assert.ErrorIs(t, err, transfer.ErrNotFound)
}

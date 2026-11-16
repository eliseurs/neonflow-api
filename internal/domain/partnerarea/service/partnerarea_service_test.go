package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/partnerarea"
	"neonflow-api/internal/domain/transfer"
)

type fakeRepo struct {
	partner partnerarea.Partner
	err     error
	listErr error
	owns    bool
	calls   []string
}

func (f *fakeRepo) Partner(context.Context, string) (partnerarea.Partner, error) {
	return f.partner, f.err
}
func (f *fakeRepo) Vehicles(context.Context, string) ([]partnerarea.Vehicle, error) {
	f.calls = append(f.calls, "vehicles")
	return []partnerarea.Vehicle{{ID: "v1"}}, f.listErr
}
func (f *fakeRepo) Bookings(context.Context, string) ([]partnerarea.Booking, error) {
	f.calls = append(f.calls, "bookings")
	return []partnerarea.Booking{{ID: "b1"}}, nil
}
func (f *fakeRepo) Rides(context.Context, string) ([]partnerarea.Ride, error) {
	f.calls = append(f.calls, "rides")
	return []partnerarea.Ride{{ID: "t1"}}, f.listErr
}
func (f *fakeRepo) Offers(context.Context) ([]partnerarea.Offer, error) {
	f.calls = append(f.calls, "offers")
	return []partnerarea.Offer{{ID: "t9"}}, nil
}
func (f *fakeRepo) Referrals(_ context.Context, id string) ([]partnerarea.Referral, error) {
	f.calls = append(f.calls, "referrals:"+id)
	return []partnerarea.Referral{}, nil
}
func (f *fakeRepo) OwnsVehicle(context.Context, string, string) (bool, error) { return f.owns, nil }

type fakeTransfers struct {
	accepted transfer.AcceptRequest
	err      error
	current  transfer.Transfer
	getErr   error
	released bool
}

func (f *fakeTransfers) Get(context.Context, string) (transfer.Transfer, error) {
	return f.current, f.getErr
}
func (f *fakeTransfers) Accept(_ context.Context, _, _ string, req transfer.AcceptRequest) (transfer.Transfer, error) {
	f.accepted = req
	return transfer.Transfer{}, f.err
}
func (f *fakeTransfers) Release(context.Context, string, string) (transfer.Transfer, error) {
	f.released = true
	return transfer.Transfer{}, f.err
}

func svcFor(repo *fakeRepo, tr *fakeTransfers) *PartnerAreaService {
	return NewPartnerAreaService(repo, tr)
}

var driver = partnerarea.Partner{ID: "p2", Kinds: []string{"motorista"}}

func TestArea_ByKind(t *testing.T) {
	owner := &fakeRepo{partner: partnerarea.Partner{ID: "p1", Kinds: []string{"proprietario"}}}
	a, err := svcFor(owner, &fakeTransfers{}).Area(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, []string{"vehicles", "bookings", "referrals:p1"}, owner.calls)
	assert.Len(t, a.Bookings, 1)
	assert.Empty(t, a.Offers, "a oferta aberta é só para motoristas")

	d := &fakeRepo{partner: partnerarea.Partner{ID: "p2", Kinds: []string{"motorista", "indicador"}}}
	a, err = svcFor(d, &fakeTransfers{}).Area(context.Background(), "u2")
	require.NoError(t, err)
	assert.Equal(t, []string{"rides", "offers", "referrals:p2"}, d.calls, "o motorista não vê carros nem reservas")
	assert.Len(t, a.Offers, 1)
}

func TestArea_Errors(t *testing.T) {
	_, err := svcFor(&fakeRepo{err: partnerarea.ErrNoPartner}, &fakeTransfers{}).Area(context.Background(), "u1")
	assert.ErrorIs(t, err, partnerarea.ErrNoPartner)
	boom := errors.New("db")
	_, err = svcFor(&fakeRepo{partner: partnerarea.Partner{Kinds: []string{"proprietario"}}, listErr: boom}, &fakeTransfers{}).Area(context.Background(), "u1")
	assert.ErrorIs(t, err, boom)
	_, err = svcFor(&fakeRepo{partner: driver, listErr: boom}, &fakeTransfers{}).Area(context.Background(), "u1")
	assert.ErrorIs(t, err, boom)
}

func TestAccept(t *testing.T) {
	tr := &fakeTransfers{}
	require.NoError(t, svcFor(&fakeRepo{partner: driver}, tr).Accept(context.Background(), "u2", "t9", partnerarea.AcceptRequest{}))
	assert.Equal(t, transfer.AcceptRequest{DriverPartnerID: "p2"}, tr.accepted, "o motorista é o da conta, sem valor novo")

	tr = &fakeTransfers{}
	require.NoError(t, svcFor(&fakeRepo{partner: driver, owns: true}, tr).Accept(context.Background(), "u2", "t9", partnerarea.AcceptRequest{VehicleID: " v1 "}))
	assert.Equal(t, "v1", tr.accepted.VehicleID)

	err := svcFor(&fakeRepo{partner: driver, owns: false}, &fakeTransfers{}).Accept(context.Background(), "u2", "t9", partnerarea.AcceptRequest{VehicleID: "v9"})
	assert.ErrorIs(t, err, partnerarea.ErrNotYourVehicle)

	err = svcFor(&fakeRepo{partner: partnerarea.Partner{ID: "p1", Kinds: []string{"proprietario"}}}, &fakeTransfers{}).Accept(context.Background(), "u1", "t9", partnerarea.AcceptRequest{})
	assert.ErrorIs(t, err, partnerarea.ErrNotDriver)

	err = svcFor(&fakeRepo{partner: driver}, &fakeTransfers{err: transfer.ErrWrongStage}).Accept(context.Background(), "u2", "t9", partnerarea.AcceptRequest{})
	assert.ErrorIs(t, err, transfer.ErrWrongStage, "outro motorista aceitou antes")

	err = svcFor(&fakeRepo{err: partnerarea.ErrNoPartner}, &fakeTransfers{}).Accept(context.Background(), "u2", "t9", partnerarea.AcceptRequest{})
	assert.ErrorIs(t, err, partnerarea.ErrNoPartner)
}

func TestRelease(t *testing.T) {
	mine := "p2"
	tr := &fakeTransfers{current: transfer.Transfer{DriverPartnerID: &mine}}
	require.NoError(t, svcFor(&fakeRepo{partner: driver}, tr).Release(context.Background(), "u2", "t1"))
	assert.True(t, tr.released)

	other := "p7"
	tr = &fakeTransfers{current: transfer.Transfer{DriverPartnerID: &other}}
	assert.ErrorIs(t, svcFor(&fakeRepo{partner: driver}, tr).Release(context.Background(), "u2", "t1"), partnerarea.ErrNotYourRide)
	assert.False(t, tr.released, "não desiste da corrida de outro")
	assert.ErrorIs(t, svcFor(&fakeRepo{partner: driver}, &fakeTransfers{}).Release(context.Background(), "u2", "t1"), partnerarea.ErrNotYourRide)
	assert.ErrorIs(t, svcFor(&fakeRepo{partner: driver}, &fakeTransfers{getErr: transfer.ErrNotFound}).Release(context.Background(), "u2", "t1"), partnerarea.ErrNotYourRide)
	boom := errors.New("db")
	assert.ErrorIs(t, svcFor(&fakeRepo{partner: driver}, &fakeTransfers{getErr: boom}).Release(context.Background(), "u2", "t1"), boom)
	assert.ErrorIs(t, svcFor(&fakeRepo{partner: partnerarea.Partner{Kinds: []string{"indicador"}}}, &fakeTransfers{}).Release(context.Background(), "u2", "t1"), partnerarea.ErrNotDriver)
}

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/partnerarea"
)

type fakeRepo struct {
	partner partnerarea.Partner
	err     error
	listErr error
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
func (f *fakeRepo) Referrals(_ context.Context, id string) ([]partnerarea.Referral, error) {
	f.calls = append(f.calls, "referrals:"+id)
	return []partnerarea.Referral{}, nil
}

func TestArea_ByKind(t *testing.T) {
	owner := &fakeRepo{partner: partnerarea.Partner{ID: "p1", Kinds: []string{"proprietario"}}}
	a, err := NewPartnerAreaService(owner).Area(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, []string{"vehicles", "bookings", "referrals:p1"}, owner.calls)
	assert.Len(t, a.Bookings, 1)
	assert.Empty(t, a.Rides)

	driver := &fakeRepo{partner: partnerarea.Partner{ID: "p2", Kinds: []string{"motorista", "indicador"}}}
	a, err = NewPartnerAreaService(driver).Area(context.Background(), "u2")
	require.NoError(t, err)
	assert.Equal(t, []string{"rides", "referrals:p2"}, driver.calls, "o motorista não vê carros nem reservas")
	assert.Empty(t, a.Vehicles)
	assert.Len(t, a.Rides, 1)
}

func TestArea_Errors(t *testing.T) {
	_, err := NewPartnerAreaService(&fakeRepo{err: partnerarea.ErrNoPartner}).Area(context.Background(), "u1")
	assert.ErrorIs(t, err, partnerarea.ErrNoPartner)
	boom := errors.New("db")
	_, err = NewPartnerAreaService(&fakeRepo{partner: partnerarea.Partner{Kinds: []string{"proprietario"}}, listErr: boom}).Area(context.Background(), "u1")
	assert.ErrorIs(t, err, boom)
	_, err = NewPartnerAreaService(&fakeRepo{partner: partnerarea.Partner{Kinds: []string{"motorista"}}, listErr: boom}).Area(context.Background(), "u1")
	assert.ErrorIs(t, err, boom)
}

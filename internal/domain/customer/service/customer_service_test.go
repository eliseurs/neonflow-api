package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/customer"
)

type fakeRepo struct {
	text, digits string
	saved        customer.Request
	anonymized   string
}

func (f *fakeRepo) List(_ context.Context, text, digits string) ([]customer.Customer, error) {
	f.text, f.digits = text, digits
	return []customer.Customer{{ID: "c1"}}, nil
}
func (f *fakeRepo) Get(_ context.Context, id string) (customer.Detail, error) {
	return customer.Detail{Customer: customer.Customer{ID: id}}, nil
}
func (f *fakeRepo) Create(_ context.Context, r customer.Request) (customer.Customer, error) {
	f.saved = r
	return customer.Customer{ID: "c1", Name: r.Name}, nil
}
func (f *fakeRepo) Update(_ context.Context, id string, r customer.Request) (customer.Customer, error) {
	f.saved = r
	return customer.Customer{ID: id, Name: r.Name}, nil
}
func (f *fakeRepo) Anonymize(_ context.Context, id string) (customer.Customer, error) {
	f.anonymized = id
	return customer.Customer{ID: id, Name: customer.AnonymizedName}, nil
}

func TestCustomerService(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{}
	svc := NewCustomerService(repo)

	_, err := svc.List(ctx, " (54) 9999 ")
	require.NoError(t, err)
	assert.Equal(t, "549999", repo.digits)
	d, err := svc.Get(ctx, "c9")
	require.NoError(t, err)
	assert.Equal(t, "c9", d.ID)

	c, err := svc.Create(ctx, customer.Request{Name: " Ana "})
	require.NoError(t, err)
	assert.Equal(t, "Ana", c.Name)
	_, err = svc.Create(ctx, customer.Request{})
	assert.ErrorIs(t, err, customer.ErrNameRequired)
	_, err = svc.Update(ctx, "c1", customer.Request{Name: "Ana", Email: "x"})
	assert.ErrorIs(t, err, customer.ErrInvalidEmail)
	u, err := svc.Update(ctx, "c1", customer.Request{Name: "Ana Maria"})
	require.NoError(t, err)
	assert.Equal(t, "Ana Maria", u.Name)

	a, err := svc.Anonymize(ctx, "c1")
	require.NoError(t, err)
	assert.Equal(t, customer.AnonymizedName, a.Name)
	assert.Equal(t, "c1", repo.anonymized)
}

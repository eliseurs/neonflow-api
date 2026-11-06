package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/partner"
)

type fakeRepo struct {
	kind      string
	codes     []string
	takenLeft int
	code      string
	updated   partner.Request
}

func (f *fakeRepo) List(_ context.Context, kind string) ([]partner.Partner, error) {
	f.kind = kind
	return []partner.Partner{{ID: "p1"}}, nil
}
func (f *fakeRepo) FindByID(_ context.Context, id string) (partner.Partner, error) {
	return partner.Partner{ID: id}, nil
}
func (f *fakeRepo) FindByCode(_ context.Context, code string) (partner.Partner, error) {
	f.code = code
	return partner.Partner{ReferralCode: code}, nil
}
func (f *fakeRepo) Create(_ context.Context, code string, r partner.Request) (partner.Partner, error) {
	f.codes = append(f.codes, code)
	if f.takenLeft > 0 {
		f.takenLeft--
		return partner.Partner{}, partner.ErrCodeTaken
	}
	return partner.Partner{Name: r.Name, ReferralCode: code}, nil
}
func (f *fakeRepo) Update(_ context.Context, id string, r partner.Request) (partner.Partner, error) {
	f.updated = r
	return partner.Partner{ID: id, Name: r.Name}, nil
}

func TestPartnerService(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{}
	svc := NewPartnerService(repo)
	n := 0
	svc.newCode = func() (string, error) { n++; return []string{"AAA222", "BBB333", "CCC444"}[n-1], nil }

	// Código repetido: sorteia de novo (RN-07.3).
	repo.takenLeft = 2
	p, err := svc.Create(ctx, partner.Request{Name: " Hotel ", Kinds: []string{"indicador"}})
	require.NoError(t, err)
	assert.Equal(t, "CCC444", p.ReferralCode)
	assert.Equal(t, []string{"AAA222", "BBB333", "CCC444"}, repo.codes)

	repo.takenLeft = 99
	svc.newCode = func() (string, error) { return "DDD555", nil }
	_, err = svc.Create(ctx, partner.Request{Name: "X", Kinds: []string{"indicador"}})
	assert.ErrorIs(t, err, partner.ErrCodeExhausted)

	svc.newCode = func() (string, error) { return "", errors.New("rand") }
	_, err = svc.Create(ctx, partner.Request{Name: "X", Kinds: []string{"indicador"}})
	assert.Error(t, err)
	_, err = svc.Create(ctx, partner.Request{Name: "", Kinds: []string{"indicador"}})
	assert.ErrorIs(t, err, partner.ErrNameRequired)

	_, err = svc.List(ctx, "motorista")
	require.NoError(t, err)
	assert.Equal(t, "motorista", repo.kind)
	_, err = svc.List(ctx, "locadora")
	assert.ErrorIs(t, err, partner.ErrInvalidKind)

	got, err := svc.Get(ctx, "p9")
	require.NoError(t, err)
	assert.Equal(t, "p9", got.ID)
	_, err = svc.ByCode(ctx, " ccc444 ")
	require.NoError(t, err)
	assert.Equal(t, "CCC444", repo.code)
	_, err = svc.ByCode(ctx, "?")
	assert.ErrorIs(t, err, partner.ErrPartnerNotFound)

	u, err := svc.Update(ctx, "p1", partner.Request{Name: " Novo ", Kinds: []string{"motorista"}})
	require.NoError(t, err)
	assert.Equal(t, "Novo", u.Name)
	_, err = svc.Update(ctx, "p1", partner.Request{Name: "Novo"})
	assert.ErrorIs(t, err, partner.ErrInvalidKind)
}

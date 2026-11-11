package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/terms"
)

type fakeRepo struct {
	current   *terms.Version
	stage     string
	stageErr  error
	linkHash  string
	linkExp   time.Time
	link      terms.LinkInfo
	linkErr   error
	accepted  string
	acceptVer int
	createErr error
	order     *terms.Order
	orderErr  error
	orderReq  terms.AcceptRequest
}

func (f *fakeRepo) Versions(context.Context) ([]terms.Version, error) {
	return []terms.Version{{Version: 1}}, nil
}
func (f *fakeRepo) Current(context.Context) (terms.Version, error) {
	if f.current == nil {
		return terms.Version{}, terms.ErrNoTerms
	}
	return *f.current, nil
}
func (f *fakeRepo) CreateVersion(_ context.Context, body, _ string) (terms.Version, error) {
	return terms.Version{Version: 2, Body: body}, nil
}
func (f *fakeRepo) BookingStage(context.Context, string) (string, error) { return f.stage, f.stageErr }
func (f *fakeRepo) CreateLink(_ context.Context, _, hash string, exp time.Time, _ string) error {
	f.linkHash, f.linkExp = hash, exp
	return f.createErr
}
func (f *fakeRepo) LinkByHash(_ context.Context, hash string) (terms.LinkInfo, error) {
	if hash != f.linkHash {
		return terms.LinkInfo{}, terms.ErrLinkNotFound
	}
	return f.link, f.linkErr
}
func (f *fakeRepo) Public(_ context.Context, l terms.LinkInfo) (terms.Public, error) {
	return terms.Public{Quadro: terms.Quadro{ClientName: "Cliente " + l.BookingID}}, nil
}
func (f *fakeRepo) Accept(_ context.Context, _ terms.LinkInfo, version int, name string) (terms.Acceptance, error) {
	f.accepted, f.acceptVer = name, version
	return terms.Acceptance{Version: version, TypedName: name}, nil
}
func (f *fakeRepo) PublishedOrder(context.Context) (*terms.Order, error) { return f.order, f.orderErr }
func (f *fakeRepo) AcceptOrder(_ context.Context, _ terms.LinkInfo, req terms.AcceptRequest) (terms.Acceptance, error) {
	f.orderReq = req
	return terms.Acceptance{Document: terms.DocOrder, Version: 3, SHA256: req.SHA256, TypedName: req.TypedName}, nil
}
func (f *fakeRepo) LastAcceptance(context.Context, string) (*terms.Acceptance, error) {
	return &terms.Acceptance{Version: 1}, nil
}

func TestTermsService(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepo{stage: "confirmada"}
	svc := NewTermsService(repo)
	svc.now = func() time.Time { return now }

	list, err := svc.Versions(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	v, err := svc.Publish(ctx, "  Texto  ", "u1")
	require.NoError(t, err)
	assert.Equal(t, "Texto", v.Body)
	_, err = svc.Publish(ctx, " ", "u1")
	assert.ErrorIs(t, err, terms.ErrBodyRequired)

	// Sem termos publicados, não há link.
	_, err = svc.CreateLink(ctx, "b1", "u1")
	assert.ErrorIs(t, err, terms.ErrNoTerms)
	repo.current = &terms.Version{Version: 1}
	link, err := svc.CreateLink(ctx, "b1", "u1")
	require.NoError(t, err)
	assert.Equal(t, "/termos/"+link.Token, link.Path)
	assert.Equal(t, terms.HashToken(link.Token), repo.linkHash, "o banco guarda só o hash")
	assert.True(t, repo.linkExp.Equal(now.Add(7*24*time.Hour)))

	repo.stage = "em_locacao"
	_, err = svc.CreateLink(ctx, "b1", "u1")
	assert.ErrorIs(t, err, terms.ErrWrongStage)
	repo.stageErr = terms.ErrBookingNotFound
	_, err = svc.CreateLink(ctx, "x", "u1")
	assert.ErrorIs(t, err, terms.ErrBookingNotFound)
	repo.stageErr, repo.stage = nil, "confirmada"
	svc.newToken = func() (string, string, error) { return "", "", errors.New("rand") }
	_, err = svc.CreateLink(ctx, "b1", "u1")
	assert.Error(t, err)
	svc.newToken = terms.NewToken

	a, err := svc.BookingAcceptance(ctx, "b1")
	require.NoError(t, err)
	assert.Equal(t, 1, a.Version)

	// Pelo código do link.
	repo.link = terms.LinkInfo{ID: "l1", BookingID: "b1", BookingStage: "confirmada"}
	p, err := svc.Public(ctx, link.Token)
	require.NoError(t, err)
	assert.Equal(t, "Cliente b1", p.ClientName)
	_, err = svc.Public(ctx, "codigo-errado")
	assert.ErrorIs(t, err, terms.ErrLinkNotFound)

	acc, err := svc.Accept(ctx, link.Token, terms.AcceptRequest{TypedName: " Ana  Teste ", Version: 1})
	require.NoError(t, err)
	assert.Equal(t, "Ana Teste", acc.TypedName)
	_, err = svc.Accept(ctx, link.Token, terms.AcceptRequest{TypedName: "A", Version: 1})
	assert.ErrorIs(t, err, terms.ErrInvalidName)
	_, err = svc.Accept(ctx, "codigo-errado", terms.AcceptRequest{TypedName: "Ana Teste", Version: 1})
	assert.ErrorIs(t, err, terms.ErrLinkNotFound)
	repo.link.BookingStage = "cancelada"
	_, err = svc.Accept(ctx, link.Token, terms.AcceptRequest{TypedName: "Ana Teste", Version: 1})
	assert.ErrorIs(t, err, terms.ErrLinkNotFound, "reserva cancelada não aceita")
}

func TestTermsService_Order(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{stage: "confirmada", order: &terms.Order{VersionID: "cv3", Version: 3, SHA256: "abc"}}
	svc := NewTermsService(repo)

	// Com a Ordem 04 publicada, o link sai mesmo sem os termos antigos.
	link, err := svc.CreateLink(ctx, "b1", "u1")
	require.NoError(t, err)
	repo.link = terms.LinkInfo{ID: "l1", BookingID: "b1", BookingStage: "confirmada"}

	req := terms.AcceptRequest{TypedName: "Ana Teste", ContractVersionID: "cv3", SHA256: " ABC ", QuadroSHA256: "q1", Confirm: true}
	acc, err := svc.Accept(ctx, link.Token, req)
	require.NoError(t, err)
	assert.Equal(t, terms.DocOrder, acc.Document)
	assert.Equal(t, "abc", repo.orderReq.SHA256, "hash normalizado")

	req.Confirm = false
	_, err = svc.Accept(ctx, link.Token, req)
	assert.ErrorIs(t, err, terms.ErrConfirmRequired, "caixa não marcada")

	// Aceite dos termos antigos depois da Ordem publicada: recarregar.
	_, err = svc.Accept(ctx, link.Token, terms.AcceptRequest{TypedName: "Ana Teste", Version: 1})
	assert.ErrorIs(t, err, terms.ErrVersionChanged)

	repo.order, repo.orderErr = nil, errors.New("db")
	_, err = svc.CreateLink(ctx, "b1", "u1")
	assert.Error(t, err)
	_, err = svc.Accept(ctx, link.Token, terms.AcceptRequest{TypedName: "Ana Teste", Version: 1})
	assert.Error(t, err)
}

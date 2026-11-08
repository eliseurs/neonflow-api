package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/contract"
)

type fakeRepo struct {
	createdSHA string
	comment    contract.CommentRequest
	action     string
	current    string
	sha        string
}

func (f *fakeRepo) Documents(context.Context) ([]contract.Document, error) {
	return []contract.Document{{Code: "01"}}, nil
}
func (f *fakeRepo) Versions(context.Context, string) ([]contract.Summary, error) {
	return []contract.Summary{{Version: 1}}, nil
}
func (f *fakeRepo) CreateVersion(_ context.Context, _ string, req contract.VersionRequest, sha string, _ contract.Actor) (contract.Summary, error) {
	f.createdSHA = sha
	return contract.Summary{Version: 1, Label: req.Label, SHA256: sha}, nil
}
func (f *fakeRepo) Version(context.Context, string) (contract.Version, error) {
	return contract.Version{Body: "x"}, nil
}
func (f *fakeRepo) AddComment(_ context.Context, _ string, req contract.CommentRequest, _ contract.Actor) (contract.Comment, error) {
	f.comment = req
	return contract.Comment{Clause: req.Clause}, nil
}
func (f *fakeRepo) AddEvent(_ context.Context, _, action, note string, _ contract.Actor,
	next func(current, sha string) (string, error)) (contract.Event, error) {
	f.action = action
	status, err := next(f.current, f.sha)
	if err != nil {
		return contract.Event{}, err
	}
	return contract.Event{Action: action, Status: status, Note: note}, nil
}

var (
	admin    = contract.Actor{UserID: "a", Role: authentication.RoleAdmin}
	reviewer = contract.Actor{UserID: "r", Role: authentication.RoleLegalReviewer}
	central  = contract.Actor{UserID: "c", Role: authentication.RoleCentral}
)

func TestReads(t *testing.T) {
	s := NewContractService(&fakeRepo{})
	docs, err := s.Documents(context.Background())
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	vs, err := s.Versions(context.Background(), "01")
	require.NoError(t, err)
	assert.Len(t, vs, 1)
	v, err := s.Version(context.Background(), "id")
	require.NoError(t, err)
	assert.Equal(t, "x", v.Body)
}

func TestCreateVersion(t *testing.T) {
	repo := &fakeRepo{}
	s := NewContractService(repo)
	v, err := s.CreateVersion(context.Background(), "04", contract.VersionRequest{Label: "v0.4", Body: " texto "}, admin)
	require.NoError(t, err)
	assert.Equal(t, contract.Hash("texto"), repo.createdSHA, "hash do texto já limpo")
	assert.Equal(t, "v0.4", v.Label)

	_, err = s.CreateVersion(context.Background(), "04", contract.VersionRequest{Body: "x"}, reviewer)
	assert.ErrorIs(t, err, contract.ErrForbiddenAction, "revisora não edita o texto")
	_, err = s.CreateVersion(context.Background(), "04", contract.VersionRequest{}, admin)
	assert.ErrorIs(t, err, contract.ErrBodyRequired)
}

func TestAddComment(t *testing.T) {
	repo := &fakeRepo{}
	s := NewContractService(repo)
	_, err := s.AddComment(context.Background(), "v", contract.CommentRequest{Clause: " 2.1 ", Kind: contract.KindCorrecao, Body: "b"}, reviewer)
	require.NoError(t, err)
	assert.Equal(t, "2.1", repo.comment.Clause)

	_, err = s.AddComment(context.Background(), "v", contract.CommentRequest{Clause: "1", Kind: contract.KindObservacao, Body: "b"}, central)
	assert.ErrorIs(t, err, contract.ErrForbiddenComment)
	_, err = s.AddComment(context.Background(), "v", contract.CommentRequest{}, admin)
	assert.ErrorIs(t, err, contract.ErrInvalidClause)
}

func TestAddEvent(t *testing.T) {
	sha := contract.Hash("texto")
	repo := &fakeRepo{current: contract.StatusEnviado, sha: sha}
	s := NewContractService(repo)

	e, err := s.AddEvent(context.Background(), "v", contract.EventRequest{Action: contract.ActionAprovar, SHA256: sha, Confirm: true}, reviewer)
	require.NoError(t, err)
	assert.Equal(t, contract.StatusAprovado, e.Status)
	assert.Equal(t, contract.ActionAprovar, repo.action)

	_, err = s.AddEvent(context.Background(), "v", contract.EventRequest{Action: contract.ActionAprovar, SHA256: sha, Confirm: true}, central)
	assert.ErrorIs(t, err, contract.ErrForbiddenAction)
	_, err = s.AddEvent(context.Background(), "v", contract.EventRequest{Action: contract.ActionAprovar, SHA256: sha}, reviewer)
	assert.ErrorIs(t, err, contract.ErrConfirmRequired)
}

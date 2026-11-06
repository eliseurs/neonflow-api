package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/user"
)

type fakeRepo struct {
	created  authentication.NewUser
	updated  user.UpdateRequest
	hash     string
	setHash  string
	findErr  error
	hashErr  error
	setCalls int
}

func (f *fakeRepo) List(context.Context) ([]user.User, error) { return []user.User{{ID: "u1"}}, nil }
func (f *fakeRepo) FindByID(_ context.Context, id string) (user.User, error) {
	return user.User{ID: id}, f.findErr
}
func (f *fakeRepo) Create(_ context.Context, n authentication.NewUser) (user.User, error) {
	f.created = n
	return user.User{ID: "new", Email: n.Email, Role: n.Role, Active: true}, nil
}
func (f *fakeRepo) Update(_ context.Context, id string, r user.UpdateRequest) (user.User, error) {
	f.updated = r
	return user.User{ID: id, Name: r.Name, Role: r.Role}, nil
}
func (f *fakeRepo) PasswordHash(context.Context, string) (string, error) { return f.hash, f.hashErr }
func (f *fakeRepo) SetPassword(_ context.Context, _ string, h string) error {
	f.setHash = h
	f.setCalls++
	return nil
}

type fakeHasher struct{ err error }

func (h fakeHasher) Hash(p string) (string, error) { return "h:" + p, h.err }
func (h fakeHasher) Compare(hash, p string) bool   { return hash == "h:"+p }

func TestUserService_Create(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewUserService(repo, fakeHasher{})
	u, err := svc.Create(context.Background(), user.CreateRequest{Name: " Ana ", Email: " Ana@NeonFlow.test ", Password: "senha-forte-123", Role: "central"})
	require.NoError(t, err)
	assert.Equal(t, "ana@neonflow.test", u.Email)
	assert.Equal(t, "h:senha-forte-123", repo.created.PasswordHash, "nunca grava a senha em texto")

	_, err = svc.Create(context.Background(), user.CreateRequest{Name: "Ana", Email: "ana@neonflow.test", Password: "curta", Role: "central"})
	assert.ErrorIs(t, err, authentication.ErrWeakPassword)
	_, err = NewUserService(repo, fakeHasher{err: errors.New("x")}).Create(context.Background(),
		user.CreateRequest{Name: "Ana", Email: "ana@neonflow.test", Password: "senha-forte-123", Role: "central"})
	assert.Error(t, err)

	list, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestUserService_Update(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewUserService(repo, fakeHasher{})
	ctx := context.Background()
	off := false

	u, err := svc.Update(ctx, "admin1", "u2", user.UpdateRequest{Name: " Ana ", Role: "central", Active: &off})
	require.NoError(t, err)
	assert.Equal(t, "Ana", u.Name)
	assert.False(t, *repo.updated.Active)

	// O admin não perde o próprio acesso.
	_, err = svc.Update(ctx, "admin1", "admin1", user.UpdateRequest{Name: "Eu", Role: "central"})
	assert.ErrorIs(t, err, user.ErrSelfLockout)
	_, err = svc.Update(ctx, "admin1", "admin1", user.UpdateRequest{Name: "Eu", Role: "admin", Active: &off})
	assert.ErrorIs(t, err, user.ErrSelfLockout)
	_, err = svc.Update(ctx, "admin1", "admin1", user.UpdateRequest{Name: "Eu de novo", Role: "admin"})
	require.NoError(t, err)

	_, err = svc.Update(ctx, "admin1", "u2", user.UpdateRequest{Name: "", Role: "central"})
	assert.ErrorIs(t, err, authentication.ErrNameRequired)
}

func TestUserService_Passwords(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{hash: "h:senha-antiga-123"}
	svc := NewUserService(repo, fakeHasher{})

	require.NoError(t, svc.ResetPassword(ctx, "u2", user.PasswordRequest{Password: "senha-nova-1234"}))
	assert.Equal(t, "h:senha-nova-1234", repo.setHash)
	assert.ErrorIs(t, svc.ResetPassword(ctx, "u2", user.PasswordRequest{Password: "curta"}), authentication.ErrWeakPassword)
	repo.findErr = authentication.ErrUserNotFound
	assert.ErrorIs(t, svc.ResetPassword(ctx, "x", user.PasswordRequest{Password: "senha-nova-1234"}), authentication.ErrUserNotFound)

	assert.ErrorIs(t, svc.ChangeOwnPassword(ctx, "u1", user.ChangeOwnPasswordRequest{Current: "errada", Password: "senha-nova-1234"}), user.ErrWrongPassword)
	assert.ErrorIs(t, svc.ChangeOwnPassword(ctx, "u1", user.ChangeOwnPasswordRequest{Current: "senha-antiga-123", Password: "curta"}), authentication.ErrWeakPassword)
	require.NoError(t, svc.ChangeOwnPassword(ctx, "u1", user.ChangeOwnPasswordRequest{Current: "senha-antiga-123", Password: "outra-senha-123"}))
	assert.Equal(t, "h:outra-senha-123", repo.setHash)

	repo.hashErr = authentication.ErrUserNotFound
	assert.ErrorIs(t, svc.ChangeOwnPassword(ctx, "x", user.ChangeOwnPasswordRequest{Current: "a", Password: "outra-senha-123"}), authentication.ErrUserNotFound)
	assert.Error(t, NewUserService(&fakeRepo{}, fakeHasher{err: errors.New("x")}).ResetPassword(ctx, "u2", user.PasswordRequest{Password: "senha-nova-1234"}))
}

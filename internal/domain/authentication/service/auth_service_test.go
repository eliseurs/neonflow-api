package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/authentication"
)

type fakeRepo struct {
	byEmail   map[string]authentication.Credentials
	findErr   error
	created   authentication.NewUser
	createErr error
}

func (f *fakeRepo) FindByEmail(_ context.Context, email string) (authentication.Credentials, error) {
	if f.findErr != nil {
		return authentication.Credentials{}, f.findErr
	}
	c, ok := f.byEmail[email]
	if !ok {
		return c, authentication.ErrUserNotFound
	}
	return c, nil
}

func (f *fakeRepo) FindByID(_ context.Context, id string) (authentication.User, error) {
	for _, c := range f.byEmail {
		if c.ID == id {
			return c.User, nil
		}
	}
	return authentication.User{}, authentication.ErrUserNotFound
}

func (f *fakeRepo) Create(_ context.Context, n authentication.NewUser) (authentication.User, error) {
	f.created = n
	if f.createErr != nil {
		return authentication.User{}, f.createErr
	}
	return authentication.User{ID: "new", Name: n.Name, Email: n.Email, Role: n.Role}, nil
}

// fakeHasher: hash = "h:" + senha. compares conta quantas comparações aconteceram.
type fakeHasher struct {
	compares int
	hashErr  error
}

func (h *fakeHasher) Hash(p string) (string, error) { return "h:" + p, h.hashErr }
func (h *fakeHasher) Compare(hash, p string) bool   { h.compares++; return hash == "h:"+p }

type fakeTokens struct{ err error }

func (f fakeTokens) Generate(userID, role string) (string, time.Time, error) {
	return "tok-" + userID + "-" + role, time.Unix(100, 0), f.err
}

type nopLog struct{}

func (nopLog) Warn(string, ...any)  {}
func (nopLog) Error(string, ...any) {}

func newSvc(repo *fakeRepo, h *fakeHasher, tok fakeTokens) *AuthService {
	return NewAuthService(repo, h, tok, nopLog{})
}

var central = authentication.Credentials{
	User:         authentication.User{ID: "u1", Name: "Central", Email: "central@neonflow.test", Role: "central"},
	PasswordHash: "h:senha-forte-123",
}

func TestLogin_Success(t *testing.T) {
	repo := &fakeRepo{byEmail: map[string]authentication.Credentials{central.Email: central}}
	res, err := newSvc(repo, &fakeHasher{}, fakeTokens{}).Login(context.Background(), authentication.LoginRequest{Email: "  CENTRAL@neonflow.test ", Password: "senha-forte-123"})
	require.NoError(t, err)
	assert.Equal(t, "tok-u1-central", res.Token)
	assert.Equal(t, "u1", res.User.ID)
}

func TestLogin_SameErrorForUnknownEmailAndWrongPassword(t *testing.T) {
	repo := &fakeRepo{byEmail: map[string]authentication.Credentials{central.Email: central}}

	h := &fakeHasher{}
	_, err := newSvc(repo, h, fakeTokens{}).Login(context.Background(), authentication.LoginRequest{Email: "nao@existe.test", Password: "x"})
	assert.ErrorIs(t, err, authentication.ErrInvalidCredentials)
	assert.Equal(t, 1, h.compares, "compara mesmo sem usuário, para igualar o tempo de resposta")

	_, err = newSvc(repo, &fakeHasher{}, fakeTokens{}).Login(context.Background(), authentication.LoginRequest{Email: central.Email, Password: "errada"})
	assert.ErrorIs(t, err, authentication.ErrInvalidCredentials)
}

func TestLogin_Errors(t *testing.T) {
	repo := &fakeRepo{byEmail: map[string]authentication.Credentials{central.Email: central}}
	_, err := newSvc(repo, &fakeHasher{}, fakeTokens{}).Login(context.Background(), authentication.LoginRequest{Email: " ", Password: ""})
	assert.ErrorIs(t, err, authentication.ErrInvalidCredentials)

	boom := errors.New("db down")
	_, err = newSvc(&fakeRepo{findErr: boom}, &fakeHasher{}, fakeTokens{}).Login(context.Background(), authentication.LoginRequest{Email: "a@b.test", Password: "x"})
	assert.ErrorIs(t, err, boom)

	_, err = newSvc(repo, &fakeHasher{}, fakeTokens{err: errors.New("sign")}).Login(context.Background(), authentication.LoginRequest{Email: central.Email, Password: "senha-forte-123"})
	assert.EqualError(t, err, "sign")

	h := &fakeHasher{hashErr: errors.New("hash")}
	_, err = newSvc(&fakeRepo{}, h, fakeTokens{}).Login(context.Background(), authentication.LoginRequest{Email: "x@y.test", Password: "x"})
	assert.ErrorIs(t, err, authentication.ErrInvalidCredentials, "falha do hash fictício não vaza")
}

func TestMe(t *testing.T) {
	repo := &fakeRepo{byEmail: map[string]authentication.Credentials{central.Email: central}}
	svc := newSvc(repo, &fakeHasher{}, fakeTokens{})
	u, err := svc.Me(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, "Central", u.Name)

	_, err = svc.Me(context.Background(), " ")
	assert.ErrorIs(t, err, authentication.ErrUserNotFound)
	_, err = svc.Me(context.Background(), "outro")
	assert.ErrorIs(t, err, authentication.ErrUserNotFound)
}

func TestCreateUser(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakeHasher{}, fakeTokens{})
	u, err := svc.CreateUser(context.Background(), authentication.CreateUserRequest{Name: "Admin", Email: "Admin@Neonflow.test", Password: strings.Repeat("x", 12), Role: "admin"})
	require.NoError(t, err)
	assert.Equal(t, "admin@neonflow.test", u.Email)
	assert.Equal(t, "h:"+strings.Repeat("x", 12), repo.created.PasswordHash, "grava o hash, nunca a senha")

	_, err = svc.CreateUser(context.Background(), authentication.CreateUserRequest{Name: "A", Email: "a@b.test", Password: "curta", Role: "admin"})
	assert.ErrorIs(t, err, authentication.ErrWeakPassword)

	_, err = newSvc(&fakeRepo{}, &fakeHasher{hashErr: errors.New("x")}, fakeTokens{}).CreateUser(context.Background(), authentication.CreateUserRequest{Name: "A", Email: "a@b.test", Password: strings.Repeat("x", 12), Role: "admin"})
	assert.ErrorContains(t, err, "hash")

	_, err = newSvc(&fakeRepo{createErr: authentication.ErrEmailTaken}, &fakeHasher{}, fakeTokens{}).CreateUser(context.Background(), authentication.CreateUserRequest{Name: "A", Email: "a@b.test", Password: strings.Repeat("x", 12), Role: "admin"})
	assert.ErrorIs(t, err, authentication.ErrEmailTaken)
}

package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"neonflow-api/internal/domain/authentication"
)

type AuthService struct {
	repo   authentication.Repository
	hasher authentication.PasswordHasher
	tokens authentication.TokenIssuer
	log    authentication.Logger

	dummyOnce sync.Once
	dummyHash string
}

func NewAuthService(repo authentication.Repository, hasher authentication.PasswordHasher, tokens authentication.TokenIssuer, log authentication.Logger) *AuthService {
	return &AuthService{repo: repo, hasher: hasher, tokens: tokens, log: log}
}

// Login responde ErrInvalidCredentials tanto para e-mail inexistente quanto para senha errada (RN-05.5).
func (s *AuthService) Login(ctx context.Context, req authentication.LoginRequest) (authentication.LoginResponse, error) {
	email := authentication.NormalizeEmail(req.Email)
	if email == "" || req.Password == "" {
		return authentication.LoginResponse{}, authentication.ErrInvalidCredentials
	}

	creds, err := s.repo.FindByEmail(ctx, email)
	if errors.Is(err, authentication.ErrUserNotFound) {
		// Compara contra um hash fixo para o tempo de resposta não revelar se o e-mail existe.
		s.hasher.Compare(s.fakeHash(), req.Password)
		return authentication.LoginResponse{}, authentication.ErrInvalidCredentials
	}
	if err != nil {
		return authentication.LoginResponse{}, err
	}
	if !s.hasher.Compare(creds.PasswordHash, req.Password) {
		return authentication.LoginResponse{}, authentication.ErrInvalidCredentials
	}

	token, expiresAt, err := s.tokens.Generate(creds.ID, creds.Role)
	if err != nil {
		return authentication.LoginResponse{}, err
	}
	return authentication.LoginResponse{Token: token, ExpiresAt: expiresAt, User: creds.User}, nil
}

func (s *AuthService) Me(ctx context.Context, userID string) (authentication.User, error) {
	if strings.TrimSpace(userID) == "" {
		return authentication.User{}, authentication.ErrUserNotFound
	}
	return s.repo.FindByID(ctx, userID)
}

// CreateUser é usado pela CLI create-admin; não há cadastro aberto (RN-05.2).
func (s *AuthService) CreateUser(ctx context.Context, req authentication.CreateUserRequest) (authentication.User, error) {
	valid, err := req.Validate()
	if err != nil {
		return authentication.User{}, err
	}
	hash, err := s.hasher.Hash(valid.Password)
	if err != nil {
		return authentication.User{}, fmt.Errorf("hash: %w", err)
	}
	return s.repo.Create(ctx, authentication.NewUser{
		Name:         valid.Name,
		Email:        valid.Email,
		PasswordHash: hash,
		Role:         valid.Role,
	})
}

func (s *AuthService) fakeHash() string {
	s.dummyOnce.Do(func() {
		h, err := s.hasher.Hash("neonflow-timing-equalizer")
		if err != nil {
			s.log.Warn("dummy hash failed", "error", err)
		}
		s.dummyHash = h
	})
	return s.dummyHash
}

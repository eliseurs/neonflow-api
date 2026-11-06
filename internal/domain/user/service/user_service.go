package service

import (
	"context"
	"fmt"

	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/user"
)

type UserService struct {
	repo   user.Repository
	hasher authentication.PasswordHasher
}

func NewUserService(repo user.Repository, hasher authentication.PasswordHasher) *UserService {
	return &UserService{repo: repo, hasher: hasher}
}

func (s *UserService) List(ctx context.Context) ([]user.User, error) {
	return s.repo.List(ctx)
}

// Create usa a mesma validação da CLI create-admin (nome, e-mail, senha de 10+ caracteres, papel).
func (s *UserService) Create(ctx context.Context, req user.CreateRequest) (user.User, error) {
	valid, err := authentication.CreateUserRequest{Name: req.Name, Email: req.Email, Password: req.Password, Role: req.Role}.Validate()
	if err != nil {
		return user.User{}, err
	}
	hash, err := s.hasher.Hash(valid.Password)
	if err != nil {
		return user.User{}, fmt.Errorf("hash: %w", err)
	}
	return s.repo.Create(ctx, authentication.NewUser{Name: valid.Name, Email: valid.Email, PasswordHash: hash, Role: valid.Role})
}

// Update troca nome, papel e ativo. O admin não pode se desativar nem deixar de ser admin.
func (s *UserService) Update(ctx context.Context, actorID, id string, req user.UpdateRequest) (user.User, error) {
	valid, err := req.Validate()
	if err != nil {
		return user.User{}, err
	}
	if actorID == id && (valid.Role != authentication.RoleAdmin || (valid.Active != nil && !*valid.Active)) {
		return user.User{}, user.ErrSelfLockout
	}
	return s.repo.Update(ctx, id, valid)
}

// ResetPassword é o admin definindo uma senha nova para outro usuário.
func (s *UserService) ResetPassword(ctx context.Context, id string, req user.PasswordRequest) error {
	if err := user.ValidPassword(req.Password); err != nil {
		return err
	}
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return err
	}
	return s.setPassword(ctx, id, req.Password)
}

// ChangeOwnPassword exige a senha atual.
func (s *UserService) ChangeOwnPassword(ctx context.Context, actorID string, req user.ChangeOwnPasswordRequest) error {
	if err := user.ValidPassword(req.Password); err != nil {
		return err
	}
	hash, err := s.repo.PasswordHash(ctx, actorID)
	if err != nil {
		return err
	}
	if !s.hasher.Compare(hash, req.Current) {
		return user.ErrWrongPassword
	}
	return s.setPassword(ctx, actorID, req.Password)
}

func (s *UserService) setPassword(ctx context.Context, id, password string) error {
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return fmt.Errorf("hash: %w", err)
	}
	return s.repo.SetPassword(ctx, id, hash)
}

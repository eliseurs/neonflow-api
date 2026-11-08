package service

import (
	"context"

	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/contract"
)

type ContractService struct {
	repo contract.Repository
}

func NewContractService(repo contract.Repository) *ContractService {
	return &ContractService{repo: repo}
}

func (s *ContractService) Documents(ctx context.Context) ([]contract.Document, error) {
	return s.repo.Documents(ctx)
}

func (s *ContractService) Versions(ctx context.Context, code string) ([]contract.Summary, error) {
	return s.repo.Versions(ctx, code)
}

// CreateVersion grava uma versão nova (minuta) com o hash do texto. Só o admin cria versões.
func (s *ContractService) CreateVersion(ctx context.Context, code string, req contract.VersionRequest, actor contract.Actor) (contract.Summary, error) {
	if actor.Role != authentication.RoleAdmin {
		return contract.Summary{}, contract.ErrForbiddenAction
	}
	valid, err := req.Validate()
	if err != nil {
		return contract.Summary{}, err
	}
	return s.repo.CreateVersion(ctx, code, valid, contract.Hash(valid.Body), actor)
}

func (s *ContractService) Version(ctx context.Context, id string) (contract.Version, error) {
	return s.repo.Version(ctx, id)
}

func (s *ContractService) AddComment(ctx context.Context, versionID string, req contract.CommentRequest, actor contract.Actor) (contract.Comment, error) {
	if !contract.CanComment(actor.Role) {
		return contract.Comment{}, contract.ErrForbiddenComment
	}
	valid, err := req.Validate()
	if err != nil {
		return contract.Comment{}, err
	}
	return s.repo.AddComment(ctx, versionID, valid, actor)
}

// AddEvent registra a decisão. Papel, situação e hash são conferidos dentro da transação do repository.
func (s *ContractService) AddEvent(ctx context.Context, versionID string, req contract.EventRequest, actor contract.Actor) (contract.Event, error) {
	valid, err := req.Validate()
	if err != nil {
		return contract.Event{}, err
	}
	return s.repo.AddEvent(ctx, versionID, valid.Action, valid.Note, actor, func(current, sha string) (string, error) {
		return contract.Next(valid, actor, current, sha)
	})
}

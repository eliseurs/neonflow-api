package service

import (
	"context"

	"neonflow-api/internal/domain/checklist"
)

type ChecklistService struct {
	repo checklist.Repository
}

func NewChecklistService(repo checklist.Repository) *ChecklistService {
	return &ChecklistService{repo: repo}
}

func (s *ChecklistService) List(ctx context.Context) ([]checklist.Item, error) {
	return s.repo.List(ctx)
}

// Create gera a chave a partir do nome; ela não muda depois (RN-01.6: itens configuráveis).
func (s *ChecklistService) Create(ctx context.Context, req checklist.ItemRequest) (checklist.Item, error) {
	valid, err := req.Validate()
	if err != nil {
		return checklist.Item{}, err
	}
	key, err := checklist.NewKey(valid.Label)
	if err != nil {
		return checklist.Item{}, err
	}
	return s.repo.Create(ctx, key, valid)
}

// Update troca nome, ordem e ativo. Desativar tira o item das próximas conferências sem apagar as já feitas.
func (s *ChecklistService) Update(ctx context.Context, id string, req checklist.ItemRequest) (checklist.Item, error) {
	valid, err := req.Validate()
	if err != nil {
		return checklist.Item{}, err
	}
	return s.repo.Update(ctx, id, valid)
}

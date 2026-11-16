package service

import (
	"context"

	"neonflow-api/internal/domain/audit"
)

type AuditService struct {
	repo audit.Repository
}

func NewAuditService(repo audit.Repository) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) List(ctx context.Context, f audit.Filter) ([]audit.Entry, error) {
	f, err := f.Normalize()
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, f)
}

package service

import (
	"context"
	"errors"
	"fmt"

	"neonflow-api/internal/domain/partner"
)

// codeAttempts: tentativas de sortear um código livre (31^6 combinações; colisão é rara).
const codeAttempts = 5

type PartnerService struct {
	repo    partner.Repository
	newCode func() (string, error)
}

func NewPartnerService(repo partner.Repository) *PartnerService {
	return &PartnerService{repo: repo, newCode: partner.NewReferralCode}
}

// List filtra por tipo (proprietario, motorista, indicador); vazio lista todos.
func (s *PartnerService) List(ctx context.Context, kind string) ([]partner.Partner, error) {
	if kind != "" && kind != partner.KindOwner && kind != partner.KindDriver && kind != partner.KindReferrer {
		return nil, partner.ErrInvalidKind
	}
	return s.repo.List(ctx, kind)
}

func (s *PartnerService) Get(ctx context.Context, id string) (partner.Partner, error) {
	return s.repo.FindByID(ctx, id)
}

// ByCode acha o parceiro pelo código de indicação (o que chega na mensagem do WhatsApp).
func (s *PartnerService) ByCode(ctx context.Context, code string) (partner.Partner, error) {
	c := partner.NormalizeReferralCode(code)
	if c == "" {
		return partner.Partner{}, partner.ErrPartnerNotFound
	}
	return s.repo.FindByCode(ctx, c)
}

// Create gera o código de indicação do parceiro; ele não muda depois (RN-07.3).
func (s *PartnerService) Create(ctx context.Context, req partner.Request) (partner.Partner, error) {
	valid, err := req.Validate()
	if err != nil {
		return partner.Partner{}, err
	}
	for i := 0; i < codeAttempts; i++ {
		code, err := s.newCode()
		if err != nil {
			return partner.Partner{}, fmt.Errorf("referral code: %w", err)
		}
		p, err := s.repo.Create(ctx, code, valid)
		if !errors.Is(err, partner.ErrCodeTaken) {
			return p, err
		}
	}
	return partner.Partner{}, partner.ErrCodeExhausted
}

func (s *PartnerService) Update(ctx context.Context, id string, req partner.Request) (partner.Partner, error) {
	valid, err := req.Validate()
	if err != nil {
		return partner.Partner{}, err
	}
	return s.repo.Update(ctx, id, valid)
}

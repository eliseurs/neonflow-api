package service

import (
	"context"
	"time"

	"neonflow-api/internal/domain/terms"
)

type TermsService struct {
	repo     terms.Repository
	now      func() time.Time
	newToken func() (string, string, error)
}

func NewTermsService(repo terms.Repository) *TermsService {
	return &TermsService{repo: repo, now: time.Now, newToken: terms.NewToken}
}

func (s *TermsService) Versions(ctx context.Context) ([]terms.Version, error) {
	return s.repo.Versions(ctx)
}

// Publish cria uma versão nova do texto (o texto é do jurídico).
func (s *TermsService) Publish(ctx context.Context, body, userID string) (terms.Version, error) {
	valid, err := terms.ValidateBody(body)
	if err != nil {
		return terms.Version{}, err
	}
	return s.repo.CreateVersion(ctx, valid, userID)
}

// CreateLink gera o link de aceite da reserva, se houver o que aceitar (Ordem 04 publicada ou termos)
// e a retirada não aconteceu.
func (s *TermsService) CreateLink(ctx context.Context, bookingID, userID string) (terms.Link, error) {
	order, err := s.repo.PublishedOrder(ctx)
	if err != nil {
		return terms.Link{}, err
	}
	if order == nil {
		if _, err := s.repo.Current(ctx); err != nil {
			return terms.Link{}, err
		}
	}
	stage, err := s.repo.BookingStage(ctx, bookingID)
	if err != nil {
		return terms.Link{}, err
	}
	if !terms.LinkAllowed(stage) {
		return terms.Link{}, terms.ErrWrongStage
	}
	token, hash, err := s.newToken()
	if err != nil {
		return terms.Link{}, err
	}
	expires := s.now().Add(terms.LinkTTL)
	if err := s.repo.CreateLink(ctx, bookingID, hash, expires, userID); err != nil {
		return terms.Link{}, err
	}
	return terms.Link{Token: token, Path: "/termos/" + token, ExpiresAt: expires}, nil
}

func (s *TermsService) BookingAcceptance(ctx context.Context, bookingID string) (*terms.Acceptance, error) {
	return s.repo.LastAcceptance(ctx, bookingID)
}

// Public é o que o cliente vê pelo link.
func (s *TermsService) Public(ctx context.Context, token string) (terms.Public, error) {
	link, err := s.repo.LinkByHash(ctx, terms.HashToken(token))
	if err != nil {
		return terms.Public{}, err
	}
	return s.repo.Public(ctx, link)
}

// Accept registra o aceite: link válido, reserva antes da retirada e a versão que o cliente leu.
func (s *TermsService) Accept(ctx context.Context, token string, req terms.AcceptRequest) (terms.Acceptance, error) {
	valid, err := req.Validate()
	if err != nil {
		return terms.Acceptance{}, err
	}
	link, err := s.repo.LinkByHash(ctx, terms.HashToken(token))
	if err != nil {
		return terms.Acceptance{}, err
	}
	if !terms.LinkAllowed(link.BookingStage) {
		return terms.Acceptance{}, terms.ErrLinkNotFound
	}
	if valid.IsOrder() {
		return s.repo.AcceptOrder(ctx, link, valid)
	}
	// Com a Ordem 04 publicada, os termos antigos não valem mais: a página tem de ser recarregada.
	order, err := s.repo.PublishedOrder(ctx)
	if err != nil {
		return terms.Acceptance{}, err
	}
	if order != nil {
		return terms.Acceptance{}, terms.ErrVersionChanged
	}
	return s.repo.Accept(ctx, link, valid.Version, valid.TypedName)
}

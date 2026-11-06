package service

import (
	"context"
	"strings"
	"time"

	"neonflow-api/internal/domain/transfer"
)

type TransferService struct {
	repo transfer.Repository
	now  func() time.Time
}

func NewTransferService(repo transfer.Repository) *TransferService {
	return &TransferService{repo: repo, now: time.Now}
}

func (s *TransferService) List(ctx context.Context) ([]transfer.Transfer, error) {
	return s.repo.List(ctx)
}

func (s *TransferService) Get(ctx context.Context, id string) (transfer.Transfer, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *TransferService) Events(ctx context.Context, id string) ([]transfer.Event, error) {
	return s.repo.Events(ctx, id)
}

func (s *TransferService) Create(ctx context.Context, req transfer.CreateRequest, userID string) (transfer.Transfer, error) {
	n, err := req.Validate(userID)
	if err != nil {
		return transfer.Transfer{}, err
	}
	return s.repo.Create(ctx, n)
}

// Drivers: quem pode fazer a corrida, com as outras corridas do dia e os carros de cada um.
func (s *TransferService) Drivers(ctx context.Context, id string) ([]transfer.Driver, error) {
	return s.repo.Drivers(ctx, id)
}

// Offer define ou aumenta o valor para o motorista; o repository confere com a corrida travada (RN-02.22).
func (s *TransferService) Offer(ctx context.Context, id, userID string, req transfer.OfferRequest) (transfer.Transfer, error) {
	if err := transfer.ValidateOffer(req.OfferCents, nil); err != nil {
		return transfer.Transfer{}, err
	}
	return s.repo.Offer(ctx, id, userID, req.OfferCents)
}

// Notify registra que a central chamou o motorista pelo WhatsApp.
func (s *TransferService) Notify(ctx context.Context, id, userID string, req transfer.NotifyRequest) (transfer.Transfer, error) {
	driver := strings.TrimSpace(req.DriverPartnerID)
	if driver == "" {
		return transfer.Transfer{}, transfer.ErrDriverNotFound
	}
	return s.repo.Notify(ctx, id, userID, driver)
}

// Accept: o repository valida com a corrida travada (valor atual, motorista e carro).
func (s *TransferService) Accept(ctx context.Context, id, userID string, req transfer.AcceptRequest) (transfer.Transfer, error) {
	return s.repo.Accept(ctx, id, userID, req)
}

func (s *TransferService) Release(ctx context.Context, id, userID string) (transfer.Transfer, error) {
	return s.repo.Release(ctx, id, userID)
}

func (s *TransferService) SetPaid(ctx context.Context, id, userID string, paid bool) (transfer.Transfer, error) {
	return s.repo.SetPaid(ctx, id, userID, paid)
}

func (s *TransferService) Start(ctx context.Context, id, userID string) (transfer.Transfer, error) {
	return s.repo.Start(ctx, id, userID)
}

func (s *TransferService) Complete(ctx context.Context, id, userID string, req transfer.CompleteRequest) (transfer.Transfer, error) {
	if req.ExtraWaitCents < 0 {
		return transfer.Transfer{}, transfer.ErrInvalidExtraWait
	}
	return s.repo.Complete(ctx, id, userID, req.ExtraWaitCents)
}

// Cancel calcula a multa pela hora de agora (RN-02.10) e a grava com a corrida.
func (s *TransferService) Cancel(ctx context.Context, id, userID string, req transfer.CancelRequest) (transfer.Transfer, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return transfer.Transfer{}, err
	}
	if !transfer.Cancellable(t.Stage) {
		return transfer.Transfer{}, transfer.ErrWrongStage
	}
	fee := transfer.CancelFee(t.PriceCents, t.PickupAt, s.now(), req.FlightCancelled)
	return s.repo.Cancel(ctx, id, userID, fee, req.FlightCancelled)
}

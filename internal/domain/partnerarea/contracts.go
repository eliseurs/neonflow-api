package partnerarea

import (
	"context"

	"neonflow-api/internal/domain/transfer"
)

type Repository interface {
	// Partner é o parceiro da conta, ativo; ErrNoPartner se não houver.
	Partner(ctx context.Context, userID string) (Partner, error)
	Vehicles(ctx context.Context, partnerID string) ([]Vehicle, error)
	Bookings(ctx context.Context, partnerID string) ([]Booking, error)
	Rides(ctx context.Context, partnerID string) ([]Ride, error)
	Referrals(ctx context.Context, partnerID string) ([]Referral, error)
	// Offers são as corridas aguardando motorista com valor definido pela central.
	Offers(ctx context.Context) ([]Offer, error)
	OwnsVehicle(ctx context.Context, partnerID, vehicleID string) (bool, error)
}

// Transfers é o transfer da central: aceite e desistência passam pelas mesmas regras (trava, valor, histórico).
type Transfers interface {
	Get(ctx context.Context, id string) (transfer.Transfer, error)
	Accept(ctx context.Context, id, userID string, req transfer.AcceptRequest) (transfer.Transfer, error)
	Release(ctx context.Context, id, userID string) (transfer.Transfer, error)
}

type Service interface {
	Area(ctx context.Context, userID string) (Area, error)
	Accept(ctx context.Context, userID, transferID string, req AcceptRequest) error
	Release(ctx context.Context, userID, transferID string) error
}

type Logger interface {
	Error(msg string, args ...any)
}

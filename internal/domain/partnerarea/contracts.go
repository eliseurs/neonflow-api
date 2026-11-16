package partnerarea

import "context"

type Repository interface {
	// Partner é o parceiro da conta, ativo; ErrNoPartner se não houver.
	Partner(ctx context.Context, userID string) (Partner, error)
	Vehicles(ctx context.Context, partnerID string) ([]Vehicle, error)
	Bookings(ctx context.Context, partnerID string) ([]Booking, error)
	Rides(ctx context.Context, partnerID string) ([]Ride, error)
	Referrals(ctx context.Context, partnerID string) ([]Referral, error)
}

type Service interface {
	Area(ctx context.Context, userID string) (Area, error)
}

type Logger interface {
	Error(msg string, args ...any)
}

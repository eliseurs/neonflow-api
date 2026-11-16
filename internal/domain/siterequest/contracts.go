package siterequest

import (
	"context"

	"neonflow-api/internal/domain/booking"
	"neonflow-api/internal/domain/fleet"
	"neonflow-api/internal/domain/transfer"
	"neonflow-api/internal/infra/email"
)

type Repository interface {
	// Account é a conta de cliente ativa, com o cadastro ligado; ErrNoAccount se não houver.
	Account(ctx context.Context, userID string) (Account, error)
	VehicleTerms(ctx context.Context, vehicleID string) (VehicleTerms, error)
	// PeriodTaken diz se o carro já tem reserva ativa no período.
	PeriodTaken(ctx context.Context, vehicleID string, p booking.Period) (bool, error)
	Create(ctx context.Context, n NewRequest) (Request, error)
	List(ctx context.Context, status, customerID string) ([]Request, error)
	Find(ctx context.Context, id string) (Detail, error)
	// Transition muda a situação se ela for uma de from, grava a mensagem e o histórico.
	Transition(ctx context.Context, id string, from []string, to, note, userID string) (Request, error)
	// Resubmit troca os dados do pedido em ajuste e o devolve para a central.
	Resubmit(ctx context.Context, id string, n NewRequest) (Request, error)
	// Claim reserva o pedido para a confirmação; Release desfaz se a reserva ou o transfer falhar.
	Claim(ctx context.Context, id string) error
	Release(ctx context.Context, id string) error
	// Complete liga a reserva ou o transfer criado e grava o histórico da confirmação.
	Complete(ctx context.Context, id, bookingID, transferID, userID string) (Request, error)
}

type Bookings interface {
	Create(ctx context.Context, req booking.CreateRequest, userID string) (booking.Booking, error)
	Available(ctx context.Context, start, end string) ([]fleet.Vehicle, error)
}

type Transfers interface {
	Create(ctx context.Context, req transfer.CreateRequest, userID string) (transfer.Transfer, error)
}

type Mailer interface {
	Enabled() bool
	Send(ctx context.Context, m email.Message) error
}

type Service interface {
	Cars(ctx context.Context, start, end string) ([]ClientCar, error)
	Create(ctx context.Context, userID string, req CreateRequest) (Request, error)
	Mine(ctx context.Context, userID string) ([]Request, error)
	MineDetail(ctx context.Context, userID, id string) (Detail, error)
	Resubmit(ctx context.Context, userID, id string, req CreateRequest) (Request, error)
	CancelMine(ctx context.Context, userID, id string) (Request, error)

	List(ctx context.Context, status string) ([]Request, error)
	Get(ctx context.Context, id string) (Detail, error)
	Confirm(ctx context.Context, userID, id string, req ConfirmRequest) (Request, error)
	Adjust(ctx context.Context, userID, id string, req NoteRequest) (Request, error)
	Refuse(ctx context.Context, userID, id string, req NoteRequest) (Request, error)
}

type Logger interface {
	Error(msg string, args ...any)
}

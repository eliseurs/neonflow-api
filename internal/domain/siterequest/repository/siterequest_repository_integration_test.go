//go:build integration

package repository

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/booking"
	bookingRepo "neonflow-api/internal/domain/booking/repository"
	bookingService "neonflow-api/internal/domain/booking/service"
	"neonflow-api/internal/domain/siterequest"
	"neonflow-api/internal/domain/siterequest/service"
	transferRepo "neonflow-api/internal/domain/transfer/repository"
	transferService "neonflow-api/internal/domain/transfer/service"
	"neonflow-api/internal/infra/email"
	"neonflow-api/internal/testutil"
)

type seed struct {
	central, client, other, customer, vehicle string
}

func seedDB(t *testing.T, db *sql.DB) seed {
	t.Helper()
	ctx := context.Background()
	var s seed
	q := func(dest *string, query string, args ...any) {
		require.NoError(t, db.QueryRowContext(ctx, query, args...).Scan(dest))
	}
	q(&s.central, `INSERT INTO users (name, email, password_hash, role) VALUES ('Central', 'central@neonflow.test', 'h', 'central') RETURNING id`)
	q(&s.customer, `INSERT INTO customers (name, email) VALUES ('Ana Teste', 'ana@neonflow.test') RETURNING id`)
	q(&s.client, `INSERT INTO users (name, email, password_hash, role, customer_id, email_verified_at) VALUES ('Ana', 'ana@neonflow.test', 'h', 'cliente', $1, now()) RETURNING id`, s.customer)
	q(&s.other, `INSERT INTO users (name, email, password_hash, role) VALUES ('Sem confirmar', 'x@neonflow.test', 'h', 'cliente') RETURNING id`)
	q(&s.vehicle, `INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents, deposit_cents) VALUES ('ABC1D23', 'Jetta Teste', 'economy', 'Dono', 5, 'manual', 20000, 35000) RETURNING id`)
	_, err := db.ExecContext(ctx, `INSERT INTO partners (name, is_referrer, referral_code) VALUES ('Hotel', true, 'HTL234')`)
	require.NoError(t, err)
	return s
}

func TestSiteRequestRepository_Integration(t *testing.T) {
	db := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewSiteRequestRepository(db)
	ctx := context.Background()
	s := seedDB(t, db)

	acc, err := repo.Account(ctx, s.client)
	require.NoError(t, err)
	assert.Equal(t, s.customer, acc.CustomerID)
	for _, id := range []string{s.other, s.central, "nao-e-uuid"} {
		_, err = repo.Account(ctx, id)
		assert.ErrorIs(t, err, siterequest.ErrNoAccount, "só cliente confirmado e ligado a um cadastro pede")
	}

	terms, err := repo.VehicleTerms(ctx, s.vehicle)
	require.NoError(t, err)
	assert.Equal(t, int64(20000), terms.DailyRateCents)
	assert.Equal(t, int64(35000), *terms.DepositCents)
	_, err = repo.VehicleTerms(ctx, "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, booking.ErrVehicleNotFound)

	start := time.Now().Add(72 * time.Hour).Truncate(time.Minute)
	p := booking.Period{Start: start, End: start.Add(48 * time.Hour)}
	quote, err := siterequest.QuoteFor(terms, p, "")
	require.NoError(t, err)
	rental, err := repo.Create(ctx, siterequest.NewRequest{Kind: "aluguel", CustomerID: s.customer, UserID: s.client, WhatsApp: "5554999990001",
		ReferralCode: "HTL234", VehicleID: s.vehicle, Period: p, Quote: quote})
	require.NoError(t, err)
	assert.Equal(t, siterequest.StatusRequested, rental.Status)
	assert.Equal(t, "Jetta Teste", rental.VehicleName)
	assert.Equal(t, "Ana Teste", rental.CustomerName)
	assert.Equal(t, "ana@neonflow.test", rental.Email)
	assert.NotNil(t, rental.ReferredBy, "o código vira o parceiro")
	assert.Equal(t, int64(40000), *rental.TotalCents)

	// O pedido não bloqueia: o período continua livre até a central confirmar (PEN-73).
	taken, err := repo.PeriodTaken(ctx, s.vehicle, p)
	require.NoError(t, err)
	assert.False(t, taken)

	trf, err := repo.Create(ctx, siterequest.NewRequest{Kind: "transfer", CustomerID: s.customer, UserID: s.client, WhatsApp: "5554999990001",
		ReferralCode: "NAOEXISTE", Origin: "POA", Destination: "Gramado", PickupAt: start, Passengers: 2})
	require.NoError(t, err)
	assert.Nil(t, trf.ReferredBy, "código desconhecido fica de fora")
	assert.Nil(t, trf.TotalCents)

	mine, err := repo.List(ctx, "", s.customer)
	require.NoError(t, err)
	assert.Len(t, mine, 2)
	only, err := repo.List(ctx, "ajuste_pedido", "")
	require.NoError(t, err)
	assert.Empty(t, only)

	// Ajuste, reenvio e histórico.
	adj, err := repo.Transition(ctx, rental.ID, []string{siterequest.StatusRequested}, siterequest.StatusAdjustment, "Pode ser um dia depois?", s.central)
	require.NoError(t, err)
	assert.Equal(t, "Pode ser um dia depois?", adj.CentralNote)
	_, err = repo.Transition(ctx, rental.ID, []string{siterequest.StatusRequested}, siterequest.StatusRefused, "x", s.central)
	assert.ErrorIs(t, err, siterequest.ErrStatusChanged)
	_, err = repo.Transition(ctx, "00000000-0000-0000-0000-000000000000", []string{siterequest.StatusRequested}, siterequest.StatusRefused, "x", s.central)
	assert.ErrorIs(t, err, siterequest.ErrNotFound)

	later := booking.Period{Start: p.Start.Add(24 * time.Hour), End: p.End.Add(24 * time.Hour)}
	back, err := repo.Resubmit(ctx, rental.ID, siterequest.NewRequest{Kind: "aluguel", UserID: s.client, WhatsApp: "5554999990002", VehicleID: s.vehicle, Period: later, Quote: quote})
	require.NoError(t, err)
	assert.Equal(t, siterequest.StatusRequested, back.Status)
	assert.Equal(t, "5554999990002", back.WhatsApp)
	_, err = repo.Resubmit(ctx, rental.ID, siterequest.NewRequest{Kind: "aluguel", UserID: s.client, WhatsApp: "5554999990002", VehicleID: s.vehicle, Period: later, Quote: quote})
	assert.ErrorIs(t, err, siterequest.ErrStatusChanged, "só reenvia o que está em ajuste")

	d, err := repo.Find(ctx, rental.ID)
	require.NoError(t, err)
	require.Len(t, d.Events, 3)
	assert.Nil(t, d.Events[0].FromStatus)
	assert.Equal(t, siterequest.StatusAdjustment, d.Events[1].ToStatus)
	assert.Equal(t, "Pode ser um dia depois?", d.Events[1].Note)
	assert.Equal(t, siterequest.StatusRequested, d.Events[2].ToStatus)
	_, err = repo.Find(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, siterequest.ErrNotFound)

	// Claim e Release.
	require.NoError(t, repo.Claim(ctx, trf.ID))
	assert.ErrorIs(t, repo.Claim(ctx, trf.ID), siterequest.ErrStatusChanged, "dois cliques não confirmam duas vezes")
	require.NoError(t, repo.Release(ctx, trf.ID))
	d, err = repo.Find(ctx, trf.ID)
	require.NoError(t, err)
	assert.Equal(t, siterequest.StatusRequested, d.Status, "o Release devolve o pedido a solicitado")
	_, err = repo.Complete(ctx, trf.ID, "", "", s.central)
	assert.ErrorIs(t, err, siterequest.ErrStatusChanged, "só completa o que foi reservado pelo Claim")

	// O banco recusa pedido de aluguel sem carro.
	_, err = db.ExecContext(ctx, `INSERT INTO site_requests (kind, customer_id, user_id, whatsapp) VALUES ('aluguel', $1, $2, '5554999990001')`, s.customer, s.client)
	assert.Error(t, err)
}

// A confirmação de ponta a ponta, com os serviços reais de reserva e de transfer.
func TestSiteRequestService_ConfirmIntegration(t *testing.T) {
	db := testutil.MigratedDB(t, "../../../../sql/migrations")
	ctx := context.Background()
	s := seedDB(t, db)
	svc := service.NewSiteRequestService(NewSiteRequestRepository(db),
		bookingService.NewBookingService(bookingRepo.NewBookingRepository(db)),
		transferService.NewTransferService(transferRepo.NewTransferRepository(db)),
		email.Disabled{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "")

	start := time.Now().Add(72 * time.Hour).In(time.UTC).Format("2006-01-02T15:04:05Z")
	end := time.Now().Add(120 * time.Hour).In(time.UTC).Format("2006-01-02T15:04:05Z")
	req := siterequest.CreateRequest{Kind: "aluguel", WhatsApp: "5554999990001", VehicleID: s.vehicle, StartAt: start, EndAt: end, ReferralCode: "htl234"}

	cars, err := svc.Cars(ctx, start, end)
	require.NoError(t, err)
	require.Len(t, cars, 1)
	assert.Equal(t, int64(40000), *cars[0].TotalCents)

	first, err := svc.Create(ctx, s.client, req)
	require.NoError(t, err)
	second, err := svc.Create(ctx, s.client, req)
	require.NoError(t, err, "dois pedidos para o mesmo período podem chegar")

	ok, err := svc.Confirm(ctx, s.central, first.ID, siterequest.ConfirmRequest{})
	require.NoError(t, err)
	require.NotNil(t, ok.BookingID)
	b, err := bookingRepo.NewBookingRepository(db).FindByID(ctx, *ok.BookingID)
	require.NoError(t, err)
	assert.Equal(t, booking.StagePreparacao, b.Stage, "a reserva nasce como a da central: bloqueia e abre as 24 horas")
	assert.Equal(t, int64(40000), b.TotalCents)
	assert.Equal(t, s.customer, *b.CustomerID)
	assert.NotNil(t, b.ReferredByPartnerID)

	_, err = svc.Confirm(ctx, s.central, second.ID, siterequest.ConfirmRequest{})
	assert.ErrorIs(t, err, booking.ErrPeriodUnavailable, "o segundo não confirma no mesmo período")
	d, err := svc.Get(ctx, second.ID)
	require.NoError(t, err)
	assert.Equal(t, siterequest.StatusRequested, d.Status, "volta a solicitado para a central recusar ou pedir ajuste")

	_, err = svc.Create(ctx, s.client, req)
	assert.ErrorIs(t, err, booking.ErrPeriodUnavailable, "depois da confirmação, o período reservado é recusado na hora")

	pickup := time.Now().Add(96 * time.Hour).In(time.UTC).Format("2006-01-02T15:04:05Z")
	trf, err := svc.Create(ctx, s.client, siterequest.CreateRequest{Kind: "transfer", WhatsApp: "5554999990001", Origin: "POA", Destination: "Gramado", PickupAt: pickup, Passengers: 2})
	require.NoError(t, err)
	done, err := svc.Confirm(ctx, s.central, trf.ID, siterequest.ConfirmRequest{PriceCents: 35000})
	require.NoError(t, err)
	require.NotNil(t, done.TransferID)
	var price int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT price_cents FROM transfers WHERE id = $1`, *done.TransferID).Scan(&price))
	assert.Equal(t, int64(35000), price)
}

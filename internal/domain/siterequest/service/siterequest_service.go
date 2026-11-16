package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"neonflow-api/internal/domain/booking"
	"neonflow-api/internal/domain/siterequest"
	"neonflow-api/internal/domain/transfer"
	"neonflow-api/internal/infra/email"
	"neonflow-api/internal/pkg/timezone"
)

type SiteRequestService struct {
	repo      siterequest.Repository
	bookings  siterequest.Bookings
	transfers siterequest.Transfers
	mail      siterequest.Mailer
	log       siterequest.Logger
	appURL    string
	now       func() time.Time
}

func NewSiteRequestService(repo siterequest.Repository, bookings siterequest.Bookings, transfers siterequest.Transfers,
	mail siterequest.Mailer, log siterequest.Logger, appURL string) *SiteRequestService {
	return &SiteRequestService{repo: repo, bookings: bookings, transfers: transfers, mail: mail, log: log, appURL: appURL, now: time.Now}
}

// Cars lista os carros livres no período com a estimativa de cada um (RN-02.34).
func (s *SiteRequestService) Cars(ctx context.Context, start, end string) ([]siterequest.ClientCar, error) {
	p, err := booking.ParsePeriod(start, end)
	if err != nil {
		return nil, err
	}
	vehicles, err := s.bookings.Available(ctx, start, end)
	if err != nil {
		return nil, err
	}
	cars := make([]siterequest.ClientCar, 0, len(vehicles))
	for _, v := range vehicles {
		q, err := siterequest.QuoteFor(siterequest.VehicleTerms{
			Name: v.Name, DailyRateCents: v.DailyRateCents, DepositCents: v.DepositCents,
			KmLimitPerDay: v.KmLimitPerDay, ExtraKmCents: v.ExtraKmCents, MinDays: 1,
		}, p, "")
		if err != nil {
			continue
		}
		cars = append(cars, siterequest.ClientCar{ID: v.ID, Name: v.Name, Category: v.Category, Seats: v.Seats, Gear: v.Gear, Quote: q})
	}
	return cars, nil
}

// Create grava o pedido do cliente logado. O aluguel leva a estimativa de hoje e não bloqueia o carro;
// um período já reservado é recusado na hora (RN-02.33).
func (s *SiteRequestService) Create(ctx context.Context, userID string, req siterequest.CreateRequest) (siterequest.Request, error) {
	acc, err := s.repo.Account(ctx, userID)
	if err != nil {
		return siterequest.Request{}, err
	}
	n, err := s.prepare(ctx, req)
	if err != nil {
		return siterequest.Request{}, err
	}
	n.CustomerID, n.UserID = acc.CustomerID, acc.UserID
	return s.repo.Create(ctx, n)
}

func (s *SiteRequestService) prepare(ctx context.Context, req siterequest.CreateRequest) (siterequest.NewRequest, error) {
	n, err := req.Validate(s.now())
	if err != nil || n.Kind != siterequest.KindRental {
		return n, err
	}
	n.Quote, err = s.quote(ctx, n.VehicleID, n.Period, n.KmPlan)
	if err != nil {
		return n, err
	}
	taken, err := s.repo.PeriodTaken(ctx, n.VehicleID, n.Period)
	if err != nil {
		return n, err
	}
	if taken {
		return n, booking.ErrPeriodUnavailable
	}
	return n, nil
}

func (s *SiteRequestService) quote(ctx context.Context, vehicleID string, p booking.Period, kmPlan string) (siterequest.Quote, error) {
	terms, err := s.repo.VehicleTerms(ctx, vehicleID)
	if err != nil {
		return siterequest.Quote{}, err
	}
	return siterequest.QuoteFor(terms, p, kmPlan)
}

func (s *SiteRequestService) Mine(ctx context.Context, userID string) ([]siterequest.Request, error) {
	acc, err := s.repo.Account(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, "", acc.CustomerID)
}

// MineDetail só abre pedido do próprio cliente; o de outro responde como inexistente.
func (s *SiteRequestService) MineDetail(ctx context.Context, userID, id string) (siterequest.Detail, error) {
	acc, err := s.repo.Account(ctx, userID)
	if err != nil {
		return siterequest.Detail{}, err
	}
	d, err := s.repo.Find(ctx, id)
	if err != nil {
		return siterequest.Detail{}, err
	}
	if d.CustomerID != acc.CustomerID {
		return siterequest.Detail{}, siterequest.ErrNotFound
	}
	return d, nil
}

// Resubmit: o cliente altera o pedido em ajuste e ele volta para a central (RN-02.37).
func (s *SiteRequestService) Resubmit(ctx context.Context, userID, id string, req siterequest.CreateRequest) (siterequest.Request, error) {
	d, err := s.MineDetail(ctx, userID, id)
	if err != nil {
		return siterequest.Request{}, err
	}
	if d.Status != siterequest.StatusAdjustment {
		return siterequest.Request{}, siterequest.ErrStatusChanged
	}
	if strings.TrimSpace(req.Kind) != d.Kind {
		return siterequest.Request{}, siterequest.ErrKindChanged
	}
	n, err := s.prepare(ctx, req)
	if err != nil {
		return siterequest.Request{}, err
	}
	n.UserID = userID
	return s.repo.Resubmit(ctx, id, n)
}

func (s *SiteRequestService) CancelMine(ctx context.Context, userID, id string) (siterequest.Request, error) {
	if _, err := s.MineDetail(ctx, userID, id); err != nil {
		return siterequest.Request{}, err
	}
	return s.repo.Transition(ctx, id, []string{siterequest.StatusRequested, siterequest.StatusAdjustment}, siterequest.StatusCanceled, "", userID)
}

func (s *SiteRequestService) List(ctx context.Context, status string) ([]siterequest.Request, error) {
	return s.repo.List(ctx, strings.TrimSpace(status), "")
}

func (s *SiteRequestService) Get(ctx context.Context, id string) (siterequest.Detail, error) {
	return s.repo.Find(ctx, id)
}

// Confirm cria a reserva (que agora bloqueia o carro e abre as 24 horas para pagar, RN-02.11) ou o transfer
// com o preço da central. O pedido fica reservado durante a criação, para não confirmar duas vezes.
func (s *SiteRequestService) Confirm(ctx context.Context, userID, id string, req siterequest.ConfirmRequest) (siterequest.Request, error) {
	d, err := s.repo.Find(ctx, id)
	if err != nil {
		return siterequest.Request{}, err
	}
	if d.Status != siterequest.StatusRequested {
		return siterequest.Request{}, siterequest.ErrStatusChanged
	}
	if d.Kind == siterequest.KindTransfer && req.PriceCents <= 0 {
		return siterequest.Request{}, siterequest.ErrPriceRequired
	}
	if d.Kind == siterequest.KindRental {
		now, err := s.quote(ctx, *d.VehicleID, booking.Period{Start: *d.StartAt, End: *d.EndAt}, d.KmPlan)
		if err != nil {
			return siterequest.Request{}, err
		}
		if !now.Same(d.Quote) {
			return siterequest.Request{}, siterequest.ErrPriceChanged
		}
	}

	if err := s.repo.Claim(ctx, id); err != nil {
		return siterequest.Request{}, err
	}
	bookingID, transferID, err := s.create(ctx, userID, d.Request, req)
	if err != nil {
		if rerr := s.repo.Release(ctx, id); rerr != nil {
			s.log.Error("site request release failed", "error", rerr)
		}
		return siterequest.Request{}, err
	}
	r, err := s.repo.Complete(ctx, id, bookingID, transferID, userID)
	if err != nil {
		return siterequest.Request{}, err
	}
	s.notify(ctx, d.Request, "Pedido confirmado", confirmedText(d.Request, req))
	return r, nil
}

func (s *SiteRequestService) create(ctx context.Context, userID string, r siterequest.Request, req siterequest.ConfirmRequest) (string, string, error) {
	referred := ""
	if r.ReferredBy != nil {
		referred = *r.ReferredBy
	}
	if r.Kind == siterequest.KindRental {
		b, err := s.bookings.Create(ctx, booking.CreateRequest{
			VehicleID: *r.VehicleID, ClientName: r.CustomerName, StartAt: r.StartAt.Format(time.RFC3339), EndAt: r.EndAt.Format(time.RFC3339),
			Place: r.Place, KmPlan: r.KmPlan, ReferredByPartnerID: referred, CustomerID: r.CustomerID,
		}, userID)
		return b.ID, "", err
	}
	t, err := s.transfers.Create(ctx, transfer.CreateRequest{
		ClientName: r.CustomerName, PickupAt: r.PickupAt.In(timezone.Location).Format(transfer.DateTimeLayout), Origin: r.Origin, Destination: r.Destination,
		Flight: r.Flight, Passengers: *r.Passengers, Notes: r.Notes, PriceCents: req.PriceCents, ReferredByPartnerID: referred, CustomerID: r.CustomerID,
	}, userID)
	return "", t.ID, err
}

// Adjust devolve o pedido ao cliente com o que mudar (RN-02.37).
func (s *SiteRequestService) Adjust(ctx context.Context, userID, id string, req siterequest.NoteRequest) (siterequest.Request, error) {
	note := strings.TrimSpace(req.Note)
	if note == "" {
		return siterequest.Request{}, siterequest.ErrNoteRequired
	}
	r, err := s.repo.Transition(ctx, id, []string{siterequest.StatusRequested}, siterequest.StatusAdjustment, note, userID)
	if err != nil {
		return r, err
	}
	s.notify(ctx, r, "A central pediu um ajuste no seu pedido",
		fmt.Sprintf("A central pediu um ajuste no seu pedido de %s:\n\n%s\n\nAltere o pedido em Minha conta e envie de novo:", describe(r), note))
	return r, nil
}

func (s *SiteRequestService) Refuse(ctx context.Context, userID, id string, req siterequest.NoteRequest) (siterequest.Request, error) {
	note := strings.TrimSpace(req.Note)
	if note == "" {
		return siterequest.Request{}, siterequest.ErrNoteRequired
	}
	r, err := s.repo.Transition(ctx, id, []string{siterequest.StatusRequested, siterequest.StatusAdjustment}, siterequest.StatusRefused, note, userID)
	if err != nil {
		return r, err
	}
	s.notify(ctx, r, "Seu pedido não pôde ser atendido",
		fmt.Sprintf("Infelizmente o seu pedido de %s não pôde ser atendido:\n\n%s\n\nSe quiser, faça um pedido novo em Minha conta:", describe(r), note))
	return r, nil
}

// notify manda o e-mail ao cliente (RN-02.36). A ação da central já valeu: a falha só vai para o log.
func (s *SiteRequestService) notify(ctx context.Context, r siterequest.Request, subject, body string) {
	if !s.mail.Enabled() || r.Email == "" {
		return
	}
	err := s.mail.Send(ctx, email.Message{
		To:      r.Email,
		Subject: subject + " · Neon Flow",
		Text:    fmt.Sprintf("Olá, %s.\n\n%s\n\n%s/minha-conta\n\nNeon Flow", r.CustomerName, body, s.appURL),
	})
	if err != nil {
		s.log.Error("site request email failed", "subject", subject, "error", err)
	}
}

func confirmedText(r siterequest.Request, req siterequest.ConfirmRequest) string {
	if r.Kind == siterequest.KindTransfer {
		return fmt.Sprintf("A central confirmou o seu %s, no valor de %s. Os próximos passos estão em Minha conta:", describe(r), transfer.FormatReais(req.PriceCents))
	}
	return fmt.Sprintf("A central confirmou o seu %s, no valor de %s. O carro está reservado; o pagamento precisa ser feito em até 24 horas. "+
		"Os próximos passos estão em Minha conta:", describe(r), transfer.FormatReais(*r.TotalCents))
}

func describe(r siterequest.Request) string {
	if r.Kind == siterequest.KindTransfer {
		return fmt.Sprintf("transfer de %s para %s em %s", r.Origin, r.Destination, local(*r.PickupAt))
	}
	return fmt.Sprintf("aluguel do %s de %s a %s", r.VehicleName, local(*r.StartAt), local(*r.EndAt))
}

func local(t time.Time) string { return t.In(timezone.Location).Format("02/01/2006 15:04") }

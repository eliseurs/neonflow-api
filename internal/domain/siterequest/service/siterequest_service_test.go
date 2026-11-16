package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/account"
	"neonflow-api/internal/domain/booking"
	"neonflow-api/internal/domain/fleet"
	"neonflow-api/internal/domain/siterequest"
	"neonflow-api/internal/domain/transfer"
	"neonflow-api/internal/infra/email"
)

const carID = "11111111-1111-1111-1111-111111111111"

var now = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

type fakeRepo struct {
	account    siterequest.Account
	accountErr error
	terms      siterequest.VehicleTerms
	taken      bool
	created    siterequest.NewRequest
	detail     siterequest.Detail
	moved      []string
	note       string
	claimErr   error
	released   bool
	completed  [2]string
	resubmit   siterequest.NewRequest
	guest      siterequest.Guest
	guestErr   error
	tokens     []account.Token
}

func (f *fakeRepo) Guest(context.Context, string, string) (siterequest.Guest, error) {
	return f.guest, f.guestErr
}
func (f *fakeRepo) SaveToken(_ context.Context, t account.Token) error {
	f.tokens = append(f.tokens, t)
	return nil
}

func (f *fakeRepo) Account(context.Context, string) (siterequest.Account, error) {
	return f.account, f.accountErr
}
func (f *fakeRepo) VehicleTerms(context.Context, string) (siterequest.VehicleTerms, error) {
	return f.terms, nil
}
func (f *fakeRepo) PeriodTaken(context.Context, string, booking.Period) (bool, error) {
	return f.taken, nil
}
func (f *fakeRepo) Create(_ context.Context, n siterequest.NewRequest) (siterequest.Request, error) {
	f.created = n
	start, end := n.Period.Start, n.Period.End
	return siterequest.Request{ID: "r1", Kind: n.Kind, Status: siterequest.StatusRequested, VehicleName: "Jetta",
		StartAt: &start, EndAt: &end, Quote: n.Quote}, nil
}
func (f *fakeRepo) List(_ context.Context, status, customerID string) ([]siterequest.Request, error) {
	f.moved = append(f.moved, "list:"+status+":"+customerID)
	return nil, nil
}
func (f *fakeRepo) Find(context.Context, string) (siterequest.Detail, error) {
	if f.detail.ID == "" {
		return siterequest.Detail{}, siterequest.ErrNotFound
	}
	return f.detail, nil
}
func (f *fakeRepo) Transition(_ context.Context, _ string, from []string, to, note, _ string) (siterequest.Request, error) {
	f.moved = append(f.moved, to)
	f.note = note
	r := f.detail.Request
	r.Status, r.CentralNote = to, note
	return r, nil
}
func (f *fakeRepo) Resubmit(_ context.Context, _ string, n siterequest.NewRequest) (siterequest.Request, error) {
	f.resubmit = n
	return siterequest.Request{Status: siterequest.StatusRequested}, nil
}
func (f *fakeRepo) Claim(context.Context, string) error   { return f.claimErr }
func (f *fakeRepo) Release(context.Context, string) error { f.released = true; return nil }
func (f *fakeRepo) Complete(_ context.Context, _, bookingID, transferID, _ string) (siterequest.Request, error) {
	f.completed = [2]string{bookingID, transferID}
	return siterequest.Request{Status: siterequest.StatusConfirmed}, nil
}

type fakeBookings struct {
	created booking.CreateRequest
	err     error
	cars    []fleet.Vehicle
}

func (f *fakeBookings) Create(_ context.Context, req booking.CreateRequest, _ string) (booking.Booking, error) {
	f.created = req
	return booking.Booking{ID: "b1"}, f.err
}
func (f *fakeBookings) Available(context.Context, string, string) ([]fleet.Vehicle, error) {
	return f.cars, nil
}

type fakeTransfers struct{ created transfer.CreateRequest }

func (f *fakeTransfers) Create(_ context.Context, req transfer.CreateRequest, _ string) (transfer.Transfer, error) {
	f.created = req
	return transfer.Transfer{ID: "t1"}, nil
}

type fakeMail struct {
	sent []email.Message
	err  error
	off  bool
}

func (m *fakeMail) Enabled() bool { return !m.off }
func (m *fakeMail) Send(_ context.Context, msg email.Message) error {
	m.sent = append(m.sent, msg)
	return m.err
}

type spyLog struct{ msgs []string }

func (s *spyLog) Error(msg string, _ ...any) { s.msgs = append(s.msgs, msg) }

type deps struct {
	repo      *fakeRepo
	bookings  *fakeBookings
	transfers *fakeTransfers
	mail      *fakeMail
	log       *spyLog
}

func newSvc(repo *fakeRepo) (*SiteRequestService, deps) {
	d := deps{repo: repo, bookings: &fakeBookings{}, transfers: &fakeTransfers{}, mail: &fakeMail{}, log: &spyLog{}}
	s := NewSiteRequestService(repo, d.bookings, d.transfers, d.mail, d.log, "https://app.neonflow.test")
	s.now = func() time.Time { return now }
	return s, d
}

var clientAccount = siterequest.Account{UserID: "u1", CustomerID: "c1", Name: "Ana"}

func rentalReq() siterequest.CreateRequest {
	return siterequest.CreateRequest{Kind: "aluguel", WhatsApp: "5554999990001", VehicleID: carID, StartAt: "2026-12-10T10:00", EndAt: "2026-12-12T10:00", ReferralCode: "htl234"}
}

func ptr[T any](v T) *T { return &v }

func TestCreate_Rental(t *testing.T) {
	repo := &fakeRepo{account: clientAccount, terms: siterequest.VehicleTerms{Name: "Jetta", DailyRateCents: 20000, MinDays: 1}}
	svc, _ := newSvc(repo)
	_, err := svc.Create(context.Background(), "u1", rentalReq())
	require.NoError(t, err)
	assert.Equal(t, "c1", repo.created.CustomerID)
	assert.Equal(t, "u1", repo.created.UserID)
	assert.Equal(t, int64(40000), *repo.created.Quote.TotalCents, "estimativa gravada no pedido (RN-02.34)")
	assert.Equal(t, "HTL234", repo.created.ReferralCode)

	repo.taken = true
	_, err = svc.Create(context.Background(), "u1", rentalReq())
	assert.ErrorIs(t, err, booking.ErrPeriodUnavailable, "período já reservado é recusado na hora")

	svc, _ = newSvc(&fakeRepo{accountErr: siterequest.ErrNoAccount})
	_, err = svc.Create(context.Background(), "u1", rentalReq())
	assert.ErrorIs(t, err, siterequest.ErrNoAccount)

	svc, _ = newSvc(&fakeRepo{account: clientAccount, terms: siterequest.VehicleTerms{DailyRateCents: 1, MinDays: 5}})
	_, err = svc.Create(context.Background(), "u1", rentalReq())
	assert.ErrorIs(t, err, booking.ErrBelowMinDays)
}

func TestCreate_Transfer(t *testing.T) {
	repo := &fakeRepo{account: clientAccount}
	svc, _ := newSvc(repo)
	_, err := svc.Create(context.Background(), "u1", siterequest.CreateRequest{Kind: "transfer", WhatsApp: "5554999990001", Origin: "POA", Destination: "Gramado",
		PickupAt: "2026-12-10T08:30", Passengers: 2})
	require.NoError(t, err)
	assert.Nil(t, repo.created.Quote.TotalCents, "transfer sem preço: a central define (RN-02.35)")
}

func TestCars(t *testing.T) {
	svc, d := newSvc(&fakeRepo{})
	d.bookings.cars = []fleet.Vehicle{{ID: "v1", Name: "Jetta", Plate: "ABC1D23", DailyRateCents: 20000}}
	cars, err := svc.Cars(context.Background(), "2026-12-10T10:00", "2026-12-11T10:00")
	require.NoError(t, err)
	require.Len(t, cars, 1)
	assert.Equal(t, int64(20000), *cars[0].TotalCents)
	_, err = svc.Cars(context.Background(), "x", "y")
	assert.ErrorIs(t, err, booking.ErrInvalidDate)
}

func rentalDetail() siterequest.Detail {
	start := time.Date(2026, 12, 10, 13, 0, 0, 0, time.UTC)
	return siterequest.Detail{Request: siterequest.Request{
		ID: "r1", Kind: "aluguel", Status: siterequest.StatusRequested, CustomerID: "c1", CustomerName: "Ana", Email: "ana@neonflow.test",
		VehicleID: ptr(carID), VehicleName: "Jetta", StartAt: &start, EndAt: ptr(start.Add(48 * time.Hour)), ReferredBy: ptr("p1"),
		Quote: siterequest.Quote{Days: ptr(2), DailyRateCents: ptr(int64(20000)), TotalCents: ptr(int64(40000))},
	}}
}

func TestConfirm_Rental(t *testing.T) {
	repo := &fakeRepo{detail: rentalDetail(), terms: siterequest.VehicleTerms{DailyRateCents: 20000, MinDays: 1}}
	svc, d := newSvc(repo)
	r, err := svc.Confirm(context.Background(), "central", "r1", siterequest.ConfirmRequest{})
	require.NoError(t, err)
	assert.Equal(t, siterequest.StatusConfirmed, r.Status)
	assert.Equal(t, [2]string{"b1", ""}, repo.completed)
	assert.Equal(t, "Ana", d.bookings.created.ClientName)
	assert.Equal(t, "c1", d.bookings.created.CustomerID)
	assert.Equal(t, "p1", d.bookings.created.ReferredByPartnerID)
	require.Len(t, d.mail.sent, 1)
	assert.Equal(t, "ana@neonflow.test", d.mail.sent[0].To)
	assert.Contains(t, d.mail.sent[0].Text, "R$ 400,00")
	assert.Contains(t, d.mail.sent[0].Text, "https://app.neonflow.test/minha-conta")
}

func TestConfirm_Errors(t *testing.T) {
	// A diária mudou depois do pedido: a central pede ajuste.
	repo := &fakeRepo{detail: rentalDetail(), terms: siterequest.VehicleTerms{DailyRateCents: 25000, MinDays: 1}}
	svc, _ := newSvc(repo)
	_, err := svc.Confirm(context.Background(), "central", "r1", siterequest.ConfirmRequest{})
	assert.ErrorIs(t, err, siterequest.ErrPriceChanged)

	// O período foi ocupado por outra reserva: devolve o pedido a "solicitado".
	repo = &fakeRepo{detail: rentalDetail(), terms: siterequest.VehicleTerms{DailyRateCents: 20000, MinDays: 1}}
	svc, d := newSvc(repo)
	d.bookings.err = booking.ErrPeriodUnavailable
	_, err = svc.Confirm(context.Background(), "central", "r1", siterequest.ConfirmRequest{})
	assert.ErrorIs(t, err, booking.ErrPeriodUnavailable)
	assert.True(t, repo.released)
	assert.Empty(t, d.mail.sent)

	// Dois cliques: o segundo não confirma.
	repo = &fakeRepo{detail: rentalDetail(), terms: siterequest.VehicleTerms{DailyRateCents: 20000, MinDays: 1}, claimErr: siterequest.ErrStatusChanged}
	svc, d = newSvc(repo)
	_, err = svc.Confirm(context.Background(), "central", "r1", siterequest.ConfirmRequest{})
	assert.ErrorIs(t, err, siterequest.ErrStatusChanged)
	assert.Empty(t, d.bookings.created.VehicleID)

	done := rentalDetail()
	done.Status = siterequest.StatusRefused
	svc, _ = newSvc(&fakeRepo{detail: done})
	_, err = svc.Confirm(context.Background(), "central", "r1", siterequest.ConfirmRequest{})
	assert.ErrorIs(t, err, siterequest.ErrStatusChanged)

	svc, _ = newSvc(&fakeRepo{})
	_, err = svc.Confirm(context.Background(), "central", "x", siterequest.ConfirmRequest{})
	assert.ErrorIs(t, err, siterequest.ErrNotFound)
}

func TestConfirm_Transfer(t *testing.T) {
	pickup := time.Date(2026, 12, 10, 11, 30, 0, 0, time.UTC)
	repo := &fakeRepo{detail: siterequest.Detail{Request: siterequest.Request{
		ID: "r2", Kind: "transfer", Status: siterequest.StatusRequested, CustomerID: "c1", CustomerName: "Ana", Email: "ana@neonflow.test",
		Origin: "POA", Destination: "Gramado", PickupAt: &pickup, Passengers: ptr(2), Flight: "G3 1",
	}}}
	svc, d := newSvc(repo)
	_, err := svc.Confirm(context.Background(), "central", "r2", siterequest.ConfirmRequest{})
	assert.ErrorIs(t, err, siterequest.ErrPriceRequired)

	_, err = svc.Confirm(context.Background(), "central", "r2", siterequest.ConfirmRequest{PriceCents: 35000})
	require.NoError(t, err)
	assert.Equal(t, [2]string{"", "t1"}, repo.completed)
	assert.Equal(t, int64(35000), d.transfers.created.PriceCents)
	assert.Equal(t, 2, d.transfers.created.Passengers)
	assert.Equal(t, "2026-12-10T08:30", d.transfers.created.PickupAt, "o transfer lê a hora de Gramado")
	assert.Contains(t, d.mail.sent[0].Text, "R$ 350,00")
	assert.Contains(t, d.mail.sent[0].Text, "transfer de POA para Gramado em 10/12/2026 08:30", "hora de Gramado")
}

func TestAdjustAndRefuse(t *testing.T) {
	repo := &fakeRepo{detail: rentalDetail()}
	svc, d := newSvc(repo)
	_, err := svc.Adjust(context.Background(), "central", "r1", siterequest.NoteRequest{Note: " "})
	assert.ErrorIs(t, err, siterequest.ErrNoteRequired)

	r, err := svc.Adjust(context.Background(), "central", "r1", siterequest.NoteRequest{Note: " Pode ser no dia 11? "})
	require.NoError(t, err)
	assert.Equal(t, siterequest.StatusAdjustment, r.Status)
	assert.Equal(t, "Pode ser no dia 11?", repo.note)
	assert.Contains(t, d.mail.sent[0].Text, "Pode ser no dia 11?")

	d.mail.err = errors.New("resend: 403")
	r, err = svc.Refuse(context.Background(), "central", "r1", siterequest.NoteRequest{Note: "Carro em manutenção"})
	require.NoError(t, err, "a falha do e-mail não desfaz a resposta da central")
	assert.Equal(t, siterequest.StatusRefused, r.Status)
	assert.Equal(t, []string{"site request email failed"}, d.log.msgs)
	_, err = svc.Refuse(context.Background(), "central", "r1", siterequest.NoteRequest{})
	assert.ErrorIs(t, err, siterequest.ErrNoteRequired)
}

func TestClientRequests(t *testing.T) {
	adjust := rentalDetail()
	adjust.Status = siterequest.StatusAdjustment
	repo := &fakeRepo{account: clientAccount, detail: adjust, terms: siterequest.VehicleTerms{DailyRateCents: 20000, MinDays: 1}}
	svc, _ := newSvc(repo)

	_, err := svc.Mine(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, []string{"list::c1"}, repo.moved, "só os pedidos do próprio cliente")

	got, err := svc.MineDetail(context.Background(), "u1", "r1")
	require.NoError(t, err)
	assert.Equal(t, "r1", got.ID)

	_, err = svc.Resubmit(context.Background(), "u1", "r1", rentalReq())
	require.NoError(t, err)
	assert.Equal(t, "u1", repo.resubmit.UserID)
	assert.Equal(t, int64(40000), *repo.resubmit.Quote.TotalCents)

	tr := rentalReq()
	tr.Kind = "transfer"
	_, err = svc.Resubmit(context.Background(), "u1", "r1", tr)
	assert.ErrorIs(t, err, siterequest.ErrKindChanged)

	_, err = svc.CancelMine(context.Background(), "u1", "r1")
	require.NoError(t, err)
	assert.Contains(t, repo.moved, siterequest.StatusCanceled)

	// Pedido de outro cliente responde como inexistente.
	other := &fakeRepo{account: siterequest.Account{UserID: "u2", CustomerID: "c2"}, detail: adjust}
	svc, _ = newSvc(other)
	_, err = svc.MineDetail(context.Background(), "u2", "r1")
	assert.ErrorIs(t, err, siterequest.ErrNotFound)
	_, err = svc.CancelMine(context.Background(), "u2", "r1")
	assert.ErrorIs(t, err, siterequest.ErrNotFound)

	// Só pedido em ajuste volta para a central.
	requested := rentalDetail()
	svc, _ = newSvc(&fakeRepo{account: clientAccount, detail: requested})
	_, err = svc.Resubmit(context.Background(), "u1", "r1", rentalReq())
	assert.ErrorIs(t, err, siterequest.ErrStatusChanged)
}

func TestListAndGet(t *testing.T) {
	repo := &fakeRepo{detail: rentalDetail()}
	svc, _ := newSvc(repo)
	_, err := svc.List(context.Background(), " solicitado ")
	require.NoError(t, err)
	assert.Equal(t, []string{"list:solicitado:"}, repo.moved)
	d, err := svc.Get(context.Background(), "r1")
	require.NoError(t, err)
	assert.Equal(t, "Jetta", d.VehicleName)
}

func TestQuote(t *testing.T) {
	repo := &fakeRepo{terms: siterequest.VehicleTerms{DailyRateCents: 20000, MinDays: 1}}
	svc, _ := newSvc(repo)
	q, err := svc.Quote(context.Background(), carID, "2026-12-10T10:00", "2026-12-12T10:00")
	require.NoError(t, err)
	assert.True(t, q.Available)
	assert.Equal(t, int64(40000), *q.TotalCents)

	repo.taken = true
	q, err = svc.Quote(context.Background(), carID, "2026-12-10T10:00", "2026-12-12T10:00")
	require.NoError(t, err)
	assert.False(t, q.Available, "período reservado: mostra o valor, mas indisponível")

	_, err = svc.Quote(context.Background(), carID, "x", "y")
	assert.ErrorIs(t, err, booking.ErrInvalidDate)
	svc, _ = newSvc(&fakeRepo{terms: siterequest.VehicleTerms{DailyRateCents: 1, MinDays: 5}})
	_, err = svc.Quote(context.Background(), carID, "2026-12-10T10:00", "2026-12-12T10:00")
	assert.ErrorIs(t, err, booking.ErrBelowMinDays)
}

var guestReq = siterequest.GuestRequest{Name: " Bia Teste ", Email: " BIA@neonflow.test ", WhatsApp: "5554999990002",
	VehicleID: carID, StartAt: "2026-12-10T10:00", EndAt: "2026-12-12T10:00"}

func TestGuest(t *testing.T) {
	repo := &fakeRepo{terms: siterequest.VehicleTerms{Name: "Jetta", DailyRateCents: 20000, MinDays: 1}, guest: siterequest.Guest{UserID: "u9", Name: "Bia Teste"}}
	svc, d := newSvc(repo)
	require.NoError(t, svc.Guest(context.Background(), guestReq))
	assert.Equal(t, siterequest.StatusAwaitingEmail, repo.created.Status, "a central só vê depois do clique no e-mail")
	assert.Equal(t, "u9", repo.created.UserID)
	assert.Empty(t, repo.created.CustomerID, "o cadastro nasce na confirmação")
	require.Len(t, repo.tokens, 1)
	assert.Equal(t, account.PurposeConfirmEmail, repo.tokens[0].Purpose)
	require.Len(t, d.mail.sent, 1)
	msg := d.mail.sent[0]
	assert.Equal(t, "bia@neonflow.test", msg.To)
	assert.Contains(t, msg.Text, "R$ 400,00")
	assert.Contains(t, msg.Text, "https://app.neonflow.test/conta/confirmar/")
	i := strings.Index(msg.Text, "/conta/confirmar/") + len("/conta/confirmar/")
	assert.Equal(t, repo.tokens[0].Hash, account.HashToken(msg.Text[i:i+43]))
}

func TestGuest_Errors(t *testing.T) {
	svc, _ := newSvc(&fakeRepo{})
	svc.mail = &fakeMail{off: true}
	assert.ErrorIs(t, svc.Guest(context.Background(), guestReq), email.ErrNotConfigured)

	svc, _ = newSvc(&fakeRepo{})
	bad := guestReq
	bad.Name = " "
	assert.ErrorIs(t, svc.Guest(context.Background(), bad), siterequest.ErrNameRequired)
	bad = guestReq
	bad.Email = "Bia <bia@neonflow.test>"
	assert.ErrorIs(t, svc.Guest(context.Background(), bad), siterequest.ErrInvalidEmail)
	bad = guestReq
	bad.WhatsApp = "1"
	assert.ErrorIs(t, svc.Guest(context.Background(), bad), siterequest.ErrInvalidWhatsApp)

	svc, _ = newSvc(&fakeRepo{terms: siterequest.VehicleTerms{DailyRateCents: 1, MinDays: 1}, taken: true})
	assert.ErrorIs(t, svc.Guest(context.Background(), guestReq), booking.ErrPeriodUnavailable)

	svc, _ = newSvc(&fakeRepo{terms: siterequest.VehicleTerms{DailyRateCents: 1, MinDays: 1}, guestErr: siterequest.ErrEmailInUse})
	assert.ErrorIs(t, svc.Guest(context.Background(), guestReq), siterequest.ErrEmailInUse)

	svc, d := newSvc(&fakeRepo{terms: siterequest.VehicleTerms{DailyRateCents: 1, MinDays: 1}})
	d.mail.err = errors.New("resend: 403")
	assert.ErrorIs(t, svc.Guest(context.Background(), guestReq), siterequest.ErrSendFailed)
	assert.Equal(t, []string{"site request email failed"}, d.log.msgs)
}

//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/booking"
	bookingRepo "neonflow-api/internal/domain/booking/repository"
	"neonflow-api/internal/domain/contract"
	contractRepo "neonflow-api/internal/domain/contract/repository"
	"neonflow-api/internal/domain/terms"
	"neonflow-api/internal/testutil"
)

func TestTermsRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewTermsRepository(database)
	bookings := bookingRepo.NewBookingRepository(database)
	ctx := context.Background()

	var userID, vehicleID string
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('Admin', 'admin@neonflow.test', 'h', 'admin') RETURNING id`).Scan(&userID))
	require.NoError(t, database.QueryRowContext(ctx, `INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents)
		VALUES ('TST1A23', 'Sedã Teste', 'economy', 'Neon Flow', 5, 'manual', 10000) RETURNING id`).Scan(&vehicleID))
	p, err := booking.ParsePeriod("2026-12-10T10:00", "2026-12-12T10:00")
	require.NoError(t, err)
	b, err := bookings.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Cliente Teste", Period: p, CreatedBy: userID})
	require.NoError(t, err)

	_, err = repo.Current(ctx)
	assert.ErrorIs(t, err, terms.ErrNoTerms)
	v1, err := repo.CreateVersion(ctx, "Termos v1", userID)
	require.NoError(t, err)
	assert.Equal(t, 1, v1.Version)
	v2, err := repo.CreateVersion(ctx, "Termos v2", userID)
	require.NoError(t, err)
	assert.Equal(t, 2, v2.Version)
	list, err := repo.Versions(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, 2, list[0].Version, "mais nova primeiro")

	stage, err := repo.BookingStage(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, booking.StagePreparacao, stage)
	_, err = repo.BookingStage(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, terms.ErrBookingNotFound)

	token, hash, err := terms.NewToken()
	require.NoError(t, err)
	require.NoError(t, repo.CreateLink(ctx, b.ID, hash, time.Now().Add(time.Hour), userID))
	link, err := repo.LinkByHash(ctx, terms.HashToken(token))
	require.NoError(t, err)
	assert.Equal(t, b.ID, link.BookingID)
	_, err = repo.LinkByHash(ctx, terms.HashToken("outro"))
	assert.ErrorIs(t, err, terms.ErrLinkNotFound)
	_, oldHash, _ := terms.NewToken()
	require.NoError(t, repo.CreateLink(ctx, b.ID, oldHash, time.Now().Add(-time.Minute), userID))
	_, err = repo.LinkByHash(ctx, oldHash)
	assert.ErrorIs(t, err, terms.ErrLinkNotFound, "link vencido")

	pub, err := repo.Public(ctx, link)
	require.NoError(t, err)
	assert.Equal(t, "Cliente Teste", pub.ClientName)
	assert.Equal(t, "Sedã Teste", pub.VehicleName)
	assert.Equal(t, int64(20000), pub.TotalCents)
	assert.Equal(t, 2, pub.Terms.Version)
	assert.Nil(t, pub.Accepted)

	// Aceitar a versão antiga é recusado; a atual marca a conferência "terms" da reserva.
	_, err = repo.Accept(ctx, link, 1, "Cliente Teste")
	assert.ErrorIs(t, err, terms.ErrVersionChanged)
	acc, err := repo.Accept(ctx, link, 2, "Cliente Teste")
	require.NoError(t, err)
	assert.Equal(t, 2, acc.Version)
	after, err := bookings.FindByID(ctx, b.ID)
	require.NoError(t, err)
	assert.True(t, after.Checks[terms.TermsCheck])
	last, err := repo.LastAcceptance(ctx, b.ID)
	require.NoError(t, err)
	require.NotNil(t, last)
	assert.Equal(t, "Cliente Teste", last.TypedName)
	pub, err = repo.Public(ctx, link)
	require.NoError(t, err)
	require.NotNil(t, pub.Accepted)

	none, err := repo.LastAcceptance(ctx, "00000000-0000-0000-0000-000000000000")
	require.NoError(t, err)
	assert.Nil(t, none)
	_, err = repo.LastAcceptance(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, terms.ErrBookingNotFound)
	_, err = repo.Public(ctx, terms.LinkInfo{BookingID: "00000000-0000-0000-0000-000000000000"})
	assert.ErrorIs(t, err, terms.ErrLinkNotFound)

	require.NoError(t, database.Close())
	_, err = repo.Versions(ctx)
	assert.Error(t, err)
	_, err = repo.Current(ctx)
	assert.Error(t, err)
	_, err = repo.CreateVersion(ctx, "x", userID)
	assert.Error(t, err)
	assert.Error(t, repo.CreateLink(ctx, b.ID, "h", time.Now(), userID))
	_, err = repo.Accept(ctx, link, 2, "Cliente Teste")
	assert.Error(t, err)
	_, err = repo.LastAcceptance(ctx, b.ID)
	assert.Error(t, err)
}

func TestTermsRepository_AcceptWithoutTerms(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewTermsRepository(database)
	_, err := repo.Accept(context.Background(), terms.LinkInfo{}, 1, "Ana Teste")
	assert.ErrorIs(t, err, terms.ErrNoTerms)
}

func TestTermsRepository_Order(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewTermsRepository(database)
	contracts := contractRepo.NewContractRepository(database)
	bookings := bookingRepo.NewBookingRepository(database)
	ctx := context.Background()

	var adminID, vehicleID string
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('Admin', 'admin@neonflow.test', 'h', 'admin') RETURNING id`).Scan(&adminID))
	require.NoError(t, database.QueryRowContext(ctx, `INSERT INTO vehicles (plate, name, category, owner, seats, gear, daily_rate_cents, deposit_cents, km_limit_per_day, extra_km_cents)
		VALUES ('TST1A23', 'Sedã Teste', 'economy', 'Neon Flow', 5, 'manual', 10000, 35000, 200, 150) RETURNING id`).Scan(&vehicleID))
	p, err := booking.ParsePeriod("2026-12-10T10:00", "2026-12-12T10:00")
	require.NoError(t, err)
	b, err := bookings.Create(ctx, booking.NewBooking{VehicleID: vehicleID, ClientName: "Cliente Teste", Period: p, KmPlan: booking.KmPlanLimitado, CreatedBy: adminID})
	require.NoError(t, err)
	_, hash, err := terms.NewToken()
	require.NoError(t, err)
	require.NoError(t, repo.CreateLink(ctx, b.ID, hash, time.Now().Add(time.Hour), adminID))
	link, err := repo.LinkByHash(ctx, hash)
	require.NoError(t, err)

	// Sem Ordem publicada e sem termos: nada a aceitar.
	order, err := repo.PublishedOrder(ctx)
	require.NoError(t, err)
	assert.Nil(t, order)
	_, err = repo.Public(ctx, link)
	assert.ErrorIs(t, err, terms.ErrNoTerms)

	// Uma versão só vale depois de publicada (aprovada não basta).
	admin := contract.Actor{UserID: adminID, Role: authentication.RoleAdmin}
	v, err := contracts.CreateVersion(ctx, terms.OrderCode, contract.VersionRequest{Label: "v0.4", Body: "Ordem v1"}, contract.Hash("Ordem v1"), admin)
	require.NoError(t, err)
	for _, action := range []string{contract.ActionEnviar, contract.ActionAprovar, contract.ActionLiberar} {
		_, err := contracts.AddEvent(ctx, v.ID, action, "", admin, func(current, sha string) (string, error) {
			return contract.Next(contract.EventRequest{Action: action, SHA256: sha}, admin, current, sha)
		})
		require.NoError(t, err, action)
	}
	order, err = repo.PublishedOrder(ctx)
	require.NoError(t, err)
	assert.Nil(t, order, "liberada ainda não é publicada")
	_, err = contracts.AddEvent(ctx, v.ID, contract.ActionPublicar, "", admin, func(current, sha string) (string, error) {
		return contract.Next(contract.EventRequest{Action: contract.ActionPublicar}, admin, current, sha)
	})
	require.NoError(t, err)

	pub, err := repo.Public(ctx, link)
	require.NoError(t, err)
	require.NotNil(t, pub.Order)
	assert.Nil(t, pub.Terms, "a Ordem substitui os termos")
	assert.Equal(t, v.ID, pub.Order.VersionID)
	assert.Equal(t, contract.Hash("Ordem v1"), pub.Order.SHA256)
	require.NotNil(t, pub.DepositCents)
	assert.Equal(t, int64(35000), *pub.DepositCents, "caução copiada na reserva")
	require.NotNil(t, pub.KmLimitPerDay)
	assert.Equal(t, 200, *pub.KmLimitPerDay)
	assert.Len(t, pub.QuadroSHA256, 64)

	req := terms.AcceptRequest{TypedName: "Cliente Teste", ContractVersionID: pub.Order.VersionID, SHA256: pub.Order.SHA256, QuadroSHA256: pub.QuadroSHA256, Confirm: true}
	bad := req
	bad.SHA256 = contract.Hash("outro texto")
	_, err = repo.AcceptOrder(ctx, link, bad)
	assert.ErrorIs(t, err, terms.ErrVersionChanged, "hash do texto diferente")
	bad = req
	bad.QuadroSHA256 = contract.Hash("outro quadro")
	_, err = repo.AcceptOrder(ctx, link, bad)
	assert.ErrorIs(t, err, terms.ErrVersionChanged, "quadro mudou desde a leitura")

	acc, err := repo.AcceptOrder(ctx, link, req)
	require.NoError(t, err)
	assert.Equal(t, terms.DocOrder, acc.Document)
	assert.Equal(t, 1, acc.Version)
	after, err := bookings.FindByID(ctx, b.ID)
	require.NoError(t, err)
	assert.True(t, after.Checks[terms.TermsCheck], "aceite marca a conferência")
	last, err := repo.LastAcceptance(ctx, b.ID)
	require.NoError(t, err)
	require.NotNil(t, last)
	assert.Equal(t, terms.DocOrder, last.Document)
	assert.Equal(t, contract.Hash("Ordem v1"), last.SHA256)

	var method string
	var confirmed bool
	var snapshot []byte
	require.NoError(t, database.QueryRowContext(ctx, `SELECT method, confirmed, snapshot FROM contract_acceptances WHERE booking_id = $1`, b.ID).Scan(&method, &confirmed, &snapshot))
	assert.Equal(t, terms.MethodLink, method)
	assert.True(t, confirmed)
	assert.Contains(t, string(snapshot), `"client_name": "Cliente Teste"`)
	_, err = database.ExecContext(ctx, `UPDATE contract_acceptances SET typed_name = 'outro' WHERE booking_id = $1`, b.ID)
	assert.Error(t, err, "aceite é append-only")

	require.NoError(t, database.Close())
	_, err = repo.PublishedOrder(ctx)
	assert.Error(t, err)
	_, err = repo.AcceptOrder(ctx, link, req)
	assert.Error(t, err)
}

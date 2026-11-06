package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"neonflow-api/internal/domain/dashboard"
)

// Pré-reserva não paga e cancelada não entram no faturamento; o pago no mês é o valor das diárias
// das reservas com pagamento marcado no mês e não canceladas (o estorno é manual, RN-03.11).
const queryTotals = `
	SELECT
		coalesce(sum(total_cents + extra_cents + excess_km_cents) FILTER (WHERE start_at >= $1 AND start_at < $2
			AND stage IN ('confirmada','em_locacao','devolvida','encerrada')), 0),
		count(*) FILTER (WHERE start_at >= $1 AND start_at < $2
			AND stage IN ('confirmada','em_locacao','devolvida','encerrada')),
		coalesce(sum(total_cents) FILTER (WHERE paid_at >= $1 AND paid_at < $2 AND stage <> 'cancelada'), 0),
		count(*) FILTER (WHERE paid_at >= $1 AND paid_at < $2 AND stage <> 'cancelada'),
		count(DISTINCT vehicle_id) FILTER (WHERE stage = 'em_locacao'),
		(SELECT count(*) FROM vehicles)
	FROM bookings`

type DashboardRepository struct {
	db *sql.DB
}

func NewDashboardRepository(db *sql.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

func (r *DashboardRepository) Totals(ctx context.Context, start, end time.Time) (dashboard.Totals, error) {
	var t dashboard.Totals
	err := r.db.QueryRowContext(ctx, queryTotals, start, end).Scan(&t.PickupRevenueCents, &t.PickupCount,
		&t.PaidCents, &t.PaidCount, &t.RentedToday, &t.FleetSize)
	if err != nil {
		return t, fmt.Errorf("dashboard totals: %w", err)
	}
	return t, nil
}

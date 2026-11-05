-- Necessário para o EXCLUDE com igualdade de UUID e sobreposição de daterange.
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Reservas de aluguel (RN-02). Período em datas, semiaberto [start_date, end_date):
-- datas adjacentes não se sobrepõem (RN-02.2). Horário e fuso dependem de PEN-29.
CREATE TABLE IF NOT EXISTS bookings (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id       UUID        NOT NULL REFERENCES vehicles (id),
    client_name      TEXT        NOT NULL,
    start_date       DATE        NOT NULL,
    end_date         DATE        NOT NULL,
    place            TEXT        NOT NULL DEFAULT '',
    stage            TEXT        NOT NULL DEFAULT 'preparacao'
                     CHECK (stage IN ('preparacao', 'confirmada', 'em_locacao', 'devolvida', 'encerrada', 'cancelada')),
    days             INT         NOT NULL CHECK (days >= 1),
    daily_rate_cents BIGINT      NOT NULL CHECK (daily_rate_cents > 0),
    total_cents      BIGINT      NOT NULL CHECK (total_cents > 0),
    checks           JSONB       NOT NULL DEFAULT '{}'::jsonb,
    expires_at       TIMESTAMPTZ NOT NULL,
    created_by       UUID        REFERENCES users (id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_date > start_date),
    -- Bloqueio atômico (RN-02.1, RN-02.2, PEN-24): duas reservas ativas do mesmo veículo não se sobrepõem.
    CONSTRAINT bookings_no_overlap EXCLUDE USING gist (
        vehicle_id WITH =,
        daterange(start_date, end_date) WITH &&
    ) WHERE (stage NOT IN ('encerrada', 'cancelada'))
);

CREATE INDEX IF NOT EXISTS bookings_vehicle_idx ON bookings (vehicle_id);
CREATE INDEX IF NOT EXISTS bookings_pending_expiry_idx ON bookings (expires_at) WHERE stage = 'preparacao';

-- Histórico de cada transição de etapa. user_id nulo = transição automática (expiração, RN-02.11).
CREATE TABLE IF NOT EXISTS booking_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID        NOT NULL REFERENCES bookings (id) ON DELETE CASCADE,
    from_stage TEXT,
    to_stage   TEXT        NOT NULL,
    user_id    UUID        REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS booking_events_booking_idx ON booking_events (booking_id, created_at);

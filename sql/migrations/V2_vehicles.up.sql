-- Frota ofertável (RN-01). A diária fica no veículo e é cadastrada pela central (RN-01.2, RN-01.5).
CREATE TABLE IF NOT EXISTS vehicles (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plate            TEXT        NOT NULL,
    name             TEXT        NOT NULL,
    category         TEXT        NOT NULL CHECK (category IN ('economy', 'premium')),
    owner            TEXT        NOT NULL,
    seats            INT         NOT NULL CHECK (seats > 0),
    gear             TEXT        NOT NULL CHECK (gear IN ('manual', 'automatico')),
    daily_rate_cents BIGINT      NOT NULL CHECK (daily_rate_cents > 0),
    in_maintenance   BOOLEAN     NOT NULL DEFAULT false,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS vehicles_plate_key ON vehicles (plate);

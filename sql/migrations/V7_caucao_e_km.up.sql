-- Caução por veículo (RN-06.4) e quilometragem (RN-02.18). Valores configuráveis pela central,
-- sem padrão no código (RN-01.5): nulo significa "não informado".
ALTER TABLE vehicles
    ADD COLUMN deposit_cents    BIGINT CHECK (deposit_cents > 0),
    ADD COLUMN km_limit_per_day INT    CHECK (km_limit_per_day > 0),
    ADD COLUMN extra_km_cents   BIGINT CHECK (extra_km_cents > 0),
    -- Limite e valor do km excedente andam juntos.
    ADD CONSTRAINT vehicles_km_plan_check CHECK ((km_limit_per_day IS NULL) = (extra_km_cents IS NULL));

-- Na reserva, cópia dos valores do veículo na criação (como a diária, RN-02.6). km_limit_per_day
-- nulo = km livre. O odômetro é registrado na entrega e na devolução; o excedente é calculado na devolução.
ALTER TABLE bookings
    ADD COLUMN deposit_cents    BIGINT CHECK (deposit_cents > 0),
    ADD COLUMN km_limit_per_day INT    CHECK (km_limit_per_day > 0),
    ADD COLUMN extra_km_cents   BIGINT CHECK (extra_km_cents > 0),
    ADD COLUMN delivered_km     INT    CHECK (delivered_km >= 0),
    ADD COLUMN returned_km      INT    CHECK (returned_km >= 0),
    ADD COLUMN excess_km        INT    NOT NULL DEFAULT 0 CHECK (excess_km >= 0),
    ADD COLUMN excess_km_cents  BIGINT NOT NULL DEFAULT 0 CHECK (excess_km_cents >= 0),
    ADD CONSTRAINT bookings_km_plan_check CHECK ((km_limit_per_day IS NULL) = (extra_km_cents IS NULL)),
    ADD CONSTRAINT bookings_km_order_check CHECK (returned_km IS NULL OR delivered_km IS NULL OR returned_km >= delivered_km);

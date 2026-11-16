-- Pedidos feitos pelo cliente no site (RN-02.33 a RN-02.37): aluguel ou transfer. O pedido não bloqueia
-- o carro; quando a central confirma, nasce a reserva (que bloqueia, PEN-73) ou o transfer.
CREATE TABLE IF NOT EXISTS site_requests (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind                   TEXT        NOT NULL CHECK (kind IN ('aluguel', 'transfer')),
    status                 TEXT        NOT NULL DEFAULT 'solicitado'
                           CHECK (status IN ('solicitado', 'ajuste_pedido', 'confirmado', 'recusado', 'cancelado')),
    customer_id            UUID        NOT NULL REFERENCES customers (id),
    user_id                UUID        NOT NULL REFERENCES users (id),
    -- WhatsApp do pedido, só dígitos com DDI: a central avisa o cliente por ele (RN-02.36).
    whatsapp               TEXT        NOT NULL CHECK (whatsapp ~ '^[0-9]{10,15}$'),
    notes                  TEXT        NOT NULL DEFAULT '',
    referred_by_partner_id UUID REFERENCES partners (id),
    -- Aluguel: carro, período e a estimativa mostrada ao cliente (RN-02.34).
    vehicle_id             UUID REFERENCES vehicles (id),
    start_at               TIMESTAMPTZ,
    end_at                 TIMESTAMPTZ,
    place                  TEXT        NOT NULL DEFAULT '',
    km_plan                TEXT        NOT NULL DEFAULT '' CHECK (km_plan IN ('', 'livre', 'limitado')),
    days                   INT,
    daily_rate_cents       BIGINT,
    total_cents            BIGINT,
    deposit_cents          BIGINT,
    km_limit_per_day       INT,
    extra_km_cents         BIGINT,
    -- Transfer: o preço é da central, na confirmação (RN-02.35).
    origin                 TEXT        NOT NULL DEFAULT '',
    destination            TEXT        NOT NULL DEFAULT '',
    pickup_at              TIMESTAMPTZ,
    passengers             INT,
    flight                 TEXT        NOT NULL DEFAULT '',
    -- Última mensagem da central ao cliente (ajuste ou recusa).
    central_note           TEXT        NOT NULL DEFAULT '',
    booking_id             UUID REFERENCES bookings (id),
    transfer_id            UUID REFERENCES transfers (id),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT site_requests_kind_fields CHECK (
        CASE kind
            WHEN 'aluguel' THEN vehicle_id IS NOT NULL AND start_at IS NOT NULL AND end_at > start_at
                                AND days > 0 AND daily_rate_cents > 0 AND total_cents > 0
            ELSE pickup_at IS NOT NULL AND length(trim(origin)) > 0 AND length(trim(destination)) > 0
                 AND passengers > 0
        END
    )
);

CREATE INDEX IF NOT EXISTS site_requests_status_idx ON site_requests (status, created_at DESC);
CREATE INDEX IF NOT EXISTS site_requests_customer_idx ON site_requests (customer_id, created_at DESC);

-- Histórico do pedido: cada mudança de situação, com a mensagem e quem fez. Só inclusão.
CREATE TABLE IF NOT EXISTS site_request_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id  UUID        NOT NULL REFERENCES site_requests (id),
    from_status TEXT,
    to_status   TEXT        NOT NULL,
    note        TEXT        NOT NULL DEFAULT '',
    user_id     UUID        NOT NULL REFERENCES users (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX IF NOT EXISTS site_request_events_request_idx ON site_request_events (request_id, created_at);

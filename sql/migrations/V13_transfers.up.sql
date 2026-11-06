-- Transfer operado pela central (RN-02.9, 02.10, 02.16, 02.17, 02.22, 02.27, 02.30, 02.31; RN-03.9, 03.14).
-- Preço sob consulta (q29). A corrida é oferecida aos motoristas com valor e prazo definidos pela central;
-- quem aceita primeiro leva e recebe o valor da oferta, mais a espera extra.
CREATE TABLE IF NOT EXISTS transfers (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_name            TEXT        NOT NULL CHECK (length(trim(client_name)) > 0),
    pickup_at              TIMESTAMPTZ NOT NULL,
    origin                 TEXT        NOT NULL CHECK (length(trim(origin)) > 0),
    destination            TEXT        NOT NULL CHECK (length(trim(destination)) > 0),
    flight                 TEXT        NOT NULL DEFAULT '',
    passengers             INT         NOT NULL DEFAULT 1 CHECK (passengers BETWEEN 1 AND 60),
    notes                  TEXT        NOT NULL DEFAULT '',
    price_cents            BIGINT      NOT NULL CHECK (price_cents > 0),
    stage                  TEXT        NOT NULL DEFAULT 'aguardando_motorista'
        CHECK (stage IN ('aguardando_motorista', 'com_motorista', 'em_andamento', 'concluido', 'cancelado')),
    offer_cents            BIGINT      CHECK (offer_cents > 0),
    offer_expires_at       TIMESTAMPTZ,
    driver_partner_id      UUID        REFERENCES partners (id),
    driver_cents           BIGINT      CHECK (driver_cents > 0),
    paid_at                TIMESTAMPTZ,
    extra_wait_cents       BIGINT      NOT NULL DEFAULT 0 CHECK (extra_wait_cents >= 0),
    cancel_fee_cents       BIGINT      NOT NULL DEFAULT 0 CHECK (cancel_fee_cents >= 0),
    flight_cancelled       BOOLEAN     NOT NULL DEFAULT false,
    referred_by_partner_id UUID        REFERENCES partners (id),
    created_by             UUID        REFERENCES users (id),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT transfers_offer_check CHECK ((offer_cents IS NULL) = (offer_expires_at IS NULL)),
    CONSTRAINT transfers_driver_check CHECK ((driver_partner_id IS NULL) = (driver_cents IS NULL))
);

CREATE INDEX IF NOT EXISTS transfers_pickup_idx ON transfers (pickup_at);

-- Histórico: criação, ofertas, aceite, desistência, pagamento, início, conclusão e cancelamento.
CREATE TABLE IF NOT EXISTS transfer_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transfer_id UUID        NOT NULL REFERENCES transfers (id),
    action      TEXT        NOT NULL,
    detail      TEXT        NOT NULL DEFAULT '',
    user_id     UUID        REFERENCES users (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS transfer_events_transfer_idx ON transfer_events (transfer_id, created_at);

-- Pagamentos a parceiros também nascem do transfer: o motorista recebe valor fixo (oferta + espera).
ALTER TABLE partner_payouts
    ALTER COLUMN booking_id DROP NOT NULL,
    ADD COLUMN transfer_id UUID REFERENCES transfers (id),
    DROP CONSTRAINT partner_payouts_kind_check,
    ADD CONSTRAINT partner_payouts_kind_check CHECK (kind IN ('comissao', 'repasse', 'motorista')),
    ADD CONSTRAINT partner_payouts_source_check CHECK ((booking_id IS NULL) <> (transfer_id IS NULL)),
    ADD CONSTRAINT partner_payouts_transfer_unique UNIQUE (transfer_id, kind),
    DROP CONSTRAINT partner_payouts_amount_check,
    ADD CONSTRAINT partner_payouts_amount_check CHECK (
        CASE WHEN kind = 'motorista' THEN amount_cents IS NOT NULL ELSE (percent_bp IS NULL) = (amount_cents IS NULL) END);

-- Aceite dos termos pelo cliente (RN-02.3, q38): o admin publica o texto em versões; a central gera um
-- link por reserva e envia pelo WhatsApp; o cliente aceita digitando o nome. Registro mínimo: versão,
-- data e hora e o nome digitado (sem IP nem navegador, decisão de 06/11/2026).
CREATE TABLE IF NOT EXISTS terms_versions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version    INT         NOT NULL UNIQUE CHECK (version > 0),
    body       TEXT        NOT NULL CHECK (length(trim(body)) > 0),
    created_by UUID        REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- O link guarda só o hash do código; o código vai na URL e não fica no banco.
CREATE TABLE IF NOT EXISTS terms_links (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID        NOT NULL REFERENCES bookings (id),
    token_hash TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_by UUID        REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS terms_acceptances (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id  UUID        NOT NULL REFERENCES bookings (id),
    version_id  UUID        NOT NULL REFERENCES terms_versions (id),
    link_id     UUID        NOT NULL REFERENCES terms_links (id),
    typed_name  TEXT        NOT NULL CHECK (length(trim(typed_name)) >= 3),
    accepted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS terms_acceptances_booking_idx ON terms_acceptances (booking_id, accepted_at DESC);

-- Aceite da Ordem 04 pela reserva, no padrão V15 (RN-02.3, 04/12.1): versão exata do contrato publicado,
-- hash do texto, quadro da reserva mostrado ao cliente (e seu hash), confirmação expressa, nome digitado
-- e método. Append-only, como os demais registros de contrato (ADR-0015).
CREATE TABLE IF NOT EXISTS contract_acceptances (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id      UUID        NOT NULL REFERENCES bookings (id),
    version_id      UUID        NOT NULL REFERENCES contract_versions (id),
    link_id         UUID        NOT NULL REFERENCES terms_links (id),
    body_sha256     TEXT        NOT NULL CHECK (body_sha256 ~ '^[0-9a-f]{64}$'),
    snapshot        JSONB       NOT NULL,
    snapshot_sha256 TEXT        NOT NULL CHECK (snapshot_sha256 ~ '^[0-9a-f]{64}$'),
    typed_name      TEXT        NOT NULL CHECK (length(trim(typed_name)) >= 3),
    confirmed       BOOLEAN     NOT NULL CHECK (confirmed),
    method          TEXT        NOT NULL CHECK (method IN ('link_reserva')),
    accepted_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS contract_acceptances_booking_idx ON contract_acceptances (booking_id, accepted_at DESC);

DROP TRIGGER IF EXISTS contract_acceptances_append_only ON contract_acceptances;
CREATE TRIGGER contract_acceptances_append_only BEFORE UPDATE OR DELETE ON contract_acceptances
    FOR EACH ROW EXECUTE FUNCTION contract_append_only();

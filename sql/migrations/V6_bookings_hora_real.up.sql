-- Hora real da entrega e da devolução e cobrança do atraso (RN-02.19, RN-02.24).
-- delivered_at e returned_at são informados pela central ao avançar a etapa (padrão: a hora do clique).
-- extra_days e extra_cents guardam a diária extra calculada na devolução, pela diária gravada na reserva.
ALTER TABLE bookings
    ADD COLUMN delivered_at TIMESTAMPTZ,
    ADD COLUMN returned_at  TIMESTAMPTZ,
    ADD COLUMN extra_days   INT    NOT NULL DEFAULT 0 CHECK (extra_days >= 0),
    ADD COLUMN extra_cents  BIGINT NOT NULL DEFAULT 0 CHECK (extra_cents >= 0),
    ADD CONSTRAINT bookings_return_after_delivery CHECK (returned_at IS NULL OR delivered_at IS NULL OR returned_at > delivered_at);

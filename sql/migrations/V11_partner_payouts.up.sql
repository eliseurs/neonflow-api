-- Pagamentos a parceiros (RN-03.12, RN-03.13), gerados quando a reserva é encerrada, ou seja, paga e com
-- o serviço finalizado (RN-03.6, RN-03.8). O percentual do parceiro é copiado na geração; nulo = a
-- definir, copiado de novo no pagamento. A central paga fora do sistema e marca como paga.
--   comissao: indicador, sobre as diárias (total_cents), sai da parte do Neon Flow.
--   repasse:  dono do carro parceiro, sobre diárias + atraso + km excedente; a caução nunca entra.
CREATE TABLE IF NOT EXISTS partner_payouts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id   UUID        NOT NULL REFERENCES bookings (id),
    partner_id   UUID        NOT NULL REFERENCES partners (id),
    kind         TEXT        NOT NULL CHECK (kind IN ('comissao', 'repasse')),
    base_cents   BIGINT      NOT NULL CHECK (base_cents >= 0),
    percent_bp   INT         CHECK (percent_bp BETWEEN 0 AND 10000),
    amount_cents BIGINT      CHECK (amount_cents >= 0),
    paid_at      TIMESTAMPTZ,
    paid_by      UUID        REFERENCES users (id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT partner_payouts_unique UNIQUE (booking_id, kind),
    CONSTRAINT partner_payouts_amount_check CHECK ((percent_bp IS NULL) = (amount_cents IS NULL)),
    CONSTRAINT partner_payouts_paid_check CHECK (paid_at IS NULL OR amount_cents IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS partner_payouts_partner_idx ON partner_payouts (partner_id);

-- Reservas já encerradas antes desta versão.
INSERT INTO partner_payouts (booking_id, partner_id, kind, base_cents, percent_bp, amount_cents)
SELECT b.id, p.id, 'comissao', b.total_cents, p.commission_bp,
       CASE WHEN p.commission_bp IS NULL THEN NULL ELSE (b.total_cents * p.commission_bp + 5000) / 10000 END
FROM bookings b JOIN partners p ON p.id = b.referred_by_partner_id
WHERE b.stage = 'encerrada'
ON CONFLICT (booking_id, kind) DO NOTHING;

INSERT INTO partner_payouts (booking_id, partner_id, kind, base_cents, percent_bp, amount_cents)
SELECT b.id, p.id, 'repasse', b.total_cents + b.extra_cents + b.excess_km_cents, p.owner_share_bp,
       CASE WHEN p.owner_share_bp IS NULL THEN NULL
            ELSE ((b.total_cents + b.extra_cents + b.excess_km_cents) * p.owner_share_bp + 5000) / 10000 END
FROM bookings b JOIN vehicles v ON v.id = b.vehicle_id JOIN partners p ON p.id = v.owner_partner_id
WHERE b.stage = 'encerrada'
ON CONFLICT (booking_id, kind) DO NOTHING;

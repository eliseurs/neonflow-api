-- Parceiros (RN-07): proprietários de veículo, motoristas e indicadores (hotéis, comércio).
-- Qualquer parceiro pode indicar, por link ou QR com o seu código. Só dados básicos:
-- documentos e chave Pix esperam a política de retenção da LGPD (PEN-48).
-- Percentuais em pontos-base (1% = 100), configuráveis e sem padrão no código (RN-03.4, 03.7, 03.9).
CREATE TABLE IF NOT EXISTS partners (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             TEXT        NOT NULL CHECK (length(trim(name)) > 0),
    is_owner         BOOLEAN     NOT NULL DEFAULT false,
    is_driver        BOOLEAN     NOT NULL DEFAULT false,
    is_referrer      BOOLEAN     NOT NULL DEFAULT false,
    whatsapp         TEXT        NOT NULL DEFAULT '',
    email            TEXT        NOT NULL DEFAULT '',
    city             TEXT        NOT NULL DEFAULT '',
    notes            TEXT        NOT NULL DEFAULT '',
    referral_code    TEXT        NOT NULL CHECK (referral_code ~ '^[A-Z0-9]{6,12}$'),
    commission_bp    INT         CHECK (commission_bp BETWEEN 0 AND 10000),
    owner_share_bp   INT         CHECK (owner_share_bp BETWEEN 0 AND 10000),
    driver_share_bp  INT         CHECK (driver_share_bp BETWEEN 0 AND 10000),
    active           BOOLEAN     NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT partners_kind_check CHECK (is_owner OR is_driver OR is_referrer)
);

CREATE UNIQUE INDEX IF NOT EXISTS partners_referral_code_key ON partners (referral_code);

-- Veículo ligado ao parceiro proprietário (o texto "owner" continua por compatibilidade).
ALTER TABLE vehicles ADD COLUMN owner_partner_id UUID REFERENCES partners (id);
-- Reserva com o parceiro que indicou o cliente (origem; comissão vem na fase 2, RN-03.2).
ALTER TABLE bookings ADD COLUMN referred_by_partner_id UUID REFERENCES partners (id);

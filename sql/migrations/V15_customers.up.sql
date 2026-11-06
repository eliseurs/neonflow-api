-- Cadastro de clientes, fase 1 (RN-04.4, RN-04.7): só contato e preferências. Documentos (CPF ou
-- passaporte, CNH, nascimento, endereço) esperam o prazo de retenção do advogado (q75, PEN-48).
CREATE TABLE IF NOT EXISTS customers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT        NOT NULL CHECK (length(trim(name)) > 0),
    whatsapp      TEXT        NOT NULL DEFAULT '',
    email         TEXT        NOT NULL DEFAULT '',
    language      TEXT        NOT NULL DEFAULT 'pt' CHECK (language IN ('pt', 'en')),
    notes         TEXT        NOT NULL DEFAULT '',
    -- Anonimizado a pedido do titular (LGPD art. 18): nome e contatos apagados, pedidos mantidos.
    anonymized_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Um cadastro por WhatsApp (sem WhatsApp, não se aplica).
CREATE UNIQUE INDEX IF NOT EXISTS customers_whatsapp_key ON customers (whatsapp) WHERE whatsapp <> '';
CREATE INDEX IF NOT EXISTS customers_name_idx ON customers (lower(name));

-- O pedido aponta para o cliente; client_name continua como o nome no dia do pedido.
ALTER TABLE bookings ADD COLUMN customer_id UUID REFERENCES customers (id);
ALTER TABLE transfers ADD COLUMN customer_id UUID REFERENCES customers (id);
CREATE INDEX IF NOT EXISTS bookings_customer_idx ON bookings (customer_id);
CREATE INDEX IF NOT EXISTS transfers_customer_idx ON transfers (customer_id);

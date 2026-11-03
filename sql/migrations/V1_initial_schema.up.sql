-- Usuários da central. Não há cadastro aberto: criados via cmd/create-admin (RN-05.2).
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT        NOT NULL,
    email         TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL CHECK (role IN ('admin', 'central')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- E-mail único sem diferenciar maiúsculas (RN-05.4).
CREATE UNIQUE INDEX IF NOT EXISTS users_email_lower_key ON users (lower(email));

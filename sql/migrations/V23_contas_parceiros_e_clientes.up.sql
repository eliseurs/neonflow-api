-- Contas de parceiros e clientes (RN-05.11 a RN-05.14): o parceiro entra por convite da central e o
-- cliente se cadastra no site, com o e-mail confirmado. Cada conta aponta para o seu cadastro.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('admin', 'central', 'legal_reviewer', 'parceiro', 'cliente'));

ALTER TABLE users ADD COLUMN IF NOT EXISTS partner_id UUID REFERENCES partners (id);
ALTER TABLE users ADD COLUMN IF NOT EXISTS customer_id UUID REFERENCES customers (id);
-- Cliente só entra com o e-mail confirmado. A equipe e os parceiros convidados já nascem confirmados.
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;
UPDATE users SET email_verified_at = created_at WHERE email_verified_at IS NULL;

-- Uma conta por cadastro. O parceiro tem sempre o seu; o cliente ganha o seu ao confirmar o e-mail.
CREATE UNIQUE INDEX IF NOT EXISTS users_partner_key ON users (partner_id) WHERE partner_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS users_customer_key ON users (customer_id) WHERE customer_id IS NOT NULL;
ALTER TABLE users ADD CONSTRAINT users_account_link_check CHECK (
    CASE role
        WHEN 'parceiro' THEN partner_id IS NOT NULL AND customer_id IS NULL
        WHEN 'cliente' THEN partner_id IS NULL
        ELSE partner_id IS NULL AND customer_id IS NULL
    END
);

-- Links de uso único enviados por e-mail. Só o hash do token fica no banco.
CREATE TABLE IF NOT EXISTS account_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purpose    TEXT        NOT NULL CHECK (purpose IN ('verificar_email', 'convite', 'nova_senha')),
    token_hash TEXT        NOT NULL UNIQUE,
    user_id    UUID REFERENCES users (id),
    partner_id UUID REFERENCES partners (id),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- O convite aponta para o parceiro (a conta ainda não existe); os outros, para a conta.
    CONSTRAINT account_tokens_target_check CHECK (
        (purpose = 'convite' AND partner_id IS NOT NULL AND user_id IS NULL)
        OR (purpose <> 'convite' AND user_id IS NOT NULL AND partner_id IS NULL)
    )
);

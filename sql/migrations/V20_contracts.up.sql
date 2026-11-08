-- Contratos V15 (pacote de 07/11/2026): seis documentos em versões com hash, revisão jurídica por cláusula
-- e decisão da revisora vinculada à versão exata (06_GUIA_AREA_JURIDICA). Versões, eventos e comentários
-- são append-only: o banco recusa UPDATE e DELETE. O texto das minutas entra pela API, não pela migration.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'central', 'legal_reviewer'));

CREATE TABLE IF NOT EXISTS contract_documents (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code       TEXT        NOT NULL UNIQUE CHECK (code ~ '^[0-9]{2}$'),
    title      TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO contract_documents (code, title) VALUES
    ('01', 'Termos Gerais de Uso e Intermediação'),
    ('02', 'Termo de Parceria do Prestador'),
    ('03', 'Termo do Programa de Indicações'),
    ('04', 'Ordem de Locação de Veículo sem Motorista'),
    ('05', 'Ordem de Serviço de Transfer'),
    ('06', 'Ordem de Serviço de Passeio Turístico')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS contract_versions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID        NOT NULL REFERENCES contract_documents (id),
    version     INT         NOT NULL CHECK (version > 0),
    label       TEXT        NOT NULL DEFAULT '',
    body        TEXT        NOT NULL CHECK (length(trim(body)) > 0),
    sha256      TEXT        NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    created_by  UUID        REFERENCES users (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (document_id, version)
);

-- A situação da versão é o último evento; sem evento, minuta.
CREATE TABLE IF NOT EXISTS contract_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version_id UUID        NOT NULL REFERENCES contract_versions (id),
    action     TEXT        NOT NULL CHECK (action IN ('enviar', 'solicitar_ajustes', 'aprovar', 'revogar', 'liberar', 'publicar')),
    status     TEXT        NOT NULL CHECK (status IN ('enviado_para_revisao', 'ajustes_solicitados', 'aprovado_pela_revisora', 'liberacao_administrativa', 'publicado')),
    note       TEXT        NOT NULL DEFAULT '',
    sha256     TEXT        NOT NULL,
    user_id    UUID        NOT NULL REFERENCES users (id),
    role       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS contract_events_version_idx ON contract_events (version_id, created_at);

CREATE TABLE IF NOT EXISTS contract_comments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version_id    UUID        NOT NULL REFERENCES contract_versions (id),
    clause        TEXT        NOT NULL CHECK (length(trim(clause)) > 0),
    kind          TEXT        NOT NULL CHECK (kind IN ('observacao', 'correcao')),
    body          TEXT        NOT NULL CHECK (length(trim(body)) > 0),
    proposed_text TEXT        NOT NULL DEFAULT '',
    user_id       UUID        NOT NULL REFERENCES users (id),
    role          TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS contract_comments_version_idx ON contract_comments (version_id, created_at);

CREATE OR REPLACE FUNCTION contract_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'registro de contrato não pode ser alterado nem apagado' USING ERRCODE = 'P0001';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS contract_versions_append_only ON contract_versions;
CREATE TRIGGER contract_versions_append_only BEFORE UPDATE OR DELETE ON contract_versions
    FOR EACH ROW EXECUTE FUNCTION contract_append_only();
DROP TRIGGER IF EXISTS contract_events_append_only ON contract_events;
CREATE TRIGGER contract_events_append_only BEFORE UPDATE OR DELETE ON contract_events
    FOR EACH ROW EXECUTE FUNCTION contract_append_only();
DROP TRIGGER IF EXISTS contract_comments_append_only ON contract_comments;
CREATE TRIGGER contract_comments_append_only BEFORE UPDATE OR DELETE ON contract_comments
    FOR EACH ROW EXECUTE FUNCTION contract_append_only();

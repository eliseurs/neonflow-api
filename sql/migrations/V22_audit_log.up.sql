-- Auditoria das rotas (RN-05.9): quem chamou, com qual papel, qual rota e o resultado.
-- Entram as chamadas que alteram dados, o login e os acessos negados por perfil.
-- Sem corpo, query string nem dados pessoais: "path" só tem os ids do recurso.
CREATE TABLE IF NOT EXISTS audit_log (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id),
    role       TEXT        NOT NULL,
    method     TEXT        NOT NULL,
    route      TEXT        NOT NULL,
    path       TEXT        NOT NULL,
    status     INT         NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS audit_log_created_idx ON audit_log (created_at DESC);
CREATE INDEX IF NOT EXISTS audit_log_user_idx ON audit_log (user_id, created_at DESC);

-- Só inclusão: a trilha não pode ser alterada nem apagada.
CREATE OR REPLACE FUNCTION audit_log_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'registro de auditoria não pode ser alterado nem apagado' USING ERRCODE = 'P0001';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS audit_log_append_only ON audit_log;
CREATE TRIGGER audit_log_append_only BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_append_only();

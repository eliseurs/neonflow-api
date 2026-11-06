-- Fotos do veículo (RN-01.7) e da reserva na entrega e na devolução (RN-01.6, item "fotos" do checklist).
-- O arquivo fica no bucket privado (ADR-0014); aqui só a chave e os metadados.
CREATE TABLE IF NOT EXISTS photos (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_type   TEXT        NOT NULL CHECK (owner_type IN ('veiculo', 'reserva')),
    owner_id     UUID        NOT NULL,
    stage        TEXT        NOT NULL DEFAULT '' CHECK (stage IN ('', 'entrega', 'devolucao')),
    storage_key  TEXT        NOT NULL UNIQUE,
    content_type TEXT        NOT NULL CHECK (content_type IN ('image/jpeg', 'image/png', 'image/webp')),
    size_bytes   INT         NOT NULL CHECK (size_bytes > 0),
    position     INT         NOT NULL DEFAULT 0,
    created_by   UUID        REFERENCES users (id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Veículo não tem etapa; reserva sempre tem.
    CONSTRAINT photos_owner_stage_check CHECK ((owner_type = 'veiculo') = (stage = ''))
);

CREATE INDEX IF NOT EXISTS photos_owner_idx ON photos (owner_type, owner_id, stage, position);

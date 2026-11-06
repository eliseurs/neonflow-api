-- Itens do checklist de devolução (RN-01.6). Configuráveis pela central; a lista inicial
-- é a do fundador (q56 em 05/11/2026 e q80 em 06/11/2026). A chave vira o nome da
-- conferência na reserva em "devolvida" e não muda depois de criada.
CREATE TABLE IF NOT EXISTS checklist_items (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key        TEXT        NOT NULL CHECK (key ~ '^[a-z0-9_]{1,40}$'),
    label      TEXT        NOT NULL CHECK (length(trim(label)) > 0),
    position   INT         NOT NULL DEFAULT 0,
    active     BOOLEAN     NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS checklist_items_key_key ON checklist_items (key);

INSERT INTO checklist_items (key, label, position) VALUES
    ('combustivel', 'Combustível', 10),
    ('estetica', 'Estética (riscos e amassados)', 20),
    ('oleo', 'Nível de óleo', 30),
    ('agua', 'Água', 40),
    ('pastilhas', 'Pastilhas de freio', 50),
    ('pneus', 'Pneus', 60),
    ('limpeza', 'Limpeza interna', 70),
    ('documentos', 'Documentos no carro', 80),
    ('fotos', 'Fotos', 90),
    ('quilometragem', 'Quilometragem', 100)
ON CONFLICT (key) DO NOTHING;

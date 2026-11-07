-- Ficha do veículo para o site e o painel (06/11/2026): ano, cor, combustível, portas, itens e descrição.
-- Todos opcionais: o carro já cadastrado continua válido.
ALTER TABLE vehicles
    ADD COLUMN year        INT  CHECK (year BETWEEN 1980 AND 2100),
    ADD COLUMN color       TEXT NOT NULL DEFAULT '',
    ADD COLUMN fuel        TEXT NOT NULL DEFAULT '' CHECK (fuel IN ('', 'flex', 'gasolina', 'etanol', 'diesel', 'eletrico', 'hibrido')),
    ADD COLUMN doors       INT  CHECK (doors BETWEEN 2 AND 5),
    ADD COLUMN features    TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN description TEXT NOT NULL DEFAULT '';

-- Histórico de manutenção (painel), base para comparar os carros: quanto custa e quando parou.
CREATE TABLE IF NOT EXISTS vehicle_maintenance (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id  UUID        NOT NULL REFERENCES vehicles (id),
    date        DATE        NOT NULL,
    kind        TEXT        NOT NULL CHECK (kind IN ('preventiva', 'corretiva', 'revisao', 'pneus', 'outros')),
    description TEXT        NOT NULL CHECK (length(trim(description)) > 0),
    cost_cents  BIGINT      CHECK (cost_cents >= 0),
    odometer_km INT         CHECK (odometer_km >= 0),
    workshop    TEXT        NOT NULL DEFAULT '',
    created_by  UUID        REFERENCES users (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS vehicle_maintenance_vehicle_idx ON vehicle_maintenance (vehicle_id, date DESC);

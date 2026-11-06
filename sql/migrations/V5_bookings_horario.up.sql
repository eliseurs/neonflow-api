-- Reserva com horário (RN-02.23 a RN-02.26, PEN-29): retirada e devolução com data e hora.
-- O bloqueio passa a ser um intervalo de horas [start_at, end_at): devolver às 10h e retirar
-- às 10h do mesmo dia não se sobrepõem (RN-02.25).
ALTER TABLE bookings
    ADD COLUMN start_at TIMESTAMPTZ,
    ADD COLUMN end_at   TIMESTAMPTZ;

-- Reservas existentes ganham 09:00 de Gramado (o horário do protótipo) na retirada e na
-- devolução: o intervalo continua com múltiplos de 24 horas e o número de diárias não muda.
UPDATE bookings
SET start_at = (start_date + time '09:00') AT TIME ZONE 'America/Sao_Paulo',
    end_at   = (end_date + time '09:00') AT TIME ZONE 'America/Sao_Paulo';

ALTER TABLE bookings
    ALTER COLUMN start_at SET NOT NULL,
    ALTER COLUMN end_at SET NOT NULL,
    DROP CONSTRAINT bookings_no_overlap;

ALTER TABLE bookings
    ADD CONSTRAINT bookings_period_check CHECK (end_at > start_at),
    -- Bloqueio atômico (RN-02.1, RN-02.2), agora por hora.
    ADD CONSTRAINT bookings_no_overlap EXCLUDE USING gist (
        vehicle_id WITH =,
        tstzrange(start_at, end_at) WITH &&
    ) WHERE (stage NOT IN ('encerrada', 'cancelada'));

ALTER TABLE bookings
    DROP COLUMN start_date,
    DROP COLUMN end_date;

CREATE INDEX IF NOT EXISTS bookings_vehicle_start_idx ON bookings (vehicle_id, start_at);

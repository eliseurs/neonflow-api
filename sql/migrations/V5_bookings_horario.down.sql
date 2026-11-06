-- Volta para reserva por data. A hora se perde; uma reserva que começa e termina no mesmo
-- dia vira 1 diária. Pode falhar se duas reservas do mesmo dia passarem a se sobrepor.
DROP INDEX IF EXISTS bookings_vehicle_start_idx;

ALTER TABLE bookings
    ADD COLUMN start_date DATE,
    ADD COLUMN end_date   DATE;

UPDATE bookings
SET start_date = (start_at AT TIME ZONE 'America/Sao_Paulo')::date,
    end_date   = (end_at AT TIME ZONE 'America/Sao_Paulo')::date;
UPDATE bookings SET end_date = start_date + 1 WHERE end_date <= start_date;

ALTER TABLE bookings
    ALTER COLUMN start_date SET NOT NULL,
    ALTER COLUMN end_date SET NOT NULL,
    DROP CONSTRAINT bookings_no_overlap,
    DROP CONSTRAINT bookings_period_check;

ALTER TABLE bookings
    ADD CONSTRAINT bookings_check CHECK (end_date > start_date),
    ADD CONSTRAINT bookings_no_overlap EXCLUDE USING gist (
        vehicle_id WITH =,
        daterange(start_date, end_date) WITH &&
    ) WHERE (stage NOT IN ('encerrada', 'cancelada'));

ALTER TABLE bookings
    DROP COLUMN start_at,
    DROP COLUMN end_at;

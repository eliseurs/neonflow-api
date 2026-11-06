-- Atraso na devolução cobrado por minuto: diária ÷ 1440 por minuto desde o horário combinado,
-- passada 1 hora de tolerância (RN-02.19, q89 e resposta de 06/11/2026). extra_days dá lugar a
-- extra_minutes; reservas antigas convertem 1 diária em 1440 minutos (o valor não muda).
ALTER TABLE bookings ADD COLUMN extra_minutes INT NOT NULL DEFAULT 0 CHECK (extra_minutes >= 0);
UPDATE bookings SET extra_minutes = extra_days * 1440 WHERE extra_days > 0;
ALTER TABLE bookings DROP COLUMN extra_days;

-- Km controlado (RN-02.18): o limite por dia pode existir sem o valor do km excedente; aí o
-- excedente é medido e o valor fica a definir pela central (RN-02.29). Valor sem limite não faz sentido.
ALTER TABLE vehicles DROP CONSTRAINT vehicles_km_plan_check,
    ADD CONSTRAINT vehicles_km_plan_check CHECK (extra_km_cents IS NULL OR km_limit_per_day IS NOT NULL);
ALTER TABLE bookings DROP CONSTRAINT bookings_km_plan_check,
    ADD CONSTRAINT bookings_km_plan_check CHECK (extra_km_cents IS NULL OR km_limit_per_day IS NOT NULL);

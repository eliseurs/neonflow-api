-- Limite sem valor não cabe na regra antiga: volta a km livre.
UPDATE bookings SET km_limit_per_day = NULL WHERE extra_km_cents IS NULL;
UPDATE vehicles SET km_limit_per_day = NULL WHERE extra_km_cents IS NULL;
ALTER TABLE bookings DROP CONSTRAINT bookings_km_plan_check,
    ADD CONSTRAINT bookings_km_plan_check CHECK ((km_limit_per_day IS NULL) = (extra_km_cents IS NULL));
ALTER TABLE vehicles DROP CONSTRAINT vehicles_km_plan_check,
    ADD CONSTRAINT vehicles_km_plan_check CHECK ((km_limit_per_day IS NULL) = (extra_km_cents IS NULL));

ALTER TABLE bookings ADD COLUMN extra_days INT NOT NULL DEFAULT 0;
UPDATE bookings SET extra_days = (extra_minutes + 1439) / 1440 WHERE extra_minutes > 0;
ALTER TABLE bookings DROP COLUMN extra_minutes;

ALTER TABLE bookings
    DROP CONSTRAINT IF EXISTS bookings_km_order_check,
    DROP CONSTRAINT IF EXISTS bookings_km_plan_check,
    DROP COLUMN IF EXISTS excess_km_cents,
    DROP COLUMN IF EXISTS excess_km,
    DROP COLUMN IF EXISTS returned_km,
    DROP COLUMN IF EXISTS delivered_km,
    DROP COLUMN IF EXISTS extra_km_cents,
    DROP COLUMN IF EXISTS km_limit_per_day,
    DROP COLUMN IF EXISTS deposit_cents;

ALTER TABLE vehicles
    DROP CONSTRAINT IF EXISTS vehicles_km_plan_check,
    DROP COLUMN IF EXISTS extra_km_cents,
    DROP COLUMN IF EXISTS km_limit_per_day,
    DROP COLUMN IF EXISTS deposit_cents;

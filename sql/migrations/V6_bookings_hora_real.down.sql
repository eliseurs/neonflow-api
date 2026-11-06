ALTER TABLE bookings
    DROP CONSTRAINT IF EXISTS bookings_return_after_delivery,
    DROP COLUMN IF EXISTS extra_cents,
    DROP COLUMN IF EXISTS extra_days,
    DROP COLUMN IF EXISTS returned_at,
    DROP COLUMN IF EXISTS delivered_at;

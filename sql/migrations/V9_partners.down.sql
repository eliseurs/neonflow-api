ALTER TABLE bookings DROP COLUMN IF EXISTS referred_by_partner_id;
ALTER TABLE vehicles DROP COLUMN IF EXISTS owner_partner_id;
DROP TABLE IF EXISTS partners;

UPDATE transfers SET offer_expires_at = now() WHERE offer_cents IS NOT NULL;
ALTER TABLE transfers
    DROP COLUMN vehicle_id,
    ADD CONSTRAINT transfers_offer_check CHECK ((offer_cents IS NULL) = (offer_expires_at IS NULL));

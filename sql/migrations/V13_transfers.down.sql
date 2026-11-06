DELETE FROM partner_payouts WHERE transfer_id IS NOT NULL;
ALTER TABLE partner_payouts
    DROP CONSTRAINT partner_payouts_amount_check,
    ADD CONSTRAINT partner_payouts_amount_check CHECK ((percent_bp IS NULL) = (amount_cents IS NULL)),
    DROP CONSTRAINT partner_payouts_transfer_unique,
    DROP CONSTRAINT partner_payouts_source_check,
    DROP CONSTRAINT partner_payouts_kind_check,
    ADD CONSTRAINT partner_payouts_kind_check CHECK (kind IN ('comissao', 'repasse')),
    DROP COLUMN transfer_id,
    ALTER COLUMN booking_id SET NOT NULL;
DROP TABLE IF EXISTS transfer_events;
DROP TABLE IF EXISTS transfers;

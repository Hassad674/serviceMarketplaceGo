ALTER TABLE payment_info
    DROP COLUMN IF EXISTS charges_enabled,
    DROP COLUMN IF EXISTS payouts_enabled;

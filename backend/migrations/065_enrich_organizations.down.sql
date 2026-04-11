BEGIN;

DROP INDEX IF EXISTS idx_organizations_name;
DROP INDEX IF EXISTS idx_organizations_kyc_enforcement;
DROP INDEX IF EXISTS idx_organizations_stripe_account_id;

ALTER TABLE organizations DROP CONSTRAINT organizations_type_check;
ALTER TABLE organizations
    ADD CONSTRAINT organizations_type_check
    CHECK (type IN ('agency', 'enterprise'));

ALTER TABLE organizations
    DROP COLUMN kyc_restriction_notified_at,
    DROP COLUMN kyc_first_earning_at,
    DROP COLUMN stripe_last_state,
    DROP COLUMN stripe_account_country,
    DROP COLUMN stripe_account_id,
    DROP COLUMN name;

COMMIT;

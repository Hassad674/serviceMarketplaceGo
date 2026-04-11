-- Phase R5 — Drop Stripe + KYC legacy columns from users
--
-- Everything these columns carried (stripe_account_id + country +
-- last_state, kyc_first_earning_at + notification state) has already
-- been moved onto the organizations table in phase R1 and is now read
-- + written exclusively through OrganizationRepository. The matching
-- columns on users are dead weight.
--
-- A one-shot "final sync" is not necessary because phase R1 copied
-- everything at that point, and R1→R5 code kept writing to the org
-- columns only (the user-side methods are gone). So a naive DROP is
-- safe here.

BEGIN;

DROP INDEX IF EXISTS idx_users_kyc_enforcement;
DROP INDEX IF EXISTS idx_users_stripe_account_id;

ALTER TABLE users
    DROP COLUMN IF EXISTS stripe_account_id,
    DROP COLUMN IF EXISTS stripe_account_country,
    DROP COLUMN IF EXISTS stripe_last_state,
    DROP COLUMN IF EXISTS kyc_first_earning_at,
    DROP COLUMN IF EXISTS kyc_restriction_notified_at;

COMMIT;

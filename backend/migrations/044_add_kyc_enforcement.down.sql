DROP INDEX IF EXISTS idx_users_kyc_enforcement;
ALTER TABLE users
    DROP COLUMN IF EXISTS kyc_first_earning_at,
    DROP COLUMN IF EXISTS kyc_restriction_notified_at;

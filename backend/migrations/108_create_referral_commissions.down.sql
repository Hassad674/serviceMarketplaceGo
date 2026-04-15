-- 108_create_referral_commissions.down.sql

BEGIN;

DROP TRIGGER IF EXISTS referral_commissions_updated_at ON referral_commissions;
DROP INDEX IF EXISTS idx_referral_commissions_pending_kyc;
DROP INDEX IF EXISTS idx_referral_commissions_status;
DROP INDEX IF EXISTS idx_referral_commissions_attribution;
DROP TABLE IF EXISTS referral_commissions;

COMMIT;

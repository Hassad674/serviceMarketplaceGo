-- 105_create_referrals.down.sql

BEGIN;

DROP TRIGGER IF EXISTS referrals_updated_at ON referrals;
DROP INDEX IF EXISTS idx_referrals_pending_last_action;
DROP INDEX IF EXISTS idx_referrals_active_expiry;
DROP INDEX IF EXISTS idx_referrals_client_status;
DROP INDEX IF EXISTS idx_referrals_provider_status;
DROP INDEX IF EXISTS idx_referrals_referrer_status;
DROP INDEX IF EXISTS idx_referrals_active_couple_unique;
DROP TABLE IF EXISTS referrals;

COMMIT;

-- 107_create_referral_attributions.down.sql

BEGIN;

DROP INDEX IF EXISTS idx_referral_attributions_provider_client;
DROP INDEX IF EXISTS idx_referral_attributions_referral;
DROP TABLE IF EXISTS referral_attributions;

COMMIT;

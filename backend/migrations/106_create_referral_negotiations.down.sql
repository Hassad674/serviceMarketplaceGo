-- 106_create_referral_negotiations.down.sql

BEGIN;

DROP INDEX IF EXISTS idx_referral_negotiations_referral_created;
DROP TABLE IF EXISTS referral_negotiations;

COMMIT;

-- 153_referral_attributions_ended_at.down.sql
--
-- Rollback of the WALLET-UNIFY "ended_at" addition. Idempotent so a
-- partially-applied up can be cleaned up in one shot.

BEGIN;

DROP INDEX IF EXISTS idx_referral_attributions_active;

ALTER TABLE referral_attributions
    DROP COLUMN IF EXISTS ended_at;

COMMIT;

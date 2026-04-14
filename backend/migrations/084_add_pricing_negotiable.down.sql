-- 084_add_pricing_negotiable.down.sql
--
-- Reverts 084 by dropping the negotiable column on profile_pricing.

BEGIN;

ALTER TABLE profile_pricing DROP COLUMN IF EXISTS negotiable;

COMMIT;

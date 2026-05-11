-- 154_invoice_platform_fee_per_milestone.down.sql
--
-- Reverses migration 154. Idempotent.

BEGIN;

DROP INDEX IF EXISTS idx_invoice_milestone_platform_fee_unique;

ALTER TABLE invoice DROP CONSTRAINT IF EXISTS invoice_source_type_check;
ALTER TABLE invoice ADD CONSTRAINT invoice_source_type_check
    CHECK (source_type IN ('subscription', 'monthly_commission'));

ALTER TABLE invoice DROP COLUMN IF EXISTS milestone_id;

COMMIT;

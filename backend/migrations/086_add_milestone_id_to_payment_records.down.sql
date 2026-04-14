DROP INDEX IF EXISTS idx_payment_records_milestone;
ALTER TABLE payment_records DROP COLUMN IF EXISTS milestone_id;

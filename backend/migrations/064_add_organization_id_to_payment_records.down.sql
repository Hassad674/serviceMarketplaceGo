DROP INDEX IF EXISTS idx_payment_records_org_created;
DROP INDEX IF EXISTS idx_payment_records_organization_id;
ALTER TABLE payment_records DROP COLUMN IF EXISTS organization_id;

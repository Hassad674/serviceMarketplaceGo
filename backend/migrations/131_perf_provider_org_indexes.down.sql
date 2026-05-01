-- 131_perf_provider_org_indexes.down.sql
--
-- Reverse migration 131 — drop the indexes, then drop the
-- payment_records.provider_organization_id column. Indexes are dropped
-- before the column they reference. Order matters: an index dependency
-- on a column blocks DROP COLUMN otherwise.

BEGIN;

DROP INDEX IF EXISTS idx_proposals_provider_org_completed;
DROP INDEX IF EXISTS idx_proposals_provider_org_status_created;
DROP INDEX IF EXISTS idx_payment_records_provider_org_created;

ALTER TABLE payment_records
    DROP COLUMN IF EXISTS provider_organization_id;

COMMIT;

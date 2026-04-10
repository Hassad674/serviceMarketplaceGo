-- Revert: drop the organization_id column and its indexes on jobs.
-- This restores the pre-Phase-4 schema. The data loss is limited to
-- the org_id denormalization — creator_id stays untouched.

DROP INDEX IF EXISTS idx_jobs_org_status_created;
DROP INDEX IF EXISTS idx_jobs_organization_id;
ALTER TABLE jobs DROP COLUMN IF EXISTS organization_id;

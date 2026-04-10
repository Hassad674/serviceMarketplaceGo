DROP INDEX IF EXISTS idx_proposals_org_status;
DROP INDEX IF EXISTS idx_proposals_organization_id;
ALTER TABLE proposals DROP COLUMN IF EXISTS organization_id;

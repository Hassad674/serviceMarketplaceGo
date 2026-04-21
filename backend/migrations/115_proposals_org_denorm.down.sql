-- 115_proposals_org_denorm.down.sql
--
-- Drops the denormalized client/provider org columns and their partial
-- indexes. Safe — the legacy "organization_id" column still carries the
-- client-side org for historical rows.

DROP INDEX IF EXISTS idx_proposals_client_org_completed;
DROP INDEX IF EXISTS idx_proposals_client_org_status;

ALTER TABLE proposals
    DROP COLUMN IF EXISTS provider_organization_id,
    DROP COLUMN IF EXISTS client_organization_id;

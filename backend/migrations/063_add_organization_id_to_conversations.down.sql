DROP INDEX IF EXISTS idx_conversations_org_updated;
DROP INDEX IF EXISTS idx_conversations_organization_id;
ALTER TABLE conversations DROP COLUMN IF EXISTS organization_id;

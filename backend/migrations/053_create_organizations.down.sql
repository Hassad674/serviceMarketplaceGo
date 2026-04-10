DROP TRIGGER IF EXISTS organizations_updated_at ON organizations;
DROP INDEX IF EXISTS idx_organizations_pending_transfer;
DROP INDEX IF EXISTS idx_organizations_type;
DROP INDEX IF EXISTS idx_organizations_owner_user_id;
DROP TABLE IF EXISTS organizations;

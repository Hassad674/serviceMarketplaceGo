DROP TRIGGER IF EXISTS organization_members_updated_at ON organization_members;
DROP INDEX IF EXISTS idx_org_members_role;
DROP INDEX IF EXISTS idx_org_members_user_id;
DROP INDEX IF EXISTS idx_org_members_organization_id;
DROP INDEX IF EXISTS idx_org_members_unique_owner;
DROP TABLE IF EXISTS organization_members;

DROP TRIGGER IF EXISTS organization_invitations_updated_at ON organization_invitations;
DROP INDEX IF EXISTS idx_org_invitations_expires_at;
DROP INDEX IF EXISTS idx_org_invitations_status;
DROP INDEX IF EXISTS idx_org_invitations_email;
DROP INDEX IF EXISTS idx_org_invitations_organization_id;
DROP INDEX IF EXISTS idx_org_invitations_unique_pending;
DROP TABLE IF EXISTS organization_invitations;

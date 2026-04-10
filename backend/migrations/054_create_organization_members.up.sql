-- Team Management V1: organization_members table
-- Links users (operators or the founding owner) to an organization with a specific role.
--
-- Roles (V1 hardcoded in domain layer):
--   owner  — the single founder with full rights (V1: exactly one per org)
--   admin  — can do everything except billing, ownership transfer, org deletion
--   member — daily operational rights (jobs, proposals, messaging, etc.)
--   viewer — read-only across all org resources
--
-- The single-Owner V1 constraint is enforced at the DB level by the partial unique
-- index idx_org_members_unique_owner below, so no race condition can create two Owners.

CREATE TABLE IF NOT EXISTS organization_members (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id  UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role             TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    title            TEXT NOT NULL DEFAULT '',
    joined_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- A user can be a member of an org at most once
    CONSTRAINT uq_org_members_org_user UNIQUE (organization_id, user_id)
);

-- V1 invariant: exactly one Owner per organization (DB-enforced)
CREATE UNIQUE INDEX IF NOT EXISTS idx_org_members_unique_owner
    ON organization_members(organization_id)
    WHERE role = 'owner';

CREATE INDEX IF NOT EXISTS idx_org_members_organization_id ON organization_members(organization_id);
CREATE INDEX IF NOT EXISTS idx_org_members_user_id ON organization_members(user_id);
CREATE INDEX IF NOT EXISTS idx_org_members_role ON organization_members(organization_id, role);

CREATE TRIGGER organization_members_updated_at
    BEFORE UPDATE ON organization_members
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

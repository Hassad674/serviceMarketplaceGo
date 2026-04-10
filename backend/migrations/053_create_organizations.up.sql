-- Team Management V1: organizations table
-- Represents the business entity (Agency or Enterprise) as a first-class entity,
-- distinct from the user who created it. Holds org-level metadata and the
-- pending ownership transfer state.
--
-- Invariant: exactly one Owner per organization (V1 constraint). This is enforced
-- at application level by the organization_members unique partial index created
-- in migration 054. The owner_user_id column here is a denormalized cache of the
-- current owner for fast lookups, kept in sync by the app layer.

CREATE TABLE IF NOT EXISTS organizations (
    id                             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id                  UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    type                           TEXT NOT NULL CHECK (type IN ('agency', 'enterprise')),

    -- Pending ownership transfer (V1 single-owner transfer flow)
    pending_transfer_to_user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    pending_transfer_initiated_at  TIMESTAMPTZ,
    pending_transfer_expires_at    TIMESTAMPTZ,

    created_at                     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                     TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- A user can own at most one organization in V1
    CONSTRAINT uq_organizations_owner UNIQUE (owner_user_id),

    -- Pending transfer fields must be all set or all null
    CONSTRAINT chk_pending_transfer_consistency CHECK (
        (pending_transfer_to_user_id IS NULL AND pending_transfer_initiated_at IS NULL AND pending_transfer_expires_at IS NULL)
        OR
        (pending_transfer_to_user_id IS NOT NULL AND pending_transfer_initiated_at IS NOT NULL AND pending_transfer_expires_at IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_organizations_owner_user_id ON organizations(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_organizations_type ON organizations(type);
CREATE INDEX IF NOT EXISTS idx_organizations_pending_transfer
    ON organizations(pending_transfer_to_user_id, pending_transfer_expires_at)
    WHERE pending_transfer_to_user_id IS NOT NULL;

CREATE TRIGGER organizations_updated_at
    BEFORE UPDATE ON organizations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

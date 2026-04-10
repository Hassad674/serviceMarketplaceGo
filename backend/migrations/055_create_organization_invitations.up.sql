-- Team Management V1: organization_invitations table
-- Pending invitations sent by Owner/Admin to a new operator.
-- On acceptance, a new user (account_type='operator') is created and a corresponding
-- organization_members row is inserted, then the invitation is marked accepted.
--
-- Invitations cannot target the Owner role: the recipient can only be invited as
-- Admin, Member, or Viewer. Promotion to Owner happens exclusively via the transfer
-- ownership flow (columns pending_transfer_* on organizations).
--
-- Token is a random 32-byte hex string (64 chars) — unguessable, single-use.
-- Expiration is 7 days by default, enforced at application layer.

CREATE TABLE IF NOT EXISTS organization_invitations (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id    UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email              TEXT NOT NULL,
    first_name         TEXT NOT NULL,
    last_name          TEXT NOT NULL,
    title              TEXT NOT NULL DEFAULT '',
    role               TEXT NOT NULL CHECK (role IN ('admin', 'member', 'viewer')),
    token              TEXT NOT NULL UNIQUE,
    invited_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status             TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'cancelled', 'expired')),
    expires_at         TIMESTAMPTZ NOT NULL,
    accepted_at        TIMESTAMPTZ,
    cancelled_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Cannot have two pending invitations for the same email in the same org
CREATE UNIQUE INDEX IF NOT EXISTS idx_org_invitations_unique_pending
    ON organization_invitations(organization_id, lower(email))
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_org_invitations_organization_id ON organization_invitations(organization_id);
CREATE INDEX IF NOT EXISTS idx_org_invitations_email ON organization_invitations(lower(email));
CREATE INDEX IF NOT EXISTS idx_org_invitations_status ON organization_invitations(status);
CREATE INDEX IF NOT EXISTS idx_org_invitations_expires_at ON organization_invitations(expires_at) WHERE status = 'pending';

CREATE TRIGGER organization_invitations_updated_at
    BEFORE UPDATE ON organization_invitations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

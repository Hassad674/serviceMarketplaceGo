-- Phase 4 — Scope proposals to organizations
--
-- Proposals have two user references: client_id (the buying side, our
-- "business" side) and provider_id (the freelance). Since Providers
-- are solo in V1, the org owning a proposal is always the client's
-- organization.
--
-- We denormalize the client's org onto the proposal so operators of
-- the same org can see proposals the Owner (or another operator)
-- sent/received. Provider-originated proposals targeting an Agency
-- client are also captured because the backfill keys on client_id.
--
-- NULLABLE column + additive OR filter for the same zero-regression
-- reasons as migration 061.

BEGIN;

ALTER TABLE proposals
    ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_proposals_organization_id
    ON proposals(organization_id)
    WHERE organization_id IS NOT NULL;

-- Composite index for (org, status) list queries.
CREATE INDEX IF NOT EXISTS idx_proposals_org_status
    ON proposals(organization_id, status, created_at DESC)
    WHERE organization_id IS NOT NULL;

-- Backfill: every existing proposal gets the client's current org.
-- If the client is a Provider (no org), organization_id stays NULL.
UPDATE proposals p
SET organization_id = u.organization_id
FROM users u
WHERE p.client_id = u.id
  AND u.organization_id IS NOT NULL
  AND p.organization_id IS NULL;

COMMIT;

-- 115_proposals_org_denorm.up.sql
--
-- Denormalizes the client-side AND provider-side organization ids onto
-- the proposals row so the new client-profile read paths (total_spent,
-- projects completed as client, project history as client) can aggregate
-- in O(1) queries keyed on a single indexed column instead of joining
-- users → organization_members on every read.
--
-- Context:
--   * The existing column "proposals.organization_id" captures the client
--     org at INSERT time via a subquery on organization_members. It is the
--     source of truth for "which org bought this proposal" and is kept.
--   * The new columns mirror that semantics explicitly:
--       - client_organization_id   — alias column pointing to the SAME
--         org as proposals.organization_id. Added for clarity at the
--         read-path layer (the client-profile queries are symmetrical
--         with the provider-profile ones that key on provider_organization_id).
--       - provider_organization_id — resolved at INSERT time from the
--         provider's users.organization_id so provider-side reads can
--         also avoid the JOIN.
--
-- Both columns are nullable (ON DELETE SET NULL) so a deleted org does not
-- cascade-delete historical proposals — the audit trail stays intact.
--
-- Backfill sources the values from users.organization_id, which is the
-- current source of truth (the R1 column). The existing "organization_id"
-- column is not touched; the two client-side columns will converge on
-- every NEW write via the updated INSERT query shipped with this migration.

ALTER TABLE proposals
    ADD COLUMN IF NOT EXISTS client_organization_id   UUID REFERENCES organizations(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS provider_organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;

-- Backfill client side from users.organization_id (same resolution the
-- existing organization_id column uses on insert).
UPDATE proposals p
SET client_organization_id = u.organization_id
FROM users u
WHERE u.id = p.client_id AND p.client_organization_id IS NULL;

-- Backfill provider side from users.organization_id.
UPDATE proposals p
SET provider_organization_id = u.organization_id
FROM users u
WHERE u.id = p.provider_id AND p.provider_organization_id IS NULL;

-- Hot index for the client profile aggregates (total_spent, projects count).
-- Partial index so the footprint stays tight: most rows are not completed,
-- and the typical read filters by status + client_organization_id.
CREATE INDEX IF NOT EXISTS idx_proposals_client_org_status
    ON proposals(client_organization_id, status)
    WHERE client_organization_id IS NOT NULL;

-- Dedicated index for the client-side project history (ordered by completed_at).
CREATE INDEX IF NOT EXISTS idx_proposals_client_org_completed
    ON proposals(client_organization_id, completed_at DESC)
    WHERE status = 'completed' AND client_organization_id IS NOT NULL;

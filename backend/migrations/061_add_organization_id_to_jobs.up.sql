-- Phase 4 — Scope jobs to organizations
--
-- Add a nullable organization_id column on jobs so operators can see
-- their org's jobs via the list endpoint. Nullable intentionally:
--   - Existing Provider-created jobs stay NULL (Providers have no org)
--   - Existing Agency/Enterprise jobs get backfilled from the creator's
--     users.organization_id (set by migration 060)
--   - Future jobs created by an operator get the op's org via the app
--     layer at INSERT time
--
-- The index is a partial index: only rows with a non-NULL org are
-- interesting for org-scoped queries. Providers' jobs are excluded.
--
-- IMPORTANT: we do NOT add NOT NULL. Zero regression for Providers
-- depends on this column being optional forever. The primary query
-- path remains `WHERE creator_id = $1`; the org path is an additive
-- OR filter that kicks in only when the caller has an org context.

BEGIN;

ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_jobs_organization_id
    ON jobs(organization_id)
    WHERE organization_id IS NOT NULL;

-- Composite index for the operator list query which filters by
-- (org_id, status) and orders by created_at DESC.
CREATE INDEX IF NOT EXISTS idx_jobs_org_status_created
    ON jobs(organization_id, status, created_at DESC)
    WHERE organization_id IS NOT NULL;

-- Backfill: propagate the creator's current users.organization_id to
-- each existing job. This is a one-shot fix-up so operators can see
-- historical jobs the Owner created before the team feature existed.
UPDATE jobs j
SET organization_id = u.organization_id
FROM users u
WHERE j.creator_id = u.id
  AND u.organization_id IS NOT NULL
  AND j.organization_id IS NULL;

COMMIT;

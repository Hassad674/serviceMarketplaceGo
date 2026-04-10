-- Phase 4 — Backfill organizations for pre-team users
--
-- Every Agency or Enterprise user that existed before the team feature
-- landed now needs:
--   1. An organization row (owned by them, typed by their role)
--   2. A membership row with role='owner'
--   3. Their users.organization_id pointing at the new org
--
-- IMPORTANT — idempotency notes:
-- Some test accounts created during Phase 0-3 smoke tests already own
-- an organization but have users.organization_id = NULL (Phase 1
-- intentionally did NOT populate the FK because the D10 decision left
-- it as an optional cache). This migration handles both cases:
--   - Users with NO organization at all → create org + member + link
--   - Users who already own an org       → just backfill users.organization_id
--
-- Providers are intentionally excluded — they are solo in V1.
--
-- The whole migration runs inside a single transaction so the
-- multi-table write either fully commits or fully rolls back.

BEGIN;

-- Case 1: users who own no organization yet.
-- Create an org + Owner membership in a CTE chain, then link the user.
WITH
    missing_users AS (
        SELECT u.id, u.role
        FROM users u
        WHERE u.role IN ('agency', 'enterprise')
          AND u.organization_id IS NULL
          AND NOT EXISTS (
              SELECT 1 FROM organizations o WHERE o.owner_user_id = u.id
          )
    ),
    new_orgs AS (
        INSERT INTO organizations (id, owner_user_id, type, created_at, updated_at)
        SELECT gen_random_uuid(), mu.id, mu.role, now(), now()
        FROM missing_users mu
        RETURNING id AS org_id, owner_user_id
    ),
    new_members AS (
        INSERT INTO organization_members (id, organization_id, user_id, role, title, joined_at, created_at, updated_at)
        SELECT gen_random_uuid(), n.org_id, n.owner_user_id, 'owner', '', now(), now(), now()
        FROM new_orgs n
        RETURNING organization_id, user_id
    )
UPDATE users u
SET organization_id = nm.organization_id,
    updated_at      = now()
FROM new_members nm
WHERE u.id = nm.user_id;

-- Case 2: users who already own an org but whose users.organization_id
-- is still NULL. Just backfill the cached FK. These users also need
-- their Owner membership row — test accounts from Phase 0-3 smoke
-- tests were created via auth.Register which called
-- orgs.CreateWithOwnerMembership, so the membership is already there,
-- but we INSERT defensively with ON CONFLICT DO NOTHING in case a
-- historical smoke test created the org without the member row.
INSERT INTO organization_members (id, organization_id, user_id, role, title, joined_at, created_at, updated_at)
SELECT
    gen_random_uuid(),
    o.id,
    u.id,
    'owner',
    '',
    now(),
    now(),
    now()
FROM users u
JOIN organizations o ON o.owner_user_id = u.id
WHERE u.role IN ('agency', 'enterprise')
  AND u.organization_id IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM organization_members om
      WHERE om.organization_id = o.id AND om.user_id = u.id
  );

UPDATE users u
SET organization_id = o.id,
    updated_at      = now()
FROM organizations o
WHERE o.owner_user_id = u.id
  AND u.role IN ('agency', 'enterprise')
  AND u.organization_id IS NULL;

COMMIT;

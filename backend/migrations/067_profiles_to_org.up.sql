-- Phase R2 — Profile becomes the organization's identity
--
-- The Stripe Dashboard model says the marketplace profile (photo,
-- presentation video, about, title) is the ORG's public face. Every
-- member of the org sees and edits the same profile, exactly like
-- every operator in a Stripe account sees the same bank details.
--
-- This migration:
--   1. Adds profiles.organization_id.
--   2. Backfills it from users.organization_id (R1 guarantees every
--      user now has an org).
--   3. Dedupes: if an org has multiple profile rows (an operator
--      visited /profile after the owner already had one), keeps only
--      the owner's row. Covered cases discovered on current prod:
--      1 org × 2 rows.
--   4. Drops profiles.user_id and the old primary key.
--   5. Makes organization_id the new primary key and the FK target.

BEGIN;

ALTER TABLE profiles
    ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;

UPDATE profiles p
SET    organization_id = u.organization_id
FROM   users u
WHERE  p.user_id = u.id;

-- Dedupe: keep the owner's profile row, discard operator rows that
-- share the same org. Runs BEFORE the UNIQUE constraint is applied.
DELETE FROM profiles p
USING  users u, organization_members m
WHERE  p.user_id = u.id
  AND  m.user_id = u.id
  AND  m.organization_id = u.organization_id
  AND  m.role <> 'owner'
  AND  EXISTS (
      SELECT 1
      FROM   profiles p2
      JOIN   users u2 ON u2.id = p2.user_id
      JOIN   organization_members m2
             ON m2.user_id = u2.id AND m2.organization_id = u2.organization_id
      WHERE  u2.organization_id = p.organization_id
        AND  m2.role = 'owner'
  );

-- Sanity: every remaining profile has a non-NULL org_id, and no org
-- has more than one profile.
DO $$
DECLARE
    orphans integer;
    dupes   integer;
BEGIN
    SELECT COUNT(*) INTO orphans
    FROM   profiles
    WHERE  organization_id IS NULL;
    IF orphans > 0 THEN
        RAISE EXCEPTION 'migration 067 left % profiles without an org', orphans;
    END IF;

    SELECT COUNT(*) INTO dupes FROM (
        SELECT 1 FROM profiles GROUP BY organization_id HAVING COUNT(*) > 1
    ) t;
    IF dupes > 0 THEN
        RAISE EXCEPTION 'migration 067 left % orgs with duplicate profile rows', dupes;
    END IF;
END $$;

ALTER TABLE profiles ALTER COLUMN organization_id SET NOT NULL;

ALTER TABLE profiles DROP CONSTRAINT profiles_pkey;
ALTER TABLE profiles DROP CONSTRAINT profiles_user_id_fkey;
ALTER TABLE profiles DROP COLUMN user_id;
ALTER TABLE profiles ADD PRIMARY KEY (organization_id);

COMMIT;

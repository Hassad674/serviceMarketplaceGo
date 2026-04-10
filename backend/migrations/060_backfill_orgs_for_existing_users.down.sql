-- Revert: drop the backfilled organizations and their Owner memberships,
-- and null out users.organization_id for anyone that was backfilled.
--
-- This is intentionally conservative: we only remove rows that exactly
-- match the shape of the backfill (empty title, role='owner', org owns
-- exactly one Owner member). If anyone has already been invited into
-- these orgs after the backfill, the cascade delete will remove them
-- as well — that is expected behavior when reverting a feature.

BEGIN;

-- Null out users.organization_id for users whose current org was
-- created by the backfill (owner_user_id = users.id means they own
-- it themselves, which is the exact shape the backfill created).
UPDATE users u
SET organization_id = NULL,
    updated_at      = now()
WHERE u.organization_id IN (
    SELECT o.id FROM organizations o WHERE o.owner_user_id = u.id
);

-- Delete organizations created by the backfill. CASCADE will wipe
-- organization_members and organization_invitations rows.
DELETE FROM organizations o
WHERE o.owner_user_id IN (
    SELECT u.id FROM users u WHERE u.role IN ('agency', 'enterprise')
);

COMMIT;

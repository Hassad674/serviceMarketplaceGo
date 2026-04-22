-- subscriptions was originally scoped to user_id (migration 117) but business
-- state in this marketplace always belongs to an organization, not a user.
-- An agency is the legal entity that subscribes — members come and go; the
-- agency's Premium must outlive any one member leaving.
--
-- Pre-flight on dev confirmed the backfill is safe:
--   * 0 subs point to a user without organization_id
--   * 0 orgs have multiple open subs (no unique-index collision)
--   * 0 subs point to a deleted user
--
-- Rollout is a single atomic transaction: add the new FK column, backfill,
-- enforce NOT NULL, swap indexes, drop the old FK + column. If anything
-- fails mid-way the whole migration rolls back and the table is untouched.

ALTER TABLE subscriptions
    ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;

-- Repoint every row to the organization of the original subscriber.
UPDATE subscriptions s
    SET organization_id = u.organization_id
    FROM users u
    WHERE u.id = s.user_id;

-- If any row ended up NULL (user without org), this statement fails loudly.
-- That surfaces bad data rather than silently letting an orphan pass.
ALTER TABLE subscriptions
    ALTER COLUMN organization_id SET NOT NULL;

-- Replace the user-scoped indexes with org-scoped equivalents. The partial
-- unique index still enforces "one open sub per owner" — now at the org
-- level, which matches the business invariant.
DROP INDEX IF EXISTS idx_subscriptions_user_open;
DROP INDEX IF EXISTS idx_subscriptions_user;

CREATE UNIQUE INDEX idx_subscriptions_org_open ON subscriptions(organization_id)
    WHERE status IN ('incomplete', 'active', 'past_due');

CREATE INDEX idx_subscriptions_org ON subscriptions(organization_id);

-- Drop the old column + its FK last — the data stayed readable through the
-- whole transition so a rollback at any earlier ALTER is harmless.
ALTER TABLE subscriptions DROP CONSTRAINT subscriptions_user_id_fkey;
ALTER TABLE subscriptions DROP COLUMN user_id;

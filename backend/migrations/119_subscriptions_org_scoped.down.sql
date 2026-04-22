-- Reverse migration 119. Dev-safety only — forward-only in production per
-- the project's immutable-migration rule.
--
-- The original user_id cannot be recovered perfectly: the migration did not
-- snapshot which specific member of an org subscribed. We use the org's
-- owner as a deterministic fallback, matching the business intent (the
-- owner is the person who signed up the org for Premium).

ALTER TABLE subscriptions
    ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;

UPDATE subscriptions s
    SET user_id = o.owner_user_id
    FROM organizations o
    WHERE o.id = s.organization_id;

ALTER TABLE subscriptions
    ALTER COLUMN user_id SET NOT NULL;

DROP INDEX IF EXISTS idx_subscriptions_org_open;
DROP INDEX IF EXISTS idx_subscriptions_org;

CREATE UNIQUE INDEX idx_subscriptions_user_open ON subscriptions(user_id)
    WHERE status IN ('incomplete', 'active', 'past_due');

CREATE INDEX idx_subscriptions_user ON subscriptions(user_id);

ALTER TABLE subscriptions DROP CONSTRAINT subscriptions_organization_id_fkey;
ALTER TABLE subscriptions DROP COLUMN organization_id;

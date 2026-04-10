-- Phase 4 — Scope conversations to organizations
--
-- Conversations have no direct user column; ownership flows through
-- conversation_participants (many-to-many). The "business side" of a
-- conversation is the participant that belongs to an organization.
-- In V1 Providers are solo, so the org side is always the Agency or
-- Enterprise participant (if there is one).
--
-- Backfill picks the first participant with a non-NULL
-- users.organization_id. Conversations between two Providers (rare
-- but possible) keep organization_id NULL and continue to work via
-- the conversation_participants join path.

BEGIN;

ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_conversations_organization_id
    ON conversations(organization_id)
    WHERE organization_id IS NOT NULL;

-- Composite index for (org, updated_at) list queries with cursor
-- pagination.
CREATE INDEX IF NOT EXISTS idx_conversations_org_updated
    ON conversations(organization_id, updated_at DESC, id DESC)
    WHERE organization_id IS NOT NULL;

-- Backfill: find the first participant with an org and denormalize
-- that org onto the conversation. LIMIT 1 inside the subquery is
-- deterministic enough for a backfill (PostgreSQL picks the first
-- match based on the query plan, which is stable within a single
-- migration run).
UPDATE conversations c
SET organization_id = sub.org_id
FROM (
    SELECT DISTINCT ON (cp.conversation_id)
        cp.conversation_id,
        u.organization_id AS org_id
    FROM conversation_participants cp
    JOIN users u ON u.id = cp.user_id
    WHERE u.organization_id IS NOT NULL
    ORDER BY cp.conversation_id, cp.user_id
) sub
WHERE c.id = sub.conversation_id
  AND c.organization_id IS NULL;

COMMIT;

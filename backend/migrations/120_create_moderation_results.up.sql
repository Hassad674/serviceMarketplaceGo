-- Phase 2 of automated text moderation: replace per-table moderation_*
-- columns (currently only on messages + reviews) with a single generic
-- table. This enables extending moderation to any user-generated text
-- (profile bio, job titles/descriptions, proposals, candidatures,
-- display_name) without polluting business tables with moderation_*
-- columns and without growing the admin moderation feed via UNION
-- queries each time we add a new content type.
--
-- A row holds the LATEST decision for a (content_type, content_id) pair
-- — Upsert, not append. The full transition history lives in audit_logs
-- (already populated by app/messaging + app/review since Phase 1).
--
-- The 'blocked' status is new in Phase 2: it captures synchronous
-- creation refusals (e.g. someone tries to register with a toxic
-- display_name). Persisting blocked attempts lets admins see what was
-- refused and detect bad actors before they bypass via other channels.

CREATE TABLE IF NOT EXISTS moderation_results (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content_type    TEXT NOT NULL,
    content_id      UUID NOT NULL,
    author_user_id  UUID REFERENCES users(id) ON DELETE SET NULL,
    status          TEXT NOT NULL,
    score           REAL NOT NULL DEFAULT 0,
    labels          JSONB NOT NULL DEFAULT '[]'::jsonb,
    reason          TEXT NOT NULL DEFAULT '',
    decided_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at     TIMESTAMPTZ,
    UNIQUE (content_type, content_id)
);

-- Hot path index: admin queue listing pending decisions newest-first.
-- Partial WHERE reviewed_at IS NULL keeps the index tight (only the
-- subset that still needs admin attention). decided_at DESC matches
-- the default sort.
CREATE INDEX IF NOT EXISTS idx_moderation_results_pending
    ON moderation_results(status, decided_at DESC)
    WHERE reviewed_at IS NULL;

-- Lookup index: "did this specific content already get moderated?"
-- Used by the service to short-circuit re-moderation on idempotent
-- re-runs and by admin handlers to fetch a single row by content ref.
-- Backed by the UNIQUE constraint above; the explicit index makes the
-- lookup explicit for `EXPLAIN` readers.
CREATE INDEX IF NOT EXISTS idx_moderation_results_content
    ON moderation_results(content_type, content_id);

-- "Show me all moderated content from this user" — used by admin user
-- detail page (futur). Partial to avoid indexing rows where author is
-- unknown (cleanup events, system-originated content, etc.).
CREATE INDEX IF NOT EXISTS idx_moderation_results_author
    ON moderation_results(author_user_id)
    WHERE author_user_id IS NOT NULL;

-- Backfill from existing per-table moderation columns. Run after the
-- table is created so there is no window where the admin queue is
-- partial. Phase 7 drops the legacy columns once everything reads from
-- moderation_results.
INSERT INTO moderation_results (content_type, content_id, author_user_id, status, score, labels, reason, decided_at)
SELECT
    'message',
    m.id,
    m.sender_id,
    m.moderation_status,
    m.moderation_score,
    COALESCE(m.moderation_labels, '[]'::jsonb),
    'backfilled_from_messages',
    m.updated_at
FROM messages m
WHERE m.moderation_status IN ('flagged', 'hidden', 'deleted')
ON CONFLICT (content_type, content_id) DO NOTHING;

INSERT INTO moderation_results (content_type, content_id, author_user_id, status, score, labels, reason, decided_at)
SELECT
    'review',
    rv.id,
    rv.reviewer_id,
    rv.moderation_status,
    rv.moderation_score,
    COALESCE(rv.moderation_labels, '[]'::jsonb),
    'backfilled_from_reviews',
    rv.updated_at
FROM reviews rv
WHERE rv.moderation_status IN ('flagged', 'hidden', 'deleted')
ON CONFLICT (content_type, content_id) DO NOTHING;

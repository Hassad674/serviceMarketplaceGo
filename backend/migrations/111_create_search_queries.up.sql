-- 111_create_search_queries.up.sql
--
-- Persists every search executed against the Typesense-backed engine so we
-- can train future learning-to-rank models, diagnose zero-result queries,
-- and surface the most clicked results in the admin dashboard.
--
-- Feature-scoped: this table references users(id) but no other feature
-- table. Dropping it is safe and breaks nothing else in the marketplace.

CREATE TABLE IF NOT EXISTS search_queries (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID REFERENCES users(id) ON DELETE SET NULL,
    session_id         TEXT,
    query              TEXT NOT NULL,
    filters            JSONB NOT NULL DEFAULT '{}'::jsonb,
    persona            TEXT NOT NULL CHECK (persona IN ('freelance','agency','referrer','all')),
    results_count      INTEGER NOT NULL DEFAULT 0,
    latency_ms         INTEGER NOT NULL DEFAULT 0,
    clicked_result_id  UUID,
    clicked_position   INTEGER,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Chronological scan for "last N searches" debug views.
CREATE INDEX IF NOT EXISTS idx_search_queries_created_at
    ON search_queries(created_at DESC);

-- Partial index: most searches are anonymous, only logged-in ones need
-- per-user lookups for personalization experiments.
CREATE INDEX IF NOT EXISTS idx_search_queries_user_id
    ON search_queries(user_id)
 WHERE user_id IS NOT NULL;

-- Partial index dedicated to zero-result queries (the thing we care about
-- most for merchandising and synonym gap analysis).
CREATE INDEX IF NOT EXISTS idx_search_queries_zero_results
    ON search_queries(created_at DESC)
 WHERE results_count = 0;

-- French-tokenised full-text index on the query text, used by the admin
-- dashboard "top searches" panel.
CREATE INDEX IF NOT EXISTS idx_search_queries_query_gin
    ON search_queries USING GIN(to_tsvector('french', query));

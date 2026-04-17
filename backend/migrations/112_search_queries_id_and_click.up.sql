-- 112_search_queries_id_and_click.up.sql
--
-- Phase 3 extends the search_queries table with two columns:
--
--   search_id  — deterministic per-query hash produced by the app
--                layer. Multi-page loads of the same query share
--                this value so a second-page fetch updates the same
--                row instead of duplicating it. Idempotency is
--                enforced by the unique index below.
--
--   clicked_at — timestamp of the click-through update. Paired with
--                the existing clicked_result_id + clicked_position
--                columns so the analytics dashboard can compute CTR
--                per search.

ALTER TABLE search_queries
    ADD COLUMN IF NOT EXISTS search_id  TEXT,
    ADD COLUMN IF NOT EXISTS clicked_at TIMESTAMPTZ;

-- Unique (NULLS DISTINCT) so legacy rows without a search_id are not
-- accidentally collapsed. Postgres 15+ defaults to NULLS DISTINCT for
-- unique indexes, which is the behaviour we want.
CREATE UNIQUE INDEX IF NOT EXISTS idx_search_queries_search_id
    ON search_queries(search_id);

-- Covering index for the click-through dashboard:
-- most queries aggregate CTR over the last 24h bucketed by search_id.
CREATE INDEX IF NOT EXISTS idx_search_queries_clicked_at
    ON search_queries(clicked_at DESC)
 WHERE clicked_at IS NOT NULL;

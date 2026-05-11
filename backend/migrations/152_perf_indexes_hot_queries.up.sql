-- PERF-E: covering indexes on hot read paths.
--
-- Source: perf-audit.md (2026-05-11). EXPLAIN ANALYZE on the local box
-- showed planning time dominating execution time on three hot queries
-- (user_sessions.ListActiveByUser, audit_logs.ListByUser,
-- profile_view_events visibility totals). The tables are small today,
-- so the planner picks Seq Scan; the indexes added here will be
-- selected once the row count grows beyond a few hundred.
--
-- CONCURRENTLY is intentionally NOT used: golang-migrate wraps each
-- migration in a transaction, and CREATE INDEX CONCURRENTLY is
-- forbidden inside a tx (we fought this on B.6.1). Tables here are
-- small enough that a short ACCESS EXCLUSIVE lock during build is
-- acceptable. When any of these tables grows past ~10k rows, switch
-- to the manual concurrent workflow described in
-- backend/migrations/README.md.
--
-- IF NOT EXISTS keeps the migration idempotent on partial-apply retry.
-- See feedback_migration_safety.md.

BEGIN;

-- ---------------------------------------------------------------------
-- user_sessions.ListActiveByUser — auth + session-management hot path.
--
-- Query (backend/internal/adapter/postgres/session_repository.go:166):
--   SELECT ... FROM user_sessions
--   WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
--   ORDER BY expires_at DESC
--   LIMIT $2;
--
-- Existing index idx_user_sessions_user_id (user_id, expires_at DESC)
-- already covers the lookup, but it includes revoked rows and still
-- pays a Filter step on revoked_at. A PARTIAL index keyed only on
-- non-revoked sessions:
--   * shrinks the index to the rows the query actually reads
--     (revoked sessions are rare and never returned by this query);
--   * drops the post-scan filter, making the plan a pure Index Scan;
--   * collapses planning time once the planner sees a tighter index
--     stat distribution (planning was 1.8 ms even on 171 rows).
-- ---------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_user_sessions_active_by_user
    ON user_sessions (user_id, expires_at DESC)
    WHERE revoked_at IS NULL;

-- ---------------------------------------------------------------------
-- audit_logs.ListByUser — admin / GDPR / self-audit pagination.
--
-- Query (backend/internal/adapter/postgres/audit_repository.go:262):
--   SELECT ... FROM audit_logs
--   WHERE user_id = $1
--   ORDER BY created_at DESC, id DESC
--   LIMIT $2;
--
-- Existing indexes:
--   * idx_audit_logs_user_id        — partial, (user_id) only
--   * idx_audit_logs_created_at     — (created_at DESC)
-- Neither supports the cursor-paginated lookup keyed on user_id then
-- sorted by (created_at, id). The planner Seq-Scans the table and then
-- Sorts the matching rows — fine at 171 rows, but linear at 10k+ and
-- log-linear at 1M+.
-- The new composite partial index handles the entire WHERE + ORDER BY
-- in a single Index Scan, and the partial clause (user_id IS NOT NULL)
-- keeps it cheap because anonymous events are still served by the
-- existing date index.
-- ---------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_time_series
    ON audit_logs (user_id, created_at DESC, id DESC)
    WHERE user_id IS NOT NULL;

-- ---------------------------------------------------------------------
-- profile_view_events visibility totals — owner-only stats dashboard.
--
-- Query (backend/internal/adapter/postgres/profile_view_repository.go:118):
--   SELECT COUNT(*),
--          COUNT(DISTINCT (viewer_ip_anonymized, viewer_ua_hash)),
--          COUNT(*) FILTER (WHERE came_from = 'search'),
--          AVG(search_position) FILTER (WHERE search_position IS NOT NULL)
--   FROM profile_view_events
--   WHERE organization_id = $1
--     AND created_at >= NOW() - $2 * INTERVAL '1 day';
--
-- Existing idx_pve_org_created (organization_id, created_at DESC)
-- already restricts the row set, but the aggregate still reads heap
-- pages for came_from / search_position. Adding came_from to the
-- composite key lets the FILTER (WHERE came_from = 'search')
-- counter become satisfiable from index-only data when only the
-- search_appearances aggregate is needed, and shaves heap fetches on
-- the wider query as the buffer-pool grows colder.
-- search_position is not added: it is mostly NULL and including it
-- would inflate the index size for marginal benefit.
-- ---------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_pve_org_time_came_from
    ON profile_view_events (organization_id, created_at DESC, came_from);

COMMIT;

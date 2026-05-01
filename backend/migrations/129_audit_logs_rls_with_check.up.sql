-- 129_audit_logs_rls_with_check.up.sql
--
-- BUG-NEW-07 — audit_logs RLS policy `audit_logs_isolation` was created
-- with USING only and no WITH CHECK clause (migration 125). PostgreSQL
-- defaults a missing WITH CHECK to mirror USING, which means INSERT
-- attempts evaluate the USING expression against the NEW row:
--
--   user_id = current_setting('app.current_user_id', true)::uuid
--
-- When the audit log is written from a code path that did NOT set
-- app.current_user_id (background workers, system-actor logs, login
-- failures BEFORE the tenant context is set), current_setting returns
-- NULL, the comparison evaluates to NULL, and the INSERT is REJECTED
-- by RLS — even though the application logic is correct.
--
-- The fix: explicitly set WITH CHECK (true) so the policy is read-only
-- enforcing. INSERTs always succeed (regardless of context), while
-- SELECT/UPDATE/DELETE remain filtered by the USING clause to the
-- caller's own audit trail.
--
-- This is correct semantically: audit_logs are append-only (migration
-- 124 revoked UPDATE/DELETE on the table) and the application is the
-- sole writer. The actor identity is preserved in the user_id column,
-- not in the RLS WITH CHECK — every audit row carries the actor id
-- the application supplies.
--
-- A side benefit: with WITH CHECK (true), background jobs (cron,
-- workers) that record system actions can write audit entries without
-- having to fake-set app.current_user_id to a sentinel uuid.

BEGIN;

-- Postgres has no ALTER POLICY ... ADD WITH CHECK syntax, so the
-- migration drops + recreates the policy. The brief "no app downtime"
-- holds because the table is append-only and the application writes
-- through INSERTs which would have failed anyway pre-fix on
-- unset-context paths.
DROP POLICY IF EXISTS audit_logs_isolation ON audit_logs;

CREATE POLICY audit_logs_isolation ON audit_logs
    USING (
        user_id = current_setting('app.current_user_id', true)::uuid
    )
    WITH CHECK (true);

COMMIT;

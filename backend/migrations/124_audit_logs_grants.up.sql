-- 124_audit_logs_grants.up.sql
--
-- SEC-13 (audit 2026-04-29): make audit_logs append-only at the
-- database layer.
--
-- Migration 078 created the table with a documentation-only "never
-- UPDATE or DELETE" invariant. That comment is not enforced — any code
-- with INSERT also has UPDATE and DELETE by default in PostgreSQL.
--
-- This migration revokes UPDATE and DELETE on audit_logs from the
-- PUBLIC role. Because PostgreSQL grants table privileges to the OWNER
-- (the migration user) by default, and grants nothing to PUBLIC for
-- new tables, the REVOKE FROM PUBLIC below is symbolic — it documents
-- intent and protects against a future grant accidentally widening the
-- scope.
--
-- The real protection requires a dedicated application database user
-- separate from the migration owner, with explicit INSERT + SELECT
-- grants and NO UPDATE/DELETE. The standard wiring is:
--
--   CREATE ROLE marketplace_app LOGIN PASSWORD '...';
--   GRANT CONNECT ON DATABASE marketplace_go TO marketplace_app;
--   GRANT USAGE ON SCHEMA public TO marketplace_app;
--   GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO marketplace_app;
--   -- Then narrow audit_logs:
--   REVOKE UPDATE, DELETE ON audit_logs FROM marketplace_app;
--   GRANT SELECT, INSERT ON audit_logs TO marketplace_app;
--
-- That role split is INFRA work (Railway / Neon dashboards), tracked
-- as a follow-up. This migration codifies the intent so any future
-- grant on audit_logs is explicit.

REVOKE UPDATE, DELETE ON audit_logs FROM PUBLIC;

-- Document the rule directly on the table comment so DB introspection
-- tools (psql \d, pgAdmin, datagrip) surface it next to the schema.
COMMENT ON TABLE audit_logs IS
    'Append-only audit trail. UPDATE and DELETE are revoked from PUBLIC '
    '(migration 124). The application database user MUST have only '
    'INSERT + SELECT grants — verify with \\dp audit_logs.';

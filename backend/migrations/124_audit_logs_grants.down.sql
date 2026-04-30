-- 124_audit_logs_grants.down.sql
--
-- Reverses the symbolic REVOKE on audit_logs and restores the original
-- table comment from migration 078.
--
-- Note: this rollback is symbolic — it re-grants UPDATE/DELETE to
-- PUBLIC, which by default has no membership of any application role.
-- A real rollback in prod would also require restoring any
-- audit_logs-specific grants on the application user.

GRANT UPDATE, DELETE ON audit_logs TO PUBLIC;

COMMENT ON TABLE audit_logs IS
    'Append-only audit trail for security-sensitive mutations. Never UPDATE or DELETE.';

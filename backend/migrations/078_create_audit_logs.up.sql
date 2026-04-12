-- 078_create_audit_logs.up.sql
--
-- Append-only audit trail for security-sensitive mutations.
--
-- This table is the permanent record of WHO did WHAT to WHICH resource
-- and WHEN. It is append-only by convention — the application code
-- NEVER issues UPDATE or DELETE against this table. In production, the
-- application database user should be granted only INSERT and SELECT
-- on audit_logs to enforce this invariant at the DB level.
--
-- The first consumer is the role-permissions editor (migration 077):
-- every Owner edit of a role override creates one audit row per
-- changed cell so the full history is preserved.
--
-- Schema mirrors the CLAUDE.md spec:
--   - user_id        — the actor (nullable for system events)
--   - action         — machine-readable event key (snake_case)
--   - resource_type  — "organization", "user", "mission", …
--   - resource_id    — the target resource's id (nullable)
--   - metadata       — free-form JSON context (old/new values, IP, UA)
--   - ip_address     — source IP (may be nullable for background jobs)
--   - created_at     — immutable timestamp
--
-- Indexes cover the three common access patterns: "show me everything
-- this user did", "show me every time this event fired", and "show me
-- the audit trail for this resource".

CREATE TABLE IF NOT EXISTS audit_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    action        TEXT NOT NULL,
    resource_type TEXT,
    resource_id   UUID,
    metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip_address    INET,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id) WHERE resource_id IS NOT NULL;

COMMENT ON TABLE audit_logs IS 'Append-only audit trail for security-sensitive mutations. Never UPDATE or DELETE.';

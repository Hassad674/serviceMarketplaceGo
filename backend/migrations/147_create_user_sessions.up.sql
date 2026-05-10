-- B.4: server-side audit trail of authentication sessions.
--
-- The refresh-token rotation (SEC-06) already prevents replay via a
-- Redis blacklist, but it leaves no auditable record of when a session
-- was created, last refreshed, or revoked. user_sessions persists that
-- audit trail so:
--
--   - The future "Sécurité" page (web/mobile task #29) can list every
--     active session attached to an account and let the user revoke
--     them individually or globally.
--   - "I logged out everywhere" can be proven by looking at revoked_at
--     across the family of sessions chained via parent_jti.
--   - Stolen-token reuse detection has a forensic surface on top of
--     the existing blacklist alarm — the SOC can replay the rotation
--     chain to spot the divergence point.
--
-- Cleanup: revoked sessions older than 30 days are purged by the
-- retention scheduler (see domain/retention/policies.go). Active
-- sessions are kept until they expire naturally.
--
-- Down migration drops the table and indexes.

CREATE TABLE IF NOT EXISTS user_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    jti             TEXT UNIQUE NOT NULL,
    parent_jti      TEXT NULL,
    user_agent_hash TEXT NOT NULL,
    ip_anonymized   INET NOT NULL,
    login_method    TEXT NOT NULL CHECK (login_method IN (
        'password',
        'invitation',
        'token_bridge',
        'refresh',
        'admin_impersonation'
    )),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ NULL
);

-- (user_id, expires_at DESC) supports "list active sessions for
-- this user, newest expiry first" — the dominant query for the
-- future Sécurité page.
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id
    ON user_sessions(user_id, expires_at DESC);

-- jti lookup is the hot path on every refresh / logout.
CREATE INDEX IF NOT EXISTS idx_user_sessions_jti
    ON user_sessions(jti);

-- (revoked_at, expires_at) supports the retention sweep filter
-- "revoked AND stale" without a Seq Scan once the table grows.
CREATE INDEX IF NOT EXISTS idx_user_sessions_revoked_expires
    ON user_sessions(revoked_at, expires_at)
    WHERE revoked_at IS NOT NULL;

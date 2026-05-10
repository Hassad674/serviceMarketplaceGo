-- B.6.1: Email 2FA backend.
--
-- Creates the two_factor_challenges table — append-only-ish ledger of
-- one-time email codes issued during the login flow when a user has
-- opted into email 2FA. Each row stores a bcrypt hash of the 6-digit
-- code (NEVER the code itself), the remaining attempts counter (default
-- 5), an expiry timestamp (10 min from issuance enforced at the app
-- layer), and forensic fingerprint (anonymized IP + hashed user-agent)
-- so admins can correlate successful and abusive attempts.
--
-- The table also gains a partial index on (user_id, used_at, expires_at)
-- where used_at IS NULL — this is the dominant query for the verify
-- handler ("find the latest pending challenge for this user").
--
-- Additionally, add the `two_factor_email_enabled` boolean to `users`
-- so the login service knows whether to issue tokens immediately or
-- gate behind a 2FA challenge. NULL/false = not enrolled (default).
--
-- Down migration drops the table, the index, and the user column.

CREATE TABLE IF NOT EXISTS two_factor_challenges (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash       TEXT NOT NULL,
    attempts_left   INT NOT NULL DEFAULT 5,
    expires_at      TIMESTAMPTZ NOT NULL,
    used_at         TIMESTAMPTZ NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    client_ip       INET NULL,
    user_agent_hash TEXT NULL
);

-- (user_id, used_at, expires_at) partial index — supports
-- "find latest pending challenge for this user" without scanning
-- expired/used rows. The verify handler calls this on every 2FA
-- submission so the index pays for itself immediately.
-- Note: CONCURRENTLY removed because golang-migrate wraps each migration
-- in a transaction by default and CREATE INDEX CONCURRENTLY cannot run
-- inside one. The table is empty at deploy time, so the brief AccessShare
-- lock is negligible.
CREATE INDEX IF NOT EXISTS idx_2fa_user_pending
    ON two_factor_challenges(user_id, used_at, expires_at)
    WHERE used_at IS NULL;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS two_factor_email_enabled BOOLEAN NOT NULL DEFAULT false;

-- Phase A.3 of the GDPR roadmap (gdpr-roadmap.md).
--
-- Records every consent decision (accept_all / refuse_all / custom)
-- made by a visitor through the cookie banner. Provides server-side
-- proof of consent for CNIL inquiries — the localStorage-only signal
-- shipping today is not auditable.
--
-- Privacy posture:
--   * user_id is nullable: anonymous visitors are tracked via a
--     short-lived session_id only.
--   * ip_anonymized stores the truncated IP (IPv4 /16, IPv6 /32) per
--     gdpr.TruncateIP — the raw IP NEVER lands in this table.
--   * user_agent_hash stores a SHA-256 (hex) of the UA so we can
--     correlate replays without persisting the raw UA string.
--   * categories is an enum-like text[] (analytics, marketing,
--     functional) so the schema does not lock to today's two-vendor
--     setup.
--
-- The action CHECK constraint pins the enum surface so a typo never
-- lands in production.

BEGIN;

CREATE TABLE IF NOT EXISTS consent_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    session_id      TEXT NULL,
    categories      TEXT[] NOT NULL,
    action          TEXT NOT NULL CHECK (action IN ('accept_all', 'refuse_all', 'custom')),
    ip_anonymized   TEXT NOT NULL,
    user_agent_hash TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_consent_log_user_id
    ON consent_log (user_id)
    WHERE user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_consent_log_created_at
    ON consent_log (created_at);

COMMIT;

-- Phase B.5 of the GDPR roadmap (gdpr-roadmap.md) — Art. 22 disclosure.
--
-- Stores user-facing appeals against the platform's three automated
-- decisions: AI moderation (text + media), search ranking, and Stripe
-- payment risk scoring. RGPD art. 22 grants every data subject the
-- right to obtain human review of a decision based solely on automated
-- processing — this table is the persistence layer for that workflow.
--
-- Privacy posture:
--   * user_id FK to users(id) with ON DELETE CASCADE — when a user is
--     purged, their appeals follow.
--   * decision_type CHECK pins the enum to the three documented surfaces
--     (moderation | ranking | payment) so a typo never lands in prod.
--   * reference_id is intentionally TEXT — moderation result IDs are
--     UUIDs, Typesense ranking traces are short hashes, Stripe payment
--     intent IDs are pi_-prefixed strings. Free-form keeps the table
--     evolutive without a join table per surface.
--   * status default 'pending' + CHECK pins the lifecycle to
--     pending | reviewing | upheld | overturned.
--   * reason is plain TEXT (no length cap at the schema level — the
--     handler caps to 5_000 bytes).

BEGIN;

CREATE TABLE IF NOT EXISTS automated_decision_appeals (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    decision_type  TEXT NOT NULL CHECK (decision_type IN ('moderation', 'ranking', 'payment')),
    reference_id   TEXT NOT NULL,
    reason         TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending', 'reviewing', 'upheld', 'overturned')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_automated_decision_appeals_user_id
    ON automated_decision_appeals (user_id);

CREATE INDEX IF NOT EXISTS idx_automated_decision_appeals_status_created_at
    ON automated_decision_appeals (status, created_at DESC);

COMMIT;

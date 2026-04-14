-- pending_events unifies two patterns into one durable table:
--   1. Scheduled events (auto-approval, fund reminders, auto-close)
--   2. Stripe outbox (transfers that must happen exactly once with retry)
--
-- Writers INSERT a row in the same transaction as the state change
-- that should trigger it. A background worker pops due events with
-- FOR UPDATE SKIP LOCKED and dispatches to type-specific handlers.
--
-- Rationale: a single observable queue beats an in-process cron +
-- scattered direct Stripe calls. See docs/ARCHITECTURE.md (to be
-- written in phase 9) for the full pattern explanation.
CREATE TABLE IF NOT EXISTS pending_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type   TEXT NOT NULL,
    payload      JSONB NOT NULL,
    fires_at     TIMESTAMPTZ NOT NULL,

    status       TEXT NOT NULL DEFAULT 'pending',
    attempts     INT  NOT NULL DEFAULT 0,
    last_error   TEXT,

    processed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER pending_events_updated_at
    BEFORE UPDATE ON pending_events
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- Partial index for the worker hot path: only look at rows that might
-- be due. Drops to near-zero cost once backlog is empty.
CREATE INDEX idx_pending_events_due
    ON pending_events(fires_at, id)
    WHERE status IN ('pending', 'failed');

CREATE INDEX idx_pending_events_type
    ON pending_events(event_type);

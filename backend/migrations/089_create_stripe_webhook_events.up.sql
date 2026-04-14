-- Idempotency table for Stripe webhook events. Stripe delivers the
-- same event multiple times in some failure scenarios; we insert the
-- event id on first receipt and short-circuit duplicates.
--
-- INSERT ... ON CONFLICT DO NOTHING + check rows-affected is the
-- atomic way to claim an event without races.
CREATE TABLE IF NOT EXISTS stripe_webhook_events (
    stripe_event_id TEXT PRIMARY KEY,
    event_type      TEXT NOT NULL,
    processed_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_stripe_webhook_events_type ON stripe_webhook_events(event_type, processed_at DESC);

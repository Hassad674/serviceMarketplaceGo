-- 133_pending_events_stripe_event_id.down.sql
--
-- Rolls back the P8 Stripe-event-id deduplication column.
--
-- Drops the partial unique index first, then the column. IF EXISTS
-- guards keep the down idempotent across partial states.

BEGIN;

DROP INDEX IF EXISTS idx_pending_events_stripe_event_id_unique;

ALTER TABLE pending_events
    DROP COLUMN IF EXISTS stripe_event_id;

COMMIT;

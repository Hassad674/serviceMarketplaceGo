-- 134_pending_events_stripe_event_id.up.sql
--
-- P8 (Stripe webhook async): adds `stripe_event_id` for deduplication
-- of Stripe webhook events enqueued onto pending_events.
--
-- Background: the Stripe webhook handler used to dispatch handlers
-- inline (PDF generation, DB writes, email sends) and could exceed the
-- 10s Stripe timeout, triggering retries. P8 moves dispatch to the
-- pending_events worker and returns 200 OK immediately. Stripe
-- re-deliveries (after a 5xx or transient timeout) MUST NOT enqueue a
-- second pending_event row for the same event — otherwise the worker
-- would process the event twice.
--
-- Why a dedicated column (not the generic `id`):
--   - pending_events.id is a synthesised UUID; we cannot use Stripe's
--     evt_* identifier directly because non-Stripe pending events
--     (milestone_auto_approve, search.reindex, etc.) need their own
--     UUIDs and would clash on a hard NOT NULL UNIQUE.
--   - Adding `stripe_event_id TEXT NULL` keeps existing event types
--     untouched and lets the partial unique index carry only Stripe
--     rows.
--
-- Why a partial unique index:
--   - The constraint applies ONLY to rows where stripe_event_id IS
--     NOT NULL. Other event types are unaffected.
--   - INSERT ... ON CONFLICT (stripe_event_id) WHERE stripe_event_id
--     IS NOT NULL DO NOTHING is the idempotency primitive: a
--     re-delivered Stripe event becomes a no-op, never a duplicate
--     row.
--
-- The ALTER TABLE ADD COLUMN is a fast metadata-only operation
-- (Postgres 11+) because the column is nullable with no default. No
-- backfill needed.

BEGIN;

ALTER TABLE pending_events
    ADD COLUMN IF NOT EXISTS stripe_event_id TEXT NULL;

-- Partial unique index used by the webhook enqueue path's
-- ON CONFLICT (stripe_event_id) clause to make Stripe re-deliveries
-- idempotent. Only Stripe rows occupy the index — non-Stripe event
-- types stay free.
CREATE UNIQUE INDEX IF NOT EXISTS idx_pending_events_stripe_event_id_unique
    ON pending_events (stripe_event_id)
    WHERE stripe_event_id IS NOT NULL;

COMMIT;

ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_pending_tuple_all_or_none;
DROP INDEX IF EXISTS idx_subscriptions_schedule_id;
ALTER TABLE subscriptions
    DROP COLUMN IF EXISTS pending_billing_cycle,
    DROP COLUMN IF EXISTS pending_cycle_effective_at,
    DROP COLUMN IF EXISTS stripe_schedule_id;

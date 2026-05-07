-- Receipt feature rollback.
ALTER TABLE payment_records
    DROP COLUMN IF EXISTS billing_snapshot;

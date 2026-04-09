ALTER TABLE disputes
    DROP COLUMN IF EXISTS cancellation_requested_by,
    DROP COLUMN IF EXISTS cancellation_requested_at;

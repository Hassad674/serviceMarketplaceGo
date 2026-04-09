-- Cancellation request flow: when the respondent has already replied to the
-- dispute, the initiator can no longer cancel directly — they must send a
-- cancellation request that the respondent accepts or refuses.
--
-- NULL = no cancellation request pending.
ALTER TABLE disputes
    ADD COLUMN IF NOT EXISTS cancellation_requested_by UUID REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS cancellation_requested_at TIMESTAMPTZ;

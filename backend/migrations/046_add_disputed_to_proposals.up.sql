-- Add active_dispute_id to proposals to track the current dispute.
-- No FK constraint: disputes is a separate feature's table.
ALTER TABLE proposals ADD COLUMN IF NOT EXISTS active_dispute_id UUID;

-- Track the most recent dispute that has ever existed on a proposal,
-- regardless of its current status. Set when a dispute is opened and
-- NEVER cleared, so the project page can display historical decisions
-- (split + admin note) even after the dispute has been resolved or
-- cancelled and the proposal restored to active/completed.
--
-- No FK constraint on disputes(id) — the project rules forbid
-- cross-feature foreign keys; integrity is maintained at the
-- application level.
ALTER TABLE proposals
    ADD COLUMN IF NOT EXISTS last_dispute_id UUID;

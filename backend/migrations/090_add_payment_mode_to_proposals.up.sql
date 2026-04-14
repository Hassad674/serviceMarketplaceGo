-- payment_mode is a pure UX hint: it tells the frontend which form to
-- render (simple single-amount vs. multi-milestone editor). The backend
-- treats all proposals uniformly via their milestones — a one_time
-- proposal just has exactly one milestone.
--
-- Default is 'one_time' for backward compatibility with every existing
-- proposal, all of which will be backfilled with a single synthetic
-- milestone in the next migration.
ALTER TABLE proposals
    ADD COLUMN IF NOT EXISTS payment_mode TEXT NOT NULL DEFAULT 'one_time'
        CHECK (payment_mode IN ('one_time', 'milestone'));

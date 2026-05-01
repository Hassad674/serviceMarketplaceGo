-- BUG-NEW-03: pending_events stuck forever in 'processing' after a worker crash.
--
-- The original PopDue query only claimed rows in ('pending', 'failed').
-- If a worker crashed (or the process was OOM-killed) AFTER the SKIP-LOCKED
-- claim flipped the status to 'processing' but BEFORE MarkDone / MarkFailed
-- ran, the row was orphaned: no other worker would ever touch it again
-- (PopDue ignored 'processing'), and there was no reaper.
--
-- The fix is twofold and lives in the application layer:
--   1) PopDue now ALSO claims rows with status='processing' whose
--      updated_at is older than a threshold (default 5 minutes). The
--      threshold is wider than any reasonable handler runtime so we do
--      not kick a still-running handler off mid-flight.
--   2) PopDue's UPDATE refreshes updated_at on every claim, so the
--      "claim age" naturally tracks the most recent worker that owned
--      the row. The pending_events_updated_at trigger already does this
--      automatically.
--
-- This migration adds a partial index supporting the stale-processing
-- branch of PopDue. The existing idx_pending_events_due index covers
-- the (pending | failed) branch and is left untouched.
--
-- Notes:
--   - We use a partial index because <0.1% of rows are ever in
--     'processing' (only between PopDue and MarkDone/Failed).
--   - The index ordering is (updated_at, id) so the planner can serve
--     ORDER BY updated_at + LIMIT directly from the index.
CREATE INDEX IF NOT EXISTS idx_pending_events_stale_processing
    ON pending_events(updated_at, id)
    WHERE status = 'processing';

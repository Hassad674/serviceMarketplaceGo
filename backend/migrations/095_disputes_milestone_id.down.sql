-- Down: relax milestone_id back to nullable and drop it. The backfill
-- step is destructive (the column itself is removed) but only safe in
-- local development — production reverts must stay in their own
-- corrective migration per the migration safety rules in CLAUDE.md.
ALTER TABLE disputes
    ALTER COLUMN milestone_id DROP NOT NULL;

DROP INDEX IF EXISTS idx_disputes_milestone;

ALTER TABLE disputes DROP COLUMN IF EXISTS milestone_id;

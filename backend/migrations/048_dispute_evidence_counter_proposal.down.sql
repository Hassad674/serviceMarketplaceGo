DROP INDEX IF EXISTS idx_dispute_evidence_cp_id;
ALTER TABLE dispute_evidence DROP COLUMN IF EXISTS counter_proposal_id;

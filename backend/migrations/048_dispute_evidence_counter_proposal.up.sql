-- Allow dispute_evidence to be linked to a specific counter-proposal.
-- NULL = evidence attached to the dispute itself (initial filing).
-- Non-NULL = evidence attached to a specific counter-proposal.
ALTER TABLE dispute_evidence
    ADD COLUMN IF NOT EXISTS counter_proposal_id UUID
    REFERENCES dispute_counter_proposals(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_dispute_evidence_cp_id
    ON dispute_evidence(counter_proposal_id)
    WHERE counter_proposal_id IS NOT NULL;

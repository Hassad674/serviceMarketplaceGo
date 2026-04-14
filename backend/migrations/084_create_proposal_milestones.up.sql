-- Milestones are the per-step funding+delivery units of a proposal.
-- Every proposal has at least one milestone; a fixed-price "one-time"
-- mission is modelled internally as a single-milestone proposal.
--
-- Amount is stored in centimes (1 EUR = 100). There is no minimum
-- amount at the DB level: the CHECK only rejects zero and negatives.
CREATE TABLE IF NOT EXISTS proposal_milestones (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    proposal_id        UUID NOT NULL REFERENCES proposals(id) ON DELETE RESTRICT,

    sequence           SMALLINT NOT NULL CHECK (sequence >= 1),
    title              TEXT NOT NULL,
    description        TEXT NOT NULL,
    amount             BIGINT NOT NULL CHECK (amount > 0),
    deadline           DATE,

    status             TEXT NOT NULL DEFAULT 'pending_funding',

    -- Optimistic concurrency counter. Every update bumps this and the
    -- WHERE clause checks it; a mismatch yields zero rows and the app
    -- layer surfaces ErrConcurrentUpdate.
    version            INT NOT NULL DEFAULT 0,

    -- Lifecycle timestamps. Set exactly once per transition except
    -- submitted_at which is cleared on Reject so the next submit
    -- restarts the auto-approval timer from zero.
    funded_at          TIMESTAMPTZ,
    submitted_at       TIMESTAMPTZ,
    approved_at        TIMESTAMPTZ,
    released_at        TIMESTAMPTZ,
    disputed_at        TIMESTAMPTZ,
    cancelled_at       TIMESTAMPTZ,

    -- Dispute refs follow the same pattern as proposals: active_* is
    -- cleared on resolution, last_* is kept forever for historical display.
    active_dispute_id  UUID,
    last_dispute_id    UUID,

    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Sequences are unique per proposal: you cannot have two "step 1"s.
    CONSTRAINT proposal_milestones_sequence_unique UNIQUE (proposal_id, sequence)
);

CREATE TRIGGER proposal_milestones_updated_at
    BEFORE UPDATE ON proposal_milestones
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- Read indexes for the common access patterns.
CREATE INDEX idx_proposal_milestones_proposal ON proposal_milestones(proposal_id);
CREATE INDEX idx_proposal_milestones_status   ON proposal_milestones(status);

-- Partial index for the scheduler: scan only milestones whose auto-
-- approval timer might need firing. Orders of magnitude faster than
-- scanning the whole table at every tick.
CREATE INDEX idx_proposal_milestones_submitted_auto_approve
    ON proposal_milestones(submitted_at)
    WHERE status = 'submitted';

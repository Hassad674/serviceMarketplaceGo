-- Append-only audit trail of every milestone state transition.
-- Written in the same transaction as the status update so that the
-- log is always consistent with the entity.
--
-- This table is NEVER updated or deleted at the application level.
-- For compliance, application DB users should only hold INSERT/SELECT
-- grants on this table (enforced in the production grant migration).
CREATE TABLE IF NOT EXISTS milestone_transitions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    milestone_id  UUID NOT NULL REFERENCES proposal_milestones(id) ON DELETE CASCADE,
    proposal_id   UUID NOT NULL REFERENCES proposals(id)           ON DELETE CASCADE,

    from_status   TEXT NOT NULL,
    to_status     TEXT NOT NULL,

    -- actor_id is NULL when the transition is performed by the system
    -- (auto-approval, auto-close, outbox handler). Non-null otherwise
    -- for the client/provider user who triggered the change.
    actor_id      UUID REFERENCES users(id),
    actor_org_id  UUID REFERENCES organizations(id),

    -- Free-form reason (e.g. "auto-approved after 7d", "rejected: needs revision").
    reason        TEXT,

    -- Arbitrary structured metadata (e.g. stripe_payment_intent_id,
    -- dispute_id, amount_delta on future contract-change amendments).
    metadata      JSONB,

    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_milestone_transitions_milestone ON milestone_transitions(milestone_id, created_at);
CREATE INDEX idx_milestone_transitions_proposal  ON milestone_transitions(proposal_id, created_at);
CREATE INDEX idx_milestone_transitions_actor     ON milestone_transitions(actor_id) WHERE actor_id IS NOT NULL;

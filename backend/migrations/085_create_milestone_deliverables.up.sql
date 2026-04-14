-- Deliverables attached to a specific milestone.
-- Proposal-level documents (the global brief, overall contract) live in
-- proposal_documents. Milestone-level deliverables are scoped to a
-- single step: clauses, references, work-in-progress artefacts.
CREATE TABLE IF NOT EXISTS milestone_deliverables (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    milestone_id  UUID NOT NULL REFERENCES proposal_milestones(id) ON DELETE CASCADE,
    filename      TEXT NOT NULL,
    url           TEXT NOT NULL,
    size          BIGINT NOT NULL CHECK (size > 0),
    mime_type     TEXT NOT NULL,
    uploaded_by   UUID NOT NULL REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_milestone_deliverables_milestone ON milestone_deliverables(milestone_id);
CREATE INDEX idx_milestone_deliverables_uploader  ON milestone_deliverables(uploaded_by);

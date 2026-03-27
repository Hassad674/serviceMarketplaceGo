CREATE TABLE IF NOT EXISTS proposals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id),
    sender_id       UUID NOT NULL REFERENCES users(id),
    recipient_id    UUID NOT NULL REFERENCES users(id),

    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    amount          BIGINT NOT NULL,
    deadline        DATE,

    status          TEXT NOT NULL DEFAULT 'pending',

    parent_id       UUID REFERENCES proposals(id),
    version         INT NOT NULL DEFAULT 1,

    client_id       UUID NOT NULL REFERENCES users(id),
    provider_id     UUID NOT NULL REFERENCES users(id),

    metadata        JSONB,

    accepted_at     TIMESTAMPTZ,
    declined_at     TIMESTAMPTZ,
    paid_at         TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER proposals_updated_at
    BEFORE UPDATE ON proposals
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_proposals_conversation ON proposals(conversation_id);
CREATE INDEX idx_proposals_sender ON proposals(sender_id);
CREATE INDEX idx_proposals_recipient ON proposals(recipient_id);
CREATE INDEX idx_proposals_client ON proposals(client_id);
CREATE INDEX idx_proposals_provider ON proposals(provider_id);
CREATE INDEX idx_proposals_parent ON proposals(parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX idx_proposals_status ON proposals(status);

CREATE TABLE IF NOT EXISTS proposal_documents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    proposal_id UUID NOT NULL REFERENCES proposals(id) ON DELETE CASCADE,
    filename    TEXT NOT NULL,
    url         TEXT NOT NULL,
    size        BIGINT NOT NULL,
    mime_type   TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_proposal_documents_proposal ON proposal_documents(proposal_id);

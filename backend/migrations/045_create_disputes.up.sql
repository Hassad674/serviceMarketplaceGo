-- Dispute system: core tables for dispute resolution between clients and providers.
-- A dispute freezes funds on an active/completion_requested proposal until resolved.

CREATE TABLE disputes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    proposal_id     UUID NOT NULL REFERENCES proposals(id),
    conversation_id UUID NOT NULL,
    initiator_id    UUID NOT NULL REFERENCES users(id),
    respondent_id   UUID NOT NULL REFERENCES users(id),
    client_id       UUID NOT NULL REFERENCES users(id),
    provider_id     UUID NOT NULL REFERENCES users(id),

    reason          TEXT NOT NULL,
    description     TEXT NOT NULL,
    requested_amount BIGINT NOT NULL,
    proposal_amount  BIGINT NOT NULL,

    status          TEXT NOT NULL DEFAULT 'open',

    resolution_type           TEXT,
    resolution_amount_client  BIGINT,
    resolution_amount_provider BIGINT,
    resolved_by               UUID REFERENCES users(id),
    resolution_note           TEXT,
    ai_summary                TEXT,

    escalated_at              TIMESTAMPTZ,
    resolved_at               TIMESTAMPTZ,
    cancelled_at              TIMESTAMPTZ,
    last_activity_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    respondent_first_reply_at TIMESTAMPTZ,

    version    INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE dispute_evidence (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dispute_id  UUID NOT NULL REFERENCES disputes(id) ON DELETE CASCADE,
    uploader_id UUID NOT NULL REFERENCES users(id),
    filename    TEXT NOT NULL,
    url         TEXT NOT NULL,
    size        BIGINT NOT NULL,
    mime_type   TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE dispute_counter_proposals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dispute_id      UUID NOT NULL REFERENCES disputes(id) ON DELETE CASCADE,
    proposer_id     UUID NOT NULL REFERENCES users(id),
    amount_client   BIGINT NOT NULL,
    amount_provider BIGINT NOT NULL,
    message         TEXT,
    status          TEXT NOT NULL DEFAULT 'pending',
    responded_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_disputes_proposal_id ON disputes(proposal_id);
CREATE INDEX idx_disputes_initiator_id ON disputes(initiator_id);
CREATE INDEX idx_disputes_respondent_id ON disputes(respondent_id);
CREATE INDEX idx_disputes_status ON disputes(status);
CREATE INDEX idx_disputes_scheduler ON disputes(status, last_activity_at)
    WHERE status IN ('open', 'negotiation');
CREATE INDEX idx_dispute_evidence_dispute_id ON dispute_evidence(dispute_id);
CREATE INDEX idx_dispute_cps_dispute_id ON dispute_counter_proposals(dispute_id);

-- Auto-update updated_at (create function if not exists, then trigger)
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_disputes_updated_at
    BEFORE UPDATE ON disputes
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

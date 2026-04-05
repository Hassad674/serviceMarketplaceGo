CREATE TABLE IF NOT EXISTS credit_bonus_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    proposal_id UUID NOT NULL,
    client_card_fingerprint TEXT,
    credits_awarded INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending_review',
    block_reason TEXT,
    proposal_created_at TIMESTAMPTZ,
    proposal_paid_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_credit_bonus_log_provider ON credit_bonus_log(provider_id);
CREATE INDEX IF NOT EXISTS idx_credit_bonus_log_client ON credit_bonus_log(client_id);
CREATE INDEX IF NOT EXISTS idx_credit_bonus_log_provider_client ON credit_bonus_log(provider_id, client_id);
CREATE INDEX IF NOT EXISTS idx_credit_bonus_log_status ON credit_bonus_log(status);

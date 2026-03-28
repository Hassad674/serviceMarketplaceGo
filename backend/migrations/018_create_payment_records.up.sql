CREATE TABLE IF NOT EXISTS payment_records (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    proposal_id              UUID NOT NULL UNIQUE,
    client_id                UUID NOT NULL REFERENCES users(id),
    provider_id              UUID NOT NULL REFERENCES users(id),
    stripe_payment_intent_id TEXT UNIQUE,
    stripe_transfer_id       TEXT,

    proposal_amount          BIGINT NOT NULL,
    stripe_fee_amount        BIGINT NOT NULL DEFAULT 0,
    platform_fee_amount      BIGINT NOT NULL,
    client_total_amount      BIGINT NOT NULL,
    provider_payout          BIGINT NOT NULL,

    currency                 TEXT NOT NULL DEFAULT 'eur',
    status                   TEXT NOT NULL DEFAULT 'pending',
    transfer_status          TEXT NOT NULL DEFAULT 'pending',

    paid_at                  TIMESTAMPTZ,
    transferred_at           TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER payment_records_updated_at
    BEFORE UPDATE ON payment_records
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_payment_records_proposal ON payment_records(proposal_id);
CREATE INDEX idx_payment_records_client ON payment_records(client_id);
CREATE INDEX idx_payment_records_provider ON payment_records(provider_id);
CREATE INDEX idx_payment_records_pi ON payment_records(stripe_payment_intent_id)
    WHERE stripe_payment_intent_id IS NOT NULL;

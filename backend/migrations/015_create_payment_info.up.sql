CREATE TABLE IF NOT EXISTS payment_info (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,

    -- Personal / Representative
    first_name           TEXT NOT NULL,
    last_name            TEXT NOT NULL,
    date_of_birth        DATE NOT NULL,
    nationality          TEXT NOT NULL,
    address              TEXT NOT NULL,
    city                 TEXT NOT NULL,
    postal_code          TEXT NOT NULL,

    -- Business (nullable)
    is_business          BOOLEAN NOT NULL DEFAULT false,
    business_name        TEXT,
    business_address     TEXT,
    business_city        TEXT,
    business_postal_code TEXT,
    business_country     TEXT,
    tax_id               TEXT,
    vat_number           TEXT,
    role_in_company      TEXT,

    -- Bank account
    iban                 TEXT,
    bic                  TEXT,
    account_number       TEXT,
    routing_number       TEXT,
    account_holder       TEXT NOT NULL,
    bank_country         TEXT,

    -- Stripe Connect (future)
    stripe_account_id    TEXT,
    stripe_verified      BOOLEAN NOT NULL DEFAULT false,

    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER payment_info_updated_at
    BEFORE UPDATE ON payment_info
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_payment_info_user ON payment_info(user_id);

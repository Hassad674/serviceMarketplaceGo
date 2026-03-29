CREATE TABLE IF NOT EXISTS business_persons (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role           TEXT NOT NULL,
    first_name     TEXT NOT NULL,
    last_name      TEXT NOT NULL,
    date_of_birth  DATE,
    email          TEXT,
    phone          TEXT,
    address        TEXT,
    city           TEXT,
    postal_code    TEXT,
    title          TEXT,
    stripe_person_id TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER business_persons_updated_at
    BEFORE UPDATE ON business_persons
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_business_persons_user ON business_persons(user_id);

ALTER TABLE payment_info
    ADD COLUMN IF NOT EXISTS is_self_representative BOOLEAN DEFAULT true,
    ADD COLUMN IF NOT EXISTS is_self_director BOOLEAN DEFAULT true,
    ADD COLUMN IF NOT EXISTS no_major_owners BOOLEAN DEFAULT true,
    ADD COLUMN IF NOT EXISTS is_self_executive BOOLEAN DEFAULT true;

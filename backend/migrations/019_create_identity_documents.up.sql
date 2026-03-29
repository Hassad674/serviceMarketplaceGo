CREATE TABLE IF NOT EXISTS identity_documents (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category         TEXT NOT NULL,
    document_type    TEXT NOT NULL,
    side             TEXT NOT NULL,
    file_key         TEXT NOT NULL,
    stripe_file_id   TEXT,
    status           TEXT NOT NULL DEFAULT 'pending',
    rejection_reason TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER identity_documents_updated_at
    BEFORE UPDATE ON identity_documents
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE INDEX idx_identity_documents_user ON identity_documents(user_id);
CREATE INDEX idx_identity_documents_pending ON identity_documents(user_id, status)
    WHERE status = 'pending';
CREATE UNIQUE INDEX idx_identity_documents_unique_side
    ON identity_documents(user_id, category, document_type, side);

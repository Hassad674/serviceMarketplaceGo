CREATE TABLE IF NOT EXISTS message_history (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id   UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    content      TEXT NOT NULL,
    action       TEXT NOT NULL CHECK (action IN ('edited', 'deleted')),
    performed_by UUID NOT NULL REFERENCES users(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_message_history_message_id ON message_history(message_id, created_at);

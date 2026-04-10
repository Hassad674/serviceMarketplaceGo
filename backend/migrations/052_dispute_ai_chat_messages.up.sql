-- Persistent storage for the admin AI chat on each dispute. Each row is
-- one turn (admin question or assistant answer). Append-only by design,
-- read in chronological order via the (dispute_id, created_at) index.
--
-- ON DELETE CASCADE: chat history has no meaning without its dispute.
CREATE TABLE IF NOT EXISTS dispute_ai_chat_messages (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dispute_id    UUID NOT NULL REFERENCES disputes(id) ON DELETE CASCADE,
    role          TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
    content       TEXT NOT NULL,
    input_tokens  INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dispute_ai_chat_dispute
    ON dispute_ai_chat_messages(dispute_id, created_at ASC);

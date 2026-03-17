CREATE TABLE IF NOT EXISTS test_words (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    word TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

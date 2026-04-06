-- Add text moderation fields to messages
ALTER TABLE messages ADD COLUMN IF NOT EXISTS moderation_status TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS moderation_score REAL NOT NULL DEFAULT 0;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS moderation_labels JSONB;

-- Add text moderation fields to reviews
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS moderation_status TEXT NOT NULL DEFAULT '';
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS moderation_score REAL NOT NULL DEFAULT 0;
ALTER TABLE reviews ADD COLUMN IF NOT EXISTS moderation_labels JSONB;

-- Index for admin queries filtering by moderation status
CREATE INDEX IF NOT EXISTS idx_messages_moderation_status ON messages(moderation_status) WHERE moderation_status != '';
CREATE INDEX IF NOT EXISTS idx_reviews_moderation_status ON reviews(moderation_status) WHERE moderation_status != '';

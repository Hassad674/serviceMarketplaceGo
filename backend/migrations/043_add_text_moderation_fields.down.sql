DROP INDEX IF EXISTS idx_reviews_moderation_status;
DROP INDEX IF EXISTS idx_messages_moderation_status;

ALTER TABLE reviews DROP COLUMN IF EXISTS moderation_labels;
ALTER TABLE reviews DROP COLUMN IF EXISTS moderation_score;
ALTER TABLE reviews DROP COLUMN IF EXISTS moderation_status;

ALTER TABLE messages DROP COLUMN IF EXISTS moderation_labels;
ALTER TABLE messages DROP COLUMN IF EXISTS moderation_score;
ALTER TABLE messages DROP COLUMN IF EXISTS moderation_status;

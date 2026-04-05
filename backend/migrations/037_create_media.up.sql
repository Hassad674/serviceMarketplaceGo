CREATE TABLE IF NOT EXISTS media (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uploader_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_url          TEXT NOT NULL,
    file_name         TEXT NOT NULL,
    file_type         TEXT NOT NULL,
    file_size         BIGINT NOT NULL DEFAULT 0,
    context           TEXT NOT NULL DEFAULT 'profile_photo',
    context_id        UUID,
    moderation_status TEXT NOT NULL DEFAULT 'pending',
    moderation_labels JSONB,
    moderation_score  REAL NOT NULL DEFAULT 0,
    reviewed_at       TIMESTAMPTZ,
    reviewed_by       UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_media_uploader_id ON media(uploader_id);
CREATE INDEX idx_media_moderation_status ON media(moderation_status);
CREATE INDEX idx_media_context ON media(context);
CREATE INDEX idx_media_created_at ON media(created_at DESC);

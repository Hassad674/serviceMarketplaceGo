-- Custom thumbnail for video portfolio media (e.g., when first frame is unattractive)
ALTER TABLE portfolio_media
    ADD COLUMN thumbnail_url TEXT NOT NULL DEFAULT '';

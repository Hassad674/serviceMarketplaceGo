CREATE TABLE profiles (
    user_id                  UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    title                    TEXT NOT NULL DEFAULT '',
    photo_url                TEXT NOT NULL DEFAULT '',
    presentation_video_url   TEXT NOT NULL DEFAULT '',
    referrer_video_url       TEXT NOT NULL DEFAULT '',
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER profiles_updated_at
    BEFORE UPDATE ON profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

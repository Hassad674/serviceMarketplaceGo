CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notification_type   TEXT NOT NULL,
    in_app              BOOLEAN NOT NULL DEFAULT true,
    push                BOOLEAN NOT NULL DEFAULT true,
    email               BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (user_id, notification_type)
);

CREATE TABLE IF NOT EXISTS device_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT NOT NULL,
    platform    TEXT NOT NULL CHECK (platform IN ('android', 'ios', 'web')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, token)
);

CREATE INDEX IF NOT EXISTS idx_device_tokens_user_id ON device_tokens(user_id);

CREATE TABLE IF NOT EXISTS notifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL DEFAULT '',
    data        JSONB DEFAULT '{}',
    read_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications(user_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread ON notifications(user_id, created_at DESC) WHERE read_at IS NULL;

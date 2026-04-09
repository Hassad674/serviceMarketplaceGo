-- Portfolio items: showcase projects for providers and agencies
CREATE TABLE IF NOT EXISTS portfolio_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    link_url    TEXT NOT NULL DEFAULT '',
    position    INT  NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_portfolio_items_user_id ON portfolio_items(user_id);
CREATE INDEX idx_portfolio_items_user_position ON portfolio_items(user_id, position);

CREATE TRIGGER update_portfolio_items_updated_at
    BEFORE UPDATE ON portfolio_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Portfolio media: images and videos attached to portfolio items
CREATE TABLE IF NOT EXISTS portfolio_media (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_item_id UUID NOT NULL REFERENCES portfolio_items(id) ON DELETE CASCADE,
    media_url         TEXT NOT NULL,
    media_type        TEXT NOT NULL CHECK (media_type IN ('image', 'video')),
    position          INT  NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_portfolio_media_item_id ON portfolio_media(portfolio_item_id);
CREATE INDEX idx_portfolio_media_item_position ON portfolio_media(portfolio_item_id, position);

BEGIN;

DROP INDEX IF EXISTS idx_portfolio_items_org_position;
DROP INDEX IF EXISTS idx_portfolio_items_org_id;

ALTER TABLE portfolio_items ADD COLUMN user_id UUID;

UPDATE portfolio_items pi
SET    user_id = o.owner_user_id
FROM   organizations o
WHERE  pi.organization_id = o.id;

ALTER TABLE portfolio_items ALTER COLUMN user_id SET NOT NULL;

ALTER TABLE portfolio_items
    ADD CONSTRAINT portfolio_items_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

CREATE INDEX idx_portfolio_items_user_id      ON portfolio_items (user_id);
CREATE INDEX idx_portfolio_items_user_position ON portfolio_items (user_id, position);

ALTER TABLE portfolio_items DROP COLUMN organization_id;

COMMIT;

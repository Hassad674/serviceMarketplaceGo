BEGIN;

DROP INDEX IF EXISTS idx_social_links_org;
DROP INDEX IF EXISTS social_links_org_platform_key;

ALTER TABLE social_links ADD COLUMN user_id UUID;

UPDATE social_links sl
SET    user_id = o.owner_user_id
FROM   organizations o
WHERE  sl.organization_id = o.id;

ALTER TABLE social_links ALTER COLUMN user_id SET NOT NULL;

ALTER TABLE social_links
    ADD CONSTRAINT social_links_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE social_links
    ADD CONSTRAINT social_links_user_id_platform_key UNIQUE (user_id, platform);

CREATE INDEX idx_social_links_user ON social_links (user_id);

ALTER TABLE social_links DROP COLUMN organization_id;

COMMIT;

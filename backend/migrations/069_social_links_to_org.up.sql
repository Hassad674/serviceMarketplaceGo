-- Phase R2 — Social links belong to the organization
--
-- The social links (website, LinkedIn, Instagram, etc.) are the ORG's
-- public profile. Identical shape to profiles and portfolio_items —
-- just moves the anchor from user_id to organization_id.

BEGIN;

ALTER TABLE social_links
    ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;

UPDATE social_links sl
SET    organization_id = u.organization_id
FROM   users u
WHERE  sl.user_id = u.id;

DO $$
DECLARE
    orphans integer;
BEGIN
    SELECT COUNT(*) INTO orphans
    FROM   social_links
    WHERE  organization_id IS NULL;
    IF orphans > 0 THEN
        RAISE EXCEPTION 'migration 069 left % social_links without an org', orphans;
    END IF;
END $$;

ALTER TABLE social_links ALTER COLUMN organization_id SET NOT NULL;

DROP INDEX IF EXISTS idx_social_links_user;
ALTER TABLE social_links DROP CONSTRAINT social_links_user_id_fkey;
ALTER TABLE social_links DROP CONSTRAINT social_links_user_id_platform_key;
ALTER TABLE social_links DROP COLUMN user_id;

CREATE UNIQUE INDEX social_links_org_platform_key ON social_links (organization_id, platform);
CREATE INDEX        idx_social_links_org           ON social_links (organization_id);

COMMIT;

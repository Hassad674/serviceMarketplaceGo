-- Reverse 109_add_persona_to_social_links: drop the composite index,
-- restore the original (organization_id, platform) uniqueness, and
-- drop the persona column. Any row whose persona is not 'freelance'
-- is deleted first so the legacy uniqueness constraint can be
-- re-established without conflict.

BEGIN;

DELETE FROM social_links WHERE persona <> 'freelance';

DROP INDEX IF EXISTS social_links_org_persona_platform_key;
DROP INDEX IF EXISTS idx_social_links_org_persona;

CREATE UNIQUE INDEX IF NOT EXISTS social_links_org_platform_key
    ON social_links (organization_id, platform);

ALTER TABLE social_links DROP COLUMN IF EXISTS persona;

COMMIT;

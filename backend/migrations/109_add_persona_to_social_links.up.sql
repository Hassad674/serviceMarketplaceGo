-- Add a persona dimension to social_links so a single organization can
-- hold multiple independent sets — one for agencies, one for the
-- freelance persona of a provider_personal user, and one for the
-- referrer persona of the same user. The composite uniqueness
-- constraint (organization_id, persona, platform) keeps each set
-- self-contained.

BEGIN;

ALTER TABLE social_links
    ADD COLUMN persona TEXT NOT NULL DEFAULT 'freelance'
        CHECK (persona IN ('freelance', 'referrer', 'agency'));

-- Backfill: agency organizations kept their existing links. For every
-- other org type the rows represent the legacy provider_personal
-- social links, which belong to the freelance persona by default
-- (already set by the column default above).
UPDATE social_links sl
   SET persona = 'agency'
  FROM organizations o
 WHERE sl.organization_id = o.id
   AND o.type = 'agency';

-- Swap the (organization_id, platform) unique constraint for the
-- persona-aware composite. Both indexes coexist briefly during the
-- rename to avoid a window where uniqueness is not enforced.
DROP INDEX IF EXISTS social_links_org_platform_key;

CREATE UNIQUE INDEX social_links_org_persona_platform_key
    ON social_links (organization_id, persona, platform);

CREATE INDEX IF NOT EXISTS idx_social_links_org_persona
    ON social_links (organization_id, persona);

COMMIT;

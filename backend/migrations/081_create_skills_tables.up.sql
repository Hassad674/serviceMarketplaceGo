-- 081_create_skills_tables.up.sql
--
-- Skills feature foundation. Two tables:
--
--   1. skills_catalog  — hybrid dictionary of every skill the marketplace
--      knows about. Admin-seeded "curated" entries power the browse-by-
--      expertise panels; user-created entries power the long tail via
--      autocomplete. Both kinds live in the same table, distinguished by
--      is_curated.
--
--   2. profile_skills  — M2M between organizations and skills, with a
--      position column to preserve display order on the public profile.
--      Mirrors the shape of organization_expertise_domains (migration
--      080) so the adapter and service layers stay consistent.
--
-- Role-based limits (40 for agency, 25 for provider_personal, 0 for
-- enterprise) are enforced in the Go application layer: CHECK constraints
-- may not reference organizations.type, and the limit is a product
-- decision that should live in the domain layer anyway. Expertise-key
-- validation for skills_catalog.expertise_keys is likewise done in Go
-- (expertise.IsValidKey) to avoid coupling this feature's schema to the
-- closed expertise catalog.

BEGIN;

-- Required for trigram fuzzy search in skill autocomplete.
-- IF NOT EXISTS is safe — other features may already have enabled it.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Hybrid catalog of skills: curated (seeded by admin) + user-created.
-- Panels (browse by expertise) show is_curated = true only.
-- Autocomplete search shows all, curated ranked first.
CREATE TABLE skills_catalog (
    skill_text      TEXT PRIMARY KEY,                -- normalized: lowercase + trimmed + collapsed spaces
    display_text    TEXT NOT NULL,                    -- user-visible casing ("React", "Next.js", "Figma")
    expertise_keys  TEXT[] NOT NULL DEFAULT '{}',     -- 1-3 expertise domain keys, validated in Go via expertise.IsValidKey
    is_curated      BOOLEAN NOT NULL DEFAULT false,   -- true = seeded, false = user-created
    usage_count     INT NOT NULL DEFAULT 0,           -- cache: number of profiles that selected this skill
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Partial GIN index for the common panel query "curated skills for expertise X".
-- Narrower than the full index below and kept hot for the curated browse path.
CREATE INDEX idx_skills_catalog_curated_expertise
    ON skills_catalog USING GIN (expertise_keys)
    WHERE is_curated = true;

-- Full GIN index for the broader search (both curated and user-created).
CREATE INDEX idx_skills_catalog_expertise
    ON skills_catalog USING GIN (expertise_keys);

-- For "popular skills" queries and autocomplete sorting.
CREATE INDEX idx_skills_catalog_usage
    ON skills_catalog (usage_count DESC);

-- Trigram index for fuzzy autocomplete (user types "re" -> matches "react", "redis"...).
CREATE INDEX idx_skills_catalog_text_trgm
    ON skills_catalog USING GIN (skill_text gin_trgm_ops);

-- Reuse the existing updated_at trigger function (defined in migration 001).
CREATE TRIGGER skills_catalog_updated_at
    BEFORE UPDATE ON skills_catalog
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- M2M between organizations and skills. Mirrors the pattern of
-- organization_expertise_domains (migration 080): (org_id, skill_text, position).
-- Role-based limits (40 agency, 25 provider_personal, 0 enterprise) are
-- enforced in the Go application layer — CHECK constraints cannot
-- reference organizations.type.
CREATE TABLE profile_skills (
    organization_id UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    skill_text      TEXT        NOT NULL REFERENCES skills_catalog(skill_text),
    position        INT         NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, skill_text)
);

CREATE INDEX idx_profile_skills_position
    ON profile_skills (organization_id, position);

CREATE INDEX idx_profile_skills_skill
    ON profile_skills (skill_text);

COMMENT ON TABLE skills_catalog IS
    'Hybrid catalog: curated admin-seeded skills + user-created skills. Panels display is_curated = true only; autocomplete searches all.';
COMMENT ON TABLE profile_skills IS
    'Skills declared by an organization, with display order (position). Max 40 for agency, 25 for provider_personal, forbidden for enterprise — enforced in the Go domain layer.';

COMMIT;

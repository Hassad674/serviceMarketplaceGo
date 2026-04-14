-- 101_migrate_provider_personal_to_freelance_profiles.up.sql
--
-- Copies the freelance half of every provider_personal profile row
-- into freelance_profiles. Idempotent via ON CONFLICT DO NOTHING — the
-- UNIQUE(organization_id) constraint catches any re-run.
--
-- Only provider_personal orgs are migrated. Agency orgs keep using the
-- legacy profiles table (the agency refactor is a follow-up).
--
-- Expertise domains live in their own table (organization_expertise_
-- domains) in the pre-split schema; we aggregate them per org via
-- array_agg so the split freelance_profiles row carries a snapshot
-- array. The organization_expertise_domains table is NOT dropped in
-- this migration — the old code paths keep working until the split
-- is wired everywhere.

BEGIN;

INSERT INTO freelance_profiles (
    organization_id,
    title,
    about,
    video_url,
    availability_status,
    expertise_domains,
    created_at,
    updated_at
)
SELECT
    p.organization_id,
    COALESCE(p.title, ''),
    COALESCE(p.about, ''),
    COALESCE(p.presentation_video_url, ''),
    COALESCE(p.availability_status, 'available_now'),
    COALESCE(
        (
            SELECT array_agg(oed.domain_key ORDER BY oed.position, oed.domain_key)
            FROM organization_expertise_domains oed
            WHERE oed.organization_id = p.organization_id
        ),
        '{}'::text[]
    ),
    COALESCE(p.created_at, now()),
    COALESCE(p.updated_at, now())
FROM profiles p
JOIN organizations o ON o.id = p.organization_id
WHERE o.type = 'provider_personal'
ON CONFLICT (organization_id) DO NOTHING;

COMMIT;

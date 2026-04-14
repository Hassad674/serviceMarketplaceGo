-- 102_migrate_provider_personal_to_referrer_profiles.up.sql
--
-- Copies the referrer half of every provider_personal profile row
-- into referrer_profiles, but only for orgs whose owner user has
-- referrer_enabled=true. A provider with the toggle off never had a
-- meaningful referrer persona on the legacy profiles row and does not
-- get one here — the referrer service will lazily create an empty
-- profile row the day the user opts in.
--
-- Title is reused from the freelance side because the legacy profiles
-- row only has one title column — the referrer page on the frontend
-- pre-split already surfaced the same title. Service-level edits of
-- the referrer title will diverge from the freelance title naturally
-- after the split.
--
-- Expertise domains are aggregated from organization_expertise_domains
-- (same source as the freelance side) so both personas start with the
-- same declared domain list and drift over time as each is edited.

BEGIN;

INSERT INTO referrer_profiles (
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
    COALESCE(p.referrer_about, ''),
    COALESCE(p.referrer_video_url, ''),
    COALESCE(p.referrer_availability_status, 'available_now'),
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
JOIN users u        ON u.id = o.owner_user_id
WHERE o.type = 'provider_personal'
  AND u.referrer_enabled = TRUE
ON CONFLICT (organization_id) DO NOTHING;

COMMIT;

-- 104_drop_provider_personal_rows_from_legacy_profiles.down.sql
--
-- WARNING — lossy reversal. The up migration deleted the legacy
-- profiles and profile_pricing rows for every provider_personal org;
-- the down rebuilds them from the split tables as best it can, but
-- the legacy shape has ONE row per profile+pricing with a single
-- title column covering both personas, whereas the split shape has
-- two independent profiles. The reconstruction favors the freelance
-- side (it is the "default" persona) and injects the referrer_* data
-- from the corresponding referrer_profiles row when one exists.
--
-- Expertise domains are NOT re-written because they live in their
-- own table (organization_expertise_domains) and never appeared as a
-- column on profiles — the original up never deleted anything from
-- the expertise table so there is nothing to restore.
--
-- This is acceptable for development rollbacks and testing. Do NOT
-- rely on this down migration in production — it is here to satisfy
-- the up/down symmetry convention, not to guarantee byte-level
-- round-trips.

BEGIN;

INSERT INTO profiles (
    organization_id,
    title,
    about,
    photo_url,
    presentation_video_url,
    referrer_about,
    referrer_video_url,
    city,
    country_code,
    latitude,
    longitude,
    work_mode,
    travel_radius_km,
    languages_professional,
    languages_conversational,
    availability_status,
    referrer_availability_status,
    created_at,
    updated_at
)
SELECT
    o.id                         AS organization_id,
    fp.title                     AS title,
    fp.about                     AS about,
    o.photo_url                  AS photo_url,
    fp.video_url                 AS presentation_video_url,
    COALESCE(rp.about, '')       AS referrer_about,
    COALESCE(rp.video_url, '')   AS referrer_video_url,
    o.city                       AS city,
    o.country_code               AS country_code,
    o.latitude                   AS latitude,
    o.longitude                  AS longitude,
    o.work_mode                  AS work_mode,
    o.travel_radius_km           AS travel_radius_km,
    o.languages_professional     AS languages_professional,
    o.languages_conversational   AS languages_conversational,
    fp.availability_status       AS availability_status,
    rp.availability_status       AS referrer_availability_status,
    fp.created_at                AS created_at,
    fp.updated_at                AS updated_at
FROM organizations o
JOIN freelance_profiles fp     ON fp.organization_id = o.id
LEFT JOIN referrer_profiles rp ON rp.organization_id = o.id
WHERE o.type = 'provider_personal'
ON CONFLICT (organization_id) DO NOTHING;

-- Reconstruct freelance pricing under direct kind.
INSERT INTO profile_pricing (
    organization_id,
    pricing_kind,
    pricing_type,
    min_amount,
    max_amount,
    currency,
    pricing_note,
    negotiable,
    created_at,
    updated_at
)
SELECT
    fp.organization_id,
    'direct',
    fpr.pricing_type,
    fpr.min_amount,
    fpr.max_amount,
    fpr.currency,
    fpr.pricing_note,
    fpr.negotiable,
    fpr.created_at,
    fpr.updated_at
FROM freelance_pricing fpr
JOIN freelance_profiles fp ON fp.id = fpr.profile_id
JOIN organizations o      ON o.id = fp.organization_id
WHERE o.type = 'provider_personal'
ON CONFLICT (organization_id, pricing_kind) DO NOTHING;

-- Reconstruct referrer pricing under referral kind.
INSERT INTO profile_pricing (
    organization_id,
    pricing_kind,
    pricing_type,
    min_amount,
    max_amount,
    currency,
    pricing_note,
    negotiable,
    created_at,
    updated_at
)
SELECT
    rp.organization_id,
    'referral',
    rpr.pricing_type,
    rpr.min_amount,
    rpr.max_amount,
    rpr.currency,
    rpr.pricing_note,
    rpr.negotiable,
    rpr.created_at,
    rpr.updated_at
FROM referrer_pricing rpr
JOIN referrer_profiles rp ON rp.id = rpr.profile_id
JOIN organizations o     ON o.id = rp.organization_id
WHERE o.type = 'provider_personal'
ON CONFLICT (organization_id, pricing_kind) DO NOTHING;

COMMIT;

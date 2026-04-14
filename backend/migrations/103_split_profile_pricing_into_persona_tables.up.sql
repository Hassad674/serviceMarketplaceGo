-- 103_split_profile_pricing_into_persona_tables.up.sql
--
-- Copies provider_personal profile_pricing rows into the persona-
-- specific tables (freelance_pricing / referrer_pricing). The legacy
-- profile_pricing table still owns the agency rows — those are left
-- untouched. Run after 101 and 102 so the persona profile rows
-- targeted by profile_id already exist.
--
-- Mapping:
--   * profile_pricing.kind='direct'   → freelance_pricing
--   * profile_pricing.kind='referral' → referrer_pricing
--
-- profile_id is looked up on the fly via
-- (SELECT id FROM freelance_profiles WHERE organization_id = pp.org_id)
-- because the new tables use a surrogate UUID PK whereas the legacy
-- row was keyed on (org_id, kind).

BEGIN;

-- Freelance (direct) pricing — provider_personal only.
INSERT INTO freelance_pricing (
    profile_id,
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
    fp.id,
    pp.pricing_type,
    pp.min_amount,
    pp.max_amount,
    pp.currency,
    COALESCE(pp.pricing_note, ''),
    COALESCE(pp.negotiable, FALSE),
    COALESCE(pp.created_at, now()),
    COALESCE(pp.updated_at, now())
FROM profile_pricing pp
JOIN organizations o      ON o.id = pp.organization_id
JOIN freelance_profiles fp ON fp.organization_id = pp.organization_id
WHERE o.type = 'provider_personal'
  AND pp.pricing_kind = 'direct'
ON CONFLICT (profile_id) DO NOTHING;

-- Referrer pricing — provider_personal + referrer_enabled only.
INSERT INTO referrer_pricing (
    profile_id,
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
    rp.id,
    pp.pricing_type,
    pp.min_amount,
    pp.max_amount,
    pp.currency,
    COALESCE(pp.pricing_note, ''),
    COALESCE(pp.negotiable, FALSE),
    COALESCE(pp.created_at, now()),
    COALESCE(pp.updated_at, now())
FROM profile_pricing pp
JOIN organizations o      ON o.id = pp.organization_id
JOIN referrer_profiles rp ON rp.organization_id = pp.organization_id
WHERE o.type = 'provider_personal'
  AND pp.pricing_kind = 'referral'
ON CONFLICT (profile_id) DO NOTHING;

COMMIT;

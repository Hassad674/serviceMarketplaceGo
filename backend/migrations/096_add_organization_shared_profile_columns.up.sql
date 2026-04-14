-- 096_add_organization_shared_profile_columns.up.sql
--
-- Splits the profile aggregate for provider_personal orgs: the shared
-- fields (photo, location, languages) move onto the organizations row
-- directly. They were previously duplicated across the freelance and
-- referrer personas of the same org via a single profiles row; hosting
-- them on the org itself removes the duplication and makes the two
-- personas (freelance, referrer) proper siblings that JOIN against the
-- same single source of truth.
--
-- Columns added:
--
--   * photo_url                — mirrors profiles.photo_url
--   * city, country_code       — mirrors profiles.city / country_code
--   * latitude, longitude      — mirrors profiles.latitude / longitude
--   * work_mode                — TEXT[] subset of {remote, on_site, hybrid}
--   * travel_radius_km         — nullable integer
--   * languages_professional   — TEXT[] ISO 639-1 codes
--   * languages_conversational — TEXT[] ISO 639-1 codes
--
-- Every column has a safe default ('' or '{}') so existing rows stay
-- valid the moment the ALTER completes. The values are then backfilled
-- from profiles via an UPDATE ... FROM join so the split migrations
-- (100, 101) that feed the new freelance_profiles / referrer_profiles
-- tables can rely on the data already being in place.
--
-- All enum validity and normalization is enforced in the Go domain
-- layer — the DB only sets a coarse shape (NOT NULL, TEXT[] arrays).

BEGIN;

ALTER TABLE organizations ADD COLUMN photo_url                TEXT   NOT NULL DEFAULT '';
ALTER TABLE organizations ADD COLUMN city                     TEXT   NOT NULL DEFAULT '';
ALTER TABLE organizations ADD COLUMN country_code             TEXT   NOT NULL DEFAULT ''; -- ISO 3166-1 alpha-2, uppercase
ALTER TABLE organizations ADD COLUMN latitude                 DOUBLE PRECISION;           -- nullable: geocoding best-effort
ALTER TABLE organizations ADD COLUMN longitude                DOUBLE PRECISION;
ALTER TABLE organizations ADD COLUMN work_mode                TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE organizations ADD COLUMN travel_radius_km         INTEGER;
ALTER TABLE organizations ADD COLUMN languages_professional   TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE organizations ADD COLUMN languages_conversational TEXT[] NOT NULL DEFAULT '{}';

-- Backfill the new shared columns from the existing profiles row of
-- every org. COALESCE the arrays so a NULL array column (which should
-- not happen but historically does on a few legacy rows) becomes an
-- empty array rather than a NOT NULL violation.
UPDATE organizations o
SET photo_url                = COALESCE(p.photo_url, ''),
    city                     = COALESCE(p.city, ''),
    country_code             = COALESCE(p.country_code, ''),
    latitude                 = p.latitude,
    longitude                = p.longitude,
    work_mode                = COALESCE(p.work_mode, '{}'),
    travel_radius_km         = p.travel_radius_km,
    languages_professional   = COALESCE(p.languages_professional, '{}'),
    languages_conversational = COALESCE(p.languages_conversational, '{}')
FROM profiles p
WHERE p.organization_id = o.id;

-- Indexes support the same filter/facet queries the profiles-side
-- indexes served prior to the split.
CREATE INDEX IF NOT EXISTS idx_organizations_country_city_ne_empty
    ON organizations (country_code, city)
    WHERE country_code <> '';

CREATE INDEX IF NOT EXISTS idx_organizations_work_mode_gin ON organizations USING GIN (work_mode);
CREATE INDEX IF NOT EXISTS idx_organizations_lang_pro_gin  ON organizations USING GIN (languages_professional);
CREATE INDEX IF NOT EXISTS idx_organizations_lang_conv_gin ON organizations USING GIN (languages_conversational);

COMMIT;

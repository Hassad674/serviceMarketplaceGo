-- 083_profile_tier1_completion.up.sql
--
-- Profile Tier 1 completion: adds the location, languages, and
-- availability blocks to the profiles table, and creates a dedicated
-- profile_pricing table for the richer pricing model.
--
-- Design notes:
--
--   * Scalar fields (city, country_code, lat/lng, work_mode arrays,
--     languages, availability) live ON the profiles row because they
--     share the profile's edit cadence, have cardinality 1, and are
--     always needed together on every profile read. Putting them
--     elsewhere would force a join on every GetProfile.
--
--   * Pricing lives in its OWN table (profile_pricing) because:
--       - cardinality is up to 2 rows per org (direct + referral kinds),
--       - the shape is richer than a scalar (7 fields + timestamps),
--       - edit cadence is independent of the main profile,
--       - bloating profiles with 14+ pricing columns would churn the
--         whole row on every pricing edit.
--
--   * All enum validity (work_mode values, availability statuses, pricing
--     kind → type compatibility, kind → org-role compatibility, min/max
--     amount relationships) is enforced in the Go domain layer. The DB
--     only enforces the coarsest CHECK constraints (value ∈ fixed set)
--     because PostgreSQL cannot cross-reference the organizations.type
--     column from a profile-level CHECK without procedural code we do
--     not want to maintain.
--
--   * Geocoding is synchronous and best-effort (see
--     adapter/nominatim/geocoder.go). NULL latitude / longitude mean
--     "coordinates unavailable, render the UI without map features."

BEGIN;

-- ============================================================
-- Location block
-- ============================================================
ALTER TABLE profiles ADD COLUMN city             TEXT NOT NULL DEFAULT '';
ALTER TABLE profiles ADD COLUMN country_code     TEXT NOT NULL DEFAULT ''; -- ISO 3166-1 alpha-2, uppercase
ALTER TABLE profiles ADD COLUMN latitude         DOUBLE PRECISION;          -- nullable: geocoding best-effort
ALTER TABLE profiles ADD COLUMN longitude        DOUBLE PRECISION;
ALTER TABLE profiles ADD COLUMN work_mode        TEXT[] NOT NULL DEFAULT '{}'; -- subset of {'remote','on_site','hybrid'}
ALTER TABLE profiles ADD COLUMN travel_radius_km INTEGER;                   -- nullable: only meaningful for on_site/hybrid

-- ============================================================
-- Languages block (two arrays, two proficiency levels)
-- ============================================================
ALTER TABLE profiles ADD COLUMN languages_professional   TEXT[] NOT NULL DEFAULT '{}'; -- ISO 639-1 codes (lowercase)
ALTER TABLE profiles ADD COLUMN languages_conversational TEXT[] NOT NULL DEFAULT '{}';

-- ============================================================
-- Availability block
-- ============================================================
ALTER TABLE profiles ADD COLUMN availability_status TEXT NOT NULL DEFAULT 'available_now'
    CHECK (availability_status IN ('available_now', 'available_soon', 'not_available'));

-- Referrer-side availability, nullable because it only applies to
-- providers with referrer_enabled = true. NULL means "not in referrer
-- mode" and the UI hides the referrer availability section.
ALTER TABLE profiles ADD COLUMN referrer_availability_status TEXT
    CHECK (referrer_availability_status IS NULL
        OR referrer_availability_status IN ('available_now', 'available_soon', 'not_available'));

-- ============================================================
-- Indexes on profiles (support filter + facet queries)
-- ============================================================
CREATE INDEX idx_profiles_country_city_ne_empty
    ON profiles (country_code, city)
    WHERE country_code <> '';

CREATE INDEX idx_profiles_work_mode_gin        ON profiles USING GIN (work_mode);
CREATE INDEX idx_profiles_lang_pro_gin         ON profiles USING GIN (languages_professional);
CREATE INDEX idx_profiles_lang_conv_gin        ON profiles USING GIN (languages_conversational);
CREATE INDEX idx_profiles_availability         ON profiles (availability_status);

-- ============================================================
-- profile_pricing table
-- ============================================================
CREATE TABLE profile_pricing (
    organization_id UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    pricing_kind    TEXT        NOT NULL CHECK (pricing_kind IN ('direct', 'referral')),
    pricing_type    TEXT        NOT NULL CHECK (pricing_type IN (
        'daily',
        'hourly',
        'project_from',
        'project_range',
        'commission_pct',
        'commission_flat'
    )),
    min_amount      BIGINT      NOT NULL, -- centimes for currency types, basis points for commission_pct
    max_amount      BIGINT,               -- NULL when no range (TJM fixe, hourly, commission flat)
    currency        TEXT        NOT NULL DEFAULT 'EUR', -- ISO 4217 OR 'pct' for commission percentages
    pricing_note    TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, pricing_kind)
);

CREATE INDEX idx_profile_pricing_type ON profile_pricing (pricing_type);

CREATE TRIGGER profile_pricing_updated_at
    BEFORE UPDATE ON profile_pricing
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

COMMENT ON TABLE profile_pricing IS
    'Pricing rows for organizations. Max 2 rows per org: one with kind=direct (freelance/agency pricing: daily/hourly/project_from/project_range) and one with kind=referral (apporteur pricing: commission_pct/commission_flat). Kind to pricing_type validity and kind to org-role validity are enforced in the Go domain layer.';
COMMENT ON COLUMN profile_pricing.min_amount IS
    'Minimum amount in minor units: centimes for currency pricings (daily/hourly/project_*/commission_flat), basis points for commission_pct (10000 = 100.00%).';
COMMENT ON COLUMN profile_pricing.currency IS
    'ISO 4217 currency code (EUR, USD, ...) for currency pricings, or the literal ''pct'' for commission_pct rows (dimensionless).';

COMMIT;

-- 098_create_referrer_profiles.up.sql
--
-- Creates the referrer_profiles table — the apporteur d'affaires
-- ("business referrer") persona-specific half of the split profile
-- aggregate for provider_personal orgs that have the referrer toggle
-- on. Shares every shared field with the freelance profile via the
-- organizations row — only the persona-specific fields (title, about,
-- video, availability, expertise domains) live here.
--
-- Cardinality: at most one row per organization (UNIQUE constraint).
-- Unlike freelance profiles, referrer profiles are NOT auto-created
-- for every provider_personal org — they are lazily created by the
-- GetByOrgID call from the referrer service, only for users who have
-- toggled referrer_enabled=true.

BEGIN;

CREATE TABLE referrer_profiles (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id     UUID        NOT NULL UNIQUE REFERENCES organizations(id) ON DELETE CASCADE,
    title               TEXT        NOT NULL DEFAULT '',
    about               TEXT        NOT NULL DEFAULT '',
    video_url           TEXT        NOT NULL DEFAULT '',
    availability_status TEXT        NOT NULL DEFAULT 'available_now'
        CHECK (availability_status IN ('available_now', 'available_soon', 'not_available')),
    expertise_domains   TEXT[]      NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_referrer_profiles_availability ON referrer_profiles (availability_status);
CREATE INDEX idx_referrer_profiles_expertise_domains_gin
    ON referrer_profiles USING GIN (expertise_domains);

CREATE TRIGGER referrer_profiles_updated_at
    BEFORE UPDATE ON referrer_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

COMMENT ON TABLE referrer_profiles IS
    'Referrer (apporteur d''affaires) persona of a provider_personal organization with referrer_enabled=true. Title, about, video, availability, expertise domains. Shared fields (photo, languages, location) live on organizations and are JOINed at read time. Lazily created on first GET.';

COMMIT;

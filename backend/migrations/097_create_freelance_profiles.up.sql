-- 097_create_freelance_profiles.up.sql
--
-- Creates the freelance_profiles table — the persona-specific half of
-- the split profile aggregate for provider_personal orgs. Fields here
-- are unique to the freelance offering: title, about text, presentation
-- video, availability status, expertise domains. Shared fields (photo,
-- location, languages) live on organizations since migration 096 and
-- are JOINed by the handler response DTO.
--
-- Cardinality: exactly one row per organization (UNIQUE constraint).
-- FK references organizations(id) with ON DELETE CASCADE so deleting
-- an org also deletes its freelance profile row. The agency path still
-- uses the legacy profiles table untouched.

BEGIN;

CREATE TABLE freelance_profiles (
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

CREATE INDEX idx_freelance_profiles_availability ON freelance_profiles (availability_status);
CREATE INDEX idx_freelance_profiles_expertise_domains_gin
    ON freelance_profiles USING GIN (expertise_domains);

CREATE TRIGGER freelance_profiles_updated_at
    BEFORE UPDATE ON freelance_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

COMMENT ON TABLE freelance_profiles IS
    'Freelance persona of a provider_personal organization. Title, about, video, availability, expertise domains. Shared fields (photo, languages, location) live on organizations and are JOINed at read time.';

COMMIT;

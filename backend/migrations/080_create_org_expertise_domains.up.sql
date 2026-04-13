-- 080_create_org_expertise_domains.up.sql
--
-- Organization expertise domains. Each row records one domain
-- specialization attached to an organization, preserving display
-- order via the `position` column (0-indexed, contiguous from 0).
--
-- Domain keys are an application-level catalog of 15 frozen values
-- (see backend/internal/domain/expertise/catalog.go). The DB stores
-- them as TEXT — validation happens in the domain layer, not via a
-- CHECK constraint, so adding new keys to the catalog is a code-only
-- change and never requires a DDL round-trip.
--
-- Invariants enforced at the schema level:
--   * (organization_id, domain_key) is the primary key, which
--     prevents the same domain from being attached twice to the
--     same organization without the application layer checking.
--   * position is NOT NULL so the display order is always known.
--   * ON DELETE CASCADE to organizations ties the lifetime of
--     these rows to the parent org: dropping an organization also
--     drops its expertise rows, preventing orphans.
--
-- Per-org-type maximums (agency=8, provider_personal=5, enterprise
-- forbidden) are enforced in the application layer — not at the DB
-- level — because the maximum depends on a column of another table
-- (organizations.type) and CHECK constraints may not reference
-- other tables. The app service looks up the org type before every
-- Replace and rejects oversize payloads with a typed domain error.
--
-- Indexes:
--   * idx_org_expertise_position speeds up the per-org ordered read
--     (SELECT ... WHERE organization_id = $1 ORDER BY position).
--   * idx_org_expertise_domain supports future reverse lookups such
--     as "which orgs declared domain_key = 'design_ui_ux'?" for
--     discovery / filtering.

CREATE TABLE IF NOT EXISTS organization_expertise_domains (
    organization_id UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    domain_key      TEXT        NOT NULL,
    position        INT         NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (organization_id, domain_key)
);

CREATE INDEX IF NOT EXISTS idx_org_expertise_position
    ON organization_expertise_domains (organization_id, position);

CREATE INDEX IF NOT EXISTS idx_org_expertise_domain
    ON organization_expertise_domains (domain_key);

COMMENT ON TABLE organization_expertise_domains IS
    'Ordered domain specializations declared by a provider organization (max 8 for agency, 5 for provider_personal, forbidden for enterprise). Keys validated by the domain/expertise catalog in Go.';

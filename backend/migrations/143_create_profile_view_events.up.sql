-- 143_create_profile_view_events.up.sql
--
-- Records every public profile detail page view so the per-org stats
-- dashboard (visibility, time series, top keywords) has a primary
-- source of truth.
--
-- Privacy posture (RGPD art. 5-1-c — data minimization):
--   * viewer_user_id is nullable: anonymous visitors are tracked
--     without identity. The FK uses ON DELETE SET NULL so a user
--     erasure does not break the event row's analytical value.
--   * viewer_ip_anonymized stores the truncated IP as an INET value
--     (IPv4 /24, IPv6 /64) — the raw IP NEVER lands in this table.
--     The truncation happens in the Go domain layer before INSERT.
--   * viewer_ua_hash stores a SHA-256 (hex) of the User-Agent so we
--     can dedupe unique-visitor counts without persisting the raw UA.
--   * search_query / search_position are populated only when
--     came_from='search' — tracks which keyword led the visitor to
--     the profile and at which result rank, for the keyword stats
--     panel.
--
-- Feature-scoped: this table references organizations(id) and users(id)
-- but no other feature table. Dropping it is safe and breaks nothing
-- else in the marketplace.

BEGIN;

CREATE TABLE IF NOT EXISTS profile_view_events (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    persona               TEXT NOT NULL CHECK (persona IN ('freelance', 'agency', 'referrer')),
    viewer_user_id        UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    viewer_ip_anonymized  INET NOT NULL,
    viewer_ua_hash        TEXT NOT NULL,
    came_from             TEXT NOT NULL CHECK (came_from IN ('search', 'list', 'direct', 'referral', 'unknown')),
    search_query          TEXT NULL,
    search_position       INTEGER NULL,
    referrer_url          TEXT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMIT;

-- Indexes are created OUTSIDE the transaction with CONCURRENTLY where
-- possible. golang-migrate runs each .up.sql in a single transaction
-- by default, so we use plain CREATE INDEX IF NOT EXISTS here — the
-- table is brand-new so there is no concurrent-write contention.

-- Time-bucket scan for "views in the last N days" stats.
CREATE INDEX IF NOT EXISTS idx_pve_org_created
    ON profile_view_events (organization_id, created_at DESC);

-- Unique-visitor dedup index (org + ip + ua + day window). Used by
-- the keyword join + the unique-count aggregation.
CREATE INDEX IF NOT EXISTS idx_pve_unique_visitor
    ON profile_view_events (organization_id, viewer_ip_anonymized, viewer_ua_hash, created_at);

-- 136_two_pool_rls_roles.up.sql
--
-- RLS hardening — two-pool model.
--
-- Up to this point the application connects to Postgres through a single
-- role. In production that role is `neondb_owner` (Neon) which is a
-- table owner and has BYPASSRLS implied by ownership; locally it is
-- `postgres` which is a superuser. In both cases RLS policies installed
-- by migration 125 fire only inside the integration tests (which SET
-- ROLE marketplace_rls_test explicitly) — every real request bypasses
-- them.
--
-- This migration installs the two roles required for the two-pool
-- routing wired in `internal/adapter/postgres/routed_db.go`:
--
--   * marketplace_app       — NOSUPERUSER NOBYPASSRLS. The pool every
--                              user-facing repository call uses. RLS
--                              policies fire normally.
--   * marketplace_scheduler — NOSUPERUSER BYPASSRLS. The pool every
--                              system-actor path uses (schedulers,
--                              webhooks, GDPR purge, search indexer,
--                              admin overrides).
--
-- Both roles share identical SCHEMA + GRANTS. The only difference is
-- BYPASSRLS. Picking the pool happens at the Go layer, keyed on
-- `system.IsSystemActor(ctx)` — see backend/docs/rls.md for the
-- rationale.
--
-- This migration is IDEMPOTENT — re-running on a partial state is safe.
-- The role passwords are NULL: in dev/CI both roles are reachable only
-- through the migration owner connection; in production the operator
-- sets a password via the rollout script (backend/docs/rls-rollout.md).

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'marketplace_app') THEN
        CREATE ROLE marketplace_app NOLOGIN NOSUPERUSER NOBYPASSRLS;
    ELSE
        -- Force the security-relevant attributes in case a prior
        -- migration / manual operation toggled them by mistake.
        ALTER ROLE marketplace_app NOSUPERUSER NOBYPASSRLS;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'marketplace_scheduler') THEN
        CREATE ROLE marketplace_scheduler NOLOGIN NOSUPERUSER BYPASSRLS;
    ELSE
        ALTER ROLE marketplace_scheduler NOSUPERUSER BYPASSRLS;
    END IF;
END $$;

-- Schema usage so the roles can resolve table names. GRANTing on
-- `public` is the typical setup for the marketplace schema; a future
-- migration that introduces a dedicated schema must repeat these
-- grants for both roles.
GRANT USAGE ON SCHEMA public TO marketplace_app;
GRANT USAGE ON SCHEMA public TO marketplace_scheduler;

-- Read/write privileges on every existing table. Done in a single
-- statement so the migration stays under one tx and keeps the role
-- attributes in sync with the grants.
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public
    TO marketplace_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public
    TO marketplace_scheduler;

-- Sequence usage — required for `gen_random_uuid()` and any future
-- nextval() call from inside the application path.
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO marketplace_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO marketplace_scheduler;

-- Default privileges so future tables created by the migration owner
-- inherit the same grants automatically. Without this every new
-- migration would need to manually GRANT to both roles.
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO marketplace_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO marketplace_scheduler;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO marketplace_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO marketplace_scheduler;

-- audit_logs is append-only (mig 124). Narrow the marketplace_app
-- privileges accordingly; the scheduler retains UPDATE/DELETE because
-- the GDPR purge legitimately rewrites historical entries via the
-- system-actor path.
REVOKE UPDATE, DELETE ON audit_logs FROM marketplace_app;

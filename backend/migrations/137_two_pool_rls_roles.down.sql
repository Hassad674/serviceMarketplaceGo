-- 136_two_pool_rls_roles.down.sql
--
-- Reverse migration 136. Drops the two roles + the default privileges
-- that grant them future-table access. Idempotent — re-running on a
-- partial state is safe.
--
-- WARNING: rolling this back invalidates any active connection that
-- authenticated as marketplace_app / marketplace_scheduler. Do NOT run
-- on a production database without first rotating every API instance
-- back to the legacy migration owner role.

-- Default-privilege grants must be revoked before the role can be
-- dropped — Postgres refuses to drop a role that still owns a default
-- ACL entry.
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM marketplace_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM marketplace_scheduler;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    REVOKE USAGE, SELECT ON SEQUENCES FROM marketplace_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    REVOKE USAGE, SELECT ON SEQUENCES FROM marketplace_scheduler;

-- Revoke explicit privileges on existing objects.
REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM marketplace_app;
REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM marketplace_scheduler;
REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM marketplace_app;
REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM marketplace_scheduler;
REVOKE USAGE ON SCHEMA public FROM marketplace_app;
REVOKE USAGE ON SCHEMA public FROM marketplace_scheduler;

DROP ROLE IF EXISTS marketplace_app;
DROP ROLE IF EXISTS marketplace_scheduler;

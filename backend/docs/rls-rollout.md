# RLS hardening — production rollout playbook

This document is the operator-side runbook for migrating production from
the single-pool model (the API connects as `neondb_owner`, which has
table ownership and therefore implicit BYPASSRLS) to the two-pool model
introduced by migration `137_two_pool_rls_roles`.

The Go-level wiring is already in place — `internal/adapter/postgres/
routed_db.go` exposes a context-aware `RoutedDB` that picks
`marketplace_app` or `marketplace_scheduler` based on
`system.IsSystemActor(ctx)`. This document covers the Postgres-side
work that has to be done by `neondb_owner` (or any role that owns the
existing tables) — those steps cannot be embedded in
`make migrate-up` because they require setting passwords and rotating
credentials on the Railway / Neon side.

## Pre-checks

Run as `neondb_owner` against the production cluster:

```sql
-- 1. Verify the two roles created by migration 137 exist with the
--    expected attributes.
SELECT rolname, rolsuper, rolbypassrls, rolcanlogin
FROM pg_roles
WHERE rolname IN ('marketplace_app', 'marketplace_scheduler')
ORDER BY rolname;
-- Expected:
--   marketplace_app        | f | f | f
--   marketplace_scheduler  | f | t | f

-- 2. Verify the default privileges actually grant new tables to both
--    roles.
SELECT defaclrole::regrole, defaclnamespace::regnamespace,
       defaclobjtype, defaclacl
FROM pg_default_acl
WHERE defaclnamespace = 'public'::regnamespace;

-- 3. Verify audit_logs has UPDATE/DELETE revoked from marketplace_app.
SELECT grantee, privilege_type
FROM information_schema.role_table_grants
WHERE table_name = 'audit_logs'
  AND grantee = 'marketplace_app';
-- Expected: only INSERT and SELECT.
```

## Step 1 — Set passwords

The migration creates the roles with `NOLOGIN` so the cluster does not
expose a half-configured login surface. Operator commands (run as
`neondb_owner`):

```sql
ALTER ROLE marketplace_app
    WITH LOGIN PASSWORD '<32-char random>';

ALTER ROLE marketplace_scheduler
    WITH LOGIN PASSWORD '<32-char random>';
```

Store both passwords in the secret manager (Railway env vars + 1Password
vault entry). The app reads them via `DATABASE_URL` (NOBYPASSRLS pool)
and `DATABASE_URL_ADMIN` (BYPASSRLS pool) — never log either value.

## Step 2 — Rotate the API connection strings

Update Railway env vars on the API service:

| Variable | New DSN |
|---|---|
| `DATABASE_URL` | `postgres://marketplace_app:<pw>@<host>/<db>?sslmode=require` |
| `DATABASE_URL_ADMIN` | `postgres://marketplace_scheduler:<pw>@<host>/<db>?sslmode=require` |

`DATABASE_URL_ADMIN` is OPTIONAL — when unset, `wireInfrastructure`
falls back to `DATABASE_URL` for the admin pool and emits a `WARN` at
boot (`admin pool falling back to app pool`). That fallback is the
acceptable mode during the rollout window: the app stays available
even if the operator has not yet configured the second DSN.

After the rotation:

```bash
railway run --service api -- env | grep DATABASE_URL
```

confirms both vars resolve.

## Step 3 — Roll the API instances

Trigger a Railway redeploy on the API service. New instances pick up
the rotated DSNs; old instances keep using `neondb_owner` until they
are drained. The warm-up window is bounded by the Railway rolling-
update setting (typically 60-120 seconds).

During the rollout, watch the structured log stream for:

- `"non-tenant repository entry point reached without system-actor tag"`
  — surfaces unmigrated callers that would have silently leaked rows
  before, or have started failing-closed since the rotation.
- `"admin pool falling back to app pool"` — should disappear once
  `DATABASE_URL_ADMIN` is set.
- `"row-level security violation"` — indicates a write path that did
  not establish tenant context. Investigate immediately.

## Step 4 — Verify under marketplace_app

Connect as `marketplace_app` and run the smoke probe:

```sql
-- Without tenant context: every RLS-protected SELECT must return 0 rows.
SELECT count(*) FROM proposals;          -- expected: 0
SELECT count(*) FROM disputes;           -- expected: 0
SELECT count(*) FROM proposal_milestones; -- expected: 0
SELECT count(*) FROM payment_records;    -- expected: 0
SELECT count(*) FROM messages;           -- expected: 0
SELECT count(*) FROM conversations;      -- expected: 0
SELECT count(*) FROM notifications;      -- expected: 0
SELECT count(*) FROM invoice;            -- expected: 0
SELECT count(*) FROM audit_logs;         -- expected: 0
SELECT count(*) FROM disputes;           -- expected: 0

-- With a tenant context, the same SELECT returns the expected rows:
BEGIN;
SELECT set_config('app.current_org_id', '<known-org-uuid>', true);
SELECT count(*) FROM proposals;
ROLLBACK;
```

The "0 rows without context" assertion is the SAFE-DEFAULT proof — the
policy filters every row when `app.current_org_id` is unset.

## Step 5 — Verify under marketplace_scheduler

```sql
-- BYPASSRLS — every SELECT returns the full table count regardless of
-- tenant context. This is intentional: schedulers and webhooks need
-- cross-tenant visibility.
SELECT count(*) FROM proposals;          -- expected: full count
SELECT count(*) FROM disputes;           -- expected: full count
```

## Rollback

If a critical incident surfaces during the rollout window:

1. Revert `DATABASE_URL` to the legacy `neondb_owner` DSN on Railway.
2. Trigger a Railway redeploy.
3. The legacy role bypasses RLS (it is the table owner), so every
   query returns rows regardless of context — same behavior as before
   the rollout.
4. Once the incident is resolved, restart the rollout from Step 2.

`make migrate-down` against migration 137 is NOT recommended on
production — it drops the two roles, which invalidates any open
connection that authenticated through them. Use the env-var revert
path above instead.

## Post-rollout cleanup

After ≥ 24 h with no `WARN` lines and no RLS violations:

1. Promote `routed_db.go`'s `warnIfNotSystemActor` from a warning to
   an error (`return ErrSystemActorOnly`) on every legacy `GetByID`.
   This converts any future drift into a fail-closed 5xx instead of a
   silent NotFound.
2. Remove the `admin pool falling back to app pool` fallback from
   `wireInfrastructure`. After this point, missing `DATABASE_URL_ADMIN`
   is a fatal configuration error.

These two follow-ups are tracked as `RLS-FOLLOWUP-1` and
`RLS-FOLLOWUP-2`.

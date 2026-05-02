# 0003. PostgreSQL Row-Level Security with system-actor split

Date: 2026-04-30

## Status

Accepted

## Context

The marketplace stores cross-tenant data (multiple agencies and
clients) in shared PostgreSQL tables. Application-level checks
(`WHERE organization_id = $1` clauses, handler ownership
verification) are the primary defense against cross-tenant data
leaks, but a single forgotten clause on a new endpoint would
expose every tenant's data.

We need a database-level guard that catches application bugs
before they leak data. PostgreSQL's Row-Level Security (RLS) is
the standard mechanism: a policy bound to each table refuses any
SELECT/UPDATE/DELETE row that does not match the current
session's tenant.

However, naive RLS breaks two legitimate access patterns:

1. **Background jobs** (cron schedulers, webhook workers,
   GDPR exports) operate without an end-user in scope. They need
   to read across tenants — a payment scheduler must reach every
   org's pending payouts.
2. **Admin endpoints** legitimately read other tenants' data.
   The admin moderation surface lists every conversation, every
   review.

A single "bypass everything" superuser is dangerous: a misrouted
request through a shared connection pool could leak. We need a
clean separation between user-driven traffic and system-driven
traffic.

## Decision

We will enable RLS on every tenant-scoped table and split callers
into two explicit categories:

1. **User-driven traffic** — every request that has an
   authenticated user. The Auth middleware sets a per-transaction
   PostgreSQL setting via `SET LOCAL app.current_user_id =
   '<uuid>'` and `SET LOCAL app.current_org_id = '<uuid>'`. RLS
   policies match against these settings:

   ```sql
   USING (organization_id = current_setting('app.current_org_id', true)::uuid)
   ```

   The `true` parameter makes `current_setting` return `NULL` if
   the setting is unset — which causes the policy to deny by
   default. Safe-default semantics.

2. **System-driven traffic** — cron jobs, webhook workers,
   long-running background tasks. They wrap their context with
   `system.WithSystemActor(ctx)` from
   `internal/system/system_actor.go`. The repository layer
   detects this marker via `system.IsSystemActor(ctx)` and skips
   the per-org `SET LOCAL`, falling back to a special
   `BYPASSRLS` connection pool dedicated to system tasks.

The application database user (`marketplace_app`) does **not**
have `BYPASSRLS`. The migration user (`marketplace_migrator`)
does, but it is only used by `make migrate-up` — never by the
running API.

Concrete enforcement:

- Migration `125_enable_row_level_security.up.sql` enables RLS on
  9 tenant-scoped tables and creates the policies.
- `internal/adapter/postgres/transaction.go` wraps every request
  in a transaction and stamps `app.current_user_id` /
  `app.current_org_id` from the request context.
- `internal/system/system_actor.go` provides
  `WithSystemActor(ctx)` and `IsSystemActor(ctx)`. Background
  jobs that need cross-tenant access must call them explicitly.
- `internal/handler/middleware/requestid.go::MustGetOrgID` panics
  if a handler tries to read tenant data without an org id —
  catches the missing-context bug at request time.
- An integration test `test/rls_caller_audit_test.go` runs the
  full suite under the `NOBYPASSRLS` role to confirm no test
  exercises the wrong path.

## Consequences

### Positive

- A forgotten `WHERE organization_id = $1` clause on a new
  endpoint cannot leak data — RLS denies the rows. We have caught
  two such bugs in pre-prod through this guard.
- `system.WithSystemActor` is grep-able. Code reviewers can audit
  every cross-tenant escape valve in seconds.
- The RLS policies are documented in-file (each `up.sql` carries a
  policy block with a comment explaining the threat model).
- New contributors writing a new feature inherit the RLS guard for
  free as long as they FK to `organization_id`.

### Negative

- One `SET LOCAL` round-trip per request adds ~0.3 ms of latency
  per query path. Negligible for our workload.
- Any contributor adding a new background job MUST remember to
  wrap the goroutine context with `system.WithSystemActor`.
  Forgetting this returns empty result sets, which is loud (the
  cron run reports "0 rows touched") but still a real footgun.
  We compensate by:
  - Documenting the rule in `backend/CLAUDE.md`.
  - The repository's `system.IsSystemActor` log: every
    cross-tenant read by a system actor is logged with the
    caller stack.
- Database superuser cannot be the application user — a
  development convenience lost. Local docker-compose provisions
  the proper non-superuser app role.

## Alternatives considered

- **Application-level checks only** — what we had before
  migration 125. A single forgotten WHERE clause was a P0 bug.
  Rejected after a near-miss in code review where a list endpoint
  briefly returned other tenants' rows.
- **Per-tenant database** — strong isolation but operationally
  expensive (N migration runs, N backups). Overkill at our scale;
  reconsidered if we onboard a regulated industry tenant.
- **Materialized views per tenant** — static partitions, no RLS
  overhead. Rejected because schema evolution becomes a
  combinatorial explosion (one DDL per tenant).

## References

- `backend/migrations/125_enable_row_level_security.up.sql` and
  `.down.sql` — the policies.
- `backend/internal/adapter/postgres/transaction.go` — the
  per-request `SET LOCAL` stamping.
- `backend/internal/system/system_actor.go` — system-actor
  helpers.
- `backend/test/rls_caller_audit_test.go` — the NOBYPASSRLS
  integration test.
- `backend/CLAUDE.md` lines 575-602 — the migration safety rule
  that complements RLS.
- `docs/adr/0002-org-scoped-business-state.md` — the org model
  this RLS pattern protects.

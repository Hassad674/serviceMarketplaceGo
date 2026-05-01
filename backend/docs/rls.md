# PostgreSQL Row-Level Security (RLS)

This document describes the per-tenant row filter installed by
migration 125 (Phase 5, SEC-10). RLS is the BACKUP defense layer
behind the application-level `WHERE org_id = $1` /
`WHERE user_id = $1` filters in repositories. With RLS on, a single
missed filter in repository code can no longer leak another tenant's
rows — Postgres itself rejects rows that do not match the policy.

## Tables under RLS (9)

| Table | Tenant column | Policy strategy |
|---|---|---|
| `conversations` | `organization_id` | Direct, plus participant escape hatch via `conversation_participants` for solo providers |
| `messages` | `conversations.organization_id` | JOIN to the parent conversation |
| `invoice` | `recipient_organization_id` | Single-side ownership |
| `proposals` | `client_organization_id` OR `provider_organization_id` | Two-sided (either party sees the row) |
| `proposal_milestones` | inherited from `proposals` via JOIN | Two-sided through the parent proposal |
| `notifications` | `user_id` | Per-recipient, NOT per-org |
| `disputes` | `client_organization_id` OR `provider_organization_id` | Two-sided |
| `audit_logs` | `user_id` | Per-actor, NOT per-org. Append-only (mig 124) |
| `payment_records` | `organization_id` | Single-side (client org) |

The `users`, `organizations`, `organization_members`, and other admin
/ auth tables are deliberately NOT under RLS. Auth flows (login,
register, password reset) run BEFORE the user/org context is
established, so a policy keyed on `app.current_user_id` would block
these flows entirely.

## How a policy works

Each policy uses `current_setting('app.current_org_id', true)` (or
`app.current_user_id` for the per-user tables). The literal `true`
arg makes `current_setting` return `NULL` when the setting is unset
— Postgres then evaluates the USING expression to `NULL`, which is
treated as `FALSE`, and the row is filtered out. This is the SAFE
DEFAULT: forgetting to call `SetCurrentOrg` / `SetCurrentUser` in
the application denies access rather than granting it.

Both `ENABLE ROW LEVEL SECURITY` and `FORCE ROW LEVEL SECURITY` are
set on every table. FORCE is critical: without it, the table OWNER
bypasses the policy. Migration 125 turns FORCE on so the migration
user is also subject to RLS — except superusers, which always
bypass (see "Production database user requirement" below).

## Setting the tenant context (Go)

Use the helpers in `backend/internal/adapter/postgres/rls.go`:

```go
// Inside an open transaction:
err := postgres.SetCurrentOrg(ctx, tx, orgID)
err  = postgres.SetCurrentUser(ctx, tx, userID)
// or both at once:
err  = postgres.SetTenantContext(ctx, tx, orgID, userID)
```

Or — preferred — go through the tenant-aware transaction wrapper
which sets the context AND opens the tx in one call:

```go
err := txRunner.RunInTxWithTenant(ctx, orgID, userID, func(tx *sql.Tx) error {
    // every query in here runs with RLS active
    return nil
})
```

The non-tenant `RunInTx` remains for transactions that exclusively
touch RLS-free tables (e.g. `pending_events`, `search_queries`,
`organizations`).

`SET LOCAL` is used (via `set_config(name, value, true)`) so the
setting is scoped to the current transaction. When the tx commits
or rolls back, the value is discarded — the setting CANNOT leak
across pooled connections.

## Adding RLS to a new tenant-scoped table — 3-step recipe

When you ship a new feature whose table holds business state, follow
this recipe in the migration that creates the table:

```sql
-- 1. Enable RLS + force on the owner.
ALTER TABLE my_new_table ENABLE ROW LEVEL SECURITY;
ALTER TABLE my_new_table FORCE ROW LEVEL SECURITY;

-- 2. Policy. Single-side ownership:
CREATE POLICY my_new_table_isolation ON my_new_table
    USING (organization_id = current_setting('app.current_org_id', true)::uuid);

-- Two-sided (e.g. a transaction record visible to both parties):
CREATE POLICY my_new_table_isolation ON my_new_table
    USING (
        client_organization_id = current_setting('app.current_org_id', true)::uuid
        OR provider_organization_id = current_setting('app.current_org_id', true)::uuid
    );

-- Per-user (NOT per-org), e.g. a user-specific feed:
CREATE POLICY my_new_table_isolation ON my_new_table
    USING (user_id = current_setting('app.current_user_id', true)::uuid);
```

Then add an integration test (see "Testing cross-tenant access"
below) so the policy is regression-proof.

## Testing cross-tenant access

The reference integration test is
`backend/internal/adapter/postgres/rls_isolation_test.go`. It is
gated on `MARKETPLACE_TEST_DATABASE_URL` (auto-skip when unset).

Pattern:

1. Insert two orgs, two users, one row per RLS table per org —
   USING the postgres superuser connection (which bypasses RLS, so
   setup is unconstrained).
2. Open a transaction.
3. `SET LOCAL ROLE marketplace_rls_test` — a non-superuser,
   non-bypass-rls role created by the test setup. RLS only fires
   for non-superusers, so this step is mandatory.
4. `SET LOCAL app.current_org_id = orgA` (and/or
   `app.current_user_id`).
5. Assert SELECT/UPDATE/DELETE on orgB's row returns 0 rows.
6. Assert SELECT on orgA's own row works (positive control).

Adding a new table requires:

- One row in the `rlsCases()` table-driven slice.
- A matching `insertX` helper.
- Cleanup in `cleanupFixture`.

The test will then automatically participate in
`TestRLS_SelectDenied_AcrossTenants`,
`TestRLS_UpdateDenied_AcrossTenants`,
`TestRLS_DeleteDenied_AcrossTenants`,
`TestRLS_NoContextSet_HidesEverything`,
`TestRLS_SameOrgAccess_PositiveControl`, and
`TestRLS_PropertyTest_AnyCrossTenantRowDenied`.

## Production database user requirement

For RLS to be effective in production, the application database
user MUST satisfy three conditions:

1. **NOT a superuser.** Superusers always bypass RLS, regardless of
   `FORCE ROW LEVEL SECURITY`. Verify with:
   ```sql
   SELECT rolsuper FROM pg_roles WHERE rolname = current_user;
   -- expected: f
   ```
2. **NOT `BYPASSRLS`.** This attribute is identical to superuser for
   RLS purposes. Verify with:
   ```sql
   SELECT rolbypassrls FROM pg_roles WHERE rolname = current_user;
   -- expected: f
   ```
3. **NOT the table owner.** With FORCE RLS, the owner is no longer
   bypassed — but for defense-in-depth, the application user should
   not own the tables. The migration user (the role that ran the
   `CREATE TABLE`) IS the owner; the application user is a separate
   role that has been granted `SELECT, INSERT, UPDATE, DELETE` on
   the tables it needs to read/write.

Recommended infra setup (Railway / Neon dashboards or psql):

```sql
-- Migration role: owns tables, runs DDL, has all privileges.
CREATE ROLE marketplace_migrate LOGIN PASSWORD '...';
GRANT CONNECT ON DATABASE marketplace TO marketplace_migrate;
GRANT ALL PRIVILEGES ON SCHEMA public TO marketplace_migrate;

-- Application role: connects from the API process. Non-superuser,
-- non-bypassrls, non-owner. RLS policies fire normally for it.
CREATE ROLE marketplace_app LOGIN PASSWORD '...' NOSUPERUSER NOBYPASSRLS;
GRANT CONNECT ON DATABASE marketplace TO marketplace_app;
GRANT USAGE ON SCHEMA public TO marketplace_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO marketplace_app;
-- Future tables created by marketplace_migrate also need to be
-- granted to marketplace_app:
ALTER DEFAULT PRIVILEGES FOR ROLE marketplace_migrate IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO marketplace_app;

-- audit_logs is append-only (mig 124): revoke UPDATE/DELETE.
REVOKE UPDATE, DELETE ON audit_logs FROM marketplace_app;
```

Run the API as `marketplace_app`; run migrations as
`marketplace_migrate`. NEVER run the API as `postgres` or as the
migration owner — RLS will silently be bypassed.

## Checklist before applying migration 125 to production

- [ ] Application database user is NOT a superuser
      (`SELECT rolsuper FROM pg_roles WHERE rolname = current_user`)
- [ ] Application database user is NOT `BYPASSRLS`
      (`SELECT rolbypassrls FROM pg_roles WHERE rolname = current_user`)
- [ ] Application database user is NOT the table owner
      (`SELECT relowner::regrole FROM pg_class WHERE relname = 'messages'`
      should NOT match `current_user`)
- [ ] Application code has been audited: every repository method that
      touches an RLS-protected table either goes through
      `RunInTxWithTenant` OR explicitly calls `SetCurrentOrg` /
      `SetCurrentUser` at the start of a manually-managed transaction.
- [ ] `MARKETPLACE_TEST_DATABASE_URL` integration tests pass on a
      copy of production schema (run `rls_isolation_test.go`).
- [ ] Rollback plan documented: `migrate-down 1` reverses 125, but
      doing so is a security regression — only run on a disposable
      environment.

## Troubleshooting

### "permission denied for table X"

The application user does not have `SELECT, INSERT, UPDATE, DELETE`
on table `X`. Grant it:
```sql
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE X TO marketplace_app;
```
This is a GRANT issue, NOT an RLS issue — RLS does not raise
permission errors, it silently filters rows.

### "Empty result set where I expected rows"

The application forgot to call `SetCurrentOrg` / `SetCurrentUser` —
or the user authenticated without an `organization_id` and is not
listed as a `conversation_participants` row for the queried
conversation. Set the context explicitly and retry. To debug:
```sql
-- Inside the transaction:
SELECT current_setting('app.current_org_id', true);
SELECT current_setting('app.current_user_id', true);
```
A blank result means the setting was never set in this tx.

### "Cross-tenant access in tests works"

The test is running as a superuser (the default `postgres` connection
from `MARKETPLACE_TEST_DATABASE_URL` is usually `postgres`). Use
`SET LOCAL ROLE marketplace_rls_test` inside the test transaction to
drop the superuser bit. See `rls_isolation_test.go` for the pattern.

## System-actor caller pattern

Some code paths legitimately need to read RLS-protected rows
without an authenticated user / tenant context: the proposal
scheduler (auto-approve / fund-reminder / auto-close), the
dispute scheduler (auto-resolve / escalate), the Stripe webhook
reconciler, the admin force-activate / force-escalate / AI
chat / resolve endpoints, and the referral cross-tenant
aggregator. These callers cannot satisfy the
`current_setting('app.current_org_id')` policy filter because
they touch rows owned by multiple orgs in sequence.

### Boundary marker

Every system-actor entry point tags its context with
`system.WithSystemActor(ctx)` from `backend/internal/system`:

```go
import "marketplace-backend/internal/system"

// Scheduler goroutine entry point.
go disputeScheduler.Run(system.WithSystemActor(ctx), interval)

// HTTP handler that runs on behalf of an admin (not the tenant).
func (h *AdminHandler) ResolveDispute(w http.ResponseWriter, r *http.Request) {
    ctx := system.WithSystemActor(r.Context())
    h.svc.AdminResolve(ctx, ...)
}
```

User-facing app services then branch on the marker when reading
RLS-protected rows:

```go
func (s *Service) loadDisputeForActor(ctx context.Context, id uuid.UUID) (*dispute.Dispute, error) {
    if system.IsSystemActor(ctx) {
        return s.disputes.GetByID(ctx, id) // legacy non-tenant path
    }
    orgID := middleware.MustGetOrgID(ctx)
    return s.disputes.GetByIDForOrg(ctx, id, orgID) // tenant-aware path
}
```

### Adapter-level guard

The legacy `GetByID` methods on the four RLS-protected
adapters (proposal, dispute, milestone, payment_record) call
`warnIfNotSystemActor(ctx, op)` as their very first line.
Reaching `GetByID` from a non-system-actor context surfaces a
structured WARN log line — the operator sees "unmigrated
caller drift" in the dashboard before an outage develops.

The guard is intentionally a warning, not a panic, because:

- Under the migration role (BYPASSRLS) the read still works,
  so promoting to a hard error would convert a latent
  unmigrated caller into a 5xx.
- Under NOSUPERUSER NOBYPASSRLS the policy filters every row
  regardless, so the call already fails closed — the WARN
  line just points at the right caller to fix.

After the prod role rotation completes and the warn log is
empty for a sustained period, a follow-up PR can promote the
guard to `return ErrSystemActorOnly` for defence in depth.

### Production deployment — privileged scheduler pool

In production the application connects through two distinct
DB roles, both rooted at the same Postgres cluster:

| Pool | Role | Surface |
|---|---|---|
| `marketplace_app` | NOSUPERUSER NOBYPASSRLS | every user-facing request, every tenant-scoped repo call |
| `marketplace_scheduler` | NOSUPERUSER **BYPASSRLS** | the pending-events worker, the dispute scheduler, the Stripe webhook reconciler, the admin override surfaces |

Both roles share the same SCHEMA + GRANTS — the only
difference is `BYPASSRLS`. The application picks the pool by
checking `system.IsSystemActor(ctx)` at the adapter boundary
(today the boundary marker is in place; the dual-pool wiring
lands in a follow-up infra PR).

Why two roles instead of one: a scheduler that runs as
NOBYPASSRLS would need to read proposals across every org in
the system, so it would either have to set the org context
inside every read (impossible — it doesn't know which orgs to
target) or leave the context unset (which the policy filters
to zero rows). Granting BYPASSRLS to the scheduler is the
narrowest possible escape hatch: it is OFF for every other
caller and ON only for the explicitly-tagged system-actor
paths.

### Migration checklist for ops

When rotating the prod role:

1. [ ] Verify `marketplace_app` has the conditions listed in
       "Production database user requirement" above.
2. [ ] Verify `marketplace_scheduler` has BYPASSRLS:
       ```sql
       SELECT rolname, rolbypassrls FROM pg_roles
        WHERE rolname IN ('marketplace_app', 'marketplace_scheduler');
       -- expected: marketplace_app=f, marketplace_scheduler=t
       ```
3. [ ] Run `rls_caller_audit_test` against a clone of prod
       schema to confirm every user-facing GetByIDForOrg path
       passes:
       ```sh
       MARKETPLACE_TEST_DATABASE_URL=<prod-clone-dsn> \
         go test ./internal/adapter/postgres/ -run TestRLSCallerAudit
       ```
4. [ ] Rotate the API process to use `marketplace_app` for the
       primary pool.
5. [ ] Watch the structured log stream for
       `"non-tenant repository entry point reached without
       system-actor tag"` WARNs — empty for ≥1h means no drift.

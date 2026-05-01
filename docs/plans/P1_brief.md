# P1 — RLS callers migration (35-38 legacy `.GetByID()` → tenant-aware)

**Phase:** F.1 CRITICAL #1 — pre-prod blocker absolu
**Source audit:** SEC-FINAL-01 (`auditsecurite.md`) + PERF-FINAL-B-14 (`auditperf.md`)
**Effort estimate:** 3 jours mécaniques
**Tool:** 1 fresh agent dispatched, séquentiel
**Branch:** `fix/p1-rls-callers-migration`

## Problem

PR #65 (BUG-NEW-04) wrapped 8 RLS-protected REPO methods with `RunInTxWithTenant`, but kept the legacy `GetByID(ctx, id)` signature alive for system-actor schedulers. **38 APP callers** still use that legacy signature. Today they work because the migration role bypasses RLS. The moment ops rotate to `marketplace_app NOSUPERUSER NOBYPASSRLS`:
- Every proposal action 404s
- Every dispute action 404s
- Every review submission 404s
- Every checkout 404s
- Every milestone auto-approve / auto-close fails silently
- Total app outage masquerading as routing bugs

## Sites (38)

Identified via:
```bash
grep -rn "\.GetByID(" backend/internal/app/ --include="*.go" \
  | grep -v "GetByIDForOrg\|GetByIDWithVersion\|_test.go\|mocks_test.go" \
  | grep -E "proposals\.GetByID|disputes\.GetByID|milestones\.GetByID|reviews\.GetByID|records\.GetByID"
```

| File | Lines | Repo |
|---|---|---|
| `app/proposal/service_actions.go` | 20, 72, 101, 165, 260, 296, 351, 434, 446, 500, 620, 662 | proposals (12) |
| `app/dispute/service_actions.go` | 119, 269, 353, 437, 478, 528, 554, 600, 686, 767, 792, 838, 849 | proposals + disputes (13) |
| `app/proposal/service_scheduler.go` | 109, 140, 153, 298, 305, 325 | milestones + proposals (6) — **system actor** |
| `app/dispute/service_helpers.go` | 80, 389 | proposals (2) |
| `app/review/service.go` | 86, 280 | proposals (2) |
| `app/dispute/scheduler.go` | 132 | proposals (1) — **system actor** |
| `app/payment/payout_request.go` | 351 | records (1) |
| `app/referral/wiring_adapters.go` | 129 | proposals (1) |

## Required new repo methods (3)

`GetByIDForOrg(ctx, id, callerOrgID)` to be added on:
1. `port/repository/dispute_repository.go` + `adapter/postgres/dispute_repository.go`
2. `port/repository/milestone_repository.go` + `adapter/postgres/milestone_repository.go`
3. `port/repository/review_repository.go` + `adapter/postgres/review_repository.go`

(`payment_record_repository.go` already has `GetByIDForOrg`. `proposal_repository.go` already has it.)

## Fix pattern (all user-facing call sites)

```go
// BEFORE
p, err := s.proposals.GetByID(ctx, input.ProposalID)
if err != nil { return err }

// AFTER
orgID := middleware.MustGetOrgID(ctx)  // panic if not set — handler must have set it
p, err := s.proposals.GetByIDForOrg(ctx, input.ProposalID, orgID)
if err != nil { return err }
```

If `middleware.MustGetOrgID(ctx)` doesn't exist, create it in `internal/handler/middleware/auth.go` — panic semantics for "this code path requires a tenant context, missing context = bug not user error".

## Scheduler/system-actor sites — DIFFERENT pattern

`service_scheduler.go` (proposal auto-approve, auto-close) + `dispute/scheduler.go` (auto-resolve) run from cron without a user context. They MUST keep the legacy `GetByID` signature, BUT:

1. Document them with a `// SYSTEM-ACTOR: bypasses tenant gate; runs on privileged DB connection only` comment.
2. Add a runtime guard `if !systemActorCtx(ctx) { return ErrSystemActorOnly }` at the top of `GetByID` that checks a context key set explicitly by the scheduler entry-point.
3. Add `internal/system/context.go` with `WithSystemActor(ctx) context.Context` + `IsSystemActor(ctx context.Context) bool`.
4. Scheduler entrypoints (in `cmd/api/main.go` or wire helper) wrap the goroutine context with `WithSystemActor` before passing to scheduler.

Don't lose the deadlock-on-prod safety: in production, systemActor connections must use a separate pool/role with `BYPASSRLS` set. Document in `backend/docs/rls.md`.

## Tests required

1. **Unit tests** on every migrated method — assert `GetByIDForOrg` is called with the correct `orgID` from context (mock-based).
2. **Integration test `rls_caller_audit_test.go`**:
   - Create a `marketplace_test_app NOSUPERUSER NOBYPASSRLS` PG role + grant SELECT/INSERT/UPDATE on the right tables.
   - Run every public service action through the role (with valid tenant context).
   - Assert: zero `ErrNotFound` on legitimate reads.
   - Run cross-tenant attempts: assert `ErrNotFound` returned correctly.
   - Run scheduler paths through privileged role (or simulate via direct sql.DB): assert success.
3. **Smoke test on isolated DB clone**: `createdb -p 5435 marketplace_test_p1 -T marketplace_go`, apply migrations, set role, run integration test suite, verify zero failures.

## Validation gates (before commit)

- `go build ./...` — clean
- `go vet ./...` — clean
- `go test ./... -count=1 -race` — green
- `gosec ./...` — 0 new issues introduced
- New integration test `rls_caller_audit_test.go` — all sites green under `NOBYPASSRLS` role

## Out-of-scope (flag, don't fix)

- LiveKit / call code — never touch
- Other 50+ legacy callers in non-RLS-protected paths (jobs, portfolios, users) — that's not P1 scope
- ISP migration to segregated interfaces — that's P7
- Stripe webhook async — that's P8
- `.github/workflows/*` — token can't push

## Branch ownership

Agent creates `fix/p1-rls-callers-migration` from clean `main` via `git worktree add`. Never touches another branch.

## Commits expected (~8-10)

One commit per logical group:
1. `feat(repo): add GetByIDForOrg on dispute_repository`
2. `feat(repo): add GetByIDForOrg on milestone_repository`
3. `feat(repo): add GetByIDForOrg on review_repository`
4. `feat(middleware): add MustGetOrgID + system-actor context helpers`
5. `refactor(proposal/service_actions): migrate 12 GetByID callers to tenant-aware`
6. `refactor(dispute/service_actions): migrate 13 GetByID callers + 2 helpers`
7. `refactor(review/service + payment/payout_request + referral/wiring_adapters): migrate 4 callers`
8. `refactor(proposal+dispute schedulers): gate system-actor paths with explicit context`
9. `test(integration): rls_caller_audit_test under NOBYPASSRLS role`
10. `docs: update backend/docs/rls.md with system-actor pattern`

## PR description requirements

- Lists every call site migrated
- Demonstrates the integration test fails on `main` and passes on the branch
- Includes a "post-merge action" section: how ops should rotate the prod role

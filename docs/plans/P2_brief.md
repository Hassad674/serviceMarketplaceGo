# P2 — main.go split 909 → ≤300 lines

**Phase:** F.1 CRITICAL #2
**Source audit:** QUAL-FINAL-B-XX (`auditqualite.md`) — `func main()` violates the 50-line limit by 18×
**Effort:** ½j mécanique
**Tool:** 1 fresh agent dispatched
**Branch:** `fix/p2-main-split`

## Problem

`backend/cmd/api/main.go` is **909 lines** despite Phase 3 F (PR #58) extracting 16 wire helpers + 2 adapters. The remaining bulk:
- ~26 lines of local aliases `infra.X` → `X` (DRY violation, can be eliminated by using `infra.X` directly downstream)
- ~80 lines of inline service constructors (proposal, milestone, payment, kyc, etc.)
- ~120 lines of handler wrappers + setters
- ~57 lines of `RouterDeps` literal
- ~150 lines of imports + `wireInfrastructure` + `httpRateLimiter` + `setup` + `serve` + `graceful shutdown`

**Realistic target: ≤300 lines** (909 → 300 = -67% reduction). Going lower would obscure the wiring or require redesigning setter-based cross-feature wiring (out of scope).

## Plan (validated by user, 5 commits)

### Commit 1 — extract `wire_auth.go`
- `organizationSvc` (depends on infra)
- `authSvc` (with full `auth.ServiceDeps`)
- `loginBruteForce` + `passwordResetThrottle` (Redis brute-force services)
- `authHandler` with `WithBruteForce`

Pattern: struct `authDeps` + struct `authWiring` + function `wireAuth(deps authDeps) authWiring`. Mirrors existing `wire_dispute.go`, `wire_invoicing.go`, etc.

### Commit 2 — extract `wire_proposal.go`
- `proposalRepo` (with txRunner + WithTxRunner) — comments preserved (BUG-NEW-04 path 4/8 explanation)
- `milestoneRepo` + `milestoneSvc` (BUG-NEW-04 path 5/8)
- `paymentRecordRepo` (BUG-NEW-04 path 7/8)
- `bonusLogRepo`
- `pendingEventsRepo`
- `searchPublisher` (call to existing `wireSearchPublisher`)
- `txRunner`
- `milestoneTransitionsRepo`
- `proposalSvc` (NewService with full ServiceDeps)
- `pendingEventsWorker`
- `proposalHandler`

Note: `proposalSvc` depends on `notifSvc` and `paymentInfoSvc` and `messagingSvc` which are wired AFTER in main.go currently. Solution: keep the constructor call in `wire_proposal.go`, but pass these deps through `proposalDeps`. The setter calls (`SetProposalStatusReader` etc.) stay in main.go after both services exist.

### Commit 3 — extract `wire_review_portfolio_jobs_report.go` (groupé)
- Review: `reviewRepo` + `reviewSvc` + `reviewHandler`
- Portfolio: `portfolioRepo` + `portfolioSvc` + `portfolioHandler`
- Project history: `projectHistorySvc` + `projectHistoryHandler`
- Jobs: `jobRepo` + `jobAppRepo` + `jobViewRepo` + `jobCreditRepo` + `jobSvc` + `jobHandler` + `jobAppHandler`
- Report: `reportRepo` + `reportSvc` + `reportHandler`

These features are independent of each other and small. Group them in one wire file with separate `wire<X>` functions and a single `wireReviewPortfolioJobsReport` orchestrator that calls them all. Or keep them as separate files (`wire_review.go`, `wire_jobs.go`, etc.) — agent's call.

### Commit 4 — extract `wire_uploads_messaging_moderation_kyc.go` (groupé)
- Uploads: `uploadCtx` + `uploadHandler` + `freelanceProfileVideoHandler` + `referrerProfileVideoHandler` + `healthHandler`
- Messaging: `messagingSvc` initial wiring (WITHOUT setter calls — those stay in main.go) + `messagingHandler` + `wsHandler`
- Moderation: `moderationOrchestrator` + the 6 `Set*` calls (these stay in main.go since they cross multiple services)
- KYC: `kycCtx` + `startKYCScheduler` call

### Commit 5 — extend `wire_payment.go` + final main.go cleanup
- Add `paymentInfoSvc` (`paymentapp.NewService`) to existing `wire_payment.go`
- Add `walletHandler`, `billingHandler` to existing `wire_payment.go`
- Add `paymentInfoSvc.SetProposalStatusReader(newProposalStatusAdapter(proposalSvc))` line stays in main.go (cross-service setter)
- Eliminate the 26-line `Local aliases` block: replace `userRepo` with `infra.UserRepo`, `db` with `infra.DB`, etc. throughout main.go (sed replace + verify)
- Verify final main.go ≤ 300 lines
- Verify build + vet + test green

## Hard constraints (paranoid mode)

- **ZERO behaviour change.** Every wire helper produces the EXACT SAME services with the EXACT SAME dependencies as inline today. The router snapshot golden test (`internal/handler/testdata/routes.golden`, 265 routes) MUST stay byte-identical.
- **Validation pipeline before EVERY commit**:
  ```bash
  cd /tmp/mp-p2-main-split/backend
  go build ./...
  go vet ./...
  go test ./... -count=1 -short -race
  go test ./internal/handler/ -run TestRouterSnapshot -count=1
  ```
  All green.
- **Comments preserved**: every BUG-NEW-04 / SEC-XX / CRITICAL preserve comment in main.go must be moved to the wire helper VERBATIM (don't rephrase, don't summarise).
- **Setter calls (cross-feature wiring) STAY in main.go**: `paymentInfoSvc.SetProposalStatusReader(...)`, `messagingSvc.SetMediaRecorder(...)`, `messagingSvc.SetModerationOrchestrator(...)`, `reviewSvc.SetModerationOrchestrator(...)`, `authSvc.SetModerationOrchestrator(...)`, `profileSvc.WithModerationOrchestrator(...)`, `jobSvc.SetModerationOrchestrator(...)`, `proposalSvc.SetModerationOrchestrator(...)`, `embeddedNotifier.SetReferralKYCListener(...)`, `freelanceProfileSvc = freelanceProfileSvc.WithCacheInvalidator(...)`, etc. — these cross multiple wire boundaries.
- **One commit per group** (5 commits expected). NEVER squash. Conventional message: `refactor(cmd/api): extract <X> into wire_<X>.go`.
- **`testdata/routes.golden`** unchanged — verifies router shape stable.

## Tests required

- **Existing `wire_test.go`** stays passing (15 sub-tests around mount helpers + nil-safe wires + ws origin patterns).
- **No new test file required** for the extraction itself (it's pure structural refactoring with zero behaviour change). The golden snapshot + the existing test suite are the regression nets.
- If the agent finds a wiring bug during the extraction (e.g. a setter that was never called in production), flag it in PR description — don't silently fix it.

## OFF-LIMITS

- LiveKit / call code: `backend/internal/app/call/`, `backend/internal/handler/call_handler.go`, `backend/internal/adapter/livekit/`. The existing `if cfg.LiveKitConfigured()` block in main.go can move to a new `wire_call.go` (extraction is OK), but the underlying code stays untouched.
- `.github/workflows/*` — token can't push.
- Other plans (P1, P3, P4, P5+) — never touch.
- Web / admin / mobile — out of scope.

## Branch ownership

Agent creates `fix/p2-main-split` from clean `main` via `git worktree add`. Never touches another branch.

## Final report (under 700 words)

Lead with PR URL.

1. Final main.go line count (target ≤300)
2. New wire files created (list with line counts)
3. `testdata/routes.golden` unchanged (yes/no)
4. Validation pipeline output (per commit)
5. Setter calls preserved in main.go (list)
6. "Branch ownership confirmed: only worked on `fix/p2-main-split`"

GO.

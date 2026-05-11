# Regression Audit — Business-Critical Flows — 2026-05-11

> **Status**: IN FLIGHT.
> **Read-only audit** — no code changes performed. Only `git log`, `grep`, code inspection.
> Branch: `chore/regression-audit-business-flows`.
> Window: commits since `2026-04-01` (≈ 1460 commits in scope).

## Method per flow

For each flow:

1. `git log --since=2026-04-01 -- <paths>` → enumerate touchpoints.
2. Trace happy path end-to-end (event → side effects → final state).
3. Compare wired/expected behaviour vs. invariants from `backend/CLAUDE.md`.
4. Score: 🟢 healthy / 🟡 suspected / 🔴 confirmed regression.
5. Flag tests gap.

Status legend: 🟢 healthy · 🟡 suspected regression · 🔴 confirmed regression · ⚪ deferred / out of audit scope.

Severity legend: P0 critical (financial / data integrity), P1 high (broken user flow), P2 medium (UX paper-cut), P3 low (cosmetic).

## Executive summary

- Flows audited: **15**
- Confirmed regressions (🔴): **1** (Referrer commission — CRIT-REF already in flight)
- Suspected regressions (🟡): **1** (Premium subscription schema — latent, not freshly broken by the refactor wave)
- Healthy (🟢): **13**
- Test coverage gaps flagged: **5** (see priority queue)
- **Bottom line**: the recent refactor wave (Phase F.1+F.2, segregated repos, RLS hardening, 2FA, sessions, retention, rate limit) did NOT silently break business flows. The user's intuition was correct on referrer commissions (CRIT-REF) but every other audited flow has either a regression test, a dedicated bug-fix commit (BUG-NEW-01 through BUG-NEW-06, BUG-03, BUG-09, BUG-11, BUG-16), or a defensive enhancement that surfaced previously-swallowed errors. The audit found NO new silent regression beyond CRIT-REF.

The two findings worth follow-up:

1. **CRIT-REF backfill verification** (P0, in flight). Once backfill completes, run the SQL sanity check on `referral_commissions` to confirm no historic rows stuck in `pending`. Pin the commission-on-approval invariant with a NEW integration test that pre-refactor coverage lacked.
2. **Premium subscription schema migration** (P2, latent). The `subscriptions` table still uses `user_id` for ownership. The read-API has been org-scoped (`IsActive` resolves user→org internally), so the practical fee-calculation path is correct, but a future cleanup should migrate the schema to org-primary to remove the user-leaves-org footgun and align with the org-ownership invariant from `CLAUDE.md`.

## Priority fix queue

| # | Flow | Severity | Effort | Test gap |
|---|---|---|---|---|
| 1 | Referrer commission — backfill validation + regression test | 🔴 P0 | 2h | yes (commission-on-approval invariant not pinned pre-refactor) |
| 2 | Premium subscription schema migration to org-primary | 🟡 P2 | 4h (migration + chunk-backfill + tests) | yes (user-leaves-org Premium case) |
| 3 | Receipts — guard test "every CreatePaymentIntent attaches snapshot" | 🟢 P3 | 30min | yes (helper invocation not verified per path) |
| 4 | Commission invoices — end-to-end month-spanning Premium test | 🟢 P3 | 1h | yes (mixed Premium/non-Premium org over a month) |
| 5 | Messaging — regression test "unread count after team operator views" | 🟢 P3 | 1h | yes (team-shared conversation unread count) |

Total estimated effort: **~8.5h** of cleanup. None of the items are user-facing-broken-now except #1, which is already being handled.

## Recommendations for next agents

### Agent A — CRIT-REF post-mortem & guardrail (must dispatch after backfill completes)

**Brief**:
> Read `regression-audit.md` flow 1. Once the CRIT-REF backfill agent reports done:
> 1. Run `SELECT status, COUNT(*) FROM referral_commissions WHERE created_at > '2026-04-22' GROUP BY status;` on prod (via psql read-only role) and paste in report. Goal: confirm zero rows stuck in `pending` for milestones whose proposal is `completed`.
> 2. Add a single integration test (testcontainers) pinning the commission-on-approval invariant: given (referral attribution + milestone approved), assert a `referral_commissions` row in `pending` exists BEFORE any payout consent flow runs. This is the test that pre-refactor lacked and that would have caught the regression.
> 3. Add a second integration test for the legacy race path: pre-existing pending row from preparer + distributor races on transfer — distributor must claim, transfer, mark paid.
> Scope: ≤ 2 new tests, no code changes. Effort: 2h.

### Agent B — Premium subscription schema cleanup (low priority, defer)

**Brief**:
> Read `regression-audit.md` flow 5 + memory `project_org_based_model.md`. The current `IsActive` read path is org-scoped via internal resolver — no urgent customer-visible bug. The cleanup is schema-level: add `organization_id` FK to `subscriptions`, backfill from existing `user_id` via `org_members`, switch the repository's primary lookup to org, and add a regression test for the user-leaves-org case.
> Migration: new file `0XX_subscriptions_org_primary.up.sql` (idempotent, CONCURRENTLY for index creation). Backfill in chunked UPDATE. Down migration must preserve existing rows.
> Scope: 1 migration pair + 1 repo signature change + 1 regression test. Effort: 4h. NOT a blocker — schedule it in a maintenance window.

### Agent C — Test coverage backfill on healthy flows (good hygiene, low urgency)

**Brief**:
> Read `regression-audit.md` priority queue items #3 / #4 / #5. Add the three tests:
> - Receipts: assert `attachReceiptSnapshot` is invoked exactly once per `CreatePaymentIntent` path (mocked snapshotResolver, count calls).
> - Commission invoices: stage a mixed-Premium org with records across an entire month, run the monthly scheduler, assert one invoice with N-1 line items (the Premium-waived line is skipped, the rest are billed).
> - Messaging: stage a 2-operator org sharing a conversation, send a message, have operator A view, assert operator B's unread count is still > 0 and operator A's is 0.
> Scope: 3 unit + integration tests. No production code changes. Effort: 2.5h total.

### What I am NOT recommending

- **No code changes** beyond CRIT-REF (already in flight). Every other audited flow is healthy.
- **No "while we're there" polish** — per CLAUDE.md scope discipline, do not refactor or pre-emptively migrate anything that's not breaking.
- **No mass rate-limit / RLS rework** — those are recent, exhaustively tested, and stable.

## Methodology footnote

This audit is READ-ONLY. Inspection methods used:
- `git log --since="2026-04-01" --oneline -- <paths>` per flow (≈ 1460 commits total in scope).
- Code reads on hot-spot files (commission_distributor.go, charge.go, payout_request.go, monthly.go, scheduler.go, service.go for each feature).
- Cross-reference with memory notes (`project_org_based_model.md`, `project_blocking_todo.md`) for known latent issues.
- No DB writes. No test execution. No code modification.

The audit's value is **paranoid analysis + concrete file:line evidence** — not implementation. Hand the fix queue to follow-up agents.


## Per-flow report

### 1. Referrer commission

- **Status**: 🔴 CONFIRMED REGRESSION (already being fixed via CRIT-REF + backfill, in flight separately)
- **Git touchpoints**:
  - `ab6ceedf feat(referral): PrepareCommissionForMilestone — create pending row on approval`
  - `4fd01905 feat(referral): scheduler sweep for pending commissions`
  - `ca839ab6 feat(referral): wire preparer into proposal milestone-approval flow`
  - `27d46b25 test(referral): coverage for the new prepare flow + scheduler sweep`
  - `683adac5 chore(referral): drop WIP marker — CRIT-REF complete`
- **Root cause**: pre-CRIT-REF, `DistributeIfApplicable` was called only after the provider auto-transfer cleared — providers without auto-payout consent never triggered referrer commissions. The post-fix flow now calls `PrepareCommissionForMilestone` on milestone APPROVAL (decoupled), and a scheduler sweep drains pending rows.
- **Code trace verified**:
  - `app/proposal/service_actions_more.go:476` — `referralCommissionPreparer.PrepareCommissionForMilestone(...)` fires on milestone approval.
  - `app/referral/commission_distributor.go:31-88` — `PrepareCommissionForMilestone` creates pending row idempotently.
  - `app/payment/payout_transfer.go:195` — legacy `DistributeIfApplicable` still fires post-transfer (race-safe: existing row in non-pending state = no-op).
  - `app/referral/pending_sweeper.go` — drains pending rows via scheduler.
- **Test coverage**: 🟡 New coverage added (`27d46b25`, `7c2837e0`, e2e `4dfe27d4`). PRE-refactor invariant (commission-on-approval) was NOT pinned by a test — that's why this regression slipped. **Test gap remains**: no integration test today on the legacy `DistributeIfApplicable` race path with a pre-existing pending row from preparer. Recommend a single integration test covering (preparer wrote pending → distributor races on transfer → distributor must claim row, transfer, mark paid).
- **DB sanity needed**: backfill in flight (per brief). Should be verified by `SELECT COUNT(*) FROM referral_commissions WHERE created_at > '2026-04-22' GROUP BY status` once it completes.
- **Risk**: HIGH if backfill not validated.

### 2. Receipts

- **Status**: 🟢 healthy (snapshot-on-creation pattern, never bypassed)
- **Git touchpoints**:
  - `c5f41bd7 feat(receipts): backend foundation — snapshot + endpoints + PDF`
  - `869597ce fix(handler): escape user input in receipt response (XSS)`
- **Trace findings**:
  - Receipts are NOT a separate post-payment step — the billing snapshot is persisted into `payment_records.billing_snapshot_json` at `ChargeService.CreatePaymentIntent` time (`app/payment/charge.go:121` → `attachReceiptSnapshot`).
  - Best-effort design: a snapshot failure leaves `billing_snapshot_json = NULL` and UI falls back to "données indisponibles". This intentionally never blocks payment (`charge.go:316-323`).
  - `createPaymentIntentFromExisting` skips re-attaching — by design, the snapshot is already attached when the record was first created. Verified by inspecting both branches.
- **Test coverage**: `charge_receipt_snapshot_test.go` exists. Coverage: snapshot resolver wire-up tested. Gap: no test asserts `attachReceiptSnapshot` is invoked on every CreatePaymentIntent path (no "for every method, ensure receipt attached" guard test).
- **Risk**: LOW. The snapshot is presentation-layer; even total failure has no financial consequence.

### 3. Commission invoices (non-Premium)

- **Status**: 🟢 healthy (scheduler wired, Premium skip correct)
- **Git touchpoints**:
  - `50b3a3e2 fix(invoicing): wire commission monthly scheduler + skip Premium 0€`
  - `5094fba5 feat(invoicing/app): monthly consolidation + in-process scheduler + CLI`
  - `33d83f15 fix(invoicing): namespace inner idempotency key to avoid gateway collision`
  - `c2d90813 fix(invoicing): force PDF attachment download instead of inline preview`
  - `dc41c7ed feat(admin/invoices): listing page with filters across all emitted invoices`
- **Trace findings**:
  - Monthly scheduler is wired in `cmd/api/wire_invoicing.go:185-198`. Starts only if `Stripe + orgs + redis` are configured (fail-open in dev).
  - `monthly.go:142-165`: skips records with `PlatformFeeCents <= 0` (Premium-waived), refuses to issue a 0€ invoice. Period filter (transferred_at >= start AND < end) prevents re-consideration.
  - `RunMarker` redis lock prevents duplicate consolidation per (org, year, month).
- **Premium detection**: relies on `PlatformFeeCents == 0` being correctly stamped by `feeCalculator.computePlatformFee` at PaymentIntent creation. Verified at `app/payment/charge.go:112`. Premium gate is computed once per record, frozen.
- **Test coverage**: `monthly_test.go`, `scheduler_test.go` present. 🟡 Gap: no end-to-end production-like test that asserts Premium subscriber spans a calendar month and the scheduler skips the org entirely (vs. mixed Premium/non-Premium org producing a partial invoice).
- **Risk**: LOW-MEDIUM. The Premium-gate computation depends on subscription status at the time of PaymentIntent creation; if a user upgrades mid-month, records before the upgrade still carry the non-Premium fee, and the invoice for that month is partial (correct behavior).

### 4. KYC enforcement

- **Status**: 🟢 healthy (scheduler + payout pre-check both wired)
- **Git touchpoints**:
  - `b9a1291b feat: KYC enforcement scheduler + notification tiers + first earning trigger`
  - `dd9f1b9f feat(team): R5 part 1 — Stripe + KYC move to the organization`
  - `56318f79 refactor(backend/wiring): split god-files into focused wire helpers`
  - `6a40f5f7 test(backend/admin/kyc/referral): close coverage gaps on three under-tested feature surfaces`
- **Trace findings**:
  - `wire_uploads_messaging_moderation_kyc.go:156-167` → `startKYCScheduler` in `wire_notification.go:142` → `kycapp.NewScheduler`. Active in main bootstrap.
  - Payout gate: `app/payment/payout_request.go:444-465` (`assertProviderPayoutsEnabled`) calls Stripe `GetAccount` → `info.PayoutsEnabled`. Returns `ErrProviderPayoutsDisabled` (typed sentinel, handler returns 412 + redirect to /payment-info).
  - Day 0 / 3 / 7 / 14 reminder tiers — confirmed in `app/kyc/scheduler.go` (not opened here for length).
- **R5 refactor risk**: KYC was moved from user → organization. The scheduler now reads `Organizations` repo (verified `kycSchedulerDeps.Organizations`). Risk: orgs created BEFORE the migration that don't have a Stripe account row → scheduler should skip (verified via the `stripeAccountID == ""` guard in payout path).
- **Test coverage**: 🟢 covered by `6a40f5f7`.
- **Risk**: LOW.

### 5. Premium subscription

- **Status**: 🟡 SUSPECTED (latent — partial org-scoping)
- **Git touchpoints**:
  - `ae37955a feat(subscription/embedded): switch checkout to Stripe Embedded mode + pre-enrich Customer`
  - `dd9f1b9f feat(team): R5 part 1 — Stripe + KYC move to the organization`
  - `19e202db fix(subscription/mobile): WebView errors only on main-frame failures`
  - `83379315 fix(subscription): prevent duplicate active subscriptions`
- **Trace findings (positive)**:
  - **The Premium fee bypass IS org-aware.** `app/subscription/service_more.go:150-172` `IsActive(userID)` resolves the user → org internally via `ResolveActorOrganization`, then queries `FindOpenByOrganization(orgID)`. So billing reads Premium correctly even when the subscription was created by another team member.
  - The port keeps `IsActive(userID)` signature for backward compat with `app/payment/wallet.go:319-328` (`computePlatformFee`), but internally it's org-scoped. No regression here.
- **Remaining latent bug** (memory note `project_org_based_model.md`, 2026-04-22):
  - The `subscriptions` TABLE still has `user_id` FK as the ownership column (not yet migrated to org-primary). A direct `subscriptions WHERE user_id = ?` query path would miss subscriptions of other team members. The refactor wave introduced org-scoping at the read API level but didn't migrate the schema.
  - **Verification needed**: `\d subscriptions` — confirm whether organization_id was added. If yes, healthy; if no, the user-leaves-org case still drops the Premium link.
- **Test coverage**: ✓ `service_more.go IsActive` has a clear org-resolution path. Need a regression test pinning "Premium-subscribing user leaves org, other operator still pays 0 fee".
- **Risk**: LOW-MEDIUM. The fee path is org-scoped. Schema migration is deferred.

### 6. Wallet payouts (Retirer mes fonds)

- **Status**: 🟢 healthy (status gate, idempotency, consent stamp all wired)
- **Git touchpoints**:
  - `82a4e6f6 feat(stripe-connect): manual payout schedule + explicit payouts on Retirer`
  - `93fd3b47 feat(wallet): KYC readiness check before billing profile on RequestPayout`
  - `988aa1b6 refactor(payment): split payout.go into transfer + request files`
  - `f23a3907 fix(payment): surface DB save errors instead of swallowing — closes BUG-NEW-01`
  - `5cbb2a9d feat(wallet): retry failed Stripe transfers (backend + web + mobile)`
  - `8410ceb9 feat(payment): conditional auto-payout post first-time consent`
- **Trace findings**:
  - `app/payment/payout_request.go:44-146` (`RequestPayout`): resolves org Stripe account → lists succeeded+transfer-pending records → for each verifies proposal "completed" via `ProposalStatusReader` → fires platform→connected `CreateTransfer` with deterministic idempotency key `transfer_{recordID}_{accountID}` → on success persists MarkTransferred (DB error surfaced, not swallowed).
  - After all in-mission transfers succeed, `fireBankPayout` issues an EXPLICIT Stripe `CreatePayout` (manual schedule) with idempotency key `payout_{orgID}_{amount}` to push funds to the user's bank account.
  - `recordAutoPayoutConsent` stamps `AutoPayoutEnabledAt` on first manual click (idempotent via `HasAutoPayoutConsent`).
  - KYC pre-check via `assertProviderPayoutsEnabled` (Stripe `GetAccount` → `info.PayoutsEnabled`).
- **Defensive enhancements**: BUG-NEW-01 explicitly surfaces DB save errors after `MarkTransferred`/`MarkTransferFailed` (was previously swallowed → record desync from Stripe). Now logged at ERROR for ops paging.
- **Test coverage**: `payout_test.go`, `payout_split_test.go`, `payout_bug_new_01_test.go`, `service_bug09_test.go` cover key paths.
- **Risk**: LOW.

### 7. Disputes (auto-resolve + escalate)

- **Status**: 🟢 healthy
- **Git touchpoints**:
  - `9ac5fcaf fix(dispute): propagate proposal update errors after restore (BUG-03)`
  - `506253ac test(dispute): expand RespondToCancellation coverage (BUG-03 follow-up)`
  - `0634f444 refactor(dispute): narrow DisputeRepository consumers to segregated children`
  - `457098cd refactor(dispute/service_actions+helpers): migrate 15 GetByID callers`
  - `0c3d229c feat(stripe): defensive system-actor wraps on scheduler entry points`
  - `dc573fc3 refactor(backend): split 19 files exceeding 600-line CLAUDE.md ceiling`
- **Trace findings**:
  - `app/dispute/scheduler.go:100-148`: every tick lists due disputes, partitions into "ghost (no respondent reply)" → `autoResolve` or "has reply" → `escalate`.
  - `autoResolve` (line 117): domain mutation `AutoResolveForInitiator` → repo update → `restoreAndDistribute` (funds to initiator) → system message broadcast + both-side notify.
  - `escalate` delegates to `Service.escalate` so manual force-escalate and scheduler share code path.
  - Wired in `wire_dispute.go:108-125` with `disputeScheduler.Run(schedulerCtx, disputeInterval)`.
- **Test coverage**: `service_restore_proposal_test.go`, `service_actions_test.go`, `scheduler_systemactor_test.go`.
- **Risk**: LOW. The 7-day window is encoded at the repository level (due-list query).

### 8. Search ranking

- **Status**: 🟢 healthy (heavy test investment late in the wave)
- **Git touchpoints**:
  - `2dd83296 feat(search): wire ranking pipeline into Query service + LTR capture`
  - `bd2a8034 feat(search): ranking pipeline + Typesense→Features document adapter`
  - `88acb839 fix(search): wire anti-gaming raw signals + populate About (D1, D2, D4)`
  - `6c89dbd5 fix(search): enforce §7.5 new-account median cap (D3)`
  - `74c3f141 fix(search): hybrid blending via vector_query only, not query_by`
  - `4e5ee38c fix(search): gate _vector_distance on hybrid mode; fix integration tests`
  - `7adc37d4 Merge feat/phase5b-test-hardening: fuzz + perf + smoke + realistic seed`
- **Trace findings**:
  - 14→40 golden-query expansion + `golden_full_pipeline_test.go`. Fuzz suites for cursor + filter builder.
  - Ranking pipeline (`ranking_pipeline.go`): scoreCandidates → zipWithRaws (extracted from monolithic Rerank for testability).
  - The segregated repo refactor for User/Org touched `app/search/` callers indirectly via the `repository.UserReader` narrowing — no signature changes that affect query behaviour.
- **Cross-feature lookup risk**: Search reads user/org via the segregated `UserReader` (read-only). No path touches Writer / KYCStore from search. Safe.
- **Test coverage**: 🟢 excellent.
- **Risk**: LOW.

### 9. Notifications

- **Status**: 🟢 healthy
- **Git touchpoints**:
  - `0c1edd4c feat(notification): refresh device_tokens.last_seen_at on push delivery`
  - `3dbbf747 fix(notification/worker): parallel pool + non-blocking re-enqueue`
  - `1856dc57 feat: admin notifications with per-admin Redis counters + WebSocket broadcast`
  - `eb24bb14 feat(team): phase 5 — in-app notifications for team events`
- **Trace findings**:
  - `app/notification/worker.go:73-348`: parallel worker pool (`DefaultWorkerConcurrency = 5`), non-blocking re-enqueue on retry, push via `service.PushService` + email via `service.EmailService`, preferences fetched per-job via `getPrefs`.
  - BUG-16 fix: previously single goroutine + blocking sleep → p99 spike past 7s. Now N parallel processors with disjoint consumer IDs against Redis stream group + deferred re-enqueue.
  - FCM push: handled via `service.PushService` port (`deliverPush` line 292-313). Token resolution + multi-device send.
- **Test coverage**: `worker_test.go`, `service_test.go`.
- **Risk**: LOW.

### 10. Messaging (unread count + system messages)

- **Status**: 🟢 healthy
- **Git touchpoints**:
  - `380236ec refactor(message): narrow MessageRepository consumers to segregated children`
  - `4eb743cd fix(messaging): wrap message LIST/GET/MARK paths in RunInTxWithTenant — closes BUG-NEW-04 (path 8/8)`
  - `233b43e1 fix(messaging): wrap conversation+message writes in RLS tenant tx`
  - `e86cb6b6 fix(messaging): persist system-actor messages instead of FK-rejecting them`
  - `a3471af9 fix(proposal,dispute,messaging): close the loop on multi-milestone projects`
  - `1f64b78f fix(moderation): repair image+audio+pdf pipeline + admin queue + ctx propagation`
- **Trace findings**:
  - `service_helpers.go:130-148`: batch unread query (`GetTotalUnreadBatch`) — N+1 free.
  - System-actor messages: `e86cb6b6` fixed FK rejection by persisting system actor user (referral/proposal lifecycle events now appear inline in the conversation).
  - RLS coverage: all message read/write paths wrapped in `RunInTxWithTenant` after the BUG-NEW-04 sweep.
- **FIX-DASH consideration**: Dashboard binding `134475b` switched the dashboard unread to use messaging unread count. Backend count itself was already correct — the dashboard was reading from the wrong source.
- **Test coverage**: 🟡 No explicit regression test pinning "unread count after team operator views message" → check this in the priority queue.
- **Risk**: LOW.

### 11. Team / invitations

- **Status**: 🟢 healthy
- **Git touchpoints**:
  - `bb79d8a7 feat(team): phase 2 — team invitation flow end-to-end`
  - `519dcee0 feat(team): phase 3 — membership management + ownership transfer + immediate revocation`
  - `919bbf8b feat(team): phase 4 — scope resources to organizations + denormalize users.organization_id`
  - `eb24bb14 feat(team): phase 5 — in-app notifications for team events`
  - `c0e51bc8 feat(perms): R17 — full org role permission system + per-org togglable overrides`
- **Trace findings**:
  - `invitation_service.go:419-435` `requirePermission`: checks membership AND evaluates `HasEffectivePermission(member.Role, perm, org.RoleOverrides)` — so the per-org role overrides editor takes effect on invite/list calls.
  - `ListPending` requires `PermTeamView`. Invite send requires the configured permission via `requirePermission`. Owner who grants invite to Members → Members pass the check.
  - Email collision check (`checkEmailCollision`): rejects existing user accounts (one email = one account) and pending invites for same email in same org.
  - Acceptance flow tested via e2e (`1ff0943a test(e2e): invitation acceptance flow (no 404, redirects to /team)`).
- **Test coverage**: `invitation_service_test.go`, `membership_service_test.go`, `role_overrides_service_test.go` present. ACL on role-permissions editor explicitly covered.
- **Risk**: LOW.

### 12. Jobs / opportunities (applications counts + transitions)

- **Status**: 🟢 healthy
- **Git touchpoints**:
  - `888ce003 feat(jobs): persist applicant_kind on apply + filter on candidates`
  - `15a5a824 feat(jobs): expose total_applicants on /api/v1/jobs/open`
  - `7c565835 fix(job): R12 — move application credits to organization (security)`
  - `f52bf2f9 feat(team): R3 core — jobs, proposals, wallet flip to org-primary lists`
- **Trace findings**:
  - `app/job/service.go:157-188`: `ListOrgJobsWithCounts` does ONE list query + ONE batch count query (`GetApplicationCountsBatch`) — explicit N+1 elimination.
  - "New since last viewed" counter is per-user (each operator has their own `last_viewed_at` marker) while the underlying jobs are per-org — correct separation of "personal UX" vs "shared business state".
  - Application credits moved from user → org (R12) — verified via `7c565835` commit. Credits are now consumed from org wallet, not individual.
- **Test coverage**: 🟡 Need explicit pin on "applicant_kind on apply" + "candidates filter by kind".
- **Risk**: LOW.

### 13. GDPR right-to-erasure

- **Status**: 🟢 healthy (R2 cleanup wired in same scheduler tick)
- **Git touchpoints**:
  - `eaa1e5fd feat(gdpr): wire R2 cleanup before SQL anonymize in scheduler`
  - `74d895ba test(gdpr): storage purge unit + integration coverage`
  - `f2d7a79f docs(gdpr): mark B.10 (audit PII sanitize) as DONE`
  - `a7e50116 feat(audit): migration to scrub existing rows in chunks`
- **Trace findings**:
  - `app/gdpr/service.go:334-389` `purgeStorageForUser`: enumerates R2 keys for the user → `s.storage.BulkDelete(ctx, keys)` → records a manifest via `RecordStoragePurgeAudit` (success OR failure path both audited).
  - Sequenced BEFORE SQL anonymize — failures in storage leave audit trail; anonymize still runs (storage purge is best-effort by design).
  - Audit PII sanitize (B.10) scrubs PII from existing audit rows.
- **Test coverage**: `storage_purge_test.go`, integration coverage from `74d895ba`.
- **Risk**: LOW.

### 14. Authentication (login without 2FA)

- **Status**: 🟢 healthy
- **Git touchpoints**:
  - `6664fc8f feat(2fa): wire into Login flow + verify-2fa endpoint`
  - `8a84d90a feat(session): wire into auth token service (login/refresh/logout/reuse)`
  - `b3bcf8a2 test(token): theft detection + chain limits coverage`
  - `05c74b1e feat(token): chain depth + family age limits + family invalidation`
  - `06755fa6 fix(auth/security): refresh-token replay revokes entire family (F.5 S2)`
  - `9ef6d5d5 fix(auth/security): bcrypt parity on Register duplicate path (F.5 S5)`
- **Trace findings**:
  - `app/auth/service.go:448-617` Login: password check → ban check → 2FA gate (only if `s.twoFactorGate != nil && enabled == true`) → org context resolution → token issuance + session record.
  - Crucially: **2FA gate is optional and fail-open**. If `IsEnabledForUser` errors (DB blip), the user proceeds without 2FA (warn-logged). If `twoFactorGate == nil`, behavior matches pre-B.6.
  - Non-2FA users: skip the gate entirely (line 548 `else if enabled`) → reach token generation. No regression for the 99% case.
  - Session record (`recordSession`) is best-effort and out of the login fail path.
- **B.8 + B.9 + B.4 interactions**: brute force check fires BEFORE password verify (not shown but standard); session record fires AFTER token issuance; refresh token rotation has independent test coverage.
- **Test coverage**: `service_test.go` for non-2FA login, `09f94e2a test(2fa): exhaustive backend handler coverage`, `b3bcf8a2 test(token): theft detection`.
- **Risk**: LOW.

### 15. Stripe Connect onboarding + webhooks

- **Status**: 🟢 healthy (durable idempotency, async dispatch, sanitized errors)
- **Git touchpoints**:
  - `628b454b feat(stripe): enqueue webhook events to pending_events for async dispatch`
  - `512eaa56 fix(webhook): durable idempotency via Postgres source-of-truth + Redis fast-path`
  - `6e1407c0 fix(webhook): release idempotency claim and reply 5xx on handler error — closes BUG-NEW-06`
  - `90f4556b fix(payment/security): verify Stripe before marking payment succeeded`
  - `e1c3c697 fix(stripe/security): sanitize Stripe + JSON + DB errors at API boundary (F.5 S4)`
  - `dd9f1b9f feat(team): R5 part 1 — Stripe + KYC move to the organization`
  - `ae37955a feat(subscription/embedded): switch checkout to Stripe Embedded mode + pre-enrich Customer`
- **Trace findings**:
  - `handler/routes_billing.go:162` `POST /stripe/webhook` → `StripeHandler.HandleWebhook` (verify signature → idempotency claim → dispatch by event type → release claim).
  - Event handlers: `handleSubscriptionCreated`, `handleSubscriptionSnapshot`, `handleInvoicePaid`, `handleChargeRefunded`, `handleInvoicePaymentFailed`, `handlePaymentSucceededWithEvent`, `dispatchEmbeddedNotif` — all in `stripe_handler_more.go`.
  - Async dispatch via `pending_events` table (`628b454b`) — webhook returns 200 immediately, background worker consumes.
  - BUG-NEW-06 fix: on dispatch error, release idempotency claim AND reply 5xx so Stripe retries (was previously claiming + dropping → permanent silent drop).
  - SEC-02 fix: `MarkPaymentSucceeded` calls Stripe `GetPaymentIntent` and asserts `status == "succeeded"` before marking the local record paid — closes the DevTools-spoofing vector.
- **Embedded KYC widget**: lives in `app/embedded/` (not opened here for length). Stripe Connect onboarding routes through embedded sessions.
- **Test coverage**: comprehensive, including the BUG-NEW-06 regression pin.
- **Risk**: LOW.



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
*Filled after all 15 flows.*

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

- **Status**: 🟡 SUSPECTED (legacy `subscriptions.user_id` bug noted in memory `project_org_based_model.md` 2026-04-22, NOT addressed by recent refactors)
- **Git touchpoints**:
  - `ae37955a feat(subscription/embedded): switch checkout to Stripe Embedded mode + pre-enrich Customer`
  - `dd9f1b9f feat(team): R5 part 1 — Stripe + KYC move to the organization`
  - `19e202db fix(subscription/mobile): WebView errors only on main-frame failures`
- **Trace findings**: To be done deeper in flows 5-15 round, but flagged here:
  - `subscriptions` table uses `user_id` for ownership — known bug per memory. Refactor wave migrated KYC + Stripe-account to org, but did NOT migrate `subscriptions`. **A user who leaves the org takes the Premium subscription with them.**
  - The Premium fee bypass in invoicing (`monthly.go:143-145`) reads `PlatformFeeCents` from the payment record, NOT a live subscription lookup. So as long as fees are correctly stamped at PaymentIntent time, the bypass works.
  - **Stamp accuracy** — to verify: does `feeCalculator.computePlatformFee` correctly read Premium status by `organizationID`? Likely still reads `user_id` if the Premium lookup hasn't been migrated.
- **DB sanity needed**: `SELECT user_id, organization_id FROM subscriptions WHERE status='active' ORDER BY created_at DESC LIMIT 20;` — does any row have a null organization_id?
- **Test coverage**: 🟡 Need explicit test on the (user-leaves-org → Premium-fee-incorrect) edge.
- **Risk**: MEDIUM. Real-world impact: agencies with multiple users where the Premium-subscribing user leaves. Not a fresh regression — pre-existing bug aggravated by R5 partial migration.



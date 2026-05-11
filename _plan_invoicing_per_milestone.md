# Plan — Invoicing per milestone

## Goal
Emit a `platform_fee` invoice immediately at milestone approval for non-Premium
provider orgs. Backfill all historical milestones. Monthly scheduler becomes a
safety net.

## Legal constraint (user-validated)
The platform-fee invoice covers ONLY the `platform_fee_amount`. Stripe fees are
NOT included — they are not legally refacturable as a separate billable line.

## Decision: trigger pattern
Direct call from `proposal.Service` after `Approve()` succeeds in
`CompleteProposal` and `AutoApproveMilestone`. No event bus exists in this
codebase for milestone lifecycle events — direct synchronous call mirrors the
existing pattern (`prepareReferrerCommission`, `payments.TransferMilestone`).

Failure of invoice emission MUST NOT roll back the approval — the safety-net
monthly scheduler will catch it on its next run.

## Files

### 1. Migration
- `backend/migrations/154_invoice_platform_fee_per_milestone.up.sql` — add
  `milestone_id UUID NULL`, extend `invoice_source_type_check` to include
  `'platform_fee'`, partial unique index on `(milestone_id)` WHERE
  `source_type = 'platform_fee'`.
- `backend/migrations/154_invoice_platform_fee_per_milestone.down.sql` —
  reverse.

### 2. Domain
- `backend/internal/domain/invoicing/value_objects.go` — add
  `SourcePlatformFee SourceType = "platform_fee"`, accept in `IsValid()`.
- `backend/internal/domain/invoicing/invoice.go` — add `MilestoneID *uuid.UUID`
  field on `Invoice` + `(*Invoice).IsPlatformFee() bool`. Extend
  `NewInvoiceInput` to optionally accept `MilestoneID`.

### 3. Port + Adapter
- `backend/internal/port/repository/invoicing.go` — add
  `FindPlatformFeeByMilestoneID(ctx, milestoneID) (*Invoice, error)`.
- `backend/internal/adapter/postgres/invoicing_repository.go` — INSERT now
  includes `milestone_id`; scan in `invoiceColumns` adds `milestone_id`.
- New file `backend/internal/adapter/postgres/invoicing_repository_milestone.go`
  with the `FindPlatformFeeByMilestoneID` method.

### 4. App service
- New file `backend/internal/app/invoicing/per_milestone.go` —
  `IssueFromMilestone(ctx, paymentRecord, providerOrgID, isPremium) (*Invoice, error)`.
  Idempotent. Skips if Premium. NO Stripe fees in amount.
- New file `backend/internal/app/invoicing/per_milestone_test.go` — unit tests.

### 5. Cross-feature port for proposal service
- New port `service.PerMilestoneInvoicer` so proposal app does not depend on
  the invoicing app type. Single method `IssueFromMilestone`.
- Proposal service grows a `perMilestoneInvoicer` field + `SetPerMilestoneInvoicer`
  setter (same pattern as `SetReferralCommissionPreparer`).

### 6. Wire trigger
- `service_actions_more.go::CompleteProposal` — after Approve+Release, fetch
  payment record + premium status, call `s.perMilestoneInvoicer.IssueFromMilestone`.
- `service_scheduler.go::AutoApproveMilestone` — same hook on the auto-approve
  path.
- `cmd/api/wire_invoicing.go` — call `proposalSvc.SetPerMilestoneInvoicer(...)`.

### 7. Monthly safety net
- `app/invoicing/monthly.go::IssueMonthlyConsolidated` — when iterating
  records, skip records that already have a `platform_fee` invoice
  (via `FindPlatformFeeByMilestoneID`). Skip whole consolidation when
  every record is already invoiced.

### 8. Backfill CLI
- `backend/cmd/invoicing-backfill/main.go` — CLI binary with `--since`,
  `--dry-run`. Queries succeeded payment_records, calls IssueFromMilestone
  for each missing invoice. Idempotent.

### 9. Admin trigger endpoint
- `POST /api/v1/admin/invoicing/backfill?since=YYYY-MM-DD` — admin-only,
  runs the backfill in-process and returns counts.

## Test count
- Domain: 2 tests (SourcePlatformFee_IsValid, Invoice.IsPlatformFee).
- App service unit tests: 6 tests (idempotent, premium skip, amount excl
  stripe, VAT, NoStripeFees, integration with safety net).
- Adapter test: 1 test (FindPlatformFeeByMilestoneID).
- Monthly safety net unit test: 1 test (skip already-invoiced).
- Backfill CLI: 1 test (parse flags) + actual local-DB run with paste.
- Total ≥ 11 tests, paste actual stdout.

## Pipeline order
1. _plan_invoicing_per_milestone.md
2. Migration 154 + applied locally
3. Domain
4. Port + adapter
5. App service `IssueFromMilestone` + tests
6. Wire trigger on milestone approval
7. Monthly scheduler safety-net mode + test
8. Backfill CLI + run it locally
9. Admin trigger endpoint
10. Final integration tests

## Out-of-scope (explicitly NOT touching)
- Web, mobile, admin UI.
- Wallet code (Run B's territory).
- LiveKit/video.
- Monthly_commission emission behavior beyond the per-milestone skip.

## Risk acceptance
- Run-D agent has not touched these files — collisions unlikely.
- All non-trivial changes have tests.
- Migration is idempotent.


# Plan — defer platform_fee invoices until transfer.completed

Branch: `fix/invoicing-defer-till-transfer`

## Why

At `milestone.approved` time, the prestataire's Stripe Connect KYC may
still be incomplete. The `billing_profile` is hydrated from KYC via
`synced_from_kyc_at`, so issuing the invoice now produces a legally
incomplete invoice (or, worse, an `ErrNotFound` on the profile).

Defer the invoice emission to the moment the platform→connected-account
Stripe transfer completes — at that point KYC is verified by Stripe AND
the billing_profile has had a chance to be hydrated. This is the
correct legal trigger.

## Scope (ship exactly this — ni plus, ni moins)

1. **Move the trigger** away from `proposal.CompleteProposal` +
   `proposal.AutoApproveMilestone` and ONTO the three payment paths
   that flip `transfer_status='completed'`:
   - `payment.PayoutService.TransferMilestone` (mid-proposal release)
   - `payment.PayoutService.RequestPayout` (wallet "Retirer")
   - `payment.PayoutService.RetryFailedTransfer`

2. **Add a `billing_profile` completeness gate** in
   `invoicing.Service.IssueFromMilestone` — defense-in-depth: when the
   profile is missing universal fields (legal_name, country,
   address_line1, postal_code, city) we log WARN and return `(nil, nil)`
   so the monthly safety-net picks the milestone up on its next run.

3. **Verify the monthly scheduler safety-net** already handles
   "previously skipped, now complete" — re-reading
   `monthly.go::IssueMonthlyConsolidated`, the probe is
   `FindPlatformFeeByMilestoneID` (no row → still eligible) so the
   incomplete-billing skip is automatically re-tried by the next
   monthly run. NO code change needed in `monthly.go`.

4. **Idempotence verification** — the backfill CLI dry-run on the same
   period must show every existing row as `already_invoiced`. Re-runs
   are no-ops thanks to the partial UNIQUE index on
   `invoice(milestone_id) WHERE source_type='platform_fee'`.

## Files

### Domain layer (new)

- `internal/domain/invoicing/billing_profile.go` — `IsComplete()` already
  exists (delegates to `CheckCompleteness`). NO new method needed —
  reuse the existing helper.

### App layer

- `internal/app/invoicing/per_milestone.go` (MODIFY):
  - After billing_profile load, call
    `invoicing.CheckCompleteness(*profile)`. If the universal fields
    (legal_name, country, address_line1, postal_code, city) are
    missing, log WARN + return `(nil, nil)`.
  - Keep the existing "ErrNotFound" failure mode as-is (no profile row
    at all = configuration bug, still propagate).

- `internal/app/payment/payout_transfer.go` (MODIFY):
  - In `TransferMilestone`, AFTER `p.records.Update(record)` succeeds
    (i.e. the row is committed as `transfer_status=completed`),
    BEFORE the referral distributor block, call the per-milestone
    invoicer via a new injected hook on PayoutService.

- `internal/app/payment/payout_request.go` (MODIFY):
  - Same hook firing in `RequestPayout` AFTER `Update` succeeds for
    each transferred record (inside the per-record loop).
  - Same hook firing in `RetryFailedTransfer` AFTER successful Update.

- `internal/app/payment/payout.go` (MODIFY):
  - Add `perMilestoneInvoicer portservice.PerMilestoneInvoicer` field
    + `SetPerMilestoneInvoicer` setter (mirrors the existing
    `SetReferralDistributor` pattern).

- `internal/app/payment/service.go` (MODIFY):
  - Add `SetPerMilestoneInvoicer` facade that forwards to
    `payout.SetPerMilestoneInvoicer`.

- `internal/app/proposal/service_actions_more.go` (MODIFY):
  - Remove `s.emitPerMilestoneInvoice(ctx, current.ID)` call.

- `internal/app/proposal/service_scheduler.go` (MODIFY):
  - Remove `s.emitPerMilestoneInvoice(ctx, m.ID)` call.

- `internal/app/proposal/per_milestone_invoice.go` (DELETE the file):
  - The `emitPerMilestoneInvoice` helper is no longer called from
    anywhere. Removing the file keeps the proposal package smaller.
  - Also remove the `perMilestoneInvoicer` field + setter from
    `service.go` since the proposal package no longer needs it.

- `cmd/api/wire_invoicing.go` (MODIFY):
  - Wire the adapter into `paymentSvc.SetPerMilestoneInvoicer(adapter)`
    instead of (or in addition to) `proposalSvc.SetPerMilestoneInvoicer`.
  - Since proposal no longer needs the setter, remove that call too.

### Tests (≥ 90% coverage on new code)

#### New tests

- `internal/app/invoicing/per_milestone_test.go` (ADD):
  - `TestIssueFromMilestone_SkipsIncompleteBillingProfile_LegalName`
  - `TestIssueFromMilestone_SkipsIncompleteBillingProfile_Country`
  - `TestIssueFromMilestone_SkipsIncompleteBillingProfile_AddressLine1`
  - `TestIssueFromMilestone_SkipsIncompleteBillingProfile_PostalCode`
  - `TestIssueFromMilestone_SkipsIncompleteBillingProfile_City`
  - Each asserts: `(nil, nil)` returned + no DB write
    (`repo.persistedInvoices` empty).

- `internal/app/payment/payout_invoice_test.go` (NEW):
  - `TestPayoutService_TransferMilestone_FiresInvoice_OnSuccess`
  - `TestPayoutService_TransferMilestone_DoesNotFireInvoice_OnStripeFail`
  - `TestPayoutService_TransferMilestone_DoesNotFireInvoice_AlreadyDone`
  - `TestPayoutService_TransferMilestone_DoesNotFireInvoice_NoStripeAccount`
  - `TestPayoutService_TransferMilestone_InvoicerErrorIsSwallowed`
  - `TestPayoutService_RequestPayout_FiresInvoicePerTransferredRecord`
  - `TestPayoutService_RetryFailedTransfer_FiresInvoice_OnSuccess`
  - Uses an in-test stub satisfying `portservice.PerMilestoneInvoicer`
    that records call count + milestone id.

#### Regression tests (proposal side — verify old wiring is gone)

- `internal/app/proposal/service_actions_more_test.go` (or
  `service_actions_test.go` extension):
  - `TestCompleteProposal_DoesNotFireInvoice` — stub
    `PerMilestoneInvoicer`, run CompleteProposal, assert 0 calls.

- `internal/app/proposal/service_scheduler_test.go` (NEW or extend):
  - `TestAutoApproveMilestone_DoesNotFireInvoice` — same.

Wait — but the proposal service NO LONGER has a `perMilestoneInvoicer`
field after the refactor. The regression test can simply assert that
`CompleteProposal` / `AutoApproveMilestone` are no longer carrying any
invoicer hook. The cleanest test is: instantiate proposal Service
*without* an invoicer (it doesn't have the setter anymore), run the
happy path, assert no panic + no compile-time call to the invoicer.
A `grep` test isn't useful — the type-level removal of the field IS
the regression guarantee. We'll just have unit tests that exercise
both flows and assert behavior, no invoicer involved.

#### Monthly scheduler (extend existing test)

- `internal/app/invoicing/monthly_test.go` — add
  `TestIssueMonthlyConsolidated_PicksUpPreviouslySkippedDueToIncompleteBilling`:
  - Pre-condition: a released milestone, no platform_fee invoice yet
    (because the per-milestone path skipped it due to billing gap).
  - Action: org's billing_profile is now complete; run the monthly.
  - Assert: monthly consolidates the milestone (one item, fee > 0).

  In practice, `monthly.go` already does this — the test just locks in
  the behavior so a future regression doesn't break the safety net.

#### Backfill idempotence

- The existing `cmd/invoicing-backfill/main_test.go` already covers
  idempotence (since the FindPlatformFeeByMilestoneID probe is in
  the service path). I'll run `-dry-run -since=2026-01-01` against
  the dev DB and confirm output shape.

## Test count (estimated)

- Invoicing per-milestone: 5 new gate tests
- Payment payout-invoicer hook: 7 new tests
- Monthly safety-net: 1 new (or existing test verified)
- Total new: ~13 unit tests

## Trigger site decision

The brief asked: "investigate which function flips `transfer_status='completed'`".

The flip happens in `payment_record.go::MarkTransferred()` (domain
method), called from THREE app sites:
1. `payout_transfer.go::TransferMilestone` (line ~181)
2. `payout_request.go::RequestPayout` (line ~118)
3. `payout_request.go::RetryFailedTransfer` (line ~328)

ALL THREE must fire the invoicer post-Update, otherwise providers who
get paid via "Retirer" or retry path don't get their invoice. The
brief mentioned `transfer.created` webhook — that's not how the
codebase works currently (transfers are created synchronously inside
`p.stripe.CreateTransfer(ctx, ...)`); the database flip happens
locally right after the Stripe call returns success and we call
`record.MarkTransferred(transferID)` + `p.records.Update(ctx, record)`.

So the correct hook point is: AFTER the successful
`p.records.Update(ctx, record)` call in each of the three flows.

## Commits (sequential)

1. `docs(invoicing-defer): plan` — this file
2. `feat(invoicing): skip platform_fee when billing profile incomplete` — gate + tests
3. `feat(invoicing-defer): wire PerMilestoneInvoicer on transfer.completed` — payout sites + tests
4. `refactor(invoicing-defer): remove premature trigger from proposal flows` — proposal cleanup + regression
5. `chore(wire): retarget invoicer adapter from proposal to payment` — main.go wiring

If commit count grows too large, fold 2+3+4+5 into a single
`fix(invoicing-defer): defer platform_fee until transfer.completed`.

## Non-goals (do NOT touch)

- LiveKit / video call code.
- Web, mobile, admin apps.
- Migrations.
- Referral domain logic.
- Wallet code.
- Monthly scheduler logic (only verify it still works).

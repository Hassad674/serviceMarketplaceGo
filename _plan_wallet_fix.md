# Plan — fix/wallet-ui-crash-and-aggregation

Branch: `fix/wallet-ui-crash-and-aggregation` (created in worktree, branched off
main HEAD `192f9d58`).

Scope: 3 critical wallet bugs + 1 regression pin. NO feature creep.

## Bug 1 (UI crash) — `wallet-unified-page.tsx:97`

Symptom: clicking "Retirer X €" on `/wallet` crashes the Next.js page with
`TypeError: Cannot read properties of undefined (reading 'length')` because
`onSuccess` destructures `result.errors.length` without defensive access.

Root cause: `WithdrawResult.errors` is typed as `WithdrawLegError[]` (always
present), but the backend can omit `errors` from the JSON when empty (`omitempty`
on the `withdrawResponse.Errors` field — see `wallet_withdraw.go:42`). The
TS type lied — at runtime `errors` was `undefined`.

### Fix (`web/src/features/wallet/components/wallet-unified-page.tsx`)

Defensive consumption + total fallback path:
- `const errors = result?.errors ?? []`
- `const drained = result?.drained_cents ?? 0`
- `const missions = result?.missions_cents ?? 0`
- `const commissions = result?.commissions_cents ?? 0`
- 200 success, drained > 0, no errors → success toast.
- 207, drained > 0, errors[].length > 0 → partial-result modal.
- 200 with zero drained AND no errors → defensive "retrait en cours" toast.
- 422 / 403 / any other ApiError → existing branches unchanged.
- Any non-ApiError → defensive `toast.error(t("toast.unknown"))`.

Also widen the `WithdrawResult.errors` TS type to `errors?: WithdrawLegError[]`
in `web/src/features/wallet/api/wallet-api.ts` so the contract matches the
backend's `omitempty`.

### Tests (`__tests__/wallet-unified-page.test.tsx`)

Add 2 new tests + verify existing 5 still pass:
- `TestWalletUnifiedPage_HandlesEmptyErrorsField` — `{drained_cents:100, ...}` (no `errors` key) → no crash, success toast, no result modal.
- `TestWalletUnifiedPage_Handles200WithEmptyResult` — `{}` → no crash, no result modal, defensive toast.

i18n: add `walletUnified.toast.unknown` to fr.json + en.json (tutoiement FR).

## Bug 2 (double-count) — `wallet_summary.go:commissionLeg`

Symptom: 1298€ commission stuck in `pending_kyc` appears as 1298€ in BOTH
`escrowed_cents` AND `available_cents` cards.

Root cause: `commissionLeg` (current) is correct in PRINCIPLE — the records
loop adds to `AvailableCents` (pending_kyc/failed), the projections loop adds
to `EscrowedCents` (Escrowed/Pending). The bug is that for a milestone in
"approved without commission row yet" state, `dispatchMilestone` emits a
**`ProjectionPending` source=projection** (safety net). When the commission row
THEN gets persisted with `pending_kyc` status, both the projection AND the row
exist for the SAME milestone → double-counted.

Decision: **projection is the canonical source of truth.** The brief says so,
and the algorithm in `dispatchMilestone` already handles all states:
- pending_funding / cancelled / refunded → SKIP
- funded/submitted/disputed → projection (escrowed)
- approved + row → from-row status (paid / pending / failed)
- approved + no row → projection (pending, safety net)
- released same as approved
- ENDED attribution → SKIP active-escrow (no commission accrues)

**Algorithm:** derive `commissionLeg` from `view.projections` ONLY, NOT from
`view.records`. The projection has `ProjectionPaid`, `ProjectionPending`,
`ProjectionEscrowed`, `ProjectionFailed` — all four are needed.

Audit: verify projection covers paid+transmitted (yes — `fromCommissionRow`
maps `referral.CommissionPaid` → `ProjectionPaid`).

### Fix

Rewrite `commissionLeg(view commissionSideView)`:
- iterate `view.projections` ONLY.
- Status → bucket mapping:
  - `ProjectionPaid` → TransmittedCents
  - `ProjectionEscrowed` → EscrowedCents
  - `ProjectionPending` → AvailableCents (the apporteur can retry)
  - `ProjectionFailed` → AvailableCents (retire-eligible — UI shows Retirer)
- `TotalCents = Available + Escrowed + Transmitted`

Keep `view.records` for the timeline (rows still feed
`recent_transactions` via `commissionTransaction`). Records list is the
authoritative history; projections own the aggregates.

Update `pickCurrency` to also fall back gracefully (already does so).

### Tests (`wallet_summary_test.go`)

Add `TestWalletSummary_NoDoubleCount` (table-driven):
- Both projection (pending) AND DB record (pending_kyc) for same milestone →
  counted ONCE (in AvailableCents), not twice.
- Projection (escrowed) only → EscrowedCents.
- Record (paid) + projection (paid, source=row) → TransmittedCents (once).
- Record (failed) + projection (failed, source=row) → AvailableCents (once).

Update `TestSummary_CommissionsOnly` — its expectations encode the OLD
algorithm; rewrite to match the new projection-only canonical path.

Update `TestSummary_RowSourcedProjection_NotDoubleCounted` — the comment about
"row counterpart counted on records side" becomes obsolete; the projection IS
the source. Adjust the assertion.

## Bug 3 (mission_title empty)

Symptom: every `recent_transactions[i].mission_title` is empty → UI falls back
to "Sans titre".

Root cause: `missionTransaction(r WalletRecord)` and
`commissionTransaction(r ReferralCommissionRecord)` never populate
`MissionTitle`. The handler has `h.proposalSvc` wired (used in `GetWallet` for
`MissionStatus`) but doesn't use it for `recent_transactions`.

### Fix (`wallet_summary.go`)

Add a single batch resolver: collect all unique `proposal_id`s across mission
records + commission records → one `GetProposalByID` lookup per unique id
(N proposals, N queries — but N is small bounded by the timeline page).

Existing `proposalSvc.GetProposalByID(ctx, id)` returns the proposal entity
which has `Title`. Use it. Wire into `buildTransactionTimeline` so each
emitted `summaryTransaction` carries `MissionTitle`.

Edge case: when `proposalSvc` is nil (test setup, degraded mode), titles stay
empty. Graceful — already covered by the FR `"Sans titre"` fallback.

Edge case: when `GetProposalByID` errors (RLS denies access, e.g. on a
commission for a proposal the apporteur doesn't own), title stays empty.

### Tests

`TestWalletSummary_RecentTransactionsHaveTitle`:
- Seed 2 mission records with proposals "Mission Alpha" and "Mission Beta".
- Stub `proposalSvc` (fake) returning the right title per id.
- Verify `recent_transactions[0].mission_title == "Mission Alpha"` etc.

Since `proposalSvc` is `*proposalapp.Service` (concrete type, not interface),
we'll introduce a narrow `proposalTitleResolver` interface on the handler
(satisfied by `*proposalapp.Service`) so tests can inject a fake without
spinning up the whole proposal service. Builder pattern matches
`WithCommissionRecorder` etc.

## Regression test — Bug 4

`TestWalletWithdraw_RefusesAmountAboveAvailable`:
- Seed wallet with 100 cents available + 200 cents escrowed.
- POST `/wallet/withdraw` with `amount_cents: 300`.
- Already implemented: backend uses `RequestPayout` + commission retry — it
  ONLY drains what's authoritatively eligible regardless of request. So:
  - assert response.drained_cents <= available (100, in this case).
  - assert state in payment_records unchanged (still escrowed = 200).

Don't change the implementation — the test pins existing behavior.

## File-by-file deltas

| File | Change | LOC |
|------|--------|-----|
| `web/.../wallet-api.ts` | `errors?: ...` widen | 1 |
| `web/.../wallet-unified-page.tsx` | defensive `result?.errors ?? []` | ~15 |
| `web/.../__tests__/wallet-unified-page.test.tsx` | +2 tests | ~50 |
| `web/messages/fr.json` + `en.json` | +`walletUnified.toast.unknown` | 2 |
| `backend/.../wallet_summary.go` | rewrite `commissionLeg`, add title resolver | ~40 |
| `backend/.../wallet_handler.go` | add `proposalTitleResolver` port + builder | ~15 |
| `backend/.../wallet_summary_test.go` | +3 new tests, adjust 2 existing | ~150 |
| `backend/.../wallet_withdraw_test.go` | +1 regression test | ~50 |

## Test count

- 2 new web vitest tests (+5 existing pass).
- 3 new backend tests + 1 regression = 4 new go tests.

## Commits (target)

1. `docs(wallet-fix): plan` — this file.
2. `fix(wallet): defensive withdraw result handling — UI no longer crashes on empty errors`
3. `fix(wallet-summary): projection is canonical source — no more double-counted commissions`
4. `feat(wallet-summary): join proposal.title onto recent_transactions`
5. `test(wallet-withdraw): regression — refuses amount above available`

Each commit ≤ 10 tool uses for the first; later commits may exceed since they
include the test stdout paste.

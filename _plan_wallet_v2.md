# Wallet escrow vs available — Fix #2 plan

## Context

Previous round fixed the **commission-side** double-count (`wallet_summary_test.go::TestSummary_NoDoubleCount`). The user's screenshot still shows the **mission-side** `EscrowAmount == AvailableAmount` (31207€ in BOTH buckets, plus 3210€ transmitted = 65624€).

## Root causes

### #1 Mission side `AvailableAmount = EscrowAmount`
`backend/internal/app/payment/wallet.go:147` literally assigns:
```go
wallet.AvailableAmount = wallet.EscrowAmount
```
And the loop above only writes to `EscrowAmount` for `(payment_status='succeeded' AND transfer_status='pending')` rows — never distinguishing whether the milestone has been approved by the client.

The legacy `GetWallet` handler (line 296+) re-computes this client-side from `MissionStatus == "completed"`. But **`/wallet/summary`** uses `missionLeg(w *paymentapp.WalletOverview)` directly, which trusts the (broken) `EscrowAmount` / `AvailableAmount` fields straight from the service.

### #2 Apporteur commission escrow invisible
`WalletHandler` exposes `WithCommissionProjector` / `WithCommissionRecorder` setters, but **they are NEVER called in `bootstrap.go` or `bootstrap_billing.go`**. The only setter wired in production is `WithCommissionRetrier`. So when `/wallet/summary` calls `loadCommissionSide`, both `commissionRecorder` and `commissionProjector` are nil → empty commission breakdown.

## Fixes

### Fix #1: split mission escrow vs available by milestone status

Milestone state machine (per `domain/milestone/status.go`):
- Active escrow (not yet retire-eligible): `funded`, `submitted`, `disputed`
- Retire-eligible (client approved): `approved`
- Terminal: `released` (already transferred — counted as `TransferredAmount`), `cancelled`, `refunded`, `pending_funding`

Bucket dispatch (single source of truth):
| transfer_status | payment_status | milestone.status | bucket |
|---|---|---|---|
| completed | * | * | TransferredAmount |
| pending | succeeded | funded, submitted, disputed | EscrowAmount |
| pending | succeeded | approved | AvailableAmount |
| pending | succeeded | released | (skip — should not exist; defensive: TransferredAmount via transfer_status) |
| pending | succeeded | pending_funding, cancelled, refunded | skip (defensive — succeeded should not be in this state) |
| * | pending/failed/refunded | * | skip |

**Implementation**:
- Add narrow port `MilestoneStatusReader` (or reuse `MilestoneStatusByIDs` shape) in `backend/internal/port/repository/payment_record_repository.go` (already loosely related) OR a new dedicated narrow port in `backend/internal/app/payment/wallet.go` (preferred — adapter-local pattern). The adapter satisfies via existing `MilestoneRepository.ListByProposals`-style batch.
- Decision: Add a NEW narrow port `MilestoneStatusByIDs` in the wallet.go file (local to the consumer, like `MilestonesByProposalLister` is in referral). The adapter wires from the existing `MilestoneRepository`.
- Modify `GetWalletOverview` to:
  1. First pass: collect `milestoneIDs` from records
  2. Single batch fetch milestone status (one SQL query → no N+1)
  3. Second pass: dispatch each record to TransferredAmount / EscrowAmount / AvailableAmount based on milestone status
- Remove `wallet.AvailableAmount = wallet.EscrowAmount` (line 147)

Update `walletStubRecords` test usage: existing test `TestWalletService_GetWalletOverview_AggregatesEscrowAndTransferred` asserts `EscrowAmount == AvailableAmount`. **Need to update** this assertion + add explicit funded/approved tests.

### Fix #2: wire CommissionProjector + CommissionRecorder in production

In `bootstrap.go` line 537 area, where `WithCommissionRetrier` is already wired:

```go
if walletHandler != nil && referralSvc != nil {
    walletHandler = walletHandler.WithCommissionRetrier(referralSvc).
        WithCommissionProjector(referralSvc).
        WithCommissionRecorder(referralSvc)
}
```

`referralSvc` satisfies both `commissionProjector` (via `ProjectedCommissions`) and `commissionRecorder` (via `RecentCommissions` which is already on the service for the wallet reader path).

This makes the apporteur's projected escrowed commissions visible on `/wallet/summary`.

## File list

### Modified
1. `backend/internal/app/payment/wallet.go` — split escrow/available logic + new port + setter
2. `backend/internal/app/payment/service.go` — new setter `SetMilestoneStatusReader` + thread through ServiceDeps
3. `backend/cmd/api/wire_payment.go` — wire milestone status reader from existing MilestoneRepository
4. `backend/cmd/api/wire_referral.go` — pass MilestoneRepository deps (already has it)
5. `backend/cmd/api/bootstrap.go` (or bootstrap_billing) — add `WithCommissionProjector(referralSvc).WithCommissionRecorder(referralSvc)`
6. `backend/internal/app/payment/wallet_test.go` — update existing tests + add escrow/available split tests

### New
7. `backend/internal/app/payment/wallet_escrow_split_test.go` — focused matrix test for the new logic

## Tests planned (≥ 90% coverage)

### Backend
1. `TestWalletList_EscrowVsAvailable_Split` — table-driven matrix (8+ cases): (transfer_status, payment_status, milestone.status) × expected bucket
2. `TestWalletList_BatchedMilestoneFetch` — seed N=10 records with mixed milestones, verify exactly ONE batch query (no N+1)
3. `TestWalletService_GetWalletOverview_NoMilestoneReader_DegradesToEscrowOnly` — when reader is nil, fallback to old behaviour (escrow only, available=0) — graceful degradation contract
4. `TestWalletService_GetWalletOverview_MilestoneReaderError_DegradesToEscrow` — soft-fail on batch error
5. Update `TestWalletService_GetWalletOverview_AggregatesEscrowAndTransferred` — explicit milestone status + assertion that available != escrow
6. `TestSummary_MissionSide_EscrowAvailable_NoDoubleCount` (handler-level) — wire mission overview with split escrow/available, verify breakdown sums correctly without double-count

### Wiring regression
7. `TestBootstrap_WalletHandler_HasCommissionProjector` — verify the new wiring (could be a thin smoke test via the bootstrap function — defer if too heavy, replace with manual code inspection commit-time assertion)

### Regression (existing tests still pass)
- `TestSummary_NoDoubleCount` (commission side) — untouched
- `TestSummary_RecordsDoNotDoubleCountProjections` — untouched
- `TestWalletWithdraw_RefusesAmountAboveAvailable` — untouched

## Commits
1. `_plan_wallet_v2.md` (this file)
2. Add MilestoneStatusReader port + wallet.go split logic + service.go setter
3. Tests for the new escrow/available split
4. Wire MilestoneStatusReader in wire_payment.go + cmd/api/bootstrap.go (commission projector/recorder)

## Status decision matrix (final)
| Record state | Milestone state | Bucket |
|---|---|---|
| TransferStatus=completed | * | TransferredAmount |
| Status=succeeded, TransferStatus=pending | funded/submitted/disputed | EscrowAmount |
| Status=succeeded, TransferStatus=pending | approved | AvailableAmount |
| Status=succeeded, TransferStatus=pending | released | TransferredAmount (defensive — should never happen; transfer_status=pending with milestone=released is data corruption) |
| Status=succeeded, TransferStatus=pending | pending_funding/cancelled/refunded | SKIP (data corruption — log warn) |
| Any other | * | SKIP |
| Milestone lookup MISSING | — | EscrowAmount (conservative — don't show as available if we cannot prove approval) |

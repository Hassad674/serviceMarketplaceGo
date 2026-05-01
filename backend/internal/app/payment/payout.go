package payment

import (
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// PayoutService owns every state transition that moves money out of
// platform escrow: per-milestone transfers, proposal-wide transfers,
// dispute partial transfers, refunds, manual payouts, retries, and the
// post-Premium fee waiver on already-funded records.
//
// SRP rationale: every method here mutates the transfer-side of a
// payment record (or the org's auto-payout consent flag). PI lifecycle
// stays on ChargeService; reads stay on WalletService.
//
// Phase 3.1 (2026-05-01) split the implementation across three files
// to keep each one well under the CLAUDE.md 600-line ceiling:
//
//   - payout.go (this file) — struct, dependencies, constructor, and
//     the post-construction setters for optional collaborators.
//
//   - payout_transfer.go — the methods that release escrow money
//     directly to providers / clients: TransferToProvider,
//     TransferMilestone, TransferPartialToProvider, RefundToClient,
//     CanProviderReceivePayouts, HasAutoPayoutConsent,
//     WaivePlatformFeeOnActiveRecords.
//
//   - payout_request.go — the manual-payout entry points:
//     RequestPayout (the wallet "Retirer" button) and
//     RetryFailedTransfer, plus their shared helpers
//     (fireBankPayout, pickPayoutCurrency, recordAutoPayoutConsent,
//     loadRetryRecord, assertRetryAllowed, resolveProviderAccount,
//     assertProviderPayoutsEnabled, maybeStampRetryConsent).
//
// Dependencies:
//   - records: every read + write on payment_records
//   - orgs:    Stripe account / KYC / consent reads + writes
//   - stripe:  CreateTransfer / GetAccount / CreateRefund / CreatePayout
//   - referralDistributor: optional fire-and-forget hook on per-milestone
//     transfer success (drives the apporteur commission split)
//   - proposalStatuses: optional gate on RequestPayout / RetryFailedTransfer
//     so escrow funds never leave the platform before the mission is
//     marked completed (prevents the wallet "Retirer" side-channel bug)
type PayoutService struct {
	records repository.PaymentRecordRepository
	orgs    repository.OrganizationRepository
	stripe  portservice.StripeService

	// referralDistributor is the apporteur commission hook fired after
	// a successful per-milestone transfer. Nil when the referral feature
	// is not active — every guard in the body checks for nil before
	// invoking.
	referralDistributor portservice.ReferralCommissionDistributor

	// proposalStatuses gates payout transfers on mission completion.
	// Wired post-construction because payment is built before proposal
	// in main.go (proposal depends on payment's PaymentProcessor). When
	// nil, RequestPayout logs a warning and falls back to the legacy
	// behaviour so the payment feature stays bootable without proposal.
	proposalStatuses portservice.ProposalStatusReader
}

// PayoutServiceDeps groups every dependency NewPayoutService needs.
type PayoutServiceDeps struct {
	Records       repository.PaymentRecordRepository
	Organizations repository.OrganizationRepository
	Stripe        portservice.StripeService
}

// NewPayoutService wires the payout / transfer sub-service. Optional
// dependencies (referral distributor, proposal status reader) are wired
// post-construction via setters.
func NewPayoutService(deps PayoutServiceDeps) *PayoutService {
	return &PayoutService{
		records: deps.Records,
		orgs:    deps.Organizations,
		stripe:  deps.Stripe,
	}
}

// SetReferralDistributor plugs the referral commission distributor in
// post-construction. Safe to call at app startup after both services
// exist. Passing nil disables the hook.
func (p *PayoutService) SetReferralDistributor(d portservice.ReferralCommissionDistributor) {
	p.referralDistributor = d
}

// SetProposalStatusReader plugs the proposal status lookup used by
// RequestPayout to keep escrow funds from being transferred before the
// mission is marked completed. Setter pattern because the proposal
// service is constructed AFTER payment in main.go (proposal depends on
// payment's PaymentProcessor). Passing nil leaves RequestPayout in a
// degraded mode that logs a warning and falls back to the pre-fix
// behaviour rather than erroring out — the feature must keep working
// in unusual wirings (tests, migrations, one-off binaries).
func (p *PayoutService) SetProposalStatusReader(r portservice.ProposalStatusReader) {
	p.proposalStatuses = r
}

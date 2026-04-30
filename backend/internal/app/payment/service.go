// Package payment handles post-KYC payment flow: charging clients,
// transferring funds to connected accounts, tracking payment records,
// computing wallet balances, and issuing payouts.
//
// KYC onboarding itself lives in internal/app/embedded (Stripe Connect
// Embedded Components). Since phase R5 the Stripe Connect account is
// owned by the ORGANIZATION, not the user — the payment service reads
// and writes it through OrganizationRepository.
//
// ─── SOLID decomposition (Phase 3, 2026-04-30) ─────────────────────────
//
// The legacy 1171-line service_stripe.go god-service has been split
// along Single Responsibility lines into three focused sub-services:
//
//   - WalletService (wallet.go)  — read paths: GetWalletOverview,
//     GetPaymentRecord, PreviewFee, computePlatformFee. Owns the
//     subscription reader (Premium fee waiver) and the referral wallet
//     reader (apporteur commission rendering).
//
//   - ChargeService (charge.go) — PaymentIntent lifecycle:
//     CreatePaymentIntent, MarkPaymentSucceeded, HandlePaymentSucceeded,
//     VerifyWebhook. Owns nothing payout-side; delegates fee calc to
//     WalletService.
//
//   - PayoutService (payout.go) — every state transition that moves
//     money out of platform escrow: TransferToProvider, TransferMilestone,
//     TransferPartialToProvider, RefundToClient, RequestPayout,
//     RetryFailedTransfer, CanProviderReceivePayouts, HasAutoPayoutConsent,
//     WaivePlatformFeeOnActiveRecords. Owns the proposal-status reader
//     and the referral commission distributor.
//
// Service is the thin composition facade that holds all three and
// delegates each public method to the right sub-service. Existing call
// sites compile unchanged because every method signature is preserved.
package payment

import (
	"context"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// Service orchestrates payment intents, transfers, payouts, and the
// wallet overview. Thread-safe; dependencies are injected via
// ServiceDeps. Public methods delegate to the appropriate sub-service
// (wallet / charge / payout). Tests and existing call sites that
// instantiate Service via NewService keep working unchanged.
type Service struct {
	// Sub-services — each owns a focused slice of the legacy
	// god-service surface. See package doc for the SRP split.
	wallet *WalletService
	charge *ChargeService
	payout *PayoutService

	// Optional sender used by upstream features. Nil when the
	// notification feature is not wired. Kept on the parent service
	// (rather than a sub-service) because it is currently unused by
	// any of the methods below — preserved for backward compatibility
	// with main.go wiring.
	notifications service.NotificationSender
	frontendURL   string

	// referralClawback is wired on the parent so any future flow
	// (refund, dispute) can reach it via the parent service, mirroring
	// the legacy field layout. Unused today by the decomposed
	// sub-services — preserved to keep the SetReferralClawback
	// signature stable.
	referralClawback service.ReferralClawback
}

// ServiceDeps groups all dependencies for the payment service.
type ServiceDeps struct {
	Records       repository.PaymentRecordRepository
	Users         repository.UserRepository
	Organizations repository.OrganizationRepository
	Stripe        service.StripeService
	Notifications service.NotificationSender
	FrontendURL   string
}

// NewService wires the payment service. All fields are optional except
// Records, Users and Organizations — the charge / transfer methods
// fail-fast when Stripe is not configured.
//
// Internally, NewService instantiates the three sub-services and wires
// them so the charge service can delegate platform-fee computation to
// the wallet service (single source of truth for the fee schedule —
// PreviewFee and CreatePaymentIntent must always agree).
func NewService(deps ServiceDeps) *Service {
	wallet := NewWalletService(WalletServiceDeps{
		Records:       deps.Records,
		Users:         deps.Users,
		Organizations: deps.Organizations,
		Stripe:        deps.Stripe,
	})

	charge := NewChargeService(ChargeServiceDeps{
		Records:       deps.Records,
		Stripe:        deps.Stripe,
		FeeCalculator: wallet,
	})

	payout := NewPayoutService(PayoutServiceDeps{
		Records:       deps.Records,
		Organizations: deps.Organizations,
		Stripe:        deps.Stripe,
	})

	return &Service{
		wallet:        wallet,
		charge:        charge,
		payout:        payout,
		notifications: deps.Notifications,
		frontendURL:   deps.FrontendURL,
	}
}

// StripeConfigured returns true when the Stripe service is available.
// Reads through the wallet sub-service, but the answer is identical
// across all three sub-services (they share the same StripeService
// pointer).
func (s *Service) StripeConfigured() bool {
	return s.wallet.stripe != nil
}

// SetReferralDistributor plugs the referral commission distributor in
// post-construction. Safe to call at app startup after both services
// exist. Passing nil disables the hook. Forwarded to the payout
// sub-service which actually fires the distribution on a successful
// per-milestone transfer.
func (s *Service) SetReferralDistributor(d service.ReferralCommissionDistributor) {
	s.payout.SetReferralDistributor(d)
}

// SetReferralClawback plugs the referral clawback hook in
// post-construction. Called from the refund / dispute-resolution flow
// when a milestone is (partially) refunded. Stored on the parent for
// backward compat with the legacy Service surface; not yet exercised by
// any decomposed sub-service.
func (s *Service) SetReferralClawback(c service.ReferralClawback) {
	s.referralClawback = c
}

// SetReferralWalletReader plugs the apporteur commission read path
// into the wallet overview. Wire AFTER both services exist so the
// wallet endpoint can return commission totals + history for
// referrers. Passing nil disables the commission section of the wallet
// DTO.
func (s *Service) SetReferralWalletReader(r service.ReferralWalletReader) {
	s.wallet.SetReferralWalletReader(r)
}

// SetProposalStatusReader plugs the proposal status lookup used by
// RequestPayout to keep escrow funds from being transferred before the
// mission is marked completed. Setter pattern because the proposal
// service is constructed AFTER payment in main.go (proposal depends on
// payment's PaymentProcessor). Passing nil leaves RequestPayout in a
// degraded mode that logs a warning and falls back to the pre-fix
// behaviour rather than erroring out — the feature must keep working
// in unusual wirings (tests, migrations, one-off binaries).
func (s *Service) SetProposalStatusReader(r service.ProposalStatusReader) {
	s.payout.SetProposalStatusReader(r)
}

// SetSubscriptionReader plugs the Premium subscription lookup. When
// the reader reports active=true for a provider, computePlatformFee
// waives the fee (returns 0). Setter pattern because the subscription
// app service is constructed AFTER payment in main.go. Passing nil
// (or never calling this) disables the feature — every milestone is
// billed at the normal grid rate.
func (s *Service) SetSubscriptionReader(r service.SubscriptionReader) {
	s.wallet.SetSubscriptionReader(r)
}

// ---------------------------------------------------------------------------
// Sub-service accessors — for callers that want the focused contract
// instead of the whole facade. Production code mostly uses the legacy
// public methods below; these accessors are wired so future call sites
// can apply Interface Segregation by depending on Wallet / Charge /
// Payout directly.
// ---------------------------------------------------------------------------

// Wallet returns the wallet sub-service for callers that only need
// read paths. Most production callers should use the legacy methods on
// *Service for backward compatibility.
func (s *Service) Wallet() *WalletService { return s.wallet }

// Charge returns the PaymentIntent lifecycle sub-service.
func (s *Service) Charge() *ChargeService { return s.charge }

// Payout returns the payout / transfer sub-service.
func (s *Service) Payout() *PayoutService { return s.payout }

// ---------------------------------------------------------------------------
// Legacy public surface — every method below delegates to the
// appropriate sub-service. Signatures are preserved so existing tests
// and callers keep working without modification.
// ---------------------------------------------------------------------------

// CreatePaymentIntent delegates to ChargeService.
func (s *Service) CreatePaymentIntent(ctx context.Context, input service.PaymentIntentInput) (*service.PaymentIntentOutput, error) {
	return s.charge.CreatePaymentIntent(ctx, input)
}

// MarkPaymentSucceeded delegates to ChargeService.
func (s *Service) MarkPaymentSucceeded(ctx context.Context, proposalID uuid.UUID) error {
	return s.charge.MarkPaymentSucceeded(ctx, proposalID)
}

// HandlePaymentSucceeded delegates to ChargeService.
func (s *Service) HandlePaymentSucceeded(ctx context.Context, paymentIntentID string) (uuid.UUID, error) {
	return s.charge.HandlePaymentSucceeded(ctx, paymentIntentID)
}

// VerifyWebhook delegates to ChargeService.
func (s *Service) VerifyWebhook(payload []byte, signature string) (*service.StripeWebhookEvent, error) {
	return s.charge.VerifyWebhook(payload, signature)
}

// TransferToProvider delegates to PayoutService.
func (s *Service) TransferToProvider(ctx context.Context, proposalID uuid.UUID) error {
	return s.payout.TransferToProvider(ctx, proposalID)
}

// TransferMilestone delegates to PayoutService.
func (s *Service) TransferMilestone(ctx context.Context, milestoneID uuid.UUID) error {
	return s.payout.TransferMilestone(ctx, milestoneID)
}

// TransferPartialToProvider delegates to PayoutService.
func (s *Service) TransferPartialToProvider(ctx context.Context, proposalID uuid.UUID, amount int64) error {
	return s.payout.TransferPartialToProvider(ctx, proposalID, amount)
}

// RefundToClient delegates to PayoutService.
func (s *Service) RefundToClient(ctx context.Context, proposalID uuid.UUID, amount int64) error {
	return s.payout.RefundToClient(ctx, proposalID, amount)
}

// RequestPayout delegates to PayoutService.
func (s *Service) RequestPayout(ctx context.Context, userID, orgID uuid.UUID) (*PayoutResult, error) {
	return s.payout.RequestPayout(ctx, userID, orgID)
}

// RetryFailedTransfer delegates to PayoutService.
func (s *Service) RetryFailedTransfer(ctx context.Context, userID, orgID, recordID uuid.UUID) (*PayoutResult, error) {
	return s.payout.RetryFailedTransfer(ctx, userID, orgID, recordID)
}

// CanProviderReceivePayouts delegates to PayoutService.
func (s *Service) CanProviderReceivePayouts(ctx context.Context, providerOrgID uuid.UUID) (bool, error) {
	return s.payout.CanProviderReceivePayouts(ctx, providerOrgID)
}

// HasAutoPayoutConsent delegates to PayoutService.
func (s *Service) HasAutoPayoutConsent(ctx context.Context, providerOrgID uuid.UUID) (bool, error) {
	return s.payout.HasAutoPayoutConsent(ctx, providerOrgID)
}

// WaivePlatformFeeOnActiveRecords delegates to PayoutService.
func (s *Service) WaivePlatformFeeOnActiveRecords(ctx context.Context, providerOrgID uuid.UUID) error {
	return s.payout.WaivePlatformFeeOnActiveRecords(ctx, providerOrgID)
}

// GetWalletOverview delegates to WalletService.
func (s *Service) GetWalletOverview(ctx context.Context, userID, orgID uuid.UUID) (*WalletOverview, error) {
	return s.wallet.GetWalletOverview(ctx, userID, orgID)
}

// GetPaymentRecord delegates to WalletService.
func (s *Service) GetPaymentRecord(ctx context.Context, proposalID uuid.UUID) (*domain.PaymentRecord, error) {
	return s.wallet.GetPaymentRecord(ctx, proposalID)
}

// PreviewFee delegates to WalletService.
func (s *Service) PreviewFee(ctx context.Context, userID uuid.UUID, amountCents int64, recipientID *uuid.UUID) (*FeePreviewResult, error) {
	return s.wallet.PreviewFee(ctx, userID, amountCents, recipientID)
}

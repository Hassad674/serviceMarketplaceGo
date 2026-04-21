// Package payment handles post-KYC payment flow: charging clients,
// transferring funds to connected accounts, tracking payment records,
// computing wallet balances, and issuing payouts.
//
// KYC onboarding itself lives in internal/app/embedded (Stripe Connect
// Embedded Components). Since phase R5 the Stripe Connect account is
// owned by the ORGANIZATION, not the user — the payment service reads
// and writes it through OrganizationRepository.
package payment

import (
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// Service orchestrates payment intents, transfers, payouts, and the
// wallet overview. Thread-safe; dependencies are injected via ServiceDeps.
type Service struct {
	records       repository.PaymentRecordRepository
	users         repository.UserRepository // still used for display-name lookups
	orgs          repository.OrganizationRepository
	stripe        service.StripeService      // nil if Stripe not configured
	notifications service.NotificationSender // nil if not configured
	frontendURL   string

	// Referral hooks — wired post-construction via setters to break the
	// import cycle with the referral app service. Nil when the referral
	// feature is not active; all call sites guard for nil before invoking.
	referralDistributor service.ReferralCommissionDistributor
	referralClawback    service.ReferralClawback
	referralWallet      service.ReferralWalletReader

	// proposalStatuses gates payout transfers on mission completion.
	// Wired post-construction because payment is built before proposal
	// in main.go (proposal depends on payment's PaymentProcessor). When
	// nil, RequestPayout logs a warning and falls back to the legacy
	// behaviour so the payment feature stays bootable without proposal.
	proposalStatuses service.ProposalStatusReader

	// subscriptions waives the platform fee for Premium subscribers.
	// Wired post-construction because the subscription feature is
	// removable — when nil, computePlatformFee falls back to the grid
	// fee for every user, matching the pre-Premium behaviour exactly.
	subscriptions service.SubscriptionReader
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
func NewService(deps ServiceDeps) *Service {
	return &Service{
		records:       deps.Records,
		users:         deps.Users,
		orgs:          deps.Organizations,
		stripe:        deps.Stripe,
		notifications: deps.Notifications,
		frontendURL:   deps.FrontendURL,
	}
}

// StripeConfigured returns true when the Stripe service is available.
func (s *Service) StripeConfigured() bool {
	return s.stripe != nil
}

// SetReferralDistributor plugs the referral commission distributor in
// post-construction. Safe to call at app startup after both services exist.
// Passing nil disables the hook.
func (s *Service) SetReferralDistributor(d service.ReferralCommissionDistributor) {
	s.referralDistributor = d
}

// SetReferralClawback plugs the referral clawback hook in post-construction.
// Called from the refund / dispute-resolution flow when a milestone is
// (partially) refunded.
func (s *Service) SetReferralClawback(c service.ReferralClawback) {
	s.referralClawback = c
}

// SetReferralWalletReader plugs the apporteur commission read path into
// the wallet overview. Wire AFTER both services exist so the wallet
// endpoint can return commission totals + history for referrers.
// Passing nil disables the commission section of the wallet DTO.
func (s *Service) SetReferralWalletReader(r service.ReferralWalletReader) {
	s.referralWallet = r
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
	s.proposalStatuses = r
}

// SetSubscriptionReader plugs the Premium subscription lookup. When the
// reader reports active=true for a provider, computePlatformFee waives
// the fee (returns 0). Setter pattern because the subscription app
// service is constructed AFTER payment in main.go. Passing nil (or
// never calling this) disables the feature — every milestone is billed
// at the normal grid rate.
func (s *Service) SetSubscriptionReader(r service.SubscriptionReader) {
	s.subscriptions = r
}

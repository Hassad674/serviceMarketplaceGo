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

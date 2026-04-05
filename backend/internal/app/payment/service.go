// Package payment handles post-KYC payment flow: charging clients,
// transferring funds to connected accounts, tracking payment records,
// computing wallet balances, and issuing payouts.
//
// KYC onboarding itself lives in internal/app/embedded (Stripe Connect
// Embedded Components). This package consumes the stripe_account_id
// stored on users.stripe_account_id (migration 040) via UserRepository.
package payment

import (
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// Service orchestrates payment intents, transfers, payouts, and the
// wallet overview. Thread-safe; dependencies are injected via ServiceDeps.
type Service struct {
	records       repository.PaymentRecordRepository
	users         repository.UserRepository
	stripe        service.StripeService      // nil if Stripe not configured
	notifications service.NotificationSender // nil if not configured
	frontendURL   string
}

// ServiceDeps groups all dependencies for the payment service.
type ServiceDeps struct {
	Records       repository.PaymentRecordRepository
	Users         repository.UserRepository
	Stripe        service.StripeService
	Notifications service.NotificationSender
	FrontendURL   string
}

// NewService wires the payment service. All fields are optional except
// Records and Users — the charge / transfer methods fail-fast when
// Stripe is not configured.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		records:       deps.Records,
		users:         deps.Users,
		stripe:        deps.Stripe,
		notifications: deps.Notifications,
		frontendURL:   deps.FrontendURL,
	}
}

// StripeConfigured returns true when the Stripe service is available.
func (s *Service) StripeConfigured() bool {
	return s.stripe != nil
}

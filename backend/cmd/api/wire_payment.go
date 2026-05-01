package main

import (
	"log/slog"

	stripeadapter "marketplace-backend/internal/adapter/stripe"
	notifapp "marketplace-backend/internal/app/notification"
	paymentapp "marketplace-backend/internal/app/payment"
	proposalapp "marketplace-backend/internal/app/proposal"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// stripeServices bundles the three faces the Stripe adapter exposes.
// One concrete *stripeadapter.Service satisfies all three interfaces
// (charges, transfer reversals, KYC snapshot reads) — we keep a typed
// reference so the referral feature can pick the narrower interfaces
// it actually consumes without re-resolving downstream.
//
// Every field stays nil when Stripe is not configured. The whole
// commerce subtree (payment, subscription, invoicing, wallet KYC,
// referral commissions, dispute payouts) short-circuits on these
// nil checks so the backend boots cleanly without Stripe credentials.
type stripeServices struct {
	Charges        service.StripeService
	Reversals      service.StripeTransferReversalService
	KYCReader      service.StripeKYCSnapshotReader
}

// wireStripe spins up the Stripe payment adapter when StripeConfigured()
// returns true. Otherwise every field stays nil — callers are expected
// to short-circuit on those nil checks (see the wallet, subscription,
// invoicing and referral wires).
func wireStripe(cfg *config.Config) stripeServices {
	if !cfg.StripeConfigured() {
		slog.Info("stripe payment adapter disabled (not configured)")
		return stripeServices{}
	}
	stripeAdapter := stripeadapter.NewService(cfg.StripeSecretKey, cfg.StripeWebhookSecret)
	slog.Info("stripe payment adapter enabled")
	return stripeServices{
		Charges:   stripeAdapter,
		Reversals: stripeAdapter,
		KYCReader: stripeAdapter,
	}
}

// paymentWiring carries the payment app service plus the wallet +
// billing HTTP handlers. The proposal service is consumed by the
// wallet handler (mission status lookup before payout) — main.go
// still attaches the proposalSvc.SetProposalStatusReader setter
// outside this helper because it crosses the proposal wire boundary.
type paymentWiring struct {
	Service *paymentapp.Service
	Wallet  *handler.WalletHandler
	Billing *handler.BillingHandler
}

// paymentDeps captures the upstream dependencies the payment service
// + wallet/billing handlers reach into.
type paymentDeps struct {
	Cfg               *config.Config
	PaymentRecordRepo repository.PaymentRecordRepository
	UserRepo          repository.UserRepository
	OrganizationRepo  repository.OrganizationRepository
	StripeSvc         service.StripeService
	Notifications     *notifapp.Service
}

// wirePayment builds the payment app service. The wallet + billing
// handlers depend on the proposal service which is wired AFTER the
// payment service, so they live on a separate helper
// (wirePaymentHandlers) called downstream by main.go.
func wirePayment(deps paymentDeps) *paymentapp.Service {
	// Payment service — charge creation + transfers + wallet overview.
	// KYC onboarding lives in internal/app/embedded (Embedded Components).
	return paymentapp.NewService(paymentapp.ServiceDeps{
		Records:       deps.PaymentRecordRepo,
		Users:         deps.UserRepo,
		Organizations: deps.OrganizationRepo,
		Stripe:        deps.StripeSvc,
		Notifications: deps.Notifications,
		FrontendURL:   deps.Cfg.FrontendURL,
	})
}

// paymentHandlersDeps captures the dependencies the wallet + billing
// handlers reach into. Both share the same payment service; the
// wallet handler also needs the proposal service for mission status
// lookups.
type paymentHandlersDeps struct {
	PaymentInfoSvc *paymentapp.Service
	ProposalSvc    *proposalapp.Service
}

// wirePaymentHandlers builds the wallet + billing HTTP handlers. The
// billing handler is read-only fee preview for the proposal creation
// flow (shares the payment service so the fee schedule stays the
// single source of truth across CreatePaymentIntent and the
// client-facing simulator). The wallet handler exposes the wallet
// overview + payout request endpoints.
func wirePaymentHandlers(deps paymentHandlersDeps) (*handler.WalletHandler, *handler.BillingHandler) {
	// Wallet handler
	walletHandler := handler.NewWalletHandler(deps.PaymentInfoSvc, deps.ProposalSvc)

	// Billing handler — read-only fee preview endpoint for the proposal
	// creation flow. Shares the payment service (no new dependencies) so
	// the fee schedule stays the single source of truth across CreatePaymentIntent
	// and the client-facing simulator.
	billingHandler := handler.NewBillingHandler(deps.PaymentInfoSvc)

	return walletHandler, billingHandler
}

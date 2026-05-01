package main

import (
	"log/slog"

	stripeadapter "marketplace-backend/internal/adapter/stripe"
	"marketplace-backend/internal/config"
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

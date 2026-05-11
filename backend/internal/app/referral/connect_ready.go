package referral

import (
	"context"
	"log/slog"
)

// connectReadyForReferrer returns true when the apporteur's Stripe
// Connect account is ready to receive a transfer. The gate combines
// two signals:
//
//  1. The connected-account id resolution must have produced a non-empty
//     account id (i.e. the apporteur has at least started onboarding and
//     we have an acct_* on file).
//  2. The live Stripe account snapshot must report `payouts_enabled=true`
//     AND `charges_enabled=true` — Stripe rejects Transfer-and-Payout on
//     accounts that do not have both capabilities active.
//
// The probe is fail-CLOSED on error: a transient network blip while
// asking Stripe for capabilities returns false, so the commission lands
// in pending_kyc instead of burning a Stripe idempotency key on a
// doomed CreateTransfer. The apporteur can then come back and retire
// the commission once their account is verified — the retry endpoint
// re-runs the same gate.
//
// When the StripeService dependency is nil (worktree without Stripe
// wiring) we degrade to "ready as soon as we have an account id" so
// the legacy distributor path still works in slim deployments. Tests
// pass a fakeStripe that returns canned info.
func (s *Service) connectReadyForReferrer(ctx context.Context, stripeAccount string) bool {
	if stripeAccount == "" {
		return false
	}
	if s.stripe == nil {
		// Degrade open — slim deployments without Stripe wiring fall
		// back to the legacy "account present" heuristic.
		return true
	}
	info, err := s.stripe.GetAccount(ctx, stripeAccount)
	if err != nil {
		slog.Warn("referral: GetAccount capability probe failed, treating as not ready",
			"stripe_account", stripeAccount, "error", err)
		return false
	}
	if info == nil {
		// No capability snapshot — fail closed.
		return false
	}
	return info.PayoutsEnabled && info.ChargesEnabled
}

package stripe

import (
	"context"
	"fmt"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/payout"

	portservice "marketplace-backend/internal/port/service"
)

// CreatePayout creates a Stripe payout from a connected account's
// available balance to the bank account configured on that connected
// account. Required because every connected account is now created on
// a manual payout schedule (see account.go) — Stripe will not auto-pay,
// the platform must trigger the bank transfer at "Retirer" time.
//
// The Stripe-Account header is set via params.SetStripeAccount so the
// payout runs *on* the connected account (a payout without that header
// would target the platform balance, which is wrong).
//
// The IdempotencyKey is required: the wallet handler may be retried by
// the user / mobile client, and a duplicate payout would double-debit
// the connected account balance.
func (s *Service) CreatePayout(ctx context.Context, input portservice.CreatePayoutInput) (string, error) {
	if input.ConnectedAccountID == "" {
		return "", fmt.Errorf("create payout: empty connected account id")
	}
	if input.Amount <= 0 {
		return "", fmt.Errorf("create payout: amount must be positive (got %d)", input.Amount)
	}
	if input.Currency == "" {
		return "", fmt.Errorf("create payout: empty currency")
	}
	if input.IdempotencyKey == "" {
		return "", fmt.Errorf("create payout: missing idempotency key")
	}

	params := &stripe.PayoutParams{
		Amount:   stripe.Int64(input.Amount),
		Currency: stripe.String(input.Currency),
		Params: stripe.Params{
			IdempotencyKey: stripe.String(input.IdempotencyKey),
			Context:        ctx,
		},
	}
	params.SetStripeAccount(input.ConnectedAccountID)

	if input.Method != "" {
		params.Method = stripe.String(input.Method)
	}
	if input.Description != "" {
		params.Description = stripe.String(input.Description)
	}

	p, err := payout.New(params)
	if err != nil {
		return "", fmt.Errorf("create payout on %s: %w", input.ConnectedAccountID, err)
	}
	return p.ID, nil
}

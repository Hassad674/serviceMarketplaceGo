package stripe

import (
	"context"
	"fmt"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentintent"

	portservice "marketplace-backend/internal/port/service"
)

func (s *Service) CreatePaymentIntent(ctx context.Context, input portservice.CreatePaymentIntentInput) (*portservice.PaymentIntentResult, error) {
	// Idempotency is scoped to the MILESTONE, not the proposal —
	// phase 4 lets a single proposal own N payment intents (one per
	// milestone), each with its own amount. Using pi_<proposalID>
	// would trigger Stripe's "same key, different params" error on
	// the second milestone and beyond.
	params := &stripe.PaymentIntentParams{
		Amount:             stripe.Int64(input.AmountCentimes),
		Currency:           stripe.String(input.Currency),
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		TransferGroup:      stripe.String(input.TransferGroup),
		Params: stripe.Params{
			IdempotencyKey: stripe.String("pi_ms_" + input.MilestoneID),
		},
	}

	params.AddMetadata("proposal_id", input.ProposalID)
	params.AddMetadata("milestone_id", input.MilestoneID)
	params.AddMetadata("client_id", input.ClientID)
	params.AddMetadata("provider_id", input.ProviderID)

	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("create payment intent: %w", err)
	}

	return &portservice.PaymentIntentResult{
		PaymentIntentID: pi.ID,
		ClientSecret:    pi.ClientSecret,
		AmountTotal:     pi.Amount,
	}, nil
}

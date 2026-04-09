package stripe

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/refund"
)

// CreateRefund creates a partial or full refund on a PaymentIntent.
func (s *Service) CreateRefund(ctx context.Context, paymentIntentID string, amount int64) (string, error) {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(paymentIntentID),
	}
	if amount > 0 {
		params.Amount = stripe.Int64(amount)
	}

	r, err := refund.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe refund: %w", err)
	}

	return r.ID, nil
}

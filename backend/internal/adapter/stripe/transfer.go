package stripe

import (
	"context"
	"fmt"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/transfer"
	"github.com/stripe/stripe-go/v82/transferreversal"

	portservice "marketplace-backend/internal/port/service"
)

func (s *Service) CreateTransfer(ctx context.Context, input portservice.CreateTransferInput) (string, error) {
	params := &stripe.TransferParams{
		Amount:        stripe.Int64(input.Amount),
		Currency:      stripe.String(input.Currency),
		Destination:   stripe.String(input.DestinationAccount),
		TransferGroup: stripe.String(input.TransferGroup),
		Params: stripe.Params{
			IdempotencyKey: stripe.String(input.IdempotencyKey),
		},
	}

	t, err := transfer.New(params)
	if err != nil {
		return "", fmt.Errorf("create transfer: %w", err)
	}

	return t.ID, nil
}

// CreateTransferReversal reverses a previously executed transfer, fully or
// partially. Used by the referral clawback flow when a milestone is refunded
// after the apporteur commission has been transferred out.
func (s *Service) CreateTransferReversal(ctx context.Context, input portservice.CreateTransferReversalInput) (string, error) {
	params := &stripe.TransferReversalParams{
		ID:     stripe.String(input.TransferID),
		Amount: stripe.Int64(input.Amount),
		Params: stripe.Params{
			IdempotencyKey: stripe.String(input.IdempotencyKey),
		},
	}

	r, err := transferreversal.New(params)
	if err != nil {
		return "", fmt.Errorf("create transfer reversal: %w", err)
	}

	return r.ID, nil
}

package stripe

import (
	"context"
	"fmt"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/transfer"

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

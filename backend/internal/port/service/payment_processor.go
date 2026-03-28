package service

import (
	"context"

	"github.com/google/uuid"
)

// PaymentProcessor is implemented by the payment app service and consumed
// by the proposal service. This avoids cross-feature imports.
type PaymentProcessor interface {
	// CreatePaymentIntent creates a Stripe PaymentIntent for a proposal.
	// Returns the client secret for Stripe Elements on the frontend.
	CreatePaymentIntent(ctx context.Context, input PaymentIntentInput) (*PaymentIntentOutput, error)

	// TransferToProvider transfers funds to the provider's connected account.
	TransferToProvider(ctx context.Context, proposalID uuid.UUID) error

	// HandlePaymentSucceeded processes a successful payment webhook.
	// Returns the proposal ID so the caller can transition the proposal.
	HandlePaymentSucceeded(ctx context.Context, paymentIntentID string) (proposalID uuid.UUID, err error)
}

type PaymentIntentInput struct {
	ProposalID     uuid.UUID
	ClientID       uuid.UUID
	ProviderID     uuid.UUID
	ProposalAmount int64 // centimes
}

type PaymentIntentOutput struct {
	ClientSecret    string
	PaymentRecordID uuid.UUID
	ProposalAmount  int64
	StripeFee       int64
	PlatformFee     int64
	ClientTotal     int64
	ProviderPayout  int64
}

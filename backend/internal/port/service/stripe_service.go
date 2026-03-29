package service

import (
	"context"
	"io"

	"marketplace-backend/internal/domain/payment"
)

// StripeService abstracts Stripe API operations for Connect Custom.
type StripeService interface {
	// CreateConnectedAccount creates a Stripe Custom connected account from payment info.
	CreateConnectedAccount(ctx context.Context, info *payment.PaymentInfo, tosIP string) (accountID string, err error)

	// GetAccountStatus checks whether a connected account is verified.
	GetAccountStatus(ctx context.Context, accountID string) (verified bool, err error)

	// CreatePaymentIntent creates a PaymentIntent on the platform account.
	CreatePaymentIntent(ctx context.Context, input CreatePaymentIntentInput) (*PaymentIntentResult, error)

	// CreateTransfer sends funds to a connected account.
	CreateTransfer(ctx context.Context, input CreateTransferInput) (transferID string, err error)

	// ConstructWebhookEvent verifies and parses a Stripe webhook event.
	ConstructWebhookEvent(payload []byte, signature string) (*StripeWebhookEvent, error)

	// GetIdentityVerificationStatus returns the verification status and the verified front file ID.
	GetIdentityVerificationStatus(ctx context.Context, accountID string) (status string, verifiedFileID string, err error)

	// UploadIdentityFile uploads a file to Stripe for identity verification.
	UploadIdentityFile(ctx context.Context, filename string, reader io.Reader, purpose string) (fileID string, err error)

	// UpdateAccountVerification attaches verification documents to a connected account.
	UpdateAccountVerification(ctx context.Context, accountID string, frontFileID, backFileID string) error
}

type CreatePaymentIntentInput struct {
	AmountCentimes int64  // total amount client pays
	Currency       string // "eur"
	ProposalID     string // metadata + idempotency
	ClientID       string // metadata
	ProviderID     string // metadata
	TransferGroup  string // groups related transfers
}

type PaymentIntentResult struct {
	PaymentIntentID string
	ClientSecret    string
	AmountTotal     int64
}

type CreateTransferInput struct {
	Amount             int64
	Currency           string
	DestinationAccount string
	TransferGroup      string
	IdempotencyKey     string
}

type StripeWebhookEvent struct {
	Type            string
	PaymentIntentID string
	AccountID       string
}

package service

import (
	"context"
)

// StripeService abstracts Stripe API operations for Connect Custom.
type StripeService interface {
	// CreateMinimalAccount creates a minimal Stripe Custom account for embedded onboarding.
	CreateMinimalAccount(ctx context.Context, country, email string) (accountID string, err error)

	// CreateAccountSession creates an Account Session for embedded components.
	CreateAccountSession(ctx context.Context, accountID string) (clientSecret string, err error)

	// GetAccountStatus checks whether a connected account is verified.
	GetAccountStatus(ctx context.Context, accountID string) (verified bool, err error)

	// GetFullAccount returns detailed account info for syncing to the database.
	GetFullAccount(ctx context.Context, accountID string) (*StripeAccountInfo, error)

	// CreatePaymentIntent creates a PaymentIntent on the platform account.
	CreatePaymentIntent(ctx context.Context, input CreatePaymentIntentInput) (*PaymentIntentResult, error)

	// CreateTransfer sends funds to a connected account.
	CreateTransfer(ctx context.Context, input CreateTransferInput) (transferID string, err error)

	// ConstructWebhookEvent verifies and parses a Stripe webhook event.
	ConstructWebhookEvent(payload []byte, signature string) (*StripeWebhookEvent, error)
}

// StripeAccountInfo holds synced account data from Stripe.
type StripeAccountInfo struct {
	ChargesEnabled bool
	PayoutsEnabled bool
	Country        string
	BusinessType   string
	DisplayName    string
	CurrentlyDue   []string
}

type CreatePaymentIntentInput struct {
	AmountCentimes int64
	Currency       string
	ProposalID     string
	ClientID       string
	ProviderID     string
	TransferGroup  string
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

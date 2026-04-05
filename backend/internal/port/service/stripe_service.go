package service

import (
	"context"
	"time"
)

// StripeService abstracts Stripe API operations for payment + webhook
// verification. KYC onboarding lives in Embedded Components — not here.
type StripeService interface {
	// CreatePaymentIntent creates a PaymentIntent on the platform account.
	CreatePaymentIntent(ctx context.Context, input CreatePaymentIntentInput) (*PaymentIntentResult, error)

	// CreateTransfer sends funds to a connected account.
	CreateTransfer(ctx context.Context, input CreateTransferInput) (transferID string, err error)

	// ConstructWebhookEvent verifies and parses a Stripe webhook event.
	ConstructWebhookEvent(payload []byte, signature string) (*StripeWebhookEvent, error)
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

type CreatePersonInput struct {
	FirstName        string
	LastName         string
	Email            string
	Phone            string
	DOB              time.Time
	Address          string
	City             string
	PostalCode       string
	State            string
	Country          string
	Title            string
	IDNumber         string
	SSNLast4         string
	IsRepresentative bool
	IsDirector       bool
	IsOwner          bool
	IsExecutive      bool
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

	// AccountSnapshot is populated for account.* events so downstream
	// handlers can react on full account state without a second API call.
	AccountSnapshot *StripeAccountSnapshot
}

// StripeAccountSnapshot captures the state of a connected account at the
// moment a webhook was received. Used to detect transitions (activated /
// suspended / requirements changed / document rejected) without needing
// to re-fetch the account from the API.
type StripeAccountSnapshot struct {
	AccountID        string
	Country          string
	BusinessType     string
	ChargesEnabled   bool
	PayoutsEnabled   bool
	DetailsSubmitted bool

	// Requirements partitions — each holds the field names Stripe needs.
	CurrentlyDue        []string
	EventuallyDue       []string
	PastDue             []string
	PendingVerification []string
	DisabledReason      string

	// Errors explains WHY a field was rejected (document blurry, name
	// mismatch, etc.). Keyed by the requirement, value is the reason.
	RequirementErrors []StripeRequirementError
}

// StripeRequirementError mirrors a single entry of Stripe's
// requirements.errors array (requirement + code + reason).
type StripeRequirementError struct {
	Requirement string
	Code        string
	Reason      string
}

// AccountFullStatus combines verification and account status from a single Stripe API call.
type AccountFullStatus struct {
	VerificationStatus string
	VerifiedFileID     string
	ChargesEnabled     bool
	PayoutsEnabled     bool
}

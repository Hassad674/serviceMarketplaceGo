package service

import (
	"context"
	"time"
)

// StripeService abstracts Stripe API operations for payment + webhook
// verification. KYC onboarding lives in Embedded Components — not here.
// StripeAccountInfo is a minimal view of a connected account's capabilities.
type StripeAccountInfo struct {
	ChargesEnabled bool
	PayoutsEnabled bool
}

type StripeService interface {
	// CreatePaymentIntent creates a PaymentIntent on the platform account.
	CreatePaymentIntent(ctx context.Context, input CreatePaymentIntentInput) (*PaymentIntentResult, error)

	// CreateTransfer sends funds to a connected account.
	CreateTransfer(ctx context.Context, input CreateTransferInput) (transferID string, err error)

	// ConstructWebhookEvent verifies and parses a Stripe webhook event.
	ConstructWebhookEvent(payload []byte, signature string) (*StripeWebhookEvent, error)

	// GetAccount retrieves a connected account's capabilities status.
	GetAccount(ctx context.Context, accountID string) (*StripeAccountInfo, error)

	// CreateRefund creates a partial or full refund on a PaymentIntent.
	CreateRefund(ctx context.Context, paymentIntentID string, amount int64) (refundID string, err error)
}

type CreatePaymentIntentInput struct {
	AmountCentimes int64  // total amount client pays
	Currency       string // "eur"
	ProposalID     string // metadata + transfer group
	MilestoneID    string // metadata + idempotency (phase 4)
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
	// EventID is the Stripe event identifier (evt_*). Used by the
	// handler as an idempotency key so replays (Stripe retries on 5xx)
	// do not double-apply a state change.
	EventID string

	Type            string
	PaymentIntentID string
	AccountID       string

	// AccountSnapshot is populated for account.* events so downstream
	// handlers can react on full account state without a second API call.
	AccountSnapshot *StripeAccountSnapshot

	// Subscription fields — populated for customer.subscription.* and
	// invoice.payment_* events. The subscription app service consumes
	// them to mirror the Stripe state into our local row.
	SubscriptionSnapshot       *SubscriptionSnapshot
	SubscriptionDeleted        bool
	// SubscriptionOrganizationID is the canonical owner identifier written
	// in subscription metadata since the org-scoped migration. Empty for
	// legacy Stripe subscriptions that predate the migration — in that
	// case SubscriptionUserID is populated and the handler resolves the
	// org via users.organization_id.
	SubscriptionOrganizationID string
	SubscriptionUserID         string // legacy metadata.user_id — transition only
	SubscriptionPlan           string // parsed from price lookup_key
	SubscriptionCycle          string // parsed from price lookup_key
	// SubscriptionCancelAtPeriodEndIntent captures the user's choice at
	// checkout time (the "auto-renew off by default" product rule).
	// Stripe Checkout doesn't expose cancel_at_period_end at creation,
	// so we propagate it via subscription metadata and let the webhook
	// handler apply the flag post-creation. True iff the metadata key
	// `cancel_at_period_end` equals "true".
	SubscriptionCancelAtPeriodEndIntent bool
	InvoiceSubscriptionID string // parent subscription id on invoice events
	InvoicePaymentFailed  bool   // true on invoice.payment_failed
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

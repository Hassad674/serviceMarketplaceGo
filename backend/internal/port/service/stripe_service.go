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

	// CreatePayout triggers a payout from a connected account's Stripe
	// balance to its external bank account. This is the second leg of
	// the wallet "Retirer" flow: CreateTransfer moved funds platform →
	// connected account, CreatePayout completes the bank transfer.
	// Required because connected accounts run on a manual payout
	// schedule (see UpdatePayoutSchedule) — Stripe will not auto-pay.
	CreatePayout(ctx context.Context, input CreatePayoutInput) (payoutID string, err error)

	// UpdatePayoutSchedule changes a connected account's payout schedule.
	// Used at account creation (interval = "manual") and by the backfill
	// CLI to bring legacy accounts in line with the manual-only policy.
	UpdatePayoutSchedule(ctx context.Context, accountID, interval string) error

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

// CreatePayoutInput drives a connected-account → bank payout. The
// Amount is the centimes to send (must not exceed the connected
// account's available balance). Method is "standard" by default;
// pass "instant" to use Stripe Instant Payouts when the destination
// debit card supports it. ConnectedAccountID is the acct_* the
// payout runs on (Stripe-Account header). IdempotencyKey is required
// to make Retirer clicks safe to replay.
type CreatePayoutInput struct {
	ConnectedAccountID string
	Amount             int64
	Currency           string
	Method             string // "standard" (default) or "instant"
	IdempotencyKey     string
	Description        string
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

	// Invoice.paid fields — populated when event.Type == "invoice.paid".
	// Consumed by the invoicing app service to issue a customer-facing
	// invoice for a successful subscription payment. Empty/zero on every
	// other event type.
	InvoicePaid                  bool
	InvoiceID                    string // Stripe invoice id (in_*)
	InvoicePaymentIntentID       string // pi_* — extracted from the invoice's payment list, may be empty
	InvoiceAmountPaidCents       int64  // amount the customer actually paid, in cents
	InvoiceCurrency              string // ISO-4217, lowercase as Stripe returns
	InvoicePeriodStart           time.Time
	InvoicePeriodEnd             time.Time
	InvoiceLineDescription       string // first line item description (e.g. plan label) — best-effort
	InvoiceSubscriptionOrgID     string // organization_id from subscription metadata, if present
	InvoiceSubscriptionUserID    string // legacy user_id from subscription metadata, if present

	// Charge.refunded fields — populated when event.Type == "charge.refunded".
	// Consumed by the invoicing app service to emit a credit note (avoir)
	// for the refunded amount. Empty/zero on every other event type.
	ChargeRefunded            bool
	ChargeID                  string // ch_*
	ChargePaymentIntentID     string // pi_* — bridges back to the original invoice
	ChargeAmountRefundedCents int64  // total refunded so far on the charge (cumulative)
	ChargeRefundID            string // re_* — most recent refund id, when available
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

package referral

import (
	"time"

	"github.com/google/uuid"
)

// CommissionStatus is the lifecycle state of a single commission row, which
// represents the apporteur's share of one milestone payment.
//
//	pending      → row inserted, transfer to the referrer not yet attempted
//	pending_kyc  → referrer has no Stripe Connect account yet, payout queued
//	               until embedded.OnStripeAccountReady fires
//	paid         → Stripe transfer succeeded, stripe_transfer_id stored
//	failed       → Stripe call failed (network, account inactive, …); operator action
//	cancelled    → milestone cancelled before transfer, no money changed hands
//	clawed_back  → milestone was refunded after payout; transfer_reversal executed
type CommissionStatus string

const (
	CommissionPending     CommissionStatus = "pending"
	CommissionPendingKYC  CommissionStatus = "pending_kyc"
	CommissionPaid        CommissionStatus = "paid"
	CommissionFailed      CommissionStatus = "failed"
	CommissionCancelled   CommissionStatus = "cancelled"
	CommissionClawedBack  CommissionStatus = "clawed_back"
)

// IsValid reports whether s is one of the known commission statuses.
func (s CommissionStatus) IsValid() bool {
	switch s {
	case CommissionPending, CommissionPendingKYC, CommissionPaid,
		CommissionFailed, CommissionCancelled, CommissionClawedBack:
		return true
	}
	return false
}

// Commission is one apporteur payout, scoped to one milestone of an attributed
// proposal. Created by the distributor BEFORE the Stripe transfer call (so DB
// idempotency wins over Stripe idempotency in case of partial failure).
type Commission struct {
	ID               uuid.UUID
	AttributionID    uuid.UUID
	MilestoneID      uuid.UUID
	GrossAmountCents int64
	CommissionCents  int64
	Currency         string
	Status           CommissionStatus
	StripeTransferID string
	StripeReversalID string
	FailureReason    string
	PaidAt           *time.Time
	ClawedBackAt     *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// NewCommissionInput is the validated input for NewCommission.
type NewCommissionInput struct {
	AttributionID    uuid.UUID
	MilestoneID      uuid.UUID
	GrossAmountCents int64
	RatePct          float64
	Currency         string
}

// NewCommission constructs a Commission in the pending state. The commission
// amount is computed from gross × rate / 100, rounded DOWN (truncation) so the
// platform never owes the apporteur more than one centime of rounding error.
func NewCommission(input NewCommissionInput) (*Commission, error) {
	if input.AttributionID == uuid.Nil || input.MilestoneID == uuid.Nil {
		return nil, ErrNotAuthorized
	}
	if input.GrossAmountCents <= 0 {
		return nil, ErrInsufficientGrossAmount
	}
	if input.RatePct < MinRatePct || input.RatePct > MaxRatePct {
		return nil, ErrRateOutOfRange
	}
	currency := input.Currency
	if currency == "" {
		currency = "EUR"
	}

	amount := computeCommissionCents(input.GrossAmountCents, input.RatePct)
	now := time.Now().UTC()
	return &Commission{
		ID:               uuid.New(),
		AttributionID:    input.AttributionID,
		MilestoneID:      input.MilestoneID,
		GrossAmountCents: input.GrossAmountCents,
		CommissionCents:  amount,
		Currency:         currency,
		Status:           CommissionPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// computeCommissionCents truncates gross × pct / 100 down to the nearest cent.
// We avoid floating-point drift on round numbers by working in basis points:
// gross_cents * (rate_pct * 100) / 10_000 = gross_cents * rate_bp / 10_000
func computeCommissionCents(grossCents int64, ratePct float64) int64 {
	rateBp := int64(ratePct * 100) // 5.0 % → 500 basis points
	return grossCents * rateBp / 10_000
}

// MarkPaid transitions a pending commission to paid after a successful Stripe transfer.
func (c *Commission) MarkPaid(stripeTransferID string) error {
	if c.Status != CommissionPending {
		return ErrCommissionNotPayable
	}
	now := time.Now().UTC()
	c.Status = CommissionPaid
	c.StripeTransferID = stripeTransferID
	c.PaidAt = &now
	c.UpdatedAt = now
	return nil
}

// MarkPendingKYC parks a pending commission until the referrer completes KYC.
// The kyc_listener service will pick it up via OnStripeAccountReady later.
func (c *Commission) MarkPendingKYC() error {
	if c.Status != CommissionPending {
		return ErrCommissionNotPayable
	}
	c.Status = CommissionPendingKYC
	c.UpdatedAt = time.Now().UTC()
	return nil
}

// MarkFailed records a Stripe transfer failure and the reason.
func (c *Commission) MarkFailed(reason string) error {
	if c.Status != CommissionPending && c.Status != CommissionPendingKYC {
		return ErrCommissionNotPayable
	}
	c.Status = CommissionFailed
	c.FailureReason = reason
	c.UpdatedAt = time.Now().UTC()
	return nil
}

// MarkCancelled is used when a milestone is cancelled before any transfer was
// attempted (e.g., dispute resolved against the provider, milestone scrapped).
func (c *Commission) MarkCancelled() error {
	switch c.Status {
	case CommissionPending, CommissionPendingKYC:
		c.Status = CommissionCancelled
		c.UpdatedAt = time.Now().UTC()
		return nil
	}
	return ErrClawbackNotApplicable
}

// ApplyClawback reverses a previously paid commission, fully or partially,
// to mirror a Stripe refund on the parent milestone. The amount is computed
// proportionally (refunded / gross) by the caller; this method just records
// the result and stamps the reversal id.
func (c *Commission) ApplyClawback(stripeReversalID string) error {
	if c.Status != CommissionPaid {
		return ErrClawbackNotApplicable
	}
	now := time.Now().UTC()
	c.Status = CommissionClawedBack
	c.StripeReversalID = stripeReversalID
	c.ClawedBackAt = &now
	c.UpdatedAt = now
	return nil
}

// ClawbackAmountCents computes the reversal amount for a partial refund of the
// parent milestone. Truncates DOWN so the platform never reverses more than the
// strict mathematical share — favours the platform's books over the apporteur,
// the same rounding direction as NewCommission.
func ClawbackAmountCents(commissionCents, grossCents, refundedCents int64) int64 {
	if grossCents <= 0 || refundedCents <= 0 || commissionCents <= 0 {
		return 0
	}
	if refundedCents >= grossCents {
		return commissionCents
	}
	return commissionCents * refundedCents / grossCents
}

package payment

import (
	"time"

	"github.com/google/uuid"
)

// PaymentInfo holds Stripe Connect account data for a user.
// All KYC/billing data is now collected by Stripe Embedded Components directly.
type PaymentInfo struct {
	ID     uuid.UUID
	UserID uuid.UUID

	// Stripe Connect
	StripeAccountID    string
	StripeVerified     bool
	ChargesEnabled     bool
	PayoutsEnabled     bool
	StripeBusinessType string
	StripeCountry      string
	StripeDisplayName  string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewPaymentInfo creates a minimal PaymentInfo for a new user.
func NewPaymentInfo(userID uuid.UUID) *PaymentInfo {
	now := time.Now()
	return &PaymentInfo{
		ID:        uuid.New(),
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetStripeAccount records the connected account ID.
func (p *PaymentInfo) SetStripeAccount(accountID string) {
	p.StripeAccountID = accountID
	p.UpdatedAt = time.Now()
}

// MarkStripeVerified marks the account as fully verified.
func (p *PaymentInfo) MarkStripeVerified() {
	p.StripeVerified = true
	p.UpdatedAt = time.Now()
}

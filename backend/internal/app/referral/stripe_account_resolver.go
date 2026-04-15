package referral

import (
	"context"

	"github.com/google/uuid"
)

// StripeAccountResolver returns the Stripe Connect account id for a given user
// id, or empty string when the user has not completed KYC yet. The empty case
// is the trigger for "park the commission as pending_kyc".
//
// Defined as a port so the referral feature is decoupled from the embedded /
// payment_info / organization features that physically own the account id.
// The wiring in cmd/api/main.go injects a thin adapter that pulls from the
// right table.
type StripeAccountResolver interface {
	ResolveStripeAccountID(ctx context.Context, userID uuid.UUID) (string, error)
}

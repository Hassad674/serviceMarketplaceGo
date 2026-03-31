package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/payment"
)

type PaymentInfoRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*payment.PaymentInfo, error)
	Upsert(ctx context.Context, info *payment.PaymentInfo) error
	UpdateStripeFields(ctx context.Context, userID uuid.UUID, stripeAccountID string, stripeVerified bool) error
	UpdateStripeSyncFields(ctx context.Context, userID uuid.UUID, input StripeSyncInput) error
	GetByStripeAccountID(ctx context.Context, stripeAccountID string) (*payment.PaymentInfo, error)
}

// StripeSyncInput holds fields synced from Stripe account.updated webhook.
type StripeSyncInput struct {
	ChargesEnabled   bool
	PayoutsEnabled   bool
	StripeVerified   bool
	BusinessType     string
	Country          string
	DisplayName      string
}

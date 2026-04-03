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
	GetByStripeAccountID(ctx context.Context, stripeAccountID string) (*payment.PaymentInfo, error)
	UpdateAccountStatus(ctx context.Context, userID uuid.UUID, chargesEnabled, payoutsEnabled bool) error
}

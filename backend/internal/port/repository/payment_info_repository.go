package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/payment"
)

type PaymentInfoRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*payment.PaymentInfo, error)
	Upsert(ctx context.Context, info *payment.PaymentInfo) error
}

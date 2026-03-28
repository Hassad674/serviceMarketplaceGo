package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/payment"
)

type PaymentRecordRepository interface {
	Create(ctx context.Context, record *payment.PaymentRecord) error
	GetByProposalID(ctx context.Context, proposalID uuid.UUID) (*payment.PaymentRecord, error)
	GetByPaymentIntentID(ctx context.Context, paymentIntentID string) (*payment.PaymentRecord, error)
	Update(ctx context.Context, record *payment.PaymentRecord) error
}

package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/payment"
)

type PaymentRecordRepository interface {
	Create(ctx context.Context, record *payment.PaymentRecord) error
	GetByProposalID(ctx context.Context, proposalID uuid.UUID) (*payment.PaymentRecord, error)
	// GetByMilestoneID is the phase-4 idempotency key for
	// CreatePaymentIntent — every payment is scoped to one milestone.
	GetByMilestoneID(ctx context.Context, milestoneID uuid.UUID) (*payment.PaymentRecord, error)
	GetByPaymentIntentID(ctx context.Context, paymentIntentID string) (*payment.PaymentRecord, error)
	ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]*payment.PaymentRecord, error)
	Update(ctx context.Context, record *payment.PaymentRecord) error
}

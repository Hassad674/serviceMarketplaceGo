package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/payment"
)

type IdentityDocumentRepository interface {
	Create(ctx context.Context, doc *payment.IdentityDocument) error
	GetByID(ctx context.Context, id uuid.UUID) (*payment.IdentityDocument, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*payment.IdentityDocument, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, rejectionReason string) error
	UpdateStripeFileID(ctx context.Context, id uuid.UUID, stripeFileID string) error
	GetByUserAndTypeSide(ctx context.Context, userID uuid.UUID, category, docType, side string) (*payment.IdentityDocument, error)
}

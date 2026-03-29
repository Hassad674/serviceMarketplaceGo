package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/payment"
)

type BusinessPersonRepository interface {
	Create(ctx context.Context, person *payment.BusinessPerson) error
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*payment.BusinessPerson, error)
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

package payment

import (
	"context"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/payment"
)

type mockPaymentInfoRepo struct {
	getByUserIDFn func(ctx context.Context, userID uuid.UUID) (*domain.PaymentInfo, error)
	upsertFn      func(ctx context.Context, info *domain.PaymentInfo) error
}

func (m *mockPaymentInfoRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.PaymentInfo, error) {
	if m.getByUserIDFn != nil {
		return m.getByUserIDFn(ctx, userID)
	}
	return nil, domain.ErrNotFound
}

func (m *mockPaymentInfoRepo) Upsert(ctx context.Context, info *domain.PaymentInfo) error {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, info)
	}
	return nil
}

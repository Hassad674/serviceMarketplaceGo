package payment

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "marketplace-backend/internal/domain/payment"
)

func TestGetPaymentInfo_NotFound(t *testing.T) {
	repo := &mockPaymentInfoRepo{}
	svc := NewService(repo, nil, nil, nil, "")

	info, err := svc.GetPaymentInfo(context.Background(), uuid.New())

	assert.NoError(t, err)
	assert.Nil(t, info, "should return nil when not found")
}

func TestGetPaymentInfo_Found(t *testing.T) {
	userID := uuid.New()
	expected := &domain.PaymentInfo{
		UserID:          userID,
		StripeAccountID: "acct_test",
		StripeVerified:  true,
	}

	repo := &mockPaymentInfoRepo{
		getByUserIDFn: func(_ context.Context, id uuid.UUID) (*domain.PaymentInfo, error) {
			if id == userID {
				return expected, nil
			}
			return nil, domain.ErrNotFound
		},
	}
	svc := NewService(repo, nil, nil, nil, "")

	info, err := svc.GetPaymentInfo(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, "acct_test", info.StripeAccountID)
	assert.True(t, info.StripeVerified)
}

func TestIsComplete_NotFound(t *testing.T) {
	repo := &mockPaymentInfoRepo{}
	svc := NewService(repo, nil, nil, nil, "")

	complete, err := svc.IsComplete(context.Background(), uuid.New())

	assert.NoError(t, err)
	assert.False(t, complete)
}

func TestIsComplete_Verified(t *testing.T) {
	userID := uuid.New()
	repo := &mockPaymentInfoRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentInfo, error) {
			return &domain.PaymentInfo{
				StripeVerified: true,
			}, nil
		},
	}
	svc := NewService(repo, nil, nil, nil, "")

	complete, err := svc.IsComplete(context.Background(), userID)

	assert.NoError(t, err)
	assert.True(t, complete)
}

func TestIsComplete_NotVerified(t *testing.T) {
	userID := uuid.New()
	repo := &mockPaymentInfoRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentInfo, error) {
			return &domain.PaymentInfo{
				StripeVerified: false,
			}, nil
		},
	}
	svc := NewService(repo, nil, nil, nil, "")

	complete, err := svc.IsComplete(context.Background(), userID)

	assert.NoError(t, err)
	assert.False(t, complete)
}

func TestCreateAccountSession_NoStripe(t *testing.T) {
	repo := &mockPaymentInfoRepo{}
	svc := NewService(repo, nil, nil, nil, "")

	result, err := svc.CreateAccountSession(context.Background(), uuid.New(), "test@example.com")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "stripe not configured")
}

func TestCreateAccountSession_NewAccount(t *testing.T) {
	userID := uuid.New()
	repo := &mockPaymentInfoRepo{
		upsertFn: func(_ context.Context, _ *domain.PaymentInfo) error {
			return nil
		},
	}
	stripeMock := &mockStripeService{
		createMinimalAccountFn: func(_ context.Context, _, _ string) (string, error) {
			return "acct_new", nil
		},
		createAccountSessionFn: func(_ context.Context, _ string) (string, error) {
			return "cas_secret_123", nil
		},
	}
	svc := NewService(repo, nil, stripeMock, nil, "")

	result, err := svc.CreateAccountSession(context.Background(), userID, "test@example.com")

	require.NoError(t, err)
	assert.Equal(t, "cas_secret_123", result.ClientSecret)
	assert.Equal(t, "acct_new", result.StripeAccountID)
}

func TestCreateAccountSession_ExistingAccount(t *testing.T) {
	userID := uuid.New()
	repo := &mockPaymentInfoRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentInfo, error) {
			return &domain.PaymentInfo{
				StripeAccountID: "acct_existing",
			}, nil
		},
	}
	stripeMock := &mockStripeService{
		createAccountSessionFn: func(_ context.Context, accountID string) (string, error) {
			assert.Equal(t, "acct_existing", accountID)
			return "cas_secret_456", nil
		},
	}
	svc := NewService(repo, nil, stripeMock, nil, "")

	result, err := svc.CreateAccountSession(context.Background(), userID, "test@example.com")

	require.NoError(t, err)
	assert.Equal(t, "cas_secret_456", result.ClientSecret)
	assert.Equal(t, "acct_existing", result.StripeAccountID)
}

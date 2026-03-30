package payment

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "marketplace-backend/internal/domain/payment"
)

func validSaveInput() SavePaymentInfoInput {
	return SavePaymentInfoInput{
		FirstName:     "Alice",
		LastName:      "Dupont",
		DateOfBirth:   time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
		Nationality:   "FR",
		Address:       "10 rue de la Paix",
		City:          "Paris",
		PostalCode:    "75001",
		AccountHolder: "Alice Dupont",
		IBAN:          "FR7630001007941234567890185",
	}
}

func TestGetPaymentInfo_NotFound(t *testing.T) {
	repo := &mockPaymentInfoRepo{}
	svc := NewService(repo, nil, nil, nil, nil, nil, nil, "")

	info, err := svc.GetPaymentInfo(context.Background(), uuid.New())

	assert.NoError(t, err)
	assert.Nil(t, info, "should return nil when not found")
}

func TestGetPaymentInfo_Found(t *testing.T) {
	userID := uuid.New()
	expected := &domain.PaymentInfo{
		UserID:    userID,
		FirstName: "Alice",
	}

	repo := &mockPaymentInfoRepo{
		getByUserIDFn: func(_ context.Context, id uuid.UUID) (*domain.PaymentInfo, error) {
			if id == userID {
				return expected, nil
			}
			return nil, domain.ErrNotFound
		},
	}
	svc := NewService(repo, nil, nil, nil, nil, nil, nil, "")

	info, err := svc.GetPaymentInfo(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, "Alice", info.FirstName)
}

func TestSavePaymentInfo_Success(t *testing.T) {
	var persisted *domain.PaymentInfo
	repo := &mockPaymentInfoRepo{
		upsertFn: func(_ context.Context, info *domain.PaymentInfo) error {
			persisted = info
			return nil
		},
	}
	svc := NewService(repo, nil, nil, nil, nil, nil, nil, "")

	info, err := svc.SavePaymentInfo(context.Background(), uuid.New(), validSaveInput(), "", "")

	require.NoError(t, err)
	require.NotNil(t, info)
	assert.NotNil(t, persisted, "upsert should have been called")
	assert.Equal(t, "Alice", info.FirstName)
}

func TestSavePaymentInfo_ValidationError(t *testing.T) {
	repo := &mockPaymentInfoRepo{}
	svc := NewService(repo, nil, nil, nil, nil, nil, nil, "")

	input := validSaveInput()
	input.FirstName = ""

	info, err := svc.SavePaymentInfo(context.Background(), uuid.New(), input, "", "")

	assert.Nil(t, info)
	assert.ErrorIs(t, err, domain.ErrFirstNameRequired)
}

func TestIsComplete_NotFound(t *testing.T) {
	repo := &mockPaymentInfoRepo{}
	svc := NewService(repo, nil, nil, nil, nil, nil, nil, "")

	complete, err := svc.IsComplete(context.Background(), uuid.New())

	assert.NoError(t, err)
	assert.False(t, complete)
}

func TestIsComplete_Complete(t *testing.T) {
	userID := uuid.New()
	repo := &mockPaymentInfoRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentInfo, error) {
			return &domain.PaymentInfo{
				FirstName:     "Alice",
				LastName:      "Dupont",
				DateOfBirth:   time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
				Nationality:   "FR",
				Address:       "10 rue",
				City:          "Paris",
				PostalCode:    "75001",
				AccountHolder: "Alice Dupont",
				IBAN:          "FR76...",
			}, nil
		},
	}
	svc := NewService(repo, nil, nil, nil, nil, nil, nil, "")

	complete, err := svc.IsComplete(context.Background(), userID)

	assert.NoError(t, err)
	assert.True(t, complete)
}

func TestIsComplete_Incomplete(t *testing.T) {
	userID := uuid.New()
	repo := &mockPaymentInfoRepo{
		getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentInfo, error) {
			return &domain.PaymentInfo{
				FirstName: "Alice",
				// Missing other required fields
			}, nil
		},
	}
	svc := NewService(repo, nil, nil, nil, nil, nil, nil, "")

	complete, err := svc.IsComplete(context.Background(), userID)

	assert.NoError(t, err)
	assert.False(t, complete)
}

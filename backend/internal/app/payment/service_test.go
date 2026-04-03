package payment

import (
	"context"
	"fmt"
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
	svc := NewService(ServiceDeps{Payments: repo})

	info, persons, err := svc.GetPaymentInfo(context.Background(), uuid.New())

	assert.NoError(t, err)
	assert.Nil(t, info, "should return nil when not found")
	assert.Nil(t, persons)
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
	svc := NewService(ServiceDeps{Payments: repo})

	info, _, err := svc.GetPaymentInfo(context.Background(), userID)

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
	svc := NewService(ServiceDeps{Payments: repo})

	info, stripeErr, err := svc.SavePaymentInfo(context.Background(), uuid.New(), validSaveInput(), "", "")

	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Empty(t, stripeErr, "no stripe error expected without stripe service")
	assert.NotNil(t, persisted, "upsert should have been called")
	assert.Equal(t, "Alice", info.FirstName)
}

func TestSavePaymentInfo_ValidationError(t *testing.T) {
	repo := &mockPaymentInfoRepo{}
	svc := NewService(ServiceDeps{Payments: repo})

	input := validSaveInput()
	input.FirstName = ""

	info, stripeErr, err := svc.SavePaymentInfo(context.Background(), uuid.New(), input, "", "")

	assert.Nil(t, info)
	assert.Empty(t, stripeErr)
	assert.ErrorIs(t, err, domain.ErrFirstNameRequired)
}

func TestExtractStripeMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "stripe json error with message",
			err:  fmt.Errorf(`create stripe account: {"status":400,"message":"You must use a test bank account number.","type":"invalid_request_error"}`),
			want: "You must use a test bank account number.",
		},
		{
			name: "plain error without json",
			err:  fmt.Errorf("connection refused"),
			want: "connection refused",
		},
		{
			name: "json without message field",
			err:  fmt.Errorf(`stripe error: {"status":500,"type":"api_error"}`),
			want: `stripe error: {"status":500,"type":"api_error"}`,
		},
		{
			name: "nested json with message",
			err:  fmt.Errorf(`update account: {"message":"The IBAN you provided is invalid."}`),
			want: "The IBAN you provided is invalid.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractStripeMessage(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsComplete_NotFound(t *testing.T) {
	repo := &mockPaymentInfoRepo{}
	svc := NewService(ServiceDeps{Payments: repo})

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
	svc := NewService(ServiceDeps{Payments: repo})

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
	svc := NewService(ServiceDeps{Payments: repo})

	complete, err := svc.IsComplete(context.Background(), userID)

	assert.NoError(t, err)
	assert.False(t, complete)
}

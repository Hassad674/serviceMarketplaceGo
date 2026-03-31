package payment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

type Service struct {
	payments      repository.PaymentInfoRepository
	records       repository.PaymentRecordRepository
	stripe        service.StripeService      // nil if Stripe not configured
	notifications service.NotificationSender // nil if not configured
	frontendURL   string
}

func NewService(
	payments repository.PaymentInfoRepository,
	records repository.PaymentRecordRepository,
	stripe service.StripeService,
	notifications service.NotificationSender,
	frontendURL string,
) *Service {
	return &Service{
		payments:      payments,
		records:       records,
		stripe:        stripe,
		notifications: notifications,
		frontendURL:   frontendURL,
	}
}

// GetPaymentInfo returns the payment info for the user, or nil if not found.
func (s *Service) GetPaymentInfo(ctx context.Context, userID uuid.UUID) (*domain.PaymentInfo, error) {
	info, err := s.payments.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get payment info: %w", err)
	}
	return info, nil
}

// IsComplete checks whether the user has a verified Stripe account.
func (s *Service) IsComplete(ctx context.Context, userID uuid.UUID) (bool, error) {
	info, err := s.payments.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("check payment info completeness: %w", err)
	}
	return info.StripeVerified, nil
}

// AccountSessionResult holds the result of creating an account session.
type AccountSessionResult struct {
	ClientSecret    string `json:"client_secret"`
	StripeAccountID string `json:"stripe_account_id"`
}

// CreateAccountSession creates a Stripe account session for embedded onboarding.
// If the user has no Stripe account, it creates a minimal one first.
func (s *Service) CreateAccountSession(ctx context.Context, userID uuid.UUID, email string) (*AccountSessionResult, error) {
	if s.stripe == nil {
		return nil, fmt.Errorf("stripe not configured")
	}

	info, err := s.payments.GetByUserID(ctx, userID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("get payment info: %w", err)
	}

	accountID := ""
	if info != nil {
		accountID = info.StripeAccountID
	}

	// Create minimal account if none exists
	if accountID == "" {
		accountID, err = s.stripe.CreateMinimalAccount(ctx, "FR", email)
		if err != nil {
			return nil, fmt.Errorf("create minimal account: %w", err)
		}
		if err := s.ensurePaymentInfoRow(ctx, userID, accountID); err != nil {
			return nil, fmt.Errorf("persist stripe account: %w", err)
		}
	}

	clientSecret, err := s.stripe.CreateAccountSession(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("create account session: %w", err)
	}

	return &AccountSessionResult{
		ClientSecret:    clientSecret,
		StripeAccountID: accountID,
	}, nil
}

// ensurePaymentInfoRow creates or updates the payment_info row with the Stripe account ID.
func (s *Service) ensurePaymentInfoRow(ctx context.Context, userID uuid.UUID, accountID string) error {
	_, err := s.payments.GetByUserID(ctx, userID)
	if errors.Is(err, domain.ErrNotFound) {
		info := domain.NewPaymentInfo(userID)
		info.StripeAccountID = accountID
		return s.payments.Upsert(ctx, info)
	}
	if err != nil {
		return err
	}
	return s.payments.UpdateStripeFields(ctx, userID, accountID, false)
}

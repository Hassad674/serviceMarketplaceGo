package payment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/repository"
)

type Service struct {
	payments repository.PaymentInfoRepository
}

func NewService(payments repository.PaymentInfoRepository) *Service {
	return &Service{payments: payments}
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

// SavePaymentInfoInput holds the data needed to create or update payment info.
type SavePaymentInfoInput struct {
	FirstName   string
	LastName    string
	DateOfBirth time.Time
	Nationality string
	Address     string
	City        string
	PostalCode  string

	IsBusiness         bool
	BusinessName       string
	BusinessAddress    string
	BusinessCity       string
	BusinessPostalCode string
	BusinessCountry    string
	TaxID              string
	VATNumber          string
	RoleInCompany      string

	IBAN          string
	BIC           string
	AccountNumber string
	RoutingNumber string
	AccountHolder string
	BankCountry   string
}

// SavePaymentInfo validates and upserts the payment info for the user.
func (s *Service) SavePaymentInfo(ctx context.Context, userID uuid.UUID, input SavePaymentInfoInput) (*domain.PaymentInfo, error) {
	info, err := domain.NewPaymentInfo(domain.NewPaymentInfoInput{
		UserID:             userID,
		FirstName:          input.FirstName,
		LastName:           input.LastName,
		DateOfBirth:        input.DateOfBirth,
		Nationality:        input.Nationality,
		Address:            input.Address,
		City:               input.City,
		PostalCode:         input.PostalCode,
		IsBusiness:         input.IsBusiness,
		BusinessName:       input.BusinessName,
		BusinessAddress:    input.BusinessAddress,
		BusinessCity:       input.BusinessCity,
		BusinessPostalCode: input.BusinessPostalCode,
		BusinessCountry:    input.BusinessCountry,
		TaxID:              input.TaxID,
		VATNumber:          input.VATNumber,
		RoleInCompany:      input.RoleInCompany,
		IBAN:               input.IBAN,
		BIC:                input.BIC,
		AccountNumber:      input.AccountNumber,
		RoutingNumber:      input.RoutingNumber,
		AccountHolder:      input.AccountHolder,
		BankCountry:        input.BankCountry,
	})
	if err != nil {
		return nil, err
	}

	if err := s.payments.Upsert(ctx, info); err != nil {
		return nil, fmt.Errorf("save payment info: %w", err)
	}

	return info, nil
}

// IsComplete checks whether the user has complete payment info on file.
func (s *Service) IsComplete(ctx context.Context, userID uuid.UUID) (bool, error) {
	info, err := s.payments.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("check payment info completeness: %w", err)
	}
	return info.IsComplete(), nil
}

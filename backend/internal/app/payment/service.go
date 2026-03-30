package payment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

type Service struct {
	payments      repository.PaymentInfoRepository
	records       repository.PaymentRecordRepository
	documents     repository.IdentityDocumentRepository
	persons       repository.BusinessPersonRepository
	stripe        service.StripeService        // nil if Stripe not configured
	storage       service.StorageService       // nil if not configured
	notifications service.NotificationSender   // nil if not configured
	frontendURL   string
}

func NewService(
	payments repository.PaymentInfoRepository,
	records repository.PaymentRecordRepository,
	documents repository.IdentityDocumentRepository,
	persons repository.BusinessPersonRepository,
	stripe service.StripeService,
	storage service.StorageService,
	notifications service.NotificationSender,
	frontendURL string,
) *Service {
	return &Service{
		payments:      payments,
		records:       records,
		documents:     documents,
		persons:       persons,
		stripe:        stripe,
		storage:       storage,
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

	Phone          string
	ActivitySector string

	// Business KYC flags
	IsSelfRepresentative bool
	IsSelfDirector       bool
	NoMajorOwners        bool
	IsSelfExecutive      bool
	BusinessPersons      []BusinessPersonInput

	IBAN          string
	BIC           string
	AccountNumber string
	RoutingNumber string
	AccountHolder string
	BankCountry   string
}

type BusinessPersonInput struct {
	Role        string
	FirstName   string
	LastName    string
	DateOfBirth time.Time
	Email       string
	Phone       string
	Address     string
	City        string
	PostalCode  string
	Title       string
}

// SavePaymentInfo validates and upserts the payment info for the user.
func (s *Service) SavePaymentInfo(ctx context.Context, userID uuid.UUID, input SavePaymentInfoInput, tosIP string, email string) (*domain.PaymentInfo, error) {
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
		Phone:                input.Phone,
		ActivitySector:       input.ActivitySector,
		IsSelfRepresentative: input.IsSelfRepresentative,
		IsSelfDirector:       input.IsSelfDirector,
		NoMajorOwners:        input.NoMajorOwners,
		IsSelfExecutive:      input.IsSelfExecutive,
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

	// Save business persons (clear and re-create)
	if info.IsBusiness && s.persons != nil {
		_ = s.persons.DeleteByUserID(ctx, userID)
		for _, bp := range input.BusinessPersons {
			person, pErr := domain.NewBusinessPerson(domain.NewBusinessPersonInput{
				UserID:      userID,
				Role:        bp.Role,
				FirstName:   bp.FirstName,
				LastName:    bp.LastName,
				DateOfBirth: bp.DateOfBirth,
				Email:       bp.Email,
				Phone:       bp.Phone,
				Address:     bp.Address,
				City:        bp.City,
				PostalCode:  bp.PostalCode,
				Title:       bp.Title,
			})
			if pErr == nil {
				_ = s.persons.Create(ctx, person)
			}
		}
	}

	// Create Stripe connected account if configured and not already created
	s.ensureStripeAccount(ctx, info, tosIP, email)

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

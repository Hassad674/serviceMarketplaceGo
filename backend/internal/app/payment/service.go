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
	countrySpecs  service.CountrySpecService   // nil if not configured
	frontendURL   string
}

// ServiceDeps groups all dependencies for the payment service.
type ServiceDeps struct {
	Payments      repository.PaymentInfoRepository
	Records       repository.PaymentRecordRepository
	Documents     repository.IdentityDocumentRepository
	Persons       repository.BusinessPersonRepository
	Stripe        service.StripeService
	Storage       service.StorageService
	Notifications service.NotificationSender
	CountrySpecs  service.CountrySpecService
	FrontendURL   string
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		payments:      deps.Payments,
		records:       deps.Records,
		documents:     deps.Documents,
		persons:       deps.Persons,
		stripe:        deps.Stripe,
		storage:       deps.Storage,
		notifications: deps.Notifications,
		countrySpecs:  deps.CountrySpecs,
		frontendURL:   deps.FrontendURL,
	}
}

// GetPaymentInfo returns the payment info for the user, or nil if not found.
func (s *Service) GetPaymentInfo(ctx context.Context, userID uuid.UUID) (*domain.PaymentInfo, []*domain.BusinessPerson, error) {
	info, err := s.payments.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("get payment info: %w", err)
	}

	var persons []*domain.BusinessPerson
	if info.IsBusiness && s.persons != nil {
		persons, _ = s.persons.ListByUserID(ctx, userID)
	}

	return info, persons, nil
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

	Country     string
	ExtraFields map[string]string
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
		Country:            input.Country,
		ExtraFields:        input.ExtraFields,
	})
	if err != nil {
		return nil, err
	}

	// Preserve existing Stripe account ID before upserting
	existing, _ := s.payments.GetByUserID(ctx, userID)
	if existing != nil && existing.StripeAccountID != "" {
		info.StripeAccountID = existing.StripeAccountID
		info.StripeVerified = existing.StripeVerified
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

	// Create or update Stripe connected account
	if info.StripeAccountID != "" {
		s.updateStripeAccount(ctx, info, tosIP, email)
	} else {
		s.ensureStripeAccount(ctx, info, tosIP, email)
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

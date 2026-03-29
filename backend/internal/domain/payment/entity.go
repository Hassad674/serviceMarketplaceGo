package payment

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// PaymentInfo holds payment and billing details for a user.
type PaymentInfo struct {
	ID     uuid.UUID
	UserID uuid.UUID

	// Personal / Representative
	FirstName   string
	LastName    string
	DateOfBirth time.Time
	Nationality string
	Address     string
	City        string
	PostalCode  string

	// Business (optional)
	IsBusiness      bool
	BusinessName    string
	BusinessAddress string
	BusinessCity    string
	BusinessPostalCode string
	BusinessCountry string
	TaxID           string
	VATNumber       string
	RoleInCompany   string

	// Contact & KYC
	Phone          string
	ActivitySector string // MCC code

	// Bank account
	IBAN          string
	BIC           string
	AccountNumber string
	RoutingNumber string
	AccountHolder string
	BankCountry   string

	// Stripe Connect (future)
	StripeAccountID string
	StripeVerified  bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewPaymentInfoInput contains the fields required to create or update payment info.
type NewPaymentInfoInput struct {
	UserID      uuid.UUID
	FirstName   string
	LastName    string
	DateOfBirth time.Time
	Nationality string
	Address     string
	City        string
	PostalCode  string

	IsBusiness      bool
	BusinessName    string
	BusinessAddress string
	BusinessCity    string
	BusinessPostalCode string
	BusinessCountry string
	TaxID           string
	VATNumber       string
	RoleInCompany   string

	Phone          string
	ActivitySector string

	IBAN          string
	BIC           string
	AccountNumber string
	RoutingNumber string
	AccountHolder string
	BankCountry   string
}

// NewPaymentInfo validates the input and returns a PaymentInfo entity.
func NewPaymentInfo(input NewPaymentInfoInput) (*PaymentInfo, error) {
	if err := validateRequired(input); err != nil {
		return nil, err
	}

	if err := validateBusiness(input); err != nil {
		return nil, err
	}

	if err := validateBankDetails(input); err != nil {
		return nil, err
	}

	now := time.Now()
	return &PaymentInfo{
		ID:                 uuid.New(),
		UserID:             input.UserID,
		FirstName:          strings.TrimSpace(input.FirstName),
		LastName:           strings.TrimSpace(input.LastName),
		DateOfBirth:        input.DateOfBirth,
		Nationality:        strings.TrimSpace(input.Nationality),
		Address:            strings.TrimSpace(input.Address),
		City:               strings.TrimSpace(input.City),
		PostalCode:         strings.TrimSpace(input.PostalCode),
		IsBusiness:         input.IsBusiness,
		BusinessName:       strings.TrimSpace(input.BusinessName),
		BusinessAddress:    strings.TrimSpace(input.BusinessAddress),
		BusinessCity:       strings.TrimSpace(input.BusinessCity),
		BusinessPostalCode: strings.TrimSpace(input.BusinessPostalCode),
		BusinessCountry:    strings.TrimSpace(input.BusinessCountry),
		TaxID:              strings.TrimSpace(input.TaxID),
		VATNumber:          strings.TrimSpace(input.VATNumber),
		RoleInCompany:      strings.TrimSpace(input.RoleInCompany),
		Phone:              strings.TrimSpace(input.Phone),
		ActivitySector:     input.ActivitySector,
		IBAN:               strings.TrimSpace(input.IBAN),
		BIC:                strings.TrimSpace(input.BIC),
		AccountNumber:      strings.TrimSpace(input.AccountNumber),
		RoutingNumber:      strings.TrimSpace(input.RoutingNumber),
		AccountHolder:      strings.TrimSpace(input.AccountHolder),
		BankCountry:        strings.TrimSpace(input.BankCountry),
		StripeVerified:     false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}, nil
}

// IsComplete returns true when all mandatory fields are filled.
func (p *PaymentInfo) IsComplete() bool {
	personal := p.FirstName != "" &&
		p.LastName != "" &&
		!p.DateOfBirth.IsZero() &&
		p.Nationality != "" &&
		p.Address != "" &&
		p.City != "" &&
		p.PostalCode != ""

	if !personal {
		return false
	}

	if p.IsBusiness && (p.BusinessName == "" || p.TaxID == "") {
		return false
	}

	bank := p.AccountHolder != "" &&
		(p.IBAN != "" || (p.AccountNumber != "" && p.RoutingNumber != ""))

	return bank
}

func validateRequired(input NewPaymentInfoInput) error {
	if strings.TrimSpace(input.FirstName) == "" {
		return ErrFirstNameRequired
	}
	if strings.TrimSpace(input.LastName) == "" {
		return ErrLastNameRequired
	}
	if input.DateOfBirth.IsZero() {
		return ErrDateOfBirthRequired
	}
	if strings.TrimSpace(input.Nationality) == "" {
		return ErrNationalityRequired
	}
	if strings.TrimSpace(input.Address) == "" {
		return ErrAddressRequired
	}
	if strings.TrimSpace(input.City) == "" {
		return ErrCityRequired
	}
	if strings.TrimSpace(input.PostalCode) == "" {
		return ErrPostalCodeRequired
	}
	if strings.TrimSpace(input.AccountHolder) == "" {
		return ErrAccountHolderRequired
	}
	return nil
}

func validateBusiness(input NewPaymentInfoInput) error {
	if !input.IsBusiness {
		return nil
	}
	if strings.TrimSpace(input.BusinessName) == "" {
		return ErrBusinessNameRequired
	}
	if strings.TrimSpace(input.TaxID) == "" {
		return ErrTaxIDRequired
	}
	return nil
}

func (p *PaymentInfo) SetStripeAccount(accountID string) {
	p.StripeAccountID = accountID
	p.UpdatedAt = time.Now()
}

func (p *PaymentInfo) MarkStripeVerified() {
	p.StripeVerified = true
	p.UpdatedAt = time.Now()
}

func validateBankDetails(input NewPaymentInfoInput) error {
	hasIBAN := strings.TrimSpace(input.IBAN) != ""
	hasLocal := strings.TrimSpace(input.AccountNumber) != "" &&
		strings.TrimSpace(input.RoutingNumber) != ""

	if !hasIBAN && !hasLocal {
		return ErrBankDetailsRequired
	}
	return nil
}

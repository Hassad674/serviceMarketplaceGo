package payment

import "errors"

var (
	ErrNotFound             = errors.New("payment info not found")
	ErrFirstNameRequired    = errors.New("first name is required")
	ErrLastNameRequired     = errors.New("last name is required")
	ErrDateOfBirthRequired  = errors.New("date of birth is required")
	ErrNationalityRequired  = errors.New("nationality is required")
	ErrAddressRequired      = errors.New("address is required")
	ErrCityRequired         = errors.New("city is required")
	ErrPostalCodeRequired   = errors.New("postal code is required")
	ErrAccountHolderRequired = errors.New("account holder is required")
	ErrBankDetailsRequired  = errors.New("IBAN or account number + routing number is required")
	ErrBusinessNameRequired = errors.New("business name is required when is_business is true")
	ErrTaxIDRequired        = errors.New("tax ID is required when is_business is true")
)

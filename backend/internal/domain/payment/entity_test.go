package payment

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validInput() NewPaymentInfoInput {
	return NewPaymentInfoInput{
		UserID:        uuid.New(),
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

func TestNewPaymentInfo_Success(t *testing.T) {
	input := validInput()

	info, err := NewPaymentInfo(input)

	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, input.UserID, info.UserID)
	assert.Equal(t, "Alice", info.FirstName)
	assert.Equal(t, "Dupont", info.LastName)
	assert.False(t, info.CreatedAt.IsZero())
	assert.False(t, info.UpdatedAt.IsZero())
}

func TestNewPaymentInfo_TrimsWhitespace(t *testing.T) {
	input := validInput()
	input.FirstName = "  Alice  "
	input.City = "  Paris  "

	info, err := NewPaymentInfo(input)

	require.NoError(t, err)
	assert.Equal(t, "Alice", info.FirstName)
	assert.Equal(t, "Paris", info.City)
}

func TestNewPaymentInfo_RequiredFieldErrors(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*NewPaymentInfoInput)
		wantErr error
	}{
		{"missing first_name", func(i *NewPaymentInfoInput) { i.FirstName = "" }, ErrFirstNameRequired},
		{"missing last_name", func(i *NewPaymentInfoInput) { i.LastName = "" }, ErrLastNameRequired},
		{"missing date_of_birth", func(i *NewPaymentInfoInput) { i.DateOfBirth = time.Time{} }, ErrDateOfBirthRequired},
		{"missing nationality", func(i *NewPaymentInfoInput) { i.Nationality = "" }, ErrNationalityRequired},
		{"missing address", func(i *NewPaymentInfoInput) { i.Address = "" }, ErrAddressRequired},
		{"missing city", func(i *NewPaymentInfoInput) { i.City = "" }, ErrCityRequired},
		{"missing postal_code", func(i *NewPaymentInfoInput) { i.PostalCode = "" }, ErrPostalCodeRequired},
		{"missing account_holder", func(i *NewPaymentInfoInput) { i.AccountHolder = "" }, ErrAccountHolderRequired},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := validInput()
			tc.modify(&input)

			info, err := NewPaymentInfo(input)

			assert.Nil(t, info)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestNewPaymentInfo_BusinessValidation(t *testing.T) {
	t.Run("business requires business_name", func(t *testing.T) {
		input := validInput()
		input.IsBusiness = true
		input.TaxID = "12345"

		info, err := NewPaymentInfo(input)

		assert.Nil(t, info)
		assert.ErrorIs(t, err, ErrBusinessNameRequired)
	})

	t.Run("business requires tax_id", func(t *testing.T) {
		input := validInput()
		input.IsBusiness = true
		input.BusinessName = "ACME Corp"

		info, err := NewPaymentInfo(input)

		assert.Nil(t, info)
		assert.ErrorIs(t, err, ErrTaxIDRequired)
	})

	t.Run("business with all fields succeeds", func(t *testing.T) {
		input := validInput()
		input.IsBusiness = true
		input.BusinessName = "ACME Corp"
		input.TaxID = "FR12345"

		info, err := NewPaymentInfo(input)

		require.NoError(t, err)
		assert.True(t, info.IsBusiness)
		assert.Equal(t, "ACME Corp", info.BusinessName)
	})
}

func TestNewPaymentInfo_BankDetailsValidation(t *testing.T) {
	t.Run("IBAN alone is valid", func(t *testing.T) {
		input := validInput()
		input.IBAN = "FR7630001007941234567890185"
		input.AccountNumber = ""
		input.RoutingNumber = ""

		_, err := NewPaymentInfo(input)
		assert.NoError(t, err)
	})

	t.Run("account_number + routing_number alone is valid", func(t *testing.T) {
		input := validInput()
		input.IBAN = ""
		input.AccountNumber = "123456789"
		input.RoutingNumber = "021000021"

		_, err := NewPaymentInfo(input)
		assert.NoError(t, err)
	})

	t.Run("neither IBAN nor local bank details fails", func(t *testing.T) {
		input := validInput()
		input.IBAN = ""
		input.AccountNumber = ""
		input.RoutingNumber = ""

		info, err := NewPaymentInfo(input)
		assert.Nil(t, info)
		assert.ErrorIs(t, err, ErrBankDetailsRequired)
	})
}

func TestIsComplete(t *testing.T) {
	t.Run("complete personal with IBAN", func(t *testing.T) {
		input := validInput()
		info, _ := NewPaymentInfo(input)
		assert.True(t, info.IsComplete())
	})

	t.Run("incomplete missing first_name", func(t *testing.T) {
		info := &PaymentInfo{
			LastName:      "Dupont",
			DateOfBirth:   time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			Nationality:   "FR",
			Address:       "10 rue",
			City:          "Paris",
			PostalCode:    "75001",
			AccountHolder: "Alice",
			IBAN:          "FR76...",
		}
		assert.False(t, info.IsComplete())
	})

	t.Run("business incomplete missing tax_id", func(t *testing.T) {
		input := validInput()
		info, _ := NewPaymentInfo(input)
		info.IsBusiness = true
		info.BusinessName = "ACME"
		info.TaxID = ""
		assert.False(t, info.IsComplete())
	})

	t.Run("incomplete missing bank details", func(t *testing.T) {
		input := validInput()
		info, _ := NewPaymentInfo(input)
		info.IBAN = ""
		info.AccountNumber = ""
		info.RoutingNumber = ""
		assert.False(t, info.IsComplete())
	})
}

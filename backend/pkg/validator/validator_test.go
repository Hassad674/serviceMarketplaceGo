package validator

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"marketplace-backend/internal/domain/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeJSON_ValidJSON(t *testing.T) {
	type payload struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	body := `{"name":"John","email":"john@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	var dst payload
	err := DecodeJSON(req, &dst)

	require.NoError(t, err)
	assert.Equal(t, "John", dst.Name)
	assert.Equal(t, "john@example.com", dst.Email)
}

func TestDecodeJSON_NilBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Body = nil // force truly nil body

	var dst struct{}
	err := DecodeJSON(req, &dst)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestDecodeJSON_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(""))

	var dst struct{}
	err := DecodeJSON(req, &dst)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestDecodeJSON_InvalidJSON(t *testing.T) {
	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))

	var dst struct {
		Name string `json:"name"`
	}
	err := DecodeJSON(req, &dst)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestDecodeJSON_UnknownFieldsRejected(t *testing.T) {
	body := `{"name":"John","unknown_field":"value"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))

	var dst struct {
		Name string `json:"name"`
	}
	err := DecodeJSON(req, &dst)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestValidateRequired_AllFieldsPresent(t *testing.T) {
	fields := map[string]string{
		"email":    "john@example.com",
		"password": "secret123",
		"name":     "John",
	}

	errs := ValidateRequired(fields)

	assert.Nil(t, errs)
}

func TestValidateRequired_MissingFields(t *testing.T) {
	fields := map[string]string{
		"email":    "",
		"password": "secret123",
		"name":     "",
	}

	errs := ValidateRequired(fields)

	require.NotNil(t, errs)
	assert.Len(t, errs, 2)
	assert.Contains(t, errs, "email")
	assert.Contains(t, errs, "name")
	assert.NotContains(t, errs, "password")
}

func TestValidateRequired_WhitespaceOnlyTreatedAsEmpty(t *testing.T) {
	fields := map[string]string{
		"name": "   ",
	}

	errs := ValidateRequired(fields)

	require.NotNil(t, errs)
	assert.Contains(t, errs, "name")
}

func TestValidateRequired_AllEmpty(t *testing.T) {
	fields := map[string]string{
		"email": "",
		"name":  "",
	}

	errs := ValidateRequired(fields)

	require.NotNil(t, errs)
	assert.Len(t, errs, 2)
}

func TestValidateRequired_EmptyMap(t *testing.T) {
	errs := ValidateRequired(map[string]string{})
	assert.Nil(t, errs)
}

func TestValidateEmail_ValidEmail(t *testing.T) {
	err := ValidateEmail("user@example.com")
	assert.NoError(t, err)
}

func TestValidateEmail_InvalidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{"empty", ""},
		{"no at sign", "userexample.com"},
		{"no domain", "user@"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			assert.ErrorIs(t, err, user.ErrInvalidEmail)
		})
	}
}

func TestValidatePassword_ValidPassword(t *testing.T) {
	err := ValidatePassword("StrongPass1")
	assert.NoError(t, err)
}

func TestValidatePassword_WeakPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"too short", "Short1"},
		{"no uppercase", "alllower1"},
		{"no digit", "NoDigitsHere"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			assert.ErrorIs(t, err, user.ErrWeakPassword)
		})
	}
}

func TestValidateRole_ValidRoles(t *testing.T) {
	validRoles := []string{"agency", "enterprise", "provider"}

	for _, role := range validRoles {
		t.Run(role, func(t *testing.T) {
			err := ValidateRole(role)
			assert.NoError(t, err)
		})
	}
}

func TestValidateRole_InvalidRoles(t *testing.T) {
	invalidRoles := []string{"", "admin", "invalid", "AGENCY", "Provider"}

	for _, role := range invalidRoles {
		t.Run("role_"+role, func(t *testing.T) {
			err := ValidateRole(role)
			assert.ErrorIs(t, err, user.ErrInvalidRole)
		})
	}
}

// ---------------------------------------------------------------------------
// SEC-19 — Validate() wrapper around go-playground/validator.
// ---------------------------------------------------------------------------

type sampleDTO struct {
	Title       string `json:"title" validate:"required,min=1,max=200"`
	Description string `json:"description" validate:"max=5000"`
	Email       string `json:"email" validate:"required,email"`
	UserID      string `json:"user_id" validate:"omitempty,uuid"`
	URL         string `json:"url" validate:"omitempty,url"`
	Amount      int64  `json:"amount" validate:"gte=0,lte=999999999"`
	Status      string `json:"status" validate:"oneof=open closed pending"`
}

func validSampleDTO() sampleDTO {
	return sampleDTO{
		Title:  "A valid title",
		Email:  "user@example.com",
		Amount: 1000,
		Status: "open",
	}
}

func TestValidate_HappyPath(t *testing.T) {
	err := Validate(validSampleDTO())
	assert.NoError(t, err)
}

func TestValidate_MissingRequired(t *testing.T) {
	dto := validSampleDTO()
	dto.Title = ""

	err := Validate(dto)
	require.Error(t, err)

	ve, ok := IsValidationError(err)
	require.True(t, ok)
	require.Len(t, ve.Fields, 1)
	assert.Equal(t, "title", ve.Fields[0].Field)
	assert.Equal(t, "required", ve.Fields[0].Rule)
}

func TestValidate_TooLong(t *testing.T) {
	dto := validSampleDTO()
	dto.Title = strings.Repeat("a", 201) // > 200

	err := Validate(dto)
	require.Error(t, err)

	ve, ok := IsValidationError(err)
	require.True(t, ok)
	require.Len(t, ve.Fields, 1)
	assert.Equal(t, "title", ve.Fields[0].Field)
	assert.Equal(t, "max", ve.Fields[0].Rule)
}

func TestValidate_InvalidEmail(t *testing.T) {
	dto := validSampleDTO()
	dto.Email = "not-an-email"

	err := Validate(dto)
	require.Error(t, err)

	ve, ok := IsValidationError(err)
	require.True(t, ok)
	assert.Equal(t, "email", ve.Fields[0].Field)
	assert.Equal(t, "email", ve.Fields[0].Rule)
}

func TestValidate_InvalidUUID(t *testing.T) {
	dto := validSampleDTO()
	dto.UserID = "not-a-uuid"

	err := Validate(dto)
	require.Error(t, err)

	ve, ok := IsValidationError(err)
	require.True(t, ok)
	assert.Equal(t, "userid", ve.Fields[0].Field)
	assert.Equal(t, "uuid", ve.Fields[0].Rule)
}

func TestValidate_InvalidURL(t *testing.T) {
	dto := validSampleDTO()
	dto.URL = "not a url"

	err := Validate(dto)
	require.Error(t, err)

	ve, ok := IsValidationError(err)
	require.True(t, ok)
	assert.Equal(t, "url", ve.Fields[0].Field)
	assert.Equal(t, "url", ve.Fields[0].Rule)
}

func TestValidate_AmountOutOfRange(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
		rule   string
	}{
		{"negative", -1, "gte"},
		{"too large", 1_000_000_000, "lte"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dto := validSampleDTO()
			dto.Amount = tt.amount
			err := Validate(dto)
			require.Error(t, err)
			ve, ok := IsValidationError(err)
			require.True(t, ok)
			assert.Equal(t, "amount", ve.Fields[0].Field)
			assert.Equal(t, tt.rule, ve.Fields[0].Rule)
		})
	}
}

func TestValidate_OneOf(t *testing.T) {
	dto := validSampleDTO()
	dto.Status = "unknown"

	err := Validate(dto)
	require.Error(t, err)
	ve, ok := IsValidationError(err)
	require.True(t, ok)
	assert.Equal(t, "status", ve.Fields[0].Field)
	assert.Equal(t, "oneof", ve.Fields[0].Rule)
}

func TestValidate_MultipleErrors(t *testing.T) {
	dto := sampleDTO{
		Title:  "",                      // required
		Email:  "bad",                   // email
		UserID: "no",                    // uuid
		Amount: -5,                      // gte
		Status: "x",                     // oneof
	}

	err := Validate(dto)
	require.Error(t, err)
	ve, ok := IsValidationError(err)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(ve.Fields), 5)
}

func TestValidate_OmitemptyAllowsEmptyOptional(t *testing.T) {
	dto := validSampleDTO()
	dto.UserID = ""
	dto.URL = ""

	err := Validate(dto)
	assert.NoError(t, err, "omitempty fields should accept empty strings")
}

func TestDecodeAndValidate_HappyPath(t *testing.T) {
	body := `{"title":"hi","email":"u@e.com","user_id":"","url":"","amount":1,"status":"open","description":""}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	var dto sampleDTO
	err := DecodeAndValidate(req, &dto)
	assert.NoError(t, err)
}

func TestDecodeAndValidate_DecodeFailure(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{not json}"))
	var dto sampleDTO
	err := DecodeAndValidate(req, &dto)
	require.Error(t, err)
	_, isVE := IsValidationError(err)
	assert.False(t, isVE, "decode errors should NOT be ValidationError")
}

func TestDecodeAndValidate_ValidationFailure(t *testing.T) {
	body := `{"title":"","email":"bad","user_id":"","url":"","amount":1,"status":"open","description":""}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	var dto sampleDTO
	err := DecodeAndValidate(req, &dto)
	require.Error(t, err)
	_, isVE := IsValidationError(err)
	assert.True(t, isVE, "validation failure should surface as ValidationError")
}

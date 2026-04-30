package validator

import (
	"bytes"
	"net/http"
	"net/http/httptest"
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
	// SEC-20: minimum 10 chars + special character required.
	err := ValidatePassword("StrongPass1!")
	assert.NoError(t, err)
}

func TestValidatePassword_WeakPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"too short", "Short1!"},
		{"no uppercase", "alllower1!"},
		{"no digit", "NoDigitsHere!"},
		{"no special character", "NoSpecial1A"}, // SEC-20 explicit case
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

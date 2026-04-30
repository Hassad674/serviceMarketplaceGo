package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmail_ValidEmails(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple email", "user@example.com", "user@example.com"},
		{"with subdomain", "user@mail.example.com", "user@mail.example.com"},
		{"with plus tag", "user+tag@example.com", "user+tag@example.com"},
		{"with dots in local", "first.last@example.com", "first.last@example.com"},
		{"with numbers", "user123@example.com", "user123@example.com"},
		{"with hyphens in domain", "user@my-domain.com", "user@my-domain.com"},
		{"uppercase gets lowered", "USER@EXAMPLE.COM", "user@example.com"},
		{"mixed case gets lowered", "John.Doe@Example.COM", "john.doe@example.com"},
		{"leading spaces trimmed", "  user@example.com", "user@example.com"},
		{"trailing spaces trimmed", "user@example.com  ", "user@example.com"},
		{"surrounding spaces trimmed", "  user@example.com  ", "user@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := NewEmail(tt.input)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, email.String())
		})
	}
}

func TestNewEmail_InvalidEmails(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"no at sign", "userexample.com"},
		{"no domain", "user@"},
		{"no local part", "@example.com"},
		{"no tld", "user@example"},
		{"double at", "user@@example.com"},
		{"spaces in middle", "user @example.com"},
		{"single tld char", "user@example.c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := NewEmail(tt.input)

			assert.ErrorIs(t, err, ErrInvalidEmail)
			assert.Empty(t, email.String())
		})
	}
}

func TestNewEmail_Lowercases(t *testing.T) {
	email, err := NewEmail("TEST@EXAMPLE.COM")
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", email.String())
}

func TestNewPassword_ValidPasswords(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		// SEC-20: minimum length is 10 and a special character is required.
		{"exact minimum length", "Abcdefgh1!"},
		{"longer password", "MySecurePassword123!"},
		{"with multiple specials", "P@ssw0rd!#"},
		{"very long", "ThisIsAVeryLongPasswordWithUpperLower123!"},
		{"unicode special — symbol category", "Passw0rdZ$"},
		{"non-ascii punctuation accepted", "Passw0rdZ¿"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pw, err := NewPassword(tt.input)

			require.NoError(t, err)
			assert.Equal(t, tt.input, pw.String())
		})
	}
}

func TestNewPassword_TooShort(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"single char", "A"},
		{"nine chars", "Abcdef1!Z"}, // SEC-20: 9 < 10, must reject
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pw, err := NewPassword(tt.input)

			assert.ErrorIs(t, err, ErrWeakPassword)
			assert.Empty(t, pw.String())
		})
	}
}

func TestNewPassword_NoUppercase(t *testing.T) {
	pw, err := NewPassword("abcdefghi1!")
	assert.ErrorIs(t, err, ErrWeakPassword)
	assert.Empty(t, pw.String())
}

func TestNewPassword_NoLowercase(t *testing.T) {
	pw, err := NewPassword("ABCDEFGHI1!")
	assert.ErrorIs(t, err, ErrWeakPassword)
	assert.Empty(t, pw.String())
}

func TestNewPassword_NoDigit(t *testing.T) {
	pw, err := NewPassword("Abcdefghij!")
	assert.ErrorIs(t, err, ErrWeakPassword)
	assert.Empty(t, pw.String())
}

func TestNewPassword_NoSpecial(t *testing.T) {
	// SEC-20: must reject passwords without a special character even
	// when every other rule is satisfied.
	pw, err := NewPassword("Abcdefghi1")
	assert.ErrorIs(t, err, ErrWeakPassword)
	assert.Empty(t, pw.String())
}

func TestNewPassword_AllLowercase(t *testing.T) {
	pw, err := NewPassword("abcdefghij")
	assert.ErrorIs(t, err, ErrWeakPassword)
	assert.Empty(t, pw.String())
}

func TestNewPassword_AllDigits(t *testing.T) {
	pw, err := NewPassword("1234567890")
	assert.ErrorIs(t, err, ErrWeakPassword)
	assert.Empty(t, pw.String())
}

func TestPassword_String_ReturnsOriginalValue(t *testing.T) {
	input := "MyPassword1!"
	pw, err := NewPassword(input)
	require.NoError(t, err)
	assert.Equal(t, input, pw.String())
}

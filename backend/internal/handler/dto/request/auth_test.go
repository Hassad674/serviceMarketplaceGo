package request

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/pkg/validator"
)

// SEC-19: every DTO has at least one negative validation test to prove
// the validate tags actually fire when the handler invokes Validate().

func TestRegisterRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterRequest
		wantErr bool
		field   string
	}{
		{
			name: "valid agency",
			req: RegisterRequest{
				Email:    "u@example.com",
				Password: "Strong123!",
				Role:     "agency",
			},
		},
		{name: "missing email", req: RegisterRequest{Password: "p", Role: "agency"}, wantErr: true, field: "email"},
		{name: "invalid email", req: RegisterRequest{Email: "not-an-email", Password: "p", Role: "agency"}, wantErr: true, field: "email"},
		{name: "missing password", req: RegisterRequest{Email: "u@e.com", Role: "agency"}, wantErr: true, field: "password"},
		{name: "password too short", req: RegisterRequest{Email: "u@e.com", Password: "short", Role: "agency"}, wantErr: true, field: "password"},
		{name: "password too long", req: RegisterRequest{Email: "u@e.com", Password: strings.Repeat("a", 200), Role: "agency"}, wantErr: true, field: "password"},
		{name: "missing role", req: RegisterRequest{Email: "u@e.com", Password: "Strong123!"}, wantErr: true, field: "role"},
		{name: "invalid role", req: RegisterRequest{Email: "u@e.com", Password: "Strong123!", Role: "admin"}, wantErr: true, field: "role"},
		{name: "first_name too long", req: RegisterRequest{
			Email: "u@e.com", Password: "Strong123!", Role: "agency", FirstName: strings.Repeat("a", 200),
		}, wantErr: true, field: "firstname"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.req)
			if tt.wantErr {
				require.Error(t, err)
				ve, ok := validator.IsValidationError(err)
				require.True(t, ok)
				if tt.field != "" {
					assert.Contains(t, fieldNames(ve), tt.field)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoginRequest_Validation(t *testing.T) {
	err := validator.Validate(LoginRequest{Email: "not-an-email", Password: "x"})
	require.Error(t, err)
	ve, _ := validator.IsValidationError(err)
	assert.Contains(t, fieldNames(ve), "email")
}

func TestRefreshRequest_Validation(t *testing.T) {
	err := validator.Validate(RefreshRequest{RefreshToken: ""})
	require.Error(t, err)
	ve, _ := validator.IsValidationError(err)
	assert.Contains(t, fieldNames(ve), "refreshtoken")
}

// fieldNames extracts field names from a ValidationError for assertion
// readability. Used across DTO test files so we keep the assertion
// shape consistent.
func fieldNames(ve *validator.ValidationError) []string {
	if ve == nil {
		return nil
	}
	out := make([]string, len(ve.Fields))
	for i, f := range ve.Fields {
		out[i] = f.Field
	}
	return out
}

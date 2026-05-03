package validator

import (
	"bytes"
	"errors"
	"fmt"
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
	// F.6 B3: an empty body now surfaces as "empty" (typed via io.EOF)
	// instead of "invalid JSON". Handlers can map both to 400 — the
	// new message is just more accurate.
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(""))

	var dst struct{}
	err := DecodeJSON(req, &dst)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
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

	require.Error(t, err)
	// F.6 B3: unknown-field errors are now typed so handlers can
	// branch on them. The wrapped message preserves the field name.
	assert.ErrorIs(t, err, ErrUnknownField)
	assert.Contains(t, err.Error(), "unknown_field")
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

// TestValidationError_ErrorString covers the Error() implementation —
// it must concatenate per-field messages so a top-level log line is
// still meaningful even without inspecting the Fields slice.
func TestValidationError_ErrorString(t *testing.T) {
	ve := &ValidationError{Fields: []FieldError{
		{Field: "email", Rule: "required", Message: "email is required"},
		{Field: "password", Rule: "min", Message: "password must be at least 10"},
	}}

	got := ve.Error()
	assert.Contains(t, got, "email: email is required")
	assert.Contains(t, got, "password: password must be at least 10")
	assert.Contains(t, got, ";", "multiple field errors must be joined")
}

func TestValidationError_ErrorString_Empty(t *testing.T) {
	ve := &ValidationError{}
	assert.Equal(t, "", ve.Error())
}

// IsValidationError must climb the wrap chain — covers the errors.As
// path from a wrapped error.
func TestIsValidationError_UnwrapsWrappedError(t *testing.T) {
	inner := &ValidationError{Fields: []FieldError{{Field: "x", Rule: "required"}}}
	wrapped := fmt.Errorf("at boundary: %w", inner)

	got, ok := IsValidationError(wrapped)
	require.True(t, ok)
	assert.Equal(t, inner, got)
}

func TestIsValidationError_OnPlainError_ReturnsFalse(t *testing.T) {
	plain := errors.New("not a validation error")
	got, ok := IsValidationError(plain)
	assert.False(t, ok)
	assert.Nil(t, got)
}

// messageFor branches we did not exercise yet: gt, lt, len, gte, lte
// edge case (extra). Each rule produces a distinct message so the
// frontend can render localised text without ambiguity.
type extraDTO struct {
	A string `json:"a" validate:"len=3"`
	B int    `json:"b" validate:"gt=10"`
	C int    `json:"c" validate:"lt=100"`
	D int    `json:"d" validate:"gte=5,lte=15"`
}

func TestValidate_LenRuleProducesMessage(t *testing.T) {
	dto := extraDTO{A: "ab", B: 11, C: 99, D: 10}
	err := Validate(dto)
	require.Error(t, err)
	ve, _ := IsValidationError(err)
	require.NotNil(t, ve)
	for _, f := range ve.Fields {
		if f.Field == "a" {
			assert.Equal(t, "len", f.Rule)
			assert.Contains(t, f.Message, "exactly")
			return
		}
	}
	t.Fatalf("expected an A field error; got %+v", ve.Fields)
}

func TestValidate_GtAndLtRules(t *testing.T) {
	dto := extraDTO{A: "abc", B: 5, C: 200, D: 10}
	err := Validate(dto)
	require.Error(t, err)
	ve, _ := IsValidationError(err)
	require.NotNil(t, ve)

	rules := map[string]string{}
	for _, f := range ve.Fields {
		rules[f.Field] = f.Rule
	}
	assert.Equal(t, "gt", rules["b"])
	assert.Equal(t, "lt", rules["c"])
}

// messageFor unknown tag falls through to the default message format.
type unknownTagDTO struct {
	X string `json:"x" validate:"alpha"` // alpha is a known go-playground tag
}

func TestValidate_AlphaTagFlowsThroughDefault(t *testing.T) {
	dto := unknownTagDTO{X: "abc123"}
	err := Validate(dto)
	require.Error(t, err)
	ve, _ := IsValidationError(err)
	require.NotNil(t, ve)
	require.Len(t, ve.Fields, 1)
	assert.Equal(t, "alpha", ve.Fields[0].Rule)
	// Default branch produces the "%s failed %q validation" format.
	assert.Contains(t, ve.Fields[0].Message, "alpha")
}

// instance() must return the same singleton across calls — proves the
// sync.Once contract.
func TestInstance_IsSingleton(t *testing.T) {
	a := instance()
	b := instance()
	assert.Same(t, a, b, "sync.Once must yield a single instance")
}

// ---------------------------------------------------------------------------
// F.6 B3 — body cap, smuggling rejection, and valid-size acceptance.
// ---------------------------------------------------------------------------

func TestDecodeJSON_BodyOverDefaultCap_Returns413Class(t *testing.T) {
	// Build a 1.5 MiB body that is otherwise valid JSON. The cap is
	// 1 MiB (DefaultMaxBodyBytes), so this MUST trip ErrBodyTooLarge —
	// closing the F.5 unbounded-body DoS surface for the helper.
	huge := strings.Repeat("a", int(DefaultMaxBodyBytes)+(512<<10))
	body := `{"name":"` + huge + `"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))

	var dst struct {
		Name string `json:"name"`
	}
	err := DecodeJSON(req, &dst)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBodyTooLarge)
}

func TestDecodeJSON_RejectsTrailingExtraData(t *testing.T) {
	// JSON-smuggling vector: two concatenated objects. Decoder accepts
	// the first; our More() check rejects the rest. Without this guard
	// an attacker could stash fields the validator never sees but the
	// handler accidentally exposes via embedded structs.
	body := `{"name":"first"}{"name":"second"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))

	var dst struct {
		Name string `json:"name"`
	}
	err := DecodeJSON(req, &dst)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnexpectedExtraData)
}

func TestDecodeJSON_AcceptsValid100KiBBody(t *testing.T) {
	// 100 KiB is well within the 1 MiB cap — must decode cleanly.
	// Sanity-checks the cap is not ridiculously low.
	desc := strings.Repeat("a", 100<<10)
	body := `{"description":"` + desc + `"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))

	var dst struct {
		Description string `json:"description"`
	}
	err := DecodeJSON(req, &dst)

	require.NoError(t, err)
	assert.Equal(t, len(desc), len(dst.Description))
}

func TestDecodeJSONWithCap_CustomCapEnforced(t *testing.T) {
	// A handler that legitimately wants a tighter cap (e.g. 1 KiB for
	// a tiny enum-toggle endpoint) can dial it down. Anything past
	// the cap MUST error.
	body := `{"name":"` + strings.Repeat("x", 2048) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))

	var dst struct {
		Name string `json:"name"`
	}
	err := DecodeJSONWithCap(nil, req, &dst, 1024)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBodyTooLarge)
}

func TestDecodeJSONWithCap_ZeroFallsBackToDefault(t *testing.T) {
	// Passing 0 must NOT mean "unlimited" — it must fall back to the
	// 1 MiB default. A typo at a call site that passes 0 should still
	// be safe.
	body := `{"name":"ok"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))

	var dst struct {
		Name string `json:"name"`
	}
	err := DecodeJSONWithCap(nil, req, &dst, 0)

	require.NoError(t, err)
	assert.Equal(t, "ok", dst.Name)
}

func TestDecodeJSON_NilRequest_ReturnsEmptyError(t *testing.T) {
	// Belt-and-suspenders: a nil *http.Request pointer must not panic.
	// Returns the same "empty" error path as a nil body.
	var dst struct{}
	err := DecodeJSON(nil, &dst)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

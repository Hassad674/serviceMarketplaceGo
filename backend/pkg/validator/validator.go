package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	playgroundvalidator "github.com/go-playground/validator/v10"

	"marketplace-backend/internal/domain/user"
)

// validatorInstance is the shared *validator.Validate. It is safe for
// concurrent use; per the upstream docs we MUST NOT create one per call
// because reflection-based tag parsing is cached on the instance.
var (
	validatorInstance *playgroundvalidator.Validate
	validatorOnce     sync.Once
)

func instance() *playgroundvalidator.Validate {
	validatorOnce.Do(func() {
		validatorInstance = playgroundvalidator.New(playgroundvalidator.WithRequiredStructEnabled())
	})
	return validatorInstance
}

// FieldError describes a single field that failed validation. Returned
// inside ValidationError.Fields — the handler layer surfaces these to
// the caller as a 400 validation_error response with a per-field map.
type FieldError struct {
	Field   string `json:"field"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// ValidationError is the typed error returned by Validate when the
// struct fails one or more rules. The handler layer maps it to
// HTTP 400 with code "validation_error".
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		parts = append(parts, fmt.Sprintf("%s: %s", f.Field, f.Message))
	}
	return strings.Join(parts, "; ")
}

// IsValidationError unwraps ValidationError from a wrapped error.
func IsValidationError(err error) (*ValidationError, bool) {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve, true
	}
	return nil, false
}

// Validate runs go-playground/validator on the struct using its
// `validate:"..."` tags. Returns nil on success, *ValidationError on
// failure. Closes SEC-19: every DTO with validate tags now gets length /
// format / range checks before the handler reaches the app layer.
func Validate(v any) error {
	if err := instance().Struct(v); err != nil {
		var verrs playgroundvalidator.ValidationErrors
		if errors.As(err, &verrs) {
			fields := make([]FieldError, 0, len(verrs))
			for _, fe := range verrs {
				fields = append(fields, FieldError{
					Field:   strings.ToLower(fe.Field()),
					Rule:    fe.Tag(),
					Message: messageFor(fe),
				})
			}
			return &ValidationError{Fields: fields}
		}
		// Unknown validator-internal failure — surface as plain error.
		return fmt.Errorf("validate: %w", err)
	}
	return nil
}

// messageFor produces a human-readable, locale-agnostic message for a
// single field error. Kept short and machine-translatable.
func messageFor(fe playgroundvalidator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", fe.Field(), fe.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s", fe.Field(), fe.Param())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", fe.Field())
	case "uuid", "uuid4":
		return fmt.Sprintf("%s must be a valid UUID", fe.Field())
	case "url":
		return fmt.Sprintf("%s must be a valid URL", fe.Field())
	case "gte":
		return fmt.Sprintf("%s must be at least %s", fe.Field(), fe.Param())
	case "lte":
		return fmt.Sprintf("%s must be at most %s", fe.Field(), fe.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", fe.Field(), fe.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", fe.Field(), fe.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", fe.Field(), fe.Param())
	default:
		return fmt.Sprintf("%s failed %q validation", fe.Field(), fe.Tag())
	}
}

// DecodeJSON decodes the request body into dst, rejecting unknown
// fields. Used by every handler before calling Validate(dst).
func DecodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return nil
}

// DecodeAndValidate is a convenience wrapper: decode + validate in a
// single call. Returns the *ValidationError unchanged so the handler
// can branch on its type.
func DecodeAndValidate(r *http.Request, dst any) error {
	if err := DecodeJSON(r, dst); err != nil {
		return err
	}
	return Validate(dst)
}

func ValidateRequired(fields map[string]string) map[string]string {
	errs := make(map[string]string)
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			errs[name] = fmt.Sprintf("%s is required", name)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

func ValidateEmail(email string) error {
	_, err := user.NewEmail(email)
	return err
}

func ValidatePassword(password string) error {
	_, err := user.NewPassword(password)
	return err
}

func ValidateRole(role string) error {
	r := user.Role(role)
	if !r.IsValid() {
		return user.ErrInvalidRole
	}
	return nil
}

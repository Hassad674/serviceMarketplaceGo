package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// DefaultMaxBodyBytes is the per-request body cap applied by DecodeJSON.
// 1 MiB is generous for an API JSON payload; handlers that legitimately
// carry larger payloads (file uploads) use streaming MaxBytesReader
// directly and do not route through this helper.
const DefaultMaxBodyBytes int64 = 1 << 20

// ErrBodyTooLarge is returned when the body exceeds DefaultMaxBodyBytes.
// Handlers translate this to HTTP 413 Payload Too Large.
var ErrBodyTooLarge = errors.New("validator: request body too large")

// ErrUnknownField is returned when the body contains a field the
// destination struct does not declare. Handlers translate this to
// HTTP 400 Bad Request — never silently drop the field.
var ErrUnknownField = errors.New("validator: unknown field")

// ErrUnexpectedExtraData is returned when the body contains a second
// JSON value after the first one decoded successfully. This is the
// JSON-smuggling vector — `{"a":1}{"b":2}` would otherwise let an
// attacker stash extra payload past the validator.
var ErrUnexpectedExtraData = errors.New("validator: unexpected extra data after JSON value")

// DecodeJSON decodes the request body into dst, rejecting unknown
// fields, capping body size at DefaultMaxBodyBytes, and rejecting any
// trailing content past the first JSON value. Used by every handler
// before calling Validate(dst).
//
// F.6 B3: previously this helper was missing the body cap and the
// trailing-content check, so the ~59 handlers calling it inherited the
// DoS body-unbounded surface that pkg/decode.DecodeBody had already
// closed for the F.5-migrated subset. Aligning the two helpers in a
// single edit retroactively protects every call site without touching
// the handlers themselves.
//
// Errors are typed when the cause is determinable:
//   - ErrBodyTooLarge: body exceeds the cap (handler → 413)
//   - ErrUnknownField: unknown field in the body (handler → 400)
//   - ErrUnexpectedExtraData: trailing content after the first JSON
//     value (handler → 400)
//
// The body MUST NOT be consumed before this function is called — the
// decoder reads it directly. http.MaxBytesReader requires a non-nil
// http.ResponseWriter to set the connection-close hint on overflow;
// since DecodeJSON does not have one in scope, the cap still works
// but the auto-reply hint is suppressed (acceptable for our use).
func DecodeJSON(r *http.Request, dst any) error {
	return DecodeJSONWithCap(nil, r, dst, DefaultMaxBodyBytes)
}

// DecodeJSONWithCap is the parameterised variant exposing the body
// cap. Most call sites use DecodeJSON; bespoke handlers that legitimately
// accept a larger body can dial the cap up here. Passing 0 falls back
// to DefaultMaxBodyBytes.
//
// w may be nil — the only consequence is that MaxBytesReader cannot
// inject the connection-close hint into the response. The cap is still
// enforced at decode time.
func DecodeJSONWithCap(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	if r == nil || r.Body == nil {
		return fmt.Errorf("request body is empty")
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBodyBytes
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		switch {
		case errors.Is(err, io.EOF):
			return fmt.Errorf("request body is empty")
		case strings.Contains(err.Error(), "http: request body too large"):
			return ErrBodyTooLarge
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			return fmt.Errorf("%w: %s", ErrUnknownField, strings.TrimPrefix(err.Error(), "json: "))
		default:
			return fmt.Errorf("invalid JSON: %w", err)
		}
	}
	// Reject any trailing content past the first JSON value. This
	// closes a JSON-smuggling vector where an attacker concatenates a
	// second object hoping the handler will pick up extra fields.
	if decoder.More() {
		return ErrUnexpectedExtraData
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

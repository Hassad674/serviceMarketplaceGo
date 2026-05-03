package handler

import (
	"errors"
	"strings"
	"testing"

	stripe "github.com/stripe/stripe-go/v82"
)

// F.5 S4 — every (potentially) Stripe error path goes through
// classifyStripeError. The contract: never forward err.Error() to the
// client; map known error shapes to stable codes; default to a generic
// "stripe_error" when the type is unknown. These tests cover the call
// sites in embedded_handler.go (lines 173, 180, 235, 246) and the new
// JSON / DB sanitizers used at lines 140 / 208.

func TestClassifyStripeError_NilFallsBackToGeneric(t *testing.T) {
	code, msg := classifyStripeError(nil)
	if code != "stripe_error" {
		t.Errorf("expected generic code on nil, got %q", code)
	}
	if msg == "" {
		t.Error("expected non-empty fallback message")
	}
}

func TestClassifyStripeError_StripeSDKErrorSurfacesCode(t *testing.T) {
	// A simulated SDK error — the Stripe SDK exposes its own error type
	// with public Code + Msg. The sanitizer must surface those AS-IS
	// (they are documented and safe) but never the request id, account
	// id, type, or any other internal field.
	se := &stripe.Error{
		Code: "amount_too_small",
		Msg:  "Amount must be at least 50 cents.",
		Type: stripe.ErrorTypeInvalidRequest,
	}
	code, msg := classifyStripeError(se)
	if code != "amount_too_small" {
		t.Errorf("expected stripe Code surfaced, got %q", code)
	}
	if !strings.Contains(msg, "Amount must be") {
		t.Errorf("expected Stripe Msg surfaced, got %q", msg)
	}
}

func TestClassifyStripeError_StripeSDKErrorWithoutCodeFallsBackToType(t *testing.T) {
	// Some Stripe errors carry only a Type (api_error, idempotency_error).
	// The sanitizer must still produce a stable code rather than empty.
	se := &stripe.Error{
		Type: stripe.ErrorTypeAPI,
		Msg:  "Stripe internal failure.",
	}
	code, _ := classifyStripeError(se)
	if !strings.HasPrefix(code, "stripe_") {
		t.Errorf("expected stripe_<type> fallback, got %q", code)
	}
}

func TestClassifyStripeError_CountryRestrictionSubstringDetected(t *testing.T) {
	// Legacy substring heuristic — the production error message is
	// preserved. Ensures the existing front-end branch on the typed
	// code still works.
	err := errors.New("operation cannot be created by platforms in the requested region")
	code, msg := classifyStripeError(err)
	if code != "country_not_supported" {
		t.Errorf("expected country_not_supported, got %q", code)
	}
	if msg == "" {
		t.Error("expected localized country-restriction message")
	}
}

func TestClassifyStripeError_UnknownErrorReturnsGenericCode(t *testing.T) {
	// A completely opaque error must NOT leak into the response.
	err := errors.New("internal: panic on goroutine 7: deadbeef")
	code, msg := classifyStripeError(err)
	if code != "stripe_error" {
		t.Errorf("expected stripe_error for unknown error, got %q", code)
	}
	if strings.Contains(msg, "deadbeef") || strings.Contains(msg, "panic") {
		t.Errorf("sanitizer leaked raw error text in message: %q", msg)
	}
}

func TestClassifyStripeError_NeverLeaksGoroutineSnippets(t *testing.T) {
	// Defensive: confirm the message NEVER contains shapes commonly
	// found in raw err.Error() outputs.
	naughty := []error{
		errors.New("pq: relation \"users\" does not exist"),
		errors.New("dial tcp 127.0.0.1:6379: connect: connection refused"),
		errors.New("context deadline exceeded"),
	}
	for _, e := range naughty {
		_, msg := classifyStripeError(e)
		for _, leak := range []string{"pq:", "127.0.0.1", "dial tcp", "context deadline"} {
			if strings.Contains(msg, leak) {
				t.Errorf("leak detected in sanitized message %q (input %q)", msg, e.Error())
			}
		}
	}
}

func TestClassifyJSONDecodeError_ReturnsGenericMessage(t *testing.T) {
	code, msg := classifyJSONDecodeError()
	if code != "invalid_json" {
		t.Errorf("expected invalid_json, got %q", code)
	}
	// Defensive: ensure the message itself contains no struct field
	// names or reflective hints — it must be a fixed constant.
	if strings.Contains(msg, "field") || strings.Contains(msg, "struct") {
		t.Errorf("sanitized JSON message must not hint at internals: %q", msg)
	}
}

func TestClassifyDBError_ReturnsGenericMessage(t *testing.T) {
	code, msg := classifyDBError()
	if code != "db_error" {
		t.Errorf("expected db_error, got %q", code)
	}
	for _, leak := range []string{"postgres", "pq:", "sql.", "ROLLBACK", "duplicate key"} {
		if strings.Contains(strings.ToLower(msg), strings.ToLower(leak)) {
			t.Errorf("sanitized DB message must not leak driver shapes: %q", msg)
		}
	}
}

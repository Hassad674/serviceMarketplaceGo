package handler

import (
	"errors"
	"strings"

	stripe "github.com/stripe/stripe-go/v82"
)

// classifyStripeError maps a (potentially Stripe) error to a stable,
// user-safe (errCode, message) pair. The original error MUST still be
// logged via slog at the call site — only the public-facing reply is
// sanitized.
//
// Closes SEC-FINAL-06 (F.5 S4): the previous pattern of returning
// `err.Error()` to the client leaked Stripe internal IDs (account_id,
// request_id), Go struct field names from JSON parser errors, and SQL
// driver chatter. Open-source means an attacker has the matching
// source ready to grep — every leaked token compounds the surface.
//
// Decision matrix (case order matters — first match wins):
//
//   1. Stripe SDK *Error : surface its public Code + Message. Stripe's
//      published error codes are documented and safe to forward
//      (e.g. "amount_too_small", "invalid_card_number"). The Message
//      is human-readable and curated by Stripe. Type / Param are not
//      forwarded — they leak SDK internals (e.g. param names that
//      align with our DTO field names).
//
//   2. Cross-border / unsupported country : already detected by the
//      caller via substring match — promoted to a typed code so the
//      frontend can localize it.
//
//   3. Anything else : "stripe_error" / generic message. The full
//      err.Error() never reaches the client.
//
// The mapping is intentionally a small switch instead of a lookup
// table — easier to read, easier to extend, and the cost of a missed
// branch is "fall through to generic" which is the safe default.
func classifyStripeError(err error) (code, message string) {
	if err == nil {
		return "stripe_error", "Stripe operation failed."
	}

	// Stripe SDK error: surface the safe-to-publish fields.
	var stripeErr *stripe.Error
	if errors.As(err, &stripeErr) {
		code = string(stripeErr.Code)
		if code == "" {
			// Some Stripe errors carry a Type but no Code (e.g. api_error).
			// Promote Type to a stable, prefixed code so the client can
			// branch on it without mistaking it for a Code.
			code = "stripe_" + string(stripeErr.Type)
		}
		message = stripeErr.Msg
		if message == "" {
			message = "Stripe rejected the request."
		}
		return code, message
	}

	// Country-restriction substring (legacy guard — keeps the existing
	// frontend code path that branches on `country_not_supported`).
	if strings.Contains(err.Error(), "cannot be created by platforms in") {
		return "country_not_supported",
			"Ce pays n'est pas disponible depuis notre plateforme. Contactez notre support si vous pensez que c'est une erreur."
	}

	return "stripe_error", "Stripe operation failed. Please retry — the incident has been logged."
}

// classifyJSONDecodeError returns a generic 400-grade reply for a body
// that failed to parse. The decoder error message — which can include
// our struct field names — is logged separately by the caller and
// must NOT be forwarded.
func classifyJSONDecodeError() (code, message string) {
	return "invalid_json", "Request body is not a valid JSON document."
}

// classifyDBError returns a generic 500-grade reply for a database
// failure. PostgreSQL driver errors include schema names, query text,
// and sometimes constraint names — none of which should reach the
// client.
func classifyDBError() (code, message string) {
	return "db_error", "Internal storage error. Please retry — the incident has been logged."
}

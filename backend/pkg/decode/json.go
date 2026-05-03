// Package decode hosts the canonical JSON request-body decoder used by
// every HTTP handler. Centralising the pattern (DisallowUnknownFields +
// MaxBytesReader) here closes a class of bugs where a handler quietly
// accepted a 100 MB body, or silently dropped extra fields without
// rejecting them.
//
// Why a tiny package instead of an option in pkg/validator:
// pkg/validator handles validation via go-playground/validator tags;
// it is not the right home for HTTP body limits and unknown-field
// rejection — those are transport concerns. Keeping decode separate
// also makes it cheap to import from any handler without pulling in
// the validator's reflect machinery.
//
// SEC (F.5 B1): the previous "json.NewDecoder(r.Body).Decode(...)" pattern
// in 13+ handlers (admin suspend / ban, admin team, billing-profile,
// subscription, skill, health, admin credit-note) accepted unbounded
// bodies and silently tolerated unknown fields. A malicious client
// could DoS the process with a huge body, or smuggle fields the
// handler did not validate but Postgres would later persist via
// embedded structs. Both surfaces are closed by routing every body
// decode through DecodeBody.
package decode

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DefaultMaxBodyBytes is the default per-body cap. 1 MiB is generous
// for an API request body — handlers that legitimately carry larger
// payloads (file uploads) should use the streaming MaxBytesReader
// directly instead of routing through this decoder.
const DefaultMaxBodyBytes int64 = 1 << 20

// ErrEmptyBody is returned when the request has no body at all. The
// HTTP layer should map this to a 400 Bad Request — separate from a
// malformed body so the caller can distinguish "you sent nothing" from
// "your JSON is broken".
var ErrEmptyBody = errors.New("decode: empty request body")

// ErrBodyTooLarge is returned when MaxBytesReader trips. Callers should
// surface this as 413 Request Entity Too Large.
var ErrBodyTooLarge = errors.New("decode: request body too large")

// ErrUnknownField is returned when the body contains a field the
// destination struct does not declare. Callers surface this as 400
// Bad Request — never silently drop the field.
var ErrUnknownField = errors.New("decode: unknown field")

// DecodeBody reads the request body into v, capping the size at
// maxBytes (use DefaultMaxBodyBytes for the standard 1 MiB cap) and
// rejecting unknown fields. It also rejects bodies with trailing
// content after the first JSON value to prevent JSON smuggling.
//
// Errors are typed when the cause is determinable:
//   - ErrEmptyBody when r.Body is nil or has length 0
//   - ErrBodyTooLarge when the cap trips
//   - ErrUnknownField when a field is unknown to v
// All other errors are wrapped as fmt.Errorf("decode: invalid JSON: %w", ...)
// so the caller can format them uniformly without inspecting the cause.
//
// MaxBytesReader requires the http.ResponseWriter to set the
// `Content-Length`-related response header on overflow. Passing nil
// for w is allowed in tests where the handler hasn't yet written
// headers; the size enforcement still works, only the auto-reply
// connection-close hint is suppressed.
func DecodeBody(w http.ResponseWriter, r *http.Request, v any, maxBytes int64) error {
	if r == nil || r.Body == nil {
		return ErrEmptyBody
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBodyBytes
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		switch {
		case errors.Is(err, io.EOF):
			return ErrEmptyBody
		case strings.Contains(err.Error(), "http: request body too large"):
			return ErrBodyTooLarge
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			// json.Decoder wraps the unknown field name in the message
			// (e.g. `json: unknown field "foo"`). Preserve that via Errorf
			// so the caller can echo the field name in a debug response.
			return fmt.Errorf("%w: %s", ErrUnknownField, strings.TrimPrefix(err.Error(), "json: "))
		default:
			return fmt.Errorf("decode: invalid JSON: %w", err)
		}
	}
	// Reject any trailing bytes — this is what protects against JSON
	// smuggling via concatenated objects ({"a":1}{"b":2}). Decoder.More
	// returns true if there's another token after the one just decoded.
	if dec.More() {
		return fmt.Errorf("decode: invalid JSON: unexpected trailing content")
	}
	return nil
}

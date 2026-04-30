package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reflect"
)

// JSON writes a JSON-encoded response with the given status code.
//
// For status codes that do not allow a response body per RFC 7230 § 3.3.3
// (204 No Content, 304 Not Modified, and all 1xx informational responses),
// JSON short-circuits: it writes the status header without any body. This
// prevents Go's net/http from returning the error
//   "http: request method or response status code does not allow body"
// when handlers call JSON(w, http.StatusNoContent, nil) — the historical
// idiom across this codebase. Existing call sites keep working unchanged.
//
// For all other statuses the body is encoded as JSON. Encoding errors are
// logged at ERROR level because they indicate a real marshalling bug.
//
// BUG-19 (empty-list null vs []) — JSON normalises a nil slice argument
// to an empty slice of the same element type before encoding so the
// wire format is `[]` instead of `null`. TS clients across the apps
// call `.length` / `.map` on list responses; receiving `null` crashes
// them at runtime. The normalisation only touches top-level slices —
// nested slices are still rendered as `null` if the caller passes a
// nil into a struct field, because at that point the contract is
// per-field and outside the response helper's scope. Handlers wanting
// nested slices to also normalise can call NilSliceToEmpty on each
// field before composing the response.
func JSON(w http.ResponseWriter, status int, data any) {
	if bodyNotAllowed(status) {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(NilSliceToEmpty(data)); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

// NilSliceToEmpty returns an empty slice of the same element type when
// the argument is a nil slice. For every other input it returns the
// argument unchanged. Closes BUG-19: nil slices in Go marshal to JSON
// `null`, breaking TS clients that call `.length` on the response. This
// helper is exported so callers that build envelopes with nested
// slices can normalise them explicitly.
//
// Implementation note: handlers across the codebase return concrete
// slice types (`[]*User`, `[]map[string]any`, `[]string`). reflect lets
// us return an empty slice of the *same element type* without losing
// the JSON type information — both `[]*User{}` and `[]string{}` encode
// to the same JSON `[]`, but preserving the element type keeps tools
// like fmt.Sprintf("%T") informative for log lines.
func NilSliceToEmpty(v any) any {
	if v == nil {
		return v
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice || !rv.IsNil() {
		return v
	}
	return reflect.MakeSlice(rv.Type(), 0, 0).Interface()
}

// bodyNotAllowed reports whether the given HTTP status code forbids a
// response body. Mirrors Go's net/http internal rules so callers can safely
// detect statuses where Encode would fail.
func bodyNotAllowed(status int) bool {
	// 1xx informational, 204 No Content, 304 Not Modified — no body allowed.
	return status == http.StatusNoContent ||
		status == http.StatusNotModified ||
		(status >= 100 && status < 200)
}

func Error(w http.ResponseWriter, status int, errCode string, message string) {
	JSON(w, status, map[string]string{
		"error":   errCode,
		"message": message,
	})
}

func ValidationError(w http.ResponseWriter, details map[string]string) {
	JSON(w, http.StatusUnprocessableEntity, map[string]any{
		"error":   "validation_error",
		"message": "one or more fields are invalid",
		"details": details,
	})
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
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
func JSON(w http.ResponseWriter, status int, data any) {
	if bodyNotAllowed(status) {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
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

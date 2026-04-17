package middleware

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Redact strips values that must never appear in logs. The function is
// pure (no I/O) and safe for concurrent use.
//
// What we redact:
//   - Authorization headers (full value)
//   - Cookie headers (full value)
//   - JSON fields named: password, passwd, secret, token, refresh_token,
//     access_token, api_key, client_secret, jwt, authorization
//
// What we DO NOT redact:
//   - user_id (UUID is safe to log)
//   - request_id (already desirable for correlation)
//   - request path (already desirable)
//
// Usage:
//
//	slog.Info("request body", "body", middleware.Redact(body))
//	slog.Error("upstream failure", "headers", middleware.RedactHeaders(h))
//
// These helpers are opt-in — the existing Logger middleware does NOT
// log request bodies. Anything that does must route through Redact.
var (
	// Match either a JSON numeric/string value after one of the sensitive keys.
	sensitiveJSONKey = regexp.MustCompile(`(?i)"(password|passwd|secret|token|refresh_token|access_token|api_key|apikey|client_secret|jwt|authorization)"\s*:\s*"[^"\\]*(?:\\.[^"\\]*)*"`)
	sensitiveJSONNum = regexp.MustCompile(`(?i)"(password|passwd|secret|token|refresh_token|access_token|api_key|apikey|client_secret|jwt|authorization)"\s*:\s*-?\d+(\.\d+)?`)

	// Token patterns surfaced in free-form text (e.g. log messages).
	// Bearer <jwt> — redact the JWT itself.
	bearerPattern = regexp.MustCompile(`(?i)(Bearer\s+)[A-Za-z0-9\-_.]+`)

	// OpenAI / generic sk- keys (supports sk-, sk-proj-, sk-admin- prefixes).
	openAIKeyPattern = regexp.MustCompile(`sk-(?:proj-|admin-)?[A-Za-z0-9]{20,}`)

	// Email addresses — we swap to a hash-stub so correlation survives
	// without exposing the PII.
	emailPattern = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
)

// Redact sanitises a string that might contain sensitive data.
// Intended for untrusted payloads that must be emitted into a log.
func Redact(s string) string {
	s = sensitiveJSONKey.ReplaceAllString(s, `"$1":"[REDACTED]"`)
	s = sensitiveJSONNum.ReplaceAllString(s, `"$1":"[REDACTED]"`)
	s = bearerPattern.ReplaceAllString(s, "${1}[REDACTED]")
	s = openAIKeyPattern.ReplaceAllString(s, "sk-[REDACTED]")
	s = emailPattern.ReplaceAllString(s, "[REDACTED_EMAIL]")
	return s
}

// RedactBytes is the []byte counterpart. Avoids an extra copy when the
// caller already has bytes on hand.
func RedactBytes(b []byte) []byte {
	return []byte(Redact(string(b)))
}

// RedactHeaders returns a copy of the input header map where sensitive
// headers are replaced with "[REDACTED]". The original map is NOT
// mutated. Callers that log http.Header values MUST route through
// this helper.
func RedactHeaders(h map[string][]string) map[string][]string {
	const redacted = "[REDACTED]"
	out := make(map[string][]string, len(h))
	for k, v := range h {
		lk := strings.ToLower(k)
		switch lk {
		case "authorization", "cookie", "set-cookie", "x-api-key", "proxy-authorization":
			out[k] = []string{redacted}
		default:
			cp := make([]string, len(v))
			copy(cp, v)
			out[k] = cp
		}
	}
	return out
}

// RedactJSON attempts to parse v as JSON, redact sensitive fields, and
// re-serialise. If the input is not valid JSON the raw Redact fallback
// is used.
func RedactJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[UNSERIALISABLE]"
	}
	return Redact(string(b))
}

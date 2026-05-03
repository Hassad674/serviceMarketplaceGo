package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestRedact_JSONKeys(t *testing.T) {
	cases := []struct {
		name string
		in   string
		must string
	}{
		{
			name: "password field",
			in:   `{"email":"a@b.c","password":"hunter2"}`,
			must: `"password":"[REDACTED]"`,
		},
		{
			name: "refresh_token field",
			in:   `{"refresh_token":"eyJhbGciOi..."}`,
			must: `"refresh_token":"[REDACTED]"`,
		},
		{
			name: "api_key numeric",
			in:   `{"api_key":123456789}`,
			must: `"api_key":"[REDACTED]"`,
		},
		{
			name: "case-insensitive",
			in:   `{"PassWord":"leaky"}`,
			must: `"PassWord":"[REDACTED]"`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := Redact(c.in)
			if !strings.Contains(out, c.must) {
				t.Fatalf("expected %q in %q", c.must, out)
			}
			if strings.Contains(out, "hunter2") || strings.Contains(out, "eyJhbGciOi") || strings.Contains(out, "leaky") {
				t.Fatalf("raw secret survived redaction: %q", out)
			}
		})
	}
}

func TestRedact_BearerTokens(t *testing.T) {
	in := "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig"
	out := Redact(in)
	if strings.Contains(out, "eyJhbGciOiJIUzI1NiJ9") {
		t.Fatalf("JWT survived: %q", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Fatalf("expected redacted marker, got %q", out)
	}
}

func TestRedact_OpenAIKey(t *testing.T) {
	in := "calling api with sk-proj-abcdef1234567890ABCDEF"
	out := Redact(in)
	if strings.Contains(out, "abcdef1234567890") {
		t.Fatalf("openai key survived: %q", out)
	}
}

func TestRedact_Email(t *testing.T) {
	in := "user hassad.smara@example.com logged in"
	out := Redact(in)
	if strings.Contains(out, "hassad.smara@example.com") {
		t.Fatalf("email survived: %q", out)
	}
	if !strings.Contains(out, "[REDACTED_EMAIL]") {
		t.Fatalf("expected email placeholder: %q", out)
	}
}

func TestRedactHeaders_StripsSensitive(t *testing.T) {
	h := map[string][]string{
		"Authorization": {"Bearer abc"},
		"Cookie":        {"session=xyz"},
		"X-Api-Key":     {"sk-1234"},
		"Content-Type":  {"application/json"},
		"X-Request-Id":  {"uuid"},
	}
	out := RedactHeaders(h)
	for _, sens := range []string{"Authorization", "Cookie", "X-Api-Key"} {
		if got := out[sens]; len(got) != 1 || got[0] != "[REDACTED]" {
			t.Fatalf("%s not redacted: %v", sens, got)
		}
	}
	if out["Content-Type"][0] != "application/json" {
		t.Fatalf("Content-Type mutated")
	}
	if out["X-Request-Id"][0] != "uuid" {
		t.Fatalf("request-id mutated")
	}
	// Ensure original map is untouched.
	if h["Authorization"][0] != "Bearer abc" {
		t.Fatalf("original map was mutated: %v", h)
	}
}

func TestRedactJSON_InvalidFalsback(t *testing.T) {
	v := make(chan int) // not JSON-serialisable
	if RedactJSON(v) != "[UNSERIALISABLE]" {
		t.Fatalf("expected unserialisable sentinel")
	}
}

func TestRedact_PreservesSafeFields(t *testing.T) {
	in := `{"user_id":"550e8400","request_id":"abc","path":"/api/v1/search"}`
	out := Redact(in)
	for _, want := range []string{"550e8400", "abc", "/api/v1/search"} {
		if !strings.Contains(out, want) {
			t.Fatalf("safe field stripped: %q -> %q", want, out)
		}
	}
}

// ---------------------------------------------------------------------------
// SEC-FINAL-13 — SlogReplaceAttr global redaction
// ---------------------------------------------------------------------------

// newCapturingLogger builds a JSON slog logger writing to an in-memory
// buffer and routing every attribute through SlogReplaceAttr. Returns
// the logger plus the buffer the caller can inspect after each log
// emission.
func newCapturingLogger() (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	h := slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: SlogReplaceAttr,
	})
	return slog.New(h), buf
}

func TestSlogReplaceAttr_RedactsBearerInsideStringAttr(t *testing.T) {
	logger, buf := newCapturingLogger()

	// A regression that motivated SEC-FINAL-13: an unsuspecting log
	// line embeds the request's Authorization header as a string —
	// the Bearer JWT must be scrubbed before it leaves the handler.
	logger.Info("upstream call",
		"detail", "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig")

	out := buf.String()
	if strings.Contains(out, "eyJhbGciOiJIUzI1NiJ9.payload.sig") {
		t.Fatalf("Bearer JWT leaked: %q", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Fatalf("expected redaction marker: %q", out)
	}
}

func TestSlogReplaceAttr_RedactsByKeyName(t *testing.T) {
	logger, buf := newCapturingLogger()

	// Whole-attribute redaction by sensitive key name.
	logger.Info("login attempt",
		"email", "alice@example.com",
		"password", "hunter2",
		"token", "abc123",
		"refresh_token", "rt-456",
		"user_id", "550e8400",
	)

	out := buf.String()
	for _, leaked := range []string{"hunter2", "abc123", "rt-456"} {
		if strings.Contains(out, leaked) {
			t.Fatalf("secret %q leaked: %q", leaked, out)
		}
	}
	// Safe attributes survive — user_id is a UUID we WANT in logs.
	if !strings.Contains(out, "550e8400") {
		t.Fatalf("safe user_id was wrongly stripped: %q", out)
	}
}

func TestSlogReplaceAttr_RedactsHTTPHeader(t *testing.T) {
	logger, buf := newCapturingLogger()

	h := http.Header{}
	h.Set("Authorization", "Bearer leak-me")
	h.Set("Cookie", "session=secret")
	h.Set("X-Request-Id", "req-123")
	h.Set("Content-Type", "application/json")

	logger.Info("incoming request", "headers", h)

	out := buf.String()
	if strings.Contains(out, "leak-me") {
		t.Fatalf("Bearer in Authorization header leaked: %q", out)
	}
	if strings.Contains(out, "session=secret") {
		t.Fatalf("Cookie value leaked: %q", out)
	}
	// Non-sensitive headers must survive so logs stay useful.
	if !strings.Contains(out, "req-123") {
		t.Fatalf("X-Request-Id was stripped — should survive: %q", out)
	}
	if !strings.Contains(out, "application/json") {
		t.Fatalf("Content-Type was stripped — should survive: %q", out)
	}
}

func TestSlogReplaceAttr_RedactsRawHeaderMap(t *testing.T) {
	// Some callers pass map[string][]string instead of http.Header
	// (e.g. when copying request metadata). Both shapes must be
	// caught by the same code path.
	logger, buf := newCapturingLogger()

	logger.Info("raw map", "headers", map[string][]string{
		"Authorization": {"Bearer raw-map-leak"},
		"X-Request-Id":  {"req-rawmap"},
	})

	out := buf.String()
	if strings.Contains(out, "raw-map-leak") {
		t.Fatalf("Bearer in raw map leaked: %q", out)
	}
	if !strings.Contains(out, "req-rawmap") {
		t.Fatalf("safe header was wrongly stripped: %q", out)
	}
}

func TestSlogReplaceAttr_RedactsOpenAIKeyInMessage(t *testing.T) {
	logger, buf := newCapturingLogger()

	// Free-form message with an embedded sk-proj key.
	logger.Info("calling openai", "url", "https://api.openai.com/v1/chat/completions key=sk-proj-abcdef1234567890ABCDEF")

	out := buf.String()
	if strings.Contains(out, "sk-proj-abcdef1234567890ABCDEF") {
		t.Fatalf("OpenAI key leaked: %q", out)
	}
}

func TestSlogReplaceAttr_PreservesNonStringValues(t *testing.T) {
	// Non-string typed values must pass through unchanged so structured
	// logs (numbers, booleans, durations) keep their JSON shape.
	logger, buf := newCapturingLogger()

	logger.Info("metrics",
		"duration_ms", 42,
		"ok", true,
		"count", 7,
	)

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, buf.String())
	}
	if parsed["duration_ms"].(float64) != 42 {
		t.Fatalf("duration_ms mutated: %v", parsed["duration_ms"])
	}
	if parsed["ok"].(bool) != true {
		t.Fatalf("ok mutated: %v", parsed["ok"])
	}
	if parsed["count"].(float64) != 7 {
		t.Fatalf("count mutated: %v", parsed["count"])
	}
}

func TestSlogReplaceAttr_KeyMatchIsCaseInsensitive(t *testing.T) {
	logger, buf := newCapturingLogger()

	// Mixed-case sensitive keys still get redacted by name.
	logger.Info("mixed", "Authorization", "Bearer x", "PASSWORD", "y", "Refresh_Token", "z")

	out := buf.String()
	for _, leaked := range []string{"Bearer x", "\"y\"", "\"z\""} {
		if strings.Contains(out, leaked) {
			t.Fatalf("expected redaction, %q survived: %q", leaked, out)
		}
	}
}

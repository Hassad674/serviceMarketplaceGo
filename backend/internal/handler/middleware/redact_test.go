package middleware

import (
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

package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// TestTruncateQueryForLog covers the rune-aware 200-char cap.
// Multi-byte characters count as one character, not N bytes.
func TestTruncateQueryForLog(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		wantTrim bool
	}{
		{
			name:     "empty_stays_empty",
			input:    "",
			want:     "",
			wantTrim: false,
		},
		{
			name:     "short_unchanged",
			input:    "react paris",
			want:     "react paris",
			wantTrim: false,
		},
		{
			name:     "at_limit_not_truncated",
			input:    strings.Repeat("a", searchQueryLogMaxChars),
			want:     strings.Repeat("a", searchQueryLogMaxChars),
			wantTrim: false,
		},
		{
			name:     "over_limit_is_truncated",
			input:    strings.Repeat("a", searchQueryLogMaxChars+10),
			want:     strings.Repeat("a", searchQueryLogMaxChars),
			wantTrim: true,
		},
		{
			name: "multibyte_counts_as_one_rune",
			// Each é is 2 bytes but should count as 1 rune.
			input:    strings.Repeat("é", searchQueryLogMaxChars+5),
			want:     strings.Repeat("é", searchQueryLogMaxChars),
			wantTrim: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, trimmed := truncateQueryForLog(tt.input)
			if got != tt.want {
				t.Errorf("truncateQueryForLog(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if trimmed != tt.wantTrim {
				t.Errorf("truncateQueryForLog(%q) truncated = %v, want %v", tt.input, trimmed, tt.wantTrim)
			}
		})
	}
}

// TestSearchLog_LogAttrs pins the exact attribute list emitted on
// every search request. If a new field is added to SearchLog it MUST
// also appear in this test's expected set — the test fails otherwise
// so the log shape cannot drift silently.
func TestSearchLog_LogAttrs(t *testing.T) {
	payload := SearchLog{
		RequestID:    "req-123",
		UserID:       "user-456",
		Persona:      "freelance",
		Query:        "react paris",
		FilterBy:     "persona:freelance",
		SortBy:       "",
		ResultsCount: 42,
		LatencyMs:    87,
		Hybrid:       true,
		CursorActive: false,
		Truncated:    false,
	}
	attrs := payload.LogAttrs()

	wantKeys := []string{
		"event", "request_id", "user_id", "persona", "query",
		"truncated", "filter_by", "sort_by", "results_count",
		"latency_ms", "hybrid", "cursor_active",
	}
	if len(attrs) != len(wantKeys) {
		t.Fatalf("expected %d attributes, got %d", len(wantKeys), len(attrs))
	}
	for i, want := range wantKeys {
		if attrs[i].Key != want {
			t.Errorf("attr[%d].Key = %q, want %q", i, attrs[i].Key, want)
		}
	}
}

// TestEmitSearchLog_JSONShape writes through a real JSON slog handler
// into a bytes.Buffer and asserts the parsed JSON contains every
// documented key with the expected value. This is the golden shape
// that operators will grep against in production.
func TestEmitSearchLog_JSONShape(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	payload := SearchLog{
		RequestID:    "req-abc",
		UserID:       "user-xyz",
		Persona:      "agency",
		Query:        "devops kubernetes",
		FilterBy:     "persona:agency && is_published:true",
		SortBy:       "rating_score:desc",
		ResultsCount: 7,
		LatencyMs:    123,
		Hybrid:       true,
		CursorActive: true,
		Truncated:    false,
	}
	emitSearchLog(logger, payload)

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatal("emitSearchLog produced no output")
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(line), &parsed); err != nil {
		t.Fatalf("log line is not valid JSON: %v\n%s", err, line)
	}

	wantValues := map[string]any{
		"event":         "search.query",
		"request_id":    "req-abc",
		"user_id":       "user-xyz",
		"persona":       "agency",
		"query":         "devops kubernetes",
		"truncated":     false,
		"filter_by":     "persona:agency && is_published:true",
		"sort_by":       "rating_score:desc",
		"results_count": float64(7), // json.Unmarshal gives numbers as float64
		"latency_ms":    float64(123),
		"hybrid":        true,
		"cursor_active": true,
	}
	for k, want := range wantValues {
		got, ok := parsed[k]
		if !ok {
			t.Errorf("log payload missing key %q", k)
			continue
		}
		if got != want {
			t.Errorf("log[%q] = %v (%T), want %v (%T)", k, got, got, want, want)
		}
	}
	// The msg field must be present for filter-by-message tooling.
	if parsed["msg"] != "search.query" {
		t.Errorf("log msg = %v, want %q", parsed["msg"], "search.query")
	}
}

// TestEmitSearchLog_NilLoggerFallback covers the nil-safe guard.
// A caller that forgets to inject a logger must not panic — the
// function falls back to slog.Default().
func TestEmitSearchLog_NilLoggerFallback(t *testing.T) {
	// Replace default logger with a bytes-backed one so the test
	// runs deterministically without touching stderr.
	buf := &bytes.Buffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	emitSearchLog(nil, SearchLog{
		RequestID: "req-fallback",
		Persona:   "freelance",
	})

	if !strings.Contains(buf.String(), "req-fallback") {
		t.Errorf("nil-logger fallback did not emit the line: %s", buf.String())
	}
}

package handler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// F.5 B1 + F.6 B3 — guardrail test: NO handler file may use the raw
// json.NewDecoder(r.Body).Decode(...) pattern. Every body decode must
// go through one of the bounded helpers:
//
//   - pkg/decode.DecodeBody (imported as jsondec or decode)
//   - pkg/validator.DecodeJSON / DecodeAndValidate / DecodeJSONWithCap
//
// Both helpers wrap r.Body with http.MaxBytesReader (1 MiB cap),
// reject unknown fields, and reject trailing JSON tokens. A regression
// re-introducing the raw decoder would silently leak an unbounded-body
// DoS surface — this test catches it on the first build.
//
// F.5 sweep covered 7 named files; F.6 B3 turns that into a glob
// across the whole handler directory so a new file added by a future
// contributor cannot escape the convention.
func TestF5B1_NoRawJSONNewDecoderInBodyHandlers(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read handler dir: %v", err)
	}

	// decode_sweep_test.go references the forbidden pattern inside a
	// string literal as the regression guard — that's fine. Test files
	// (*_test.go) generate fixture bodies and may use json.NewDecoder
	// against test-controlled byte readers; we only police production
	// handlers.
	greenlist := map[string]struct{}{
		"decode_sweep_test.go": {},
	}

	violations := make([]string, 0)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		if _, skip := greenlist[name]; skip {
			continue
		}

		path := filepath.Join(".", name)
		bodyBytes, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read %s: %v", path, readErr)
		}
		src := string(bodyBytes)
		if strings.Contains(src, "json.NewDecoder(r.Body)") {
			violations = append(violations, name)
		}
	}

	if len(violations) > 0 {
		t.Errorf("F.5 B1 + F.6 B3 regression: handler files still use raw json.NewDecoder(r.Body) — replace with jsondec.DecodeBody or validator.DecodeJSON: %v", violations)
	}
}

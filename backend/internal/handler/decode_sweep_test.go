package handler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// F.5 B1 — guardrail test: 13 handler call sites that decode JSON
// bodies must route through pkg/decode.DecodeBody (imported as
// `jsondec` to avoid colliding with a local `decode` function in
// referral_handler_test.go). A future contributor copy-pasting the
// raw json.NewDecoder(r.Body).Decode(...) pattern would silently
// reintroduce the unbounded-body + unknown-field-tolerated surface;
// this test keeps the convention enforced.
//
// The test reads the source files (no reflection — handler functions
// are not reflectable for body-decoder calls), greps for the
// forbidden pattern, and lists the files that violate. We greenlist
// upload_handler.go family because they use streaming readers, not
// json.Decoder, on purpose.
func TestF5B1_NoRawJSONNewDecoderInBodyHandlers(t *testing.T) {
	files := []string{
		"admin_handler.go",
		"admin_credit_note_handler.go",
		"admin_team_handler.go",
		"billing_profile_handler.go",
		"subscription_handler.go",
		"skill_handler.go",
		"health_handler.go",
	}
	for _, f := range files {
		path := filepath.Join(".", f)
		bodyBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		src := string(bodyBytes)
		if strings.Contains(src, "json.NewDecoder(r.Body)") {
			t.Errorf("F.5 B1 regression: %s still uses raw json.NewDecoder(r.Body) — replace with jsondec.DecodeBody", f)
		}
		// The files MUST still call DecodeBody at least once — otherwise
		// the test would silently pass on a future refactor that simply
		// removed the body decoder.
		if !strings.Contains(src, "jsondec.DecodeBody(") && !strings.Contains(src, "decode.DecodeBody(") {
			t.Errorf("F.5 B1: %s does not call jsondec.DecodeBody — verify the sweep was applied", f)
		}
	}
}

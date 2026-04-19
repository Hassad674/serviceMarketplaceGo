package antigaming

import (
	"strings"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/search/features"
)

// tokenise behaviour locked by table.
func TestTokenise(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"single word", "react", []string{"react"}},
		{"multi word space", "go rust", []string{"go", "rust"}},
		{"punctuation split", "go, rust; python!", []string{"go", "rust", "python"}},
		{"lowercased", "React TypeScript", []string{"react", "typescript"}},
		{"digits preserved", "go 1.25", []string{"go", "1", "25"}},
		{"unicode letters", "café naïve", []string{"café", "naïve"}},
		{"whitespace trimmed", "  hello  world  ", []string{"hello", "world"}},
		{"only punctuation returns empty slice", "!!!...", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenise(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}

// detectStuffing — spec examples from §7.1.
func TestDetectStuffing(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name         string
		text         string
		wantDetected bool
	}{
		{
			name:         "empty text -> not detected",
			text:         "",
			wantDetected: false,
		},
		{
			name:         "short text (< 5 tokens) -> not detected",
			text:         "hello world",
			wantDetected: false,
		},
		{
			name:         "normal about text -> not detected",
			text:         "Senior React developer based in Paris, focused on fintech products with 10 years experience.",
			wantDetected: false,
		},
		{
			name:         "attack: 10x react repetition",
			text:         strings.Repeat("react ", 10),
			wantDetected: true,
		},
		{
			name:         "attack: low distinct ratio via 2 words",
			text:         "react react react react react react react react react react react react react react java",
			wantDetected: true,
		},
		{
			name:         "borderline: exactly 5 reps (threshold inclusive)",
			text:         "react react react react react java flutter go rust kotlin",
			wantDetected: false,
		},
		{
			name:         "borderline: 6 reps fires",
			text:         "react react react react react react java flutter go rust kotlin",
			wantDetected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			det := detectStuffing(tt.text, cfg)
			assert.Equal(t, tt.wantDetected, det.Detected,
				"detection: max_rep=%d distinct=%f total=%d",
				det.MaxRepetition, det.DistinctRatio, det.TotalTokenCount)
		})
	}
}

// Stuffing rule mutates text_match_score, reports a Penalty.
func TestStuffingRule_Penalty(t *testing.T) {
	cfg := DefaultConfig()
	f := &features.Features{TextMatchScore: 0.9}
	raw := RawSignals{
		ProfileID: "p1",
		Persona:   features.PersonaFreelance,
		Text:      strings.Repeat("react ", 20),
	}
	pen := stuffingRule(f, raw, cfg)
	assert.NotNil(t, pen)
	assert.Equal(t, RuleKeywordStuffing, pen.Rule)
	assert.InDelta(t, 0.45, f.TextMatchScore, 1e-9) // halved
	assert.Equal(t, "p1", pen.ProfileID)
}

// Clean text -> no penalty, features untouched.
func TestStuffingRule_Clean(t *testing.T) {
	cfg := DefaultConfig()
	f := &features.Features{TextMatchScore: 0.9}
	raw := RawSignals{
		Text: "senior developer with a decade of experience in distributed systems",
	}
	pen := stuffingRule(f, raw, cfg)
	assert.Nil(t, pen)
	assert.InDelta(t, 0.9, f.TextMatchScore, 1e-9)
}

// Fuzz : detectStuffing must never panic + always returns a valid boolean,
// regardless of input string up to 10 KB.
func TestDetectStuffing_Fuzz_NoPanic(t *testing.T) {
	cfg := DefaultConfig()
	fn := func(s string) bool {
		// Cap input at 10_000 chars to keep runtime reasonable.
		if len(s) > 10_000 {
			s = s[:10_000]
		}
		det := detectStuffing(s, cfg)
		// Detected is bool already. Verify derived fields are sane.
		if det.TotalTokenCount < 0 {
			return false
		}
		if det.MaxRepetition < 0 {
			return false
		}
		if det.DistinctRatio < 0 || det.DistinctRatio > 1 {
			return false
		}
		return true
	}
	if err := quick.Check(fn, &quick.Config{MaxCount: 200}); err != nil {
		t.Fatalf("detectStuffing panicked or returned bogus values: %v", err)
	}
}

// FuzzDetectStuffing is the Go native fuzz test — runs "go test -fuzz" style.
// It exercises the stuffing detector with arbitrary byte input.
func FuzzDetectStuffing(f *testing.F) {
	cfg := DefaultConfig()
	seeds := []string{
		"",
		"react",
		strings.Repeat("react ", 100),
		"!!!!!!!!!!!!",
		"unicode 你好 тест",
		strings.Repeat("a", 10_000),
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 10_000 {
			s = s[:10_000]
		}
		det := detectStuffing(s, cfg)
		if det.TotalTokenCount < 0 || det.MaxRepetition < 0 {
			t.Fatalf("negative count: %+v", det)
		}
		if det.DistinctRatio < 0 || det.DistinctRatio > 1 {
			t.Fatalf("distinct ratio out of bounds: %f", det.DistinctRatio)
		}
	})
}

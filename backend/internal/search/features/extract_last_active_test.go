package features

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ExtractLastActiveDays — hyperbolic decay, spec table pinned.
func TestExtractLastActiveDays(t *testing.T) {
	cfg := DefaultConfig()
	const secondsPerDay = 86400
	const now = 1700000000 // arbitrary Unix epoch

	tests := []struct {
		name     string
		ageDays  int64
		wantMin  float64
		wantMax  float64
	}{
		{"today", 0, 0.99, 1.00},
		{"15 days", 15, 0.65, 0.69},
		{"30 days", 30, 0.49, 0.51},
		{"90 days", 90, 0.24, 0.26},
		{"180 days", 180, 0.13, 0.15},
		{"365 days", 365, 0.07, 0.09},
		{"1000 days", 1000, 0, 0.03},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := SearchDocumentLite{
				LastActiveAt: now - tt.ageDays*secondsPerDay,
				NowUnix:      now,
			}
			got := ExtractLastActiveDays(doc, cfg)
			assert.GreaterOrEqual(t, got, tt.wantMin)
			assert.LessOrEqual(t, got, tt.wantMax)
		})
	}
}

// Missing signals -> 0 (never a negative bias).
func TestExtractLastActiveDays_MissingSignals(t *testing.T) {
	cfg := DefaultConfig()
	tests := []struct {
		name string
		doc  SearchDocumentLite
	}{
		{"last_active_at unset", SearchDocumentLite{NowUnix: 1700000000}},
		{"now_unix unset", SearchDocumentLite{LastActiveAt: 1700000000}},
		{"both unset", SearchDocumentLite{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, 0, ExtractLastActiveDays(tt.doc, cfg), 1e-9)
		})
	}
}

// Clock skew (future timestamp) clamps to "right now".
func TestExtractLastActiveDays_FutureClockSkew(t *testing.T) {
	cfg := DefaultConfig()
	doc := SearchDocumentLite{
		LastActiveAt: 1700000000 + 86400*5, // 5 days in the future
		NowUnix:      1700000000,
	}
	got := ExtractLastActiveDays(doc, cfg)
	assert.InDelta(t, 1, got, 1e-9)
}

// Decay days override reshapes the curve.
func TestExtractLastActiveDays_DecayOverride(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LastActiveDecayDays = 7 // faster decay
	const now = 1700000000
	doc := SearchDocumentLite{
		LastActiveAt: now - 7*86400,
		NowUnix:      now,
	}
	got := ExtractLastActiveDays(doc, cfg)
	assert.InDelta(t, 0.5, got, 0.01)
}

// Decay days <= 0 short-circuits to 0.
func TestExtractLastActiveDays_InvalidDecay(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LastActiveDecayDays = 0
	doc := SearchDocumentLite{LastActiveAt: 1700000000, NowUnix: 1700000000}
	assert.Equal(t, 0.0, ExtractLastActiveDays(doc, cfg))
}

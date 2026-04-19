package features

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ExtractNegativeSignals — bounded penalty 0..cap.
func TestExtractNegativeSignals(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name    string
		count   int32
		want    float64
		wantRaw int
	}{
		{"zero state", 0, 0, 0},
		{"one loss", 1, 0.10, 1},
		{"two losses", 2, 0.20, 2},
		{"three losses saturate cap", 3, 0.30, 3},
		{"ten losses still capped", 10, 0.30, 10},
		{"negative defensively coerced", -1, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := SearchDocumentLite{LostDisputesCount: tt.count}
			pen, raw := ExtractNegativeSignals(doc, cfg)
			assert.InDelta(t, tt.want, pen, 1e-9)
			assert.Equal(t, tt.wantRaw, raw)
			assert.GreaterOrEqual(t, pen, 0.0)
			assert.LessOrEqual(t, pen, cfg.DisputePenaltyCap)
		})
	}
}

// DisputePenalty <= 0 short-circuits to 0 regardless of raw count.
func TestExtractNegativeSignals_DisabledPenalty(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DisputePenalty = 0
	doc := SearchDocumentLite{LostDisputesCount: 20}
	pen, raw := ExtractNegativeSignals(doc, cfg)
	assert.Equal(t, 0.0, pen)
	assert.Equal(t, 20, raw) // raw still surfaced for logging
}

// Custom penalty cap honored.
func TestExtractNegativeSignals_CustomCap(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DisputePenalty = 0.20
	cfg.DisputePenaltyCap = 0.80
	doc := SearchDocumentLite{LostDisputesCount: 5}
	pen, _ := ExtractNegativeSignals(doc, cfg)
	assert.InDelta(t, 0.80, pen, 1e-9)
}

// Negative cap defensively coerced to 0 (output must always be >= 0).
func TestExtractNegativeSignals_NegativeCap(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DisputePenalty = 0.10
	cfg.DisputePenaltyCap = -0.5
	doc := SearchDocumentLite{LostDisputesCount: 3}
	pen, _ := ExtractNegativeSignals(doc, cfg)
	assert.Equal(t, 0.0, pen)
}

package features

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ExtractResponseRate is a pass-through clamp : signal in [0,1] out, junk in
// out coerced to a safe default.
func TestExtractResponseRate(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{"zero", 0, 0},
		{"low", 0.3, 0.3},
		{"typical", 0.8, 0.8},
		{"perfect", 1, 1},
		{"above 1 clamped", 1.5, 1},
		{"negative clamped", -0.5, 0},
		{"NaN coerced", math.NaN(), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := SearchDocumentLite{ResponseRate: tt.in}
			assert.InDelta(t, tt.want, ExtractResponseRate(doc), 1e-9)
		})
	}
}

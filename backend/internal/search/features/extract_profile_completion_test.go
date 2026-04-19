package features

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ExtractProfileCompletion maps 0-100 -> 0-1, clamped.
func TestExtractProfileCompletion(t *testing.T) {
	tests := []struct {
		name string
		in   int32
		want float64
	}{
		{"zero", 0, 0},
		{"25", 25, 0.25},
		{"50", 50, 0.5},
		{"75", 75, 0.75},
		{"100", 100, 1},
		{"above 100 clamps", 150, 1},
		{"negative clamps", -20, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := SearchDocumentLite{ProfileCompletionScore: tt.in}
			assert.InDelta(t, tt.want, ExtractProfileCompletion(doc), 1e-9)
		})
	}
}

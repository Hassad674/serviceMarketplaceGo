package features

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ExtractTextMatch normalises the bucketed Typesense score to [0, 1] and
// surfaces the raw bucket so anti-gaming can apply the stuffing penalty.
func TestExtractTextMatch(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name        string
		bucket      int
		wantScore   float64
		wantRawKept int
	}{
		{"zero bucket -> 0", 0, 0.0, 0},
		{"bucket 1 -> 0.1", 1, 0.1, 1},
		{"bucket 5 -> 0.5", 5, 0.5, 5},
		{"bucket 10 -> 1.0", 10, 1.0, 10},
		{"bucket above 10 clamps", 12, 1.0, 10},
		{"negative bucket coerced to 0", -3, 0.0, 0},
		{"typical mid-bucket", 7, 0.7, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := SearchDocumentLite{TextMatchBucket: tt.bucket}
			score, raw := ExtractTextMatch(Query{}, doc, cfg)
			assert.InDelta(t, tt.wantScore, score, 1e-9)
			assert.Equal(t, tt.wantRawKept, raw)
		})
	}
}

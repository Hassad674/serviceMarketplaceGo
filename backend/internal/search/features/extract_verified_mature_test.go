package features

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ExtractVerifiedMature is {0, 1} — both conditions must hold.
func TestExtractVerifiedMature(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name       string
		verified   bool
		ageDays    int32
		want       float64
	}{
		{"verified + 30 days -> 1", true, 30, 1},
		{"verified + 29 days -> 0 (below threshold)", true, 29, 0},
		{"verified + 100 days -> 1", true, 100, 1},
		{"verified + 0 days -> 0", true, 0, 0},
		{"unverified + 400 days -> 0", false, 400, 0},
		{"zero state -> 0", false, 0, 0},
		{"verified + negative days -> 0", true, -5, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := SearchDocumentLite{
				IsVerified:     tt.verified,
				AccountAgeDays: tt.ageDays,
			}
			assert.Equal(t, tt.want, ExtractVerifiedMature(doc, cfg))
		})
	}
}

// Env override tightens / loosens threshold.
func TestExtractVerifiedMature_Configurable(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VerifiedMatureMinAgeDays = 7
	doc := SearchDocumentLite{IsVerified: true, AccountAgeDays: 10}
	assert.Equal(t, 1.0, ExtractVerifiedMature(doc, cfg))

	cfg.VerifiedMatureMinAgeDays = 60
	assert.Equal(t, 0.0, ExtractVerifiedMature(doc, cfg))
}

package features

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ExtractAccountAgeBonus — log-scaled, capped at 1 year.
func TestExtractAccountAgeBonus(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name    string
		days    int32
		wantMin float64
		wantMax float64
	}{
		{"zero", 0, 0, 0.001},
		{"7 days ≈ 0.35", 7, 0.33, 0.37},
		{"30 days ≈ 0.58", 30, 0.56, 0.60},
		{"90 days ≈ 0.77", 90, 0.75, 0.79},
		{"365 days ≈ 1.00", 365, 0.99, 1.00},
		{"730 days capped", 730, 1.00, 1.00},
		{"5 years capped", 1825, 1.00, 1.00},
		{"negative coerced to 0", -10, 0, 0.001},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := SearchDocumentLite{AccountAgeDays: tt.days}
			got := ExtractAccountAgeBonus(doc, cfg)
			assert.GreaterOrEqual(t, got, tt.wantMin)
			assert.LessOrEqual(t, got, tt.wantMax)
		})
	}
}

// Zero cap in Config short-circuits to 0 (defensive).
func TestExtractAccountAgeBonus_InvalidCap(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AccountAgeCapDays = 0
	doc := SearchDocumentLite{AccountAgeDays: 100}
	assert.Equal(t, 0.0, ExtractAccountAgeBonus(doc, cfg))
}

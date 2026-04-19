package antigaming

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/search/features"
)

// newAccountRule flags the candidate + zeros out AccountAgeBonus when the
// profile is younger than cfg.NewAccountAgeDays.
func TestNewAccountRule(t *testing.T) {
	cfg := DefaultConfig() // NewAccountAgeDays = 7

	tests := []struct {
		name         string
		ageDays      int
		bonusBefore  float64
		wantCapped   bool
		wantPen      bool
		wantBonus    float64
	}{
		{"0 days ignored (no data)", 0, 0.35, false, false, 0.35},
		{"1 day -> capped", 1, 0.2, true, true, 0},
		{"6 days -> capped", 6, 0.35, true, true, 0},
		{"7 days -> not capped (spec: 'younger than 7')", 7, 0.4, false, false, 0.4},
		{"30 days -> not capped", 30, 0.6, false, false, 0.6},
		{"negative days ignored", -5, 0.3, false, false, 0.3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &features.Features{AccountAgeBonus: tt.bonusBefore}
			raw := RawSignals{
				ProfileID:      "p1",
				Persona:        features.PersonaFreelance,
				AccountAgeDays: tt.ageDays,
			}
			pen, capped := newAccountRule(f, raw, cfg)
			assert.Equal(t, tt.wantCapped, capped)
			if tt.wantPen {
				assert.NotNil(t, pen)
				assert.Equal(t, RuleNewAccount, pen.Rule)
			} else {
				assert.Nil(t, pen)
			}
			assert.InDelta(t, tt.wantBonus, f.AccountAgeBonus, 1e-9)
		})
	}
}

// NewAccountAgeDays <= 0 disables the rule.
func TestNewAccountRule_DisabledViaConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NewAccountAgeDays = 0
	f := &features.Features{AccountAgeBonus: 0.5}
	raw := RawSignals{AccountAgeDays: 1}
	pen, capped := newAccountRule(f, raw, cfg)
	assert.Nil(t, pen)
	assert.False(t, capped)
	assert.InDelta(t, 0.5, f.AccountAgeBonus, 1e-9)
}

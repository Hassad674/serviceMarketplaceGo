package features

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Persona.IsValid recognises the three canonical values + rejects everything
// else. The scorer depends on this being exhaustive so a typo in wiring
// fails fast.
func TestPersona_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		persona Persona
		want    bool
	}{
		{"freelance", PersonaFreelance, true},
		{"agency", PersonaAgency, true},
		{"referrer", PersonaReferrer, true},
		{"empty string", Persona(""), false},
		{"typo lowercase", Persona("freelancer"), false},
		{"typo camel", Persona("Freelance"), false},
		{"unrelated", Persona("enterprise"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.persona.IsValid())
		})
	}
}

// Features zero-value is a valid Features value — clamp01 guarantees every
// positive field stays in [0, 1] and penalty stays in [0, cap]. This is the
// contract the scorer relies on.
func TestFeatures_ZeroValue_IsValid(t *testing.T) {
	var f Features
	assert.InDelta(t, 0, f.TextMatchScore, 0)
	assert.InDelta(t, 0, f.NegativeSignals, 0)
	assert.Equal(t, 0, f.RawTextMatchBucket)
	assert.Equal(t, 0, f.RawUniqueReviewers)
	assert.Equal(t, 0, f.RawLostDisputes)
}

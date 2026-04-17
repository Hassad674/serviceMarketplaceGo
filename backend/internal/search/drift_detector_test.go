package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectDrift_NoDrift(t *testing.T) {
	postgres := []PersonaCount{
		{PersonaFreelance, 100},
		{PersonaAgency, 50},
		{PersonaReferrer, 20},
	}
	typesense := []PersonaCount{
		{PersonaFreelance, 100},
		{PersonaAgency, 50},
		{PersonaReferrer, 20},
	}
	report := DetectDrift(postgres, typesense, DetectDriftOpts{})
	assert.False(t, report.IsCritical)
	assert.Equal(t, 0.0, report.MaxRatio)
}

func TestDetectDrift_BelowThreshold(t *testing.T) {
	// 100 vs 100 → 0% drift
	// 500 vs 498 → 0.4% drift (under 0.5%)
	postgres := []PersonaCount{{PersonaFreelance, 500}}
	typesense := []PersonaCount{{PersonaFreelance, 498}}
	report := DetectDrift(postgres, typesense, DetectDriftOpts{})
	assert.False(t, report.IsCritical)
	assert.InDelta(t, 2.0/500.0, report.MaxRatio, 0.0001)
}

func TestDetectDrift_AboveThreshold(t *testing.T) {
	// 100 vs 95 → 5% drift (above 0.5%)
	postgres := []PersonaCount{{PersonaFreelance, 100}}
	typesense := []PersonaCount{{PersonaFreelance, 95}}
	report := DetectDrift(postgres, typesense, DetectDriftOpts{})
	assert.True(t, report.IsCritical)
	assert.InDelta(t, 0.05, report.MaxRatio, 0.0001)
}

func TestDetectDrift_CustomThreshold(t *testing.T) {
	postgres := []PersonaCount{{PersonaFreelance, 100}}
	typesense := []PersonaCount{{PersonaFreelance, 99}}
	// 1% drift, threshold 2% → not critical.
	report := DetectDrift(postgres, typesense, DetectDriftOpts{Threshold: 0.02})
	assert.False(t, report.IsCritical)
}

func TestDetectDrift_MissingPersonaOnOneSide(t *testing.T) {
	// Typesense has no agency entries but Postgres does → 100% drift.
	postgres := []PersonaCount{
		{PersonaFreelance, 50},
		{PersonaAgency, 10},
	}
	typesense := []PersonaCount{
		{PersonaFreelance, 50},
	}
	report := DetectDrift(postgres, typesense, DetectDriftOpts{})
	assert.True(t, report.IsCritical)
	assert.Equal(t, 1.0, report.Ratios[PersonaAgency])
}

func TestComputeDriftRatio_Cases(t *testing.T) {
	cases := []struct {
		a, b int64
		want float64
	}{
		{0, 0, 0},
		{0, 10, 1},
		{10, 0, 1},
		{100, 100, 0},
		{100, 95, 0.05},
		{95, 100, 0.05},
	}
	for _, tc := range cases {
		got := computeDriftRatio(tc.a, tc.b)
		assert.InDelta(t, tc.want, got, 0.0001, "a=%d b=%d", tc.a, tc.b)
	}
}

package antigaming

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// DefaultConfig returns the safe public values documented in §11.4.
func TestDefaultConfig_Values(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 5, cfg.MaxTokenRepetition)
	assert.InDelta(t, 0.3, cfg.MinDistinctRatio, 1e-9)
	assert.InDelta(t, 0.5, cfg.StuffingPenalty, 1e-9)
	assert.Equal(t, 5, cfg.VelocityCap24h)
	assert.Equal(t, 14, cfg.VelocityCooldownDays)
	assert.InDelta(t, 0.3, cfg.LinkedMaxFraction, 1e-9)
	assert.Equal(t, 3, cfg.UniqueReviewerFloor)
	assert.InDelta(t, 0.4, cfg.FewReviewerCap, 1e-9)
	assert.Equal(t, 7, cfg.NewAccountAgeDays)
}

// LoadConfigFromEnv reads every env var, defaults on invalid values.
func TestLoadConfigFromEnv(t *testing.T) {
	tests := []struct {
		name   string
		env    map[string]string
		assert func(t *testing.T, cfg Config)
	}{
		{
			name: "no env vars -> defaults",
			env:  nil,
			assert: func(t *testing.T, cfg Config) {
				assert.Equal(t, DefaultConfig(), cfg)
			},
		},
		{
			name: "override stuffing thresholds",
			env: map[string]string{
				"RANKING_AG_MAX_TOKEN_REPETITION": "10",
				"RANKING_AG_MIN_DISTINCT_RATIO":   "0.5",
				"RANKING_STUFFING_PENALTY":        "0.25",
			},
			assert: func(t *testing.T, cfg Config) {
				assert.Equal(t, 10, cfg.MaxTokenRepetition)
				assert.InDelta(t, 0.5, cfg.MinDistinctRatio, 1e-9)
				assert.InDelta(t, 0.25, cfg.StuffingPenalty, 1e-9)
			},
		},
		{
			name: "override velocity + linked + floor",
			env: map[string]string{
				"RANKING_AG_VELOCITY_CAP_24H":      "10",
				"RANKING_AG_VELOCITY_COOLDOWN_DAYS": "7",
				"RANKING_AG_LINKED_MAX_FRACTION":   "0.1",
				"RANKING_AG_UNIQUE_REVIEWER_FLOOR": "5",
				"RANKING_AG_FEW_REVIEWER_CAP":      "0.2",
				"RANKING_AG_NEW_ACCOUNT_AGE_DAYS":  "14",
			},
			assert: func(t *testing.T, cfg Config) {
				assert.Equal(t, 10, cfg.VelocityCap24h)
				assert.Equal(t, 7, cfg.VelocityCooldownDays)
				assert.InDelta(t, 0.1, cfg.LinkedMaxFraction, 1e-9)
				assert.Equal(t, 5, cfg.UniqueReviewerFloor)
				assert.InDelta(t, 0.2, cfg.FewReviewerCap, 1e-9)
				assert.Equal(t, 14, cfg.NewAccountAgeDays)
			},
		},
		{
			name: "malformed values fall back",
			env: map[string]string{
				"RANKING_AG_MAX_TOKEN_REPETITION": "abc",
				"RANKING_AG_MIN_DISTINCT_RATIO":   "not-a-float",
			},
			assert: func(t *testing.T, cfg Config) {
				assert.Equal(t, DefaultConfig().MaxTokenRepetition, cfg.MaxTokenRepetition)
				assert.InDelta(t, DefaultConfig().MinDistinctRatio, cfg.MinDistinctRatio, 1e-9)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			cfg := LoadConfigFromEnv()
			tt.assert(t, cfg)
		})
	}
}

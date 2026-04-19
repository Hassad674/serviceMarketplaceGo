package features

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// DefaultConfig returns the safe public defaults documented in §11. Locking
// each value pin prevents accidental drift from the spec.
func TestDefaultConfig_Defaults(t *testing.T) {
	cfg := DefaultConfig()
	assert.InDelta(t, 4.0, cfg.BayesianPriorMean, 1e-9)
	assert.InDelta(t, 8.0, cfg.BayesianPriorWeight, 1e-9)
	assert.InDelta(t, 0.15, cfg.ColdStartFloor, 1e-9)
	assert.Equal(t, 50, cfg.ReviewCountCap)
	assert.Equal(t, 100, cfg.ProjectCountCap)
	assert.Equal(t, 365, cfg.AccountAgeCapDays)
	assert.Equal(t, 30, cfg.VerifiedMatureMinAgeDays)
	assert.Equal(t, 30, cfg.LastActiveDecayDays)
	assert.InDelta(t, 0.10, cfg.DisputePenalty, 1e-9)
	assert.InDelta(t, 0.30, cfg.DisputePenaltyCap, 1e-9)
}

// LoadConfigFromEnv reads every env var, falls back gracefully on invalid
// input, and leaves the defaults untouched when nothing is set.
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
			name: "override bayesian prior",
			env: map[string]string{
				"RANKING_BAYESIAN_PRIOR_MEAN":   "3.5",
				"RANKING_BAYESIAN_PRIOR_WEIGHT": "12",
			},
			assert: func(t *testing.T, cfg Config) {
				assert.InDelta(t, 3.5, cfg.BayesianPriorMean, 1e-9)
				assert.InDelta(t, 12, cfg.BayesianPriorWeight, 1e-9)
			},
		},
		{
			name: "override caps",
			env: map[string]string{
				"RANKING_REVIEW_COUNT_CAP":    "25",
				"RANKING_PROJECT_COUNT_CAP":   "250",
				"RANKING_ACCOUNT_AGE_CAP_DAYS": "730",
			},
			assert: func(t *testing.T, cfg Config) {
				assert.Equal(t, 25, cfg.ReviewCountCap)
				assert.Equal(t, 250, cfg.ProjectCountCap)
				assert.Equal(t, 730, cfg.AccountAgeCapDays)
			},
		},
		{
			name: "malformed values fall back to defaults",
			env: map[string]string{
				"RANKING_BAYESIAN_PRIOR_MEAN": "not-a-number",
				"RANKING_REVIEW_COUNT_CAP":    "abc",
			},
			assert: func(t *testing.T, cfg Config) {
				assert.InDelta(t, DefaultConfig().BayesianPriorMean, cfg.BayesianPriorMean, 1e-9)
				assert.Equal(t, DefaultConfig().ReviewCountCap, cfg.ReviewCountCap)
			},
		},
		{
			name: "override dispute penalty",
			env: map[string]string{
				"RANKING_DISPUTE_PENALTY":     "0.20",
				"RANKING_DISPUTE_PENALTY_CAP": "0.50",
			},
			assert: func(t *testing.T, cfg Config) {
				assert.InDelta(t, 0.20, cfg.DisputePenalty, 1e-9)
				assert.InDelta(t, 0.50, cfg.DisputePenaltyCap, 1e-9)
			},
		},
		{
			name: "empty string treated as unset",
			env: map[string]string{
				"RANKING_COLD_START_FLOOR": "",
			},
			assert: func(t *testing.T, cfg Config) {
				assert.InDelta(t, 0.15, cfg.ColdStartFloor, 1e-9)
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

package features

import (
	"os"
	"strconv"
)

// Config is the immutable parameter bundle read from env vars at backend
// startup and then handed to every extractor by value. We pass a Config value
// (not a pointer) so extractors cannot accidentally mutate it — formula
// parameters are part of the ranking contract and must stay stable across
// concurrent requests.
//
// All env vars follow the RANKING_* namespace documented in
// `docs/ranking-v1.md` §11. Unset or malformed variables fall back to the safe
// open-source defaults ; the real production values live in the Railway /
// Vercel environment.
type Config struct {
	// Rating — §3.2-3
	BayesianPriorMean   float64 // RANKING_BAYESIAN_PRIOR_MEAN       default 4.0
	BayesianPriorWeight float64 // RANKING_BAYESIAN_PRIOR_WEIGHT     default 8
	ColdStartFloor      float64 // RANKING_COLD_START_FLOOR          default 0.15
	ReviewCountCap      int     // RANKING_REVIEW_COUNT_CAP          default 50

	// Proven-work — §3.2-4
	ProjectCountCap int // RANKING_PROJECT_COUNT_CAP          default 100

	// Account-age — §3.2-9
	AccountAgeCapDays int // RANKING_ACCOUNT_AGE_CAP_DAYS       default 365

	// Verified-mature — §3.2-6
	VerifiedMatureMinAgeDays int // implicit constant in spec (30), kept env-tunable

	// Last-active — §3.2-8 decay denominator
	LastActiveDecayDays int // spec-fixed 30, kept env-tunable for future A/B

	// Negative penalty — §5.3
	DisputePenalty    float64 // RANKING_DISPUTE_PENALTY           default 0.10
	DisputePenaltyCap float64 // RANKING_DISPUTE_PENALTY_CAP       default 0.30
}

// DefaultConfig returns the safe public defaults listed in
// `docs/ranking-v1.md` §11. Used as the starting point of LoadConfigFromEnv so
// any unset variable falls through to the documented value.
func DefaultConfig() Config {
	return Config{
		BayesianPriorMean:        4.0,
		BayesianPriorWeight:      8,
		ColdStartFloor:           0.15,
		ReviewCountCap:           50,
		ProjectCountCap:          100,
		AccountAgeCapDays:        365,
		VerifiedMatureMinAgeDays: 30,
		LastActiveDecayDays:      30,
		DisputePenalty:           0.10,
		DisputePenaltyCap:        0.30,
	}
}

// LoadConfigFromEnv reads the RANKING_* env vars + returns a populated Config.
// Malformed values fall back to the default (so a typo in production never
// takes down the ranking pipeline). The function is deliberately small + pure
// so tests can substitute it with a table-driven equivalent.
func LoadConfigFromEnv() Config {
	cfg := DefaultConfig()
	cfg.BayesianPriorMean = floatEnv("RANKING_BAYESIAN_PRIOR_MEAN", cfg.BayesianPriorMean)
	cfg.BayesianPriorWeight = floatEnv("RANKING_BAYESIAN_PRIOR_WEIGHT", cfg.BayesianPriorWeight)
	cfg.ColdStartFloor = floatEnv("RANKING_COLD_START_FLOOR", cfg.ColdStartFloor)
	cfg.ReviewCountCap = intEnv("RANKING_REVIEW_COUNT_CAP", cfg.ReviewCountCap)
	cfg.ProjectCountCap = intEnv("RANKING_PROJECT_COUNT_CAP", cfg.ProjectCountCap)
	cfg.AccountAgeCapDays = intEnv("RANKING_ACCOUNT_AGE_CAP_DAYS", cfg.AccountAgeCapDays)
	cfg.VerifiedMatureMinAgeDays = intEnv("RANKING_VERIFIED_MATURE_MIN_AGE_DAYS", cfg.VerifiedMatureMinAgeDays)
	cfg.LastActiveDecayDays = intEnv("RANKING_LAST_ACTIVE_DECAY_DAYS", cfg.LastActiveDecayDays)
	cfg.DisputePenalty = floatEnv("RANKING_DISPUTE_PENALTY", cfg.DisputePenalty)
	cfg.DisputePenaltyCap = floatEnv("RANKING_DISPUTE_PENALTY_CAP", cfg.DisputePenaltyCap)
	return cfg
}

// floatEnv reads a float64 env var, returning fallback on missing or invalid.
func floatEnv(name string, fallback float64) float64 {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return v
}

// intEnv reads an int env var, returning fallback on missing or invalid.
func intEnv(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

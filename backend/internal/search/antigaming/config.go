// Package antigaming implements Stage 3 of the ranking V1 pipeline described
// in `docs/ranking-v1.md` §7 — silent caps and penalties that run between
// feature extraction and scoring.
//
// The package receives a `*features.Features` value + a raw-signal snapshot
// of the candidate document and mutates the features in place when a rule
// fires. Every firing emits a structured log line so an admin dashboard can
// inspect the detection rate later ; the attacker never sees an HTTP error
// that would let them probe thresholds.
//
// Five rules ship in V1 :
//
//  1. Keyword stuffing  — §7.1 : halves text_match_score when the document
//     text is a stuffed cloud of one token.
//  2. Review velocity   — §7.2 : reviews uploaded in bursts within 24h see
//     their effective count reduced, dampening rating_score_diverse.
//  3. Linked accounts   — §7.3 : reviews from users on the same IP / email
//     domain / device fingerprint are discounted. V1 provides a no-op hook
//     and the default detector ships "empty".
//  4. Unique reviewer floor — §7.4 : fewer than 3 distinct reviewers caps
//     rating_score_diverse at 0.4.
//  5. New account cap   — §7.5 : a profile younger than 7 days can at best
//     rank at the persona median (enforced in the scorer ; this package
//     flags the candidate with a marker the scorer reads).
//
// Because the detection interfaces matter more than the numbers, every
// threshold lives in env vars (RANKING_AG_*) — attackers can read the
// formula in the open-source code but not the exact parameters.
package antigaming

import (
	"os"
	"strconv"
)

// Config holds every anti-gaming threshold in the RANKING_AG_* namespace
// documented in `docs/ranking-v1.md` §11.4. Defaults are safe public values.
type Config struct {
	// Stuffing — §7.1
	MaxTokenRepetition int     // RANKING_AG_MAX_TOKEN_REPETITION   default 5
	MinDistinctRatio   float64 // RANKING_AG_MIN_DISTINCT_RATIO     default 0.3
	StuffingPenalty    float64 // RANKING_STUFFING_PENALTY          default 0.5

	// Velocity — §7.2
	VelocityCap24h       int // RANKING_AG_VELOCITY_CAP_24H       default 5
	VelocityCooldownDays int // RANKING_AG_VELOCITY_COOLDOWN_DAYS default 14

	// Linked accounts — §7.3
	LinkedMaxFraction float64 // RANKING_AG_LINKED_MAX_FRACTION    default 0.3

	// Unique reviewer floor — §7.4
	UniqueReviewerFloor int     // RANKING_AG_UNIQUE_REVIEWER_FLOOR default 3
	FewReviewerCap      float64 // RANKING_AG_FEW_REVIEWER_CAP      default 0.4

	// New account cap — §7.5
	NewAccountAgeDays int // RANKING_AG_NEW_ACCOUNT_AGE_DAYS   default 7
}

// DefaultConfig returns the safe public defaults from §11.4.
func DefaultConfig() Config {
	return Config{
		MaxTokenRepetition:   5,
		MinDistinctRatio:     0.3,
		StuffingPenalty:      0.5,
		VelocityCap24h:       5,
		VelocityCooldownDays: 14,
		LinkedMaxFraction:    0.3,
		UniqueReviewerFloor:  3,
		FewReviewerCap:       0.4,
		NewAccountAgeDays:    7,
	}
}

// LoadConfigFromEnv reads the RANKING_AG_* + RANKING_STUFFING_PENALTY env
// vars. Malformed values fall back to the default.
func LoadConfigFromEnv() Config {
	cfg := DefaultConfig()
	cfg.MaxTokenRepetition = intEnv("RANKING_AG_MAX_TOKEN_REPETITION", cfg.MaxTokenRepetition)
	cfg.MinDistinctRatio = floatEnv("RANKING_AG_MIN_DISTINCT_RATIO", cfg.MinDistinctRatio)
	cfg.StuffingPenalty = floatEnv("RANKING_STUFFING_PENALTY", cfg.StuffingPenalty)
	cfg.VelocityCap24h = intEnv("RANKING_AG_VELOCITY_CAP_24H", cfg.VelocityCap24h)
	cfg.VelocityCooldownDays = intEnv("RANKING_AG_VELOCITY_COOLDOWN_DAYS", cfg.VelocityCooldownDays)
	cfg.LinkedMaxFraction = floatEnv("RANKING_AG_LINKED_MAX_FRACTION", cfg.LinkedMaxFraction)
	cfg.UniqueReviewerFloor = intEnv("RANKING_AG_UNIQUE_REVIEWER_FLOOR", cfg.UniqueReviewerFloor)
	cfg.FewReviewerCap = floatEnv("RANKING_AG_FEW_REVIEWER_CAP", cfg.FewReviewerCap)
	cfg.NewAccountAgeDays = intEnv("RANKING_AG_NEW_ACCOUNT_AGE_DAYS", cfg.NewAccountAgeDays)
	return cfg
}

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

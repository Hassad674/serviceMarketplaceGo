// Package rules implements the business-rules layer of the V1 search
// ranking pipeline (§6 + §8 of docs/ranking-v1.md). It sits after the
// composite scorer and transforms a scored candidate list into the
// final top-20 rendered to the user.
//
// Pipeline stages (Apply in order):
//
//  1. Tier sort   — split into Tier A (available_now/soon) and Tier B.
//  2. Randomise   — gaussian noise with rank-dependent sigma.
//  3. Re-sort     — within each tier by noise-adjusted final score.
//  4. Diversity   — break 3+ consecutive same primary_skill runs.
//  5. Rising talent — inject an eligible new+verified candidate at
//     positions 5, 10, 15, 20 when the delta allows.
//  6. Featured    — optional is_featured boost (dormant V1).
//
// The package is pure Go: no I/O, no database, no clock except a
// caller-injectable random source. This matters for testing —
// deterministic seeds make the entire pipeline reproducible.
//
// Open/Closed: adding a new rule means a new file + one call inside
// Apply. The public surface (Candidate, Config, BusinessRules) stays
// stable. The Reranker and Feature contracts live in parallel
// packages; this package stays decoupled by modelling the inputs as
// plain value types (Features + Score below) that the wiring layer
// (cmd/api/main.go, app/search) fills in from the real sources.
package rules

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"marketplace-backend/internal/search/features"
	"marketplace-backend/internal/search/scorer"
)

// Persona re-exports the features.Persona type so callers can use a
// single canonical identifier across the pipeline. Added here as a
// type alias — keeps the rules package's public signatures readable
// without forcing callers to switch imports.
type Persona = features.Persona

// Re-exported persona constants for convenience. Values are
// byte-identical to features.Persona* by contract (single source of
// truth lives in the features package).
const (
	PersonaFreelance = features.PersonaFreelance
	PersonaAgency    = features.PersonaAgency
	PersonaReferrer  = features.PersonaReferrer
)

// Features is a type alias for features.Features. The rules layer
// consumes the exact vector the extractor produces — no conversion
// layer, no field drift.
type Features = features.Features

// Score is a type alias for scorer.RankedScore. Kept as an alias so
// callers can pass the scorer's direct output without an adapter.
type Score = scorer.RankedScore

// Candidate is the unit the business rules operate on. It bundles
// the feature vector + composite score + the raw SearchDocument
// fields the rules actually read.
//
// OrganizationID is the real marketplace org UUID (used for LTR
// joins downstream). DocumentID is the Typesense primary key.
type Candidate struct {
	DocumentID         string
	OrganizationID     string
	Persona            Persona
	Feat               Features
	Score              Score
	AvailabilityStatus string // available_now | available_soon | not_available
	PrimarySkill       string // first skill in the profile — used by diversity
	AccountAgeDays     int    // convenience copy of Feat.AccountAgeDays
	IsFeatured         bool   // admin override (dormant V1 unless enabled)
	IsVerified         bool   // KYC result
}

// Config bundles every tuneable knob the rules layer respects.
// Defaults track §6 and §8 of docs/ranking-v1.md; production ops
// override them via env vars loaded by LoadConfigFromEnv.
//
// RandSeed: when non-zero the pipeline uses a deterministic Mulberry-
// style source so unit tests are reproducible. Zero (the default)
// falls back to a time-seeded *rand.Rand during New construction.
type Config struct {
	NoiseCoefficient      float64
	NoiseTop3Multiplier   float64
	NoiseMidMultiplier    float64
	NoiseTailMultiplier   float64
	RisingTalentMaxAge    int     // days
	RisingTalentSlotEvery int     // every N slots
	RisingTalentDelta     float64 // slot swap threshold (§6.3) — default 5
	FeaturedBoost         float64
	FeaturedEnabled       bool
	TopN                  int   // final output size. Default 20.
	RandSeed              int64 // 0 = nondeterministic seed at init
}

// DefaultConfig returns the published-default knob set. Pure function
// so tests can grab a known-safe baseline without parsing env vars.
func DefaultConfig() Config {
	return Config{
		NoiseCoefficient:      0.006,
		NoiseTop3Multiplier:   0.3,
		NoiseMidMultiplier:    0.8,
		NoiseTailMultiplier:   1.5,
		RisingTalentMaxAge:    60,
		RisingTalentSlotEvery: 5,
		RisingTalentDelta:     5.0,
		FeaturedBoost:         0.0,
		FeaturedEnabled:       false,
		TopN:                  20,
		RandSeed:              0,
	}
}

// LoadConfigFromEnv reads every RANKING_* variable the rules package
// understands and overlays it on top of DefaultConfig. Missing or
// empty vars keep their default. Malformed numbers return an error
// so the backend boot-time fails loud instead of limping with a
// zero-valued knob.
//
// Parsed vars:
//
//	RANKING_NOISE_COEFFICIENT         float  (default 0.006)
//	RANKING_NOISE_TOP3_MULTIPLIER     float  (default 0.3)
//	RANKING_NOISE_MID_MULTIPLIER      float  (default 0.8)
//	RANKING_NOISE_TAIL_MULTIPLIER     float  (default 1.5)
//	RANKING_RISING_TALENT_MAX_AGE     int    (default 60)
//	RANKING_RISING_TALENT_SLOT_EVERY  int    (default 5)
//	RANKING_RISING_TALENT_DELTA       float  (default 5.0)
//	RANKING_FEATURED_BOOST            float  (default 0.0)
//	RANKING_FEATURED_ENABLED          bool   (default false)
//	RANKING_RULES_TOP_N               int    (default 20)
//	RANKING_RULES_SEED                int    (default 0 → nondeterministic)
func LoadConfigFromEnv() (Config, error) {
	cfg := DefaultConfig()
	if err := readFloat("RANKING_NOISE_COEFFICIENT", &cfg.NoiseCoefficient); err != nil {
		return cfg, err
	}
	if err := readFloat("RANKING_NOISE_TOP3_MULTIPLIER", &cfg.NoiseTop3Multiplier); err != nil {
		return cfg, err
	}
	if err := readFloat("RANKING_NOISE_MID_MULTIPLIER", &cfg.NoiseMidMultiplier); err != nil {
		return cfg, err
	}
	if err := readFloat("RANKING_NOISE_TAIL_MULTIPLIER", &cfg.NoiseTailMultiplier); err != nil {
		return cfg, err
	}
	if err := readInt("RANKING_RISING_TALENT_MAX_AGE", &cfg.RisingTalentMaxAge); err != nil {
		return cfg, err
	}
	if err := readInt("RANKING_RISING_TALENT_SLOT_EVERY", &cfg.RisingTalentSlotEvery); err != nil {
		return cfg, err
	}
	if err := readFloat("RANKING_RISING_TALENT_DELTA", &cfg.RisingTalentDelta); err != nil {
		return cfg, err
	}
	if err := readFloat("RANKING_FEATURED_BOOST", &cfg.FeaturedBoost); err != nil {
		return cfg, err
	}
	if err := readBool("RANKING_FEATURED_ENABLED", &cfg.FeaturedEnabled); err != nil {
		return cfg, err
	}
	if err := readInt("RANKING_RULES_TOP_N", &cfg.TopN); err != nil {
		return cfg, err
	}
	if err := readInt64("RANKING_RULES_SEED", &cfg.RandSeed); err != nil {
		return cfg, err
	}
	return cfg, cfg.validate()
}

// validate guards against obviously-wrong knob combinations. Run
// after LoadConfigFromEnv so prod ops see a meaningful error at
// boot time rather than a silent mis-ranking.
func (c Config) validate() error {
	if c.NoiseCoefficient < 0 {
		return errors.New("rules config: NoiseCoefficient must be >= 0")
	}
	if c.RisingTalentSlotEvery <= 0 {
		return errors.New("rules config: RisingTalentSlotEvery must be > 0")
	}
	if c.RisingTalentMaxAge < 0 {
		return errors.New("rules config: RisingTalentMaxAge must be >= 0")
	}
	if c.TopN <= 0 {
		return errors.New("rules config: TopN must be > 0")
	}
	if c.FeaturedBoost < 0 {
		return errors.New("rules config: FeaturedBoost must be >= 0")
	}
	return nil
}

func readFloat(key string, dst *float64) error {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fmt.Errorf("rules config: %s: %w", key, err)
	}
	*dst = v
	return nil
}

func readInt(key string, dst *int) error {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fmt.Errorf("rules config: %s: %w", key, err)
	}
	*dst = v
	return nil
}

func readInt64(key string, dst *int64) error {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fmt.Errorf("rules config: %s: %w", key, err)
	}
	*dst = v
	return nil
}

func readBool(key string, dst *bool) error {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return fmt.Errorf("rules config: %s: %w", key, err)
	}
	*dst = v
	return nil
}

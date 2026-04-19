package scorer

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
)

// PersonaWeights holds the 9 per-feature weights for a single persona.
// Weights MUST sum to 1.0 (within floatTolerance) — enforced by Validate
// at config load time and asserted by unit tests on the hardcoded
// defaults. Field ordering mirrors the docs/ranking-v1.md §4 tables so
// reviewers can cross-reference weights.go against the spec without
// scrolling.
type PersonaWeights struct {
	TextMatch      float64
	SkillsOverlap  float64
	Rating         float64
	ProvenWork     float64
	ResponseRate   float64
	VerifiedMature float64
	Completion     float64
	LastActive     float64
	AccountAge     float64
}

// Config is the full tri-persona weight table loaded from environment
// variables. It is the only type NewWeightedScorer consumes.
type Config struct {
	Freelance PersonaWeights
	Agency    PersonaWeights
	Referrer  PersonaWeights
}

// floatTolerance is the absolute error accepted when asserting that the
// 9 weights of a persona sum to 1.0. Chosen at 1e-9 so rounding noise
// across IEEE 754 additions does not spuriously reject a well-formed
// weight table while still catching typos (a 1% drift is 1e-2, nine
// orders of magnitude above the tolerance).
const floatTolerance = 1e-9

// ErrWeightsSum is returned by Validate when a persona's nine weights
// do not sum to 1.0 within floatTolerance. Emitted by LoadConfigFromEnv
// so the backend fails fast at startup rather than silently producing
// biased scores.
var ErrWeightsSum = errors.New("scorer: persona weights must sum to 1.0")

// Sum returns the algebraic sum of the nine per-feature weights. Used
// by Validate and by the empty-query redistribution invariant test.
func (w PersonaWeights) Sum() float64 {
	return w.TextMatch + w.SkillsOverlap + w.Rating + w.ProvenWork +
		w.ResponseRate + w.VerifiedMature + w.Completion +
		w.LastActive + w.AccountAge
}

// Validate returns ErrWeightsSum (wrapped with the persona name) when
// the weights do not total 1.0. name is used only for the error
// message, so callers pass "freelance", "agency", or "referrer" to get
// a readable diagnostic.
func (w PersonaWeights) Validate(name string) error {
	delta := math.Abs(w.Sum() - 1.0)
	if delta > floatTolerance {
		return fmt.Errorf("%w: persona=%s sum=%.12f delta=%.2e",
			ErrWeightsSum, name, w.Sum(), delta)
	}
	return nil
}

// Validate checks all three personas at once. Returns the first error
// encountered so the operator sees one issue per start-up attempt and
// can fix-and-retry quickly.
func (c Config) Validate() error {
	if err := c.Freelance.Validate("freelance"); err != nil {
		return err
	}
	if err := c.Agency.Validate("agency"); err != nil {
		return err
	}
	if err := c.Referrer.Validate("referrer"); err != nil {
		return err
	}
	return nil
}

// DefaultFreelanceWeights returns the freelance weight table locked in
// docs/ranking-v1.md §4.1 / §11.1. The sum is guaranteed to be 1.0 by
// a unit test (TestDefaultWeights_SumToOne).
func DefaultFreelanceWeights() PersonaWeights {
	return PersonaWeights{
		TextMatch:      0.20,
		SkillsOverlap:  0.15,
		Rating:         0.20,
		ProvenWork:     0.15,
		ResponseRate:   0.10,
		VerifiedMature: 0.08,
		Completion:     0.07,
		LastActive:     0.03,
		AccountAge:     0.02,
	}
}

// DefaultAgencyWeights returns the agency weight table from §4.2.
func DefaultAgencyWeights() PersonaWeights {
	return PersonaWeights{
		TextMatch:      0.15,
		SkillsOverlap:  0.10,
		Rating:         0.25,
		ProvenWork:     0.25,
		ResponseRate:   0.05,
		VerifiedMature: 0.10,
		Completion:     0.07,
		LastActive:     0.02,
		AccountAge:     0.01,
	}
}

// DefaultReferrerWeights returns the referrer weight table from §4.3.
// Notably, SkillsOverlap and ProvenWork are zero — referrers don't sell
// skills and don't complete projects themselves.
func DefaultReferrerWeights() PersonaWeights {
	return PersonaWeights{
		TextMatch:      0.20,
		SkillsOverlap:  0.00,
		Rating:         0.35,
		ProvenWork:     0.00,
		ResponseRate:   0.20,
		VerifiedMature: 0.10,
		Completion:     0.10,
		LastActive:     0.03,
		AccountAge:     0.02,
	}
}

// DefaultConfig returns the full three-persona Config with hardcoded
// defaults from §11.1. Used as the fallback by LoadConfigFromEnv when
// no environment override is present.
func DefaultConfig() Config {
	return Config{
		Freelance: DefaultFreelanceWeights(),
		Agency:    DefaultAgencyWeights(),
		Referrer:  DefaultReferrerWeights(),
	}
}

// envWeightKey maps a persona + feature pair to its env-var name. Kept
// as a small helper so new personas or features can be added without
// sprinkling string-concat across the file.
func envWeightKey(persona, feature string) string {
	return "RANKING_WEIGHTS_" + persona + "_" + feature
}

// loadFloatOrDefault reads the env var at key and returns its float64
// value. If the var is unset or empty it returns fallback with nil
// error. If the var is set but not parseable as a float, returns an
// error so startup fails loudly.
func loadFloatOrDefault(key string, fallback float64) (float64, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("scorer: env %s=%q is not a valid float: %w", key, raw, err)
	}
	return v, nil
}

// loadPersonaFromEnv reads the nine env vars for a single persona,
// falling back to the provided defaults field by field. The persona
// argument is the uppercase string stamped into the env-var name
// ("FREELANCE", "AGENCY", "REFERRER").
func loadPersonaFromEnv(persona string, defaults PersonaWeights) (PersonaWeights, error) {
	type fieldBinding struct {
		suffix string
		target *float64
		fallbk float64
	}
	out := defaults
	bindings := []fieldBinding{
		{"TEXT_MATCH", &out.TextMatch, defaults.TextMatch},
		{"SKILLS_OVERLAP", &out.SkillsOverlap, defaults.SkillsOverlap},
		{"RATING", &out.Rating, defaults.Rating},
		{"PROVEN_WORK", &out.ProvenWork, defaults.ProvenWork},
		{"RESPONSE_RATE", &out.ResponseRate, defaults.ResponseRate},
		{"VERIFIED_MATURE", &out.VerifiedMature, defaults.VerifiedMature},
		{"COMPLETION", &out.Completion, defaults.Completion},
		{"LAST_ACTIVE", &out.LastActive, defaults.LastActive},
		{"ACCOUNT_AGE", &out.AccountAge, defaults.AccountAge},
	}
	for _, b := range bindings {
		v, err := loadFloatOrDefault(envWeightKey(persona, b.suffix), b.fallbk)
		if err != nil {
			return PersonaWeights{}, err
		}
		*b.target = v
	}
	return out, nil
}

// LoadConfigFromEnv builds a Config by reading RANKING_WEIGHTS_* env
// vars, falling back to the hardcoded defaults for any unset key. The
// resulting Config is validated (each persona sums to 1.0) before
// being returned; a failed validation surfaces ErrWeightsSum so the
// operator knows exactly which persona is misconfigured.
func LoadConfigFromEnv() (Config, error) {
	freelance, err := loadPersonaFromEnv("FREELANCE", DefaultFreelanceWeights())
	if err != nil {
		return Config{}, err
	}
	agency, err := loadPersonaFromEnv("AGENCY", DefaultAgencyWeights())
	if err != nil {
		return Config{}, err
	}
	referrer, err := loadPersonaFromEnv("REFERRER", DefaultReferrerWeights())
	if err != nil {
		return Config{}, err
	}
	cfg := Config{Freelance: freelance, Agency: agency, Referrer: referrer}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Select returns the weight table for the requested persona. An unknown
// persona falls back to the freelance weights (safe default: freelance
// is the most populated persona in the dataset) but this should be
// considered a programmer error — the caller typed the persona from a
// canonical enum that should never drift.
func (c Config) Select(persona Persona) PersonaWeights {
	switch persona {
	case PersonaAgency:
		return c.Agency
	case PersonaReferrer:
		return c.Referrer
	case PersonaFreelance:
		return c.Freelance
	default:
		return c.Freelance
	}
}

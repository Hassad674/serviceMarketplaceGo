package scorer

import (
	"errors"
	"math"
	"strings"
	"testing"
)

// TestDefaultWeights_SumToOne is the single most important assertion
// in this package. Every persona's nine weights MUST total 1.0 exactly
// within floatTolerance. A failure here means the hardcoded defaults
// in weights.go drifted from docs/ranking-v1.md §11.1. Fix by editing
// the file and re-running this test — NEVER relax the tolerance.
func TestDefaultWeights_SumToOne(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		w    PersonaWeights
	}{
		{"freelance", DefaultFreelanceWeights()},
		{"agency", DefaultAgencyWeights()},
		{"referrer", DefaultReferrerWeights()},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			sum := c.w.Sum()
			if math.Abs(sum-1.0) > floatTolerance {
				t.Fatalf("%s weights sum = %.12f, want 1.0 (delta %.2e > tolerance %.2e)",
					c.name, sum, math.Abs(sum-1.0), floatTolerance)
			}
		})
	}
}

// TestDefaultConfig_Validate asserts that the Config returned by
// DefaultConfig() passes Validate. This is a structural guarantee:
// if every DefaultXxxWeights returns a valid table, Config.Validate
// should never reject them. Broken by any per-persona drift.
func TestDefaultConfig_Validate(t *testing.T) {
	t.Parallel()
	if err := DefaultConfig().Validate(); err != nil {
		t.Fatalf("DefaultConfig().Validate() = %v, want nil", err)
	}
}

// TestPersonaWeights_Validate_Errors covers the unhappy path: a weight
// table that does NOT sum to 1.0 must surface ErrWeightsSum with the
// persona name embedded in the error message. Split into two cases
// (too-low / too-high) to prove the tolerance check is bidirectional.
func TestPersonaWeights_Validate_Errors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		weights  PersonaWeights
		contains string
	}{
		{
			name: "sum_too_low",
			weights: PersonaWeights{
				TextMatch: 0.20, SkillsOverlap: 0.15, Rating: 0.20,
				ProvenWork: 0.15, ResponseRate: 0.10, VerifiedMature: 0.08,
				Completion: 0.07, LastActive: 0.03, AccountAge: 0.01, // 0.99
			},
			contains: "freelance",
		},
		{
			name: "sum_too_high",
			weights: PersonaWeights{
				TextMatch: 0.20, SkillsOverlap: 0.15, Rating: 0.20,
				ProvenWork: 0.15, ResponseRate: 0.10, VerifiedMature: 0.08,
				Completion: 0.07, LastActive: 0.03, AccountAge: 0.03, // 1.01
			},
			contains: "freelance",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			err := c.weights.Validate("freelance")
			if err == nil {
				t.Fatalf("want error, got nil")
			}
			if !errors.Is(err, ErrWeightsSum) {
				t.Fatalf("want ErrWeightsSum, got %v", err)
			}
			if !strings.Contains(err.Error(), c.contains) {
				t.Fatalf("error %q should contain %q", err.Error(), c.contains)
			}
		})
	}
}

// TestPersonaWeights_Validate_Tolerance proves the tolerance is both
// tight (rejects a 0.01 drift) and permissive (accepts the IEEE 754
// rounding dust typical of adding nine float64 literals).
func TestPersonaWeights_Validate_Tolerance(t *testing.T) {
	t.Parallel()

	// This specific table sums to 1.0 only because we include the
	// matching 0.02 at the tail. Proves the canonical table passes
	// regardless of floating-point associativity.
	good := PersonaWeights{
		TextMatch: 0.2, SkillsOverlap: 0.15, Rating: 0.2,
		ProvenWork: 0.15, ResponseRate: 0.1, VerifiedMature: 0.08,
		Completion: 0.07, LastActive: 0.03, AccountAge: 0.02,
	}
	if err := good.Validate("freelance"); err != nil {
		t.Fatalf("good weights rejected: %v", err)
	}

	// Drift by twice the tolerance should be rejected.
	bad := good
	bad.TextMatch += 10 * floatTolerance
	if err := bad.Validate("freelance"); err == nil {
		t.Fatalf("drift of 10×tolerance was accepted, want ErrWeightsSum")
	}
}

// TestConfig_Validate_PropagatesFirstError ensures Validate returns
// the FIRST misconfigured persona so the operator fixes one thing at
// a time rather than getting spam for multiple drift issues. Using a
// valid freelance + invalid agency + invalid referrer, the error must
// name the agency.
func TestConfig_Validate_PropagatesFirstError(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	cfg.Agency.TextMatch += 0.5     // break agency
	cfg.Referrer.ResponseRate -= 0.5 // also break referrer

	err := cfg.Validate()
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "agency") {
		t.Fatalf("error %q should call out 'agency' (the first broken persona)", err.Error())
	}
}

// TestLoadConfigFromEnv_NoOverrides confirms that with a clean env,
// the returned Config matches DefaultConfig byte-for-byte.
func TestLoadConfigFromEnv_NoOverrides(t *testing.T) {
	// Cannot t.Parallel because we unset env vars. Test is still fast.
	clearRankingWeightEnv(t)

	got, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() = %v", err)
	}
	want := DefaultConfig()
	if got != want {
		t.Fatalf("LoadConfigFromEnv() = %#v, want %#v", got, want)
	}
}

// TestLoadConfigFromEnv_Override proves that an env var override is
// applied AND the validation still runs (here we override both
// TextMatch and AccountAge by ±delta to keep the sum at 1.0).
func TestLoadConfigFromEnv_Override(t *testing.T) {
	clearRankingWeightEnv(t)
	t.Setenv("RANKING_WEIGHTS_FREELANCE_TEXT_MATCH", "0.25")
	t.Setenv("RANKING_WEIGHTS_FREELANCE_ACCOUNT_AGE", "-0.03")

	// The pair 0.25 + (-0.03) increments TextMatch by 0.05 and
	// decrements AccountAge by 0.05 against the 0.20/0.02 defaults,
	// keeping the sum at 1.0 and letting us assert the override lands.
	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() = %v, want nil", err)
	}
	if cfg.Freelance.TextMatch != 0.25 {
		t.Fatalf("TextMatch = %v, want 0.25", cfg.Freelance.TextMatch)
	}
	if cfg.Freelance.AccountAge != -0.03 {
		t.Fatalf("AccountAge = %v, want -0.03", cfg.Freelance.AccountAge)
	}
}

// TestLoadConfigFromEnv_InvalidFloat asserts parse errors bubble up
// with the env-var name so the operator knows which key to fix. One
// table entry per persona so each error path (freelance / agency /
// referrer) is exercised.
func TestLoadConfigFromEnv_InvalidFloat(t *testing.T) {
	cases := []struct {
		name string
		key  string
	}{
		{"freelance", "RANKING_WEIGHTS_FREELANCE_TEXT_MATCH"},
		{"agency", "RANKING_WEIGHTS_AGENCY_RATING"},
		{"referrer", "RANKING_WEIGHTS_REFERRER_RESPONSE_RATE"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			clearRankingWeightEnv(t)
			t.Setenv(c.key, "not-a-number")

			_, err := LoadConfigFromEnv()
			if err == nil {
				t.Fatal("want error, got nil")
			}
			if !strings.Contains(err.Error(), c.key) {
				t.Fatalf("error %q should name the bad env var %q", err.Error(), c.key)
			}
		})
	}
}

// TestLoadConfigFromEnv_DriftRejected proves LoadConfigFromEnv wires
// Validate on its return value: an override that causes a persona to
// stop summing to 1.0 returns ErrWeightsSum, preventing a silently
// biased scorer at startup.
func TestLoadConfigFromEnv_DriftRejected(t *testing.T) {
	clearRankingWeightEnv(t)
	t.Setenv("RANKING_WEIGHTS_REFERRER_RATING", "0.99")

	_, err := LoadConfigFromEnv()
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !errors.Is(err, ErrWeightsSum) {
		t.Fatalf("want ErrWeightsSum, got %v", err)
	}
}

// TestConfig_Select_AllPersonas covers the three known personas plus
// the unknown-persona fallback to freelance. The fallback is a safety
// net, not an advertised feature — but it must not panic.
func TestConfig_Select_AllPersonas(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()

	cases := []struct {
		persona Persona
		want    PersonaWeights
	}{
		{PersonaFreelance, cfg.Freelance},
		{PersonaAgency, cfg.Agency},
		{PersonaReferrer, cfg.Referrer},
		{Persona("unknown"), cfg.Freelance},
	}

	for _, c := range cases {
		c := c
		t.Run(string(c.persona), func(t *testing.T) {
			t.Parallel()
			got := cfg.Select(c.persona)
			if got != c.want {
				t.Fatalf("Select(%q) = %#v, want %#v", c.persona, got, c.want)
			}
		})
	}
}

// clearRankingWeightEnv wipes every RANKING_WEIGHTS_* env var for the
// duration of the test so siblings' leftovers cannot poison the
// current test. t.Setenv restores the previous value on cleanup.
func clearRankingWeightEnv(t *testing.T) {
	t.Helper()
	personas := []string{"FREELANCE", "AGENCY", "REFERRER"}
	features := []string{
		"TEXT_MATCH", "SKILLS_OVERLAP", "RATING", "PROVEN_WORK",
		"RESPONSE_RATE", "VERIFIED_MATURE", "COMPLETION",
		"LAST_ACTIVE", "ACCOUNT_AGE",
	}
	for _, p := range personas {
		for _, f := range features {
			t.Setenv(envWeightKey(p, f), "")
		}
	}
}

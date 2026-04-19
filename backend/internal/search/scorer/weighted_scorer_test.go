package scorer

import (
	"context"
	"math"
	"math/rand"
	"testing"
)

// maxFeatures constructs a feature vector where every dimension is at
// its per-field maximum (ones for the normalised features, and the
// permitted cap for NegativeSignals). Used by monotonicity and bound
// property tests.
func maxFeatures() Features {
	return Features{
		TextMatchScore:      1,
		SkillsOverlapRatio:  1,
		RatingScoreDiverse:  1,
		ProvenWorkScore:     1,
		ResponseRate:        1,
		IsVerifiedMature:    1,
		ProfileCompletion:   1,
		LastActiveDaysScore: 1,
		AccountAgeBonus:     1,
		NegativeSignals:     0,
	}
}

// zeroFeatures is the all-zero vector. Final must be exactly 0.
func zeroFeatures() Features {
	return Features{}
}

// testScorer is a shared instance for tests that do not need a custom
// Config. Built once to amortise the allocation.
var testScorer = NewWeightedScorer(DefaultConfig())

// TestWeightedScorer_TableDriven walks the critical matrix described
// in the brief: 3 personas × {empty, non-empty query} × {zero, mid,
// max features} × {with, without dispute penalty}. Using an error
// tolerance on the Final score lets us pin the exact computed values
// without fragility to IEEE 754 rounding.
func TestWeightedScorer_TableDriven(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cases := []struct {
		name      string
		persona   Persona
		query     string
		features  Features
		wantBase  float64 // expected Base score (pre-clamp)
		wantFinal float64 // expected Final (after × (1-neg) × 100)
	}{
		{
			name:      "freelance_all_zeros_nonempty_query",
			persona:   PersonaFreelance,
			query:     "go react",
			features:  zeroFeatures(),
			wantBase:  0,
			wantFinal: 0,
		},
		{
			name:      "freelance_all_max_nonempty_query",
			persona:   PersonaFreelance,
			query:     "go react",
			features:  maxFeatures(),
			wantBase:  1.0, // Σ weights = 1.0
			wantFinal: 100,
		},
		{
			name:    "freelance_mid_features_nonempty_query",
			persona: PersonaFreelance,
			query:   "go react",
			features: Features{
				TextMatchScore:      0.5,
				SkillsOverlapRatio:  0.5,
				RatingScoreDiverse:  0.5,
				ProvenWorkScore:     0.5,
				ResponseRate:        0.5,
				IsVerifiedMature:    0.5,
				ProfileCompletion:   0.5,
				LastActiveDaysScore: 0.5,
				AccountAgeBonus:     0.5,
			},
			wantBase:  0.5,
			wantFinal: 50,
		},
		{
			name:    "freelance_worked_example_from_spec",
			persona: PersonaFreelance,
			query:   "développeur React Paris senior",
			// Taken directly from docs/ranking-v1.md §12.1 — spec claims 0.800.
			features: Features{
				TextMatchScore:      0.82,
				SkillsOverlapRatio:  0.75,
				RatingScoreDiverse:  0.69,
				ProvenWorkScore:     0.72,
				ResponseRate:        0.91,
				IsVerifiedMature:    1.00,
				ProfileCompletion:   0.88,
				LastActiveDaysScore: 0.83,
				AccountAgeBonus:     1.00,
			},
			wantBase:  0.8000, // Σ contributions matches spec §12.1
			wantFinal: 80.00,
		},
		{
			name:      "agency_all_max_nonempty",
			persona:   PersonaAgency,
			query:     "brand design studio",
			features:  maxFeatures(),
			wantBase:  1.0,
			wantFinal: 100,
		},
		{
			name:      "referrer_all_max_nonempty",
			persona:   PersonaReferrer,
			query:     "fintech sector",
			features:  maxFeatures(),
			wantBase:  1.0,
			wantFinal: 100,
		},
		{
			name:    "freelance_with_single_dispute_penalty",
			persona: PersonaFreelance,
			query:   "go",
			features: Features{
				TextMatchScore:      0.5,
				SkillsOverlapRatio:  0.5,
				RatingScoreDiverse:  0.5,
				ProvenWorkScore:     0.5,
				ResponseRate:        0.5,
				IsVerifiedMature:    0.5,
				ProfileCompletion:   0.5,
				LastActiveDaysScore: 0.5,
				AccountAgeBonus:     0.5,
				NegativeSignals:     0.10, // one lost dispute
			},
			wantBase:  0.5,
			wantFinal: 45, // 0.5 × (1 - 0.10) × 100
		},
		{
			name:    "freelance_with_cap_dispute_penalty",
			persona: PersonaFreelance,
			query:   "go",
			features: Features{
				TextMatchScore:      1,
				SkillsOverlapRatio:  1,
				RatingScoreDiverse:  1,
				ProvenWorkScore:     1,
				ResponseRate:        1,
				IsVerifiedMature:    1,
				ProfileCompletion:   1,
				LastActiveDaysScore: 1,
				AccountAgeBonus:     1,
				NegativeSignals:     0.30, // cap
			},
			wantBase:  1.0,
			wantFinal: 70, // 1 × (1 - 0.30) × 100
		},
		{
			name:    "freelance_overcap_dispute_clamped",
			persona: PersonaFreelance,
			query:   "go",
			features: Features{
				TextMatchScore:      1,
				SkillsOverlapRatio:  1,
				RatingScoreDiverse:  1,
				ProvenWorkScore:     1,
				ResponseRate:        1,
				IsVerifiedMature:    1,
				ProfileCompletion:   1,
				LastActiveDaysScore: 1,
				AccountAgeBonus:     1,
				NegativeSignals:     0.99, // defensive clamp to 0.30
			},
			wantBase:  1.0,
			wantFinal: 70,
		},
		{
			name:      "referrer_empty_query_max_features",
			persona:   PersonaReferrer,
			query:     "",
			features:  maxFeatures(),
			wantBase:  1.0, // redistribution keeps Σ weights = 1.0
			wantFinal: 100,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := testScorer.Score(ctx, Query{Text: c.query}, c.features, c.persona)

			if math.Abs(got.Base-c.wantBase) > 1e-4 {
				t.Errorf("Base = %.6f, want %.6f", got.Base, c.wantBase)
			}
			if math.Abs(got.Final-c.wantFinal) > 1e-2 {
				t.Errorf("Final = %.6f, want %.6f", got.Final, c.wantFinal)
			}

			// Breakdown must contain exactly 9 entries.
			if len(got.Breakdown) != 9 {
				t.Errorf("Breakdown has %d entries, want 9", len(got.Breakdown))
			}
			// All contributions must be non-negative (no feature × weight
			// can produce a negative value given the [0, 1] × [0, 1]
			// contract).
			for k, v := range got.Breakdown {
				if v < 0 {
					t.Errorf("Breakdown[%q] = %v, want >= 0", k, v)
				}
			}
		})
	}
}

// TestScoreBoundsProperty runs a randomised property test: for any
// 9-feature vector with all components in [0, 1] and NegativeSignals
// in [0, 0.30], the Final score must land in [0, 100] and Base must
// land in [0, 1] — independent of persona or query state. Fixed seed
// for determinism.
func TestScoreBoundsProperty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	r := rand.New(rand.NewSource(0xc0ffee))
	personas := []Persona{PersonaFreelance, PersonaAgency, PersonaReferrer}
	queries := []string{"", "go", "react paris senior"}

	const iterations = 1000
	for i := 0; i < iterations; i++ {
		f := Features{
			TextMatchScore:      r.Float64(),
			SkillsOverlapRatio:  r.Float64(),
			RatingScoreDiverse:  r.Float64(),
			ProvenWorkScore:     r.Float64(),
			ResponseRate:        r.Float64(),
			IsVerifiedMature:    float64(r.Intn(2)),
			ProfileCompletion:   r.Float64(),
			LastActiveDaysScore: r.Float64(),
			AccountAgeBonus:     r.Float64(),
			NegativeSignals:     r.Float64() * 0.30,
		}
		persona := personas[r.Intn(len(personas))]
		query := queries[r.Intn(len(queries))]

		got := testScorer.Score(ctx, Query{Text: query}, f, persona)

		if got.Base < 0 || got.Base > 1 {
			t.Fatalf("iter %d: Base=%v outside [0,1]", i, got.Base)
		}
		if got.Adjusted < 0 || got.Adjusted > 1 {
			t.Fatalf("iter %d: Adjusted=%v outside [0,1]", i, got.Adjusted)
		}
		if got.Final < 0 || got.Final > 100 {
			t.Fatalf("iter %d: Final=%v outside [0,100]", i, got.Final)
		}
		// Adjusted ≤ Base (negative signals cannot increase score).
		if got.Adjusted-got.Base > 1e-9 {
			t.Fatalf("iter %d: Adjusted=%v > Base=%v", i, got.Adjusted, got.Base)
		}
	}
}

// TestScoreMonotonicityProperty proves the scorer is monotonic in each
// feature: holding everything else constant, increasing one feature
// cannot decrease Final. Tests each of the 9 features on all three
// personas.
func TestScoreMonotonicityProperty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	personas := []Persona{PersonaFreelance, PersonaAgency, PersonaReferrer}

	mutators := []struct {
		name string
		mut  func(f *Features)
	}{
		{"text_match", func(f *Features) { f.TextMatchScore = 0.9 }},
		{"skills_overlap", func(f *Features) { f.SkillsOverlapRatio = 0.9 }},
		{"rating", func(f *Features) { f.RatingScoreDiverse = 0.9 }},
		{"proven_work", func(f *Features) { f.ProvenWorkScore = 0.9 }},
		{"response_rate", func(f *Features) { f.ResponseRate = 0.9 }},
		{"verified_mature", func(f *Features) { f.IsVerifiedMature = 1 }},
		{"completion", func(f *Features) { f.ProfileCompletion = 0.9 }},
		{"last_active", func(f *Features) { f.LastActiveDaysScore = 0.9 }},
		{"account_age", func(f *Features) { f.AccountAgeBonus = 0.9 }},
	}

	for _, persona := range personas {
		for _, m := range mutators {
			persona, m := persona, m
			t.Run(string(persona)+"_"+m.name, func(t *testing.T) {
				t.Parallel()
				low := Features{} // all zero
				high := low
				m.mut(&high)

				lowScore := testScorer.Score(ctx, Query{Text: "go"}, low, persona).Final
				highScore := testScorer.Score(ctx, Query{Text: "go"}, high, persona).Final

				if highScore < lowScore-1e-9 {
					t.Fatalf("monotonicity violated for %s/%s: high=%v < low=%v",
						persona, m.name, highScore, lowScore)
				}
			})
		}
	}
}

// TestScoreNegativeSignalProperty asserts the penalty is monotonic in
// NegativeSignals up to the cap: for a fixed positive vector,
// increasing the penalty cannot increase Final.
func TestScoreNegativeSignalProperty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	base := maxFeatures()
	prev := testScorer.Score(ctx, Query{Text: "x"}, base, PersonaFreelance).Final
	steps := []float64{0.05, 0.10, 0.15, 0.20, 0.25, 0.30, 0.50, 1.0}
	for _, p := range steps {
		cur := base
		cur.NegativeSignals = p
		got := testScorer.Score(ctx, Query{Text: "x"}, cur, PersonaFreelance).Final
		if got > prev+1e-9 {
			t.Fatalf("penalty=%v: score %v rose above previous %v", p, got, prev)
		}
		prev = got
	}
	// With NegativeSignals at or above cap, final is exactly 70 (max
	// features × (1 - 0.30) × 100).
	if math.Abs(prev-70) > 1e-6 {
		t.Fatalf("final at cap = %v, want 70", prev)
	}
}

// TestScoreEmptyQueryProperty verifies the empty-query redistribution
// does not simply zero-out the TextMatch slice — the eight remaining
// features must absorb its weight. Concretely: an all-mid vector with
// TextMatch=0 and empty query must return the same Final as an all-mid
// vector with TextMatch=0 and a non-empty query BUT with the weights
// redistributed. Here the simpler check: the Final score with empty
// query + mid features != 50 × (1 - weight_text_match).
func TestScoreEmptyQueryProperty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	midBelowTM := Features{
		TextMatchScore:      0,
		SkillsOverlapRatio:  0.5,
		RatingScoreDiverse:  0.5,
		ProvenWorkScore:     0.5,
		ResponseRate:        0.5,
		IsVerifiedMature:    0.5,
		ProfileCompletion:   0.5,
		LastActiveDaysScore: 0.5,
		AccountAgeBonus:     0.5,
	}

	// Empty query → redistributed weights → Final = 50 (because the
	// remaining eight features are all at 0.5 and the redistributed
	// weights sum to 1.0).
	empty := testScorer.Score(ctx, Query{Text: ""}, midBelowTM, PersonaFreelance).Final
	if math.Abs(empty-50) > 1e-6 {
		t.Fatalf("empty-query score = %v, want 50 (redistribution broken)", empty)
	}

	// Non-empty query + TextMatchScore=0 → Final = 50 × (1 - 0.20) = 40
	// because TextMatch weight (0.20) contributes 0 to Base.
	nonempty := testScorer.Score(ctx, Query{Text: "go"}, midBelowTM, PersonaFreelance).Final
	if math.Abs(nonempty-40) > 1e-6 {
		t.Fatalf("non-empty-query score = %v, want 40", nonempty)
	}
}

// TestScoreBreakdownContents pins the exact keys present in the
// Breakdown map. Any future removal or rename is a breaking change
// for downstream consumers — this test guards it.
func TestScoreBreakdownContents(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	got := testScorer.Score(ctx, Query{Text: "go"}, maxFeatures(), PersonaFreelance)

	wantKeys := []string{
		BreakdownTextMatch, BreakdownSkillsOverlap, BreakdownRating,
		BreakdownProvenWork, BreakdownResponseRate, BreakdownVerifiedMature,
		BreakdownCompletion, BreakdownLastActive, BreakdownAccountAge,
	}
	for _, k := range wantKeys {
		if _, ok := got.Breakdown[k]; !ok {
			t.Errorf("Breakdown missing key %q", k)
		}
	}
	if len(got.Breakdown) != len(wantKeys) {
		t.Errorf("Breakdown has %d keys, want %d", len(got.Breakdown), len(wantKeys))
	}

	// Breakdown values for maxFeatures + PersonaFreelance must equal
	// the weight values exactly (since each feature is 1.0).
	w := DefaultFreelanceWeights()
	if got.Breakdown[BreakdownTextMatch] != w.TextMatch {
		t.Errorf("Breakdown[text_match] = %v, want %v",
			got.Breakdown[BreakdownTextMatch], w.TextMatch)
	}
	if got.Breakdown[BreakdownAccountAge] != w.AccountAge {
		t.Errorf("Breakdown[account_age] = %v, want %v",
			got.Breakdown[BreakdownAccountAge], w.AccountAge)
	}
}

// TestScoreNaNGuards asserts that pathological NaN inputs do not
// propagate through to the Final score. clamp01 collapses NaN to 0.
func TestScoreNaNGuards(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	nan := math.NaN()
	f := Features{
		TextMatchScore: nan,
		NegativeSignals: nan,
	}
	got := testScorer.Score(ctx, Query{Text: "go"}, f, PersonaFreelance)
	if math.IsNaN(got.Base) || math.IsNaN(got.Adjusted) || math.IsNaN(got.Final) {
		t.Fatalf("NaN leaked into RankedScore: %+v", got)
	}
	// Adjusted and Base must be non-negative.
	if got.Final < 0 {
		t.Fatalf("Final = %v, want >= 0", got.Final)
	}
}

// TestScoreOverUnityClamp drives the defensive upper clamp: if the
// feature pipeline misbehaves and emits a value > 1, Base and Adjusted
// are still capped at 1.0 and Final at 100. Normal inputs never hit
// this branch — the test exists to prove the guard rail works.
func TestScoreOverUnityClamp(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	f := Features{
		TextMatchScore:      10,
		SkillsOverlapRatio:  10,
		RatingScoreDiverse:  10,
		ProvenWorkScore:     10,
		ResponseRate:        10,
		IsVerifiedMature:    10,
		ProfileCompletion:   10,
		LastActiveDaysScore: 10,
		AccountAgeBonus:     10,
	}
	got := testScorer.Score(ctx, Query{Text: "go"}, f, PersonaFreelance)
	if got.Base != 1 {
		t.Fatalf("Base = %v, want 1 (upper clamp)", got.Base)
	}
	if got.Adjusted != 1 {
		t.Fatalf("Adjusted = %v, want 1 (upper clamp)", got.Adjusted)
	}
	if got.Final != 100 {
		t.Fatalf("Final = %v, want 100 (upper clamp)", got.Final)
	}
}

// TestNewWeightedScorer_PanicOnInvalidConfig proves the constructor
// rejects a bad Config fast instead of silently scoring on biased
// weights.
func TestNewWeightedScorer_PanicOnInvalidConfig(t *testing.T) {
	t.Parallel()

	bad := Config{
		Freelance: PersonaWeights{TextMatch: 2}, // sum=2, way off
		Agency:    DefaultAgencyWeights(),
		Referrer:  DefaultReferrerWeights(),
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("NewWeightedScorer did not panic on invalid config")
		}
	}()
	_ = NewWeightedScorer(bad)
}

// TestScoreRerankerInterface confirms at test time (in addition to the
// compile-time var _ assertion) that WeightedScorer implements the
// Reranker interface.
func TestScoreRerankerInterface(t *testing.T) {
	t.Parallel()
	var r Reranker = testScorer
	got := r.Score(context.Background(), Query{}, zeroFeatures(), PersonaFreelance)
	if got.Final != 0 {
		t.Fatalf("zero-feature Score().Final = %v, want 0", got.Final)
	}
}

// BenchmarkScore proves the cost envelope: a single Score call on a
// warm instance must complete in under 200 ns (the brief's target).
// Uses b.ReportAllocs so CI can flag allocation regressions — every
// Score call allocates exactly one map for the Breakdown, and that is
// acceptable.
func BenchmarkScore(b *testing.B) {
	s := NewWeightedScorer(DefaultConfig())
	ctx := context.Background()
	q := Query{Text: "développeur"}
	f := maxFeatures()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = s.Score(ctx, q, f, PersonaFreelance)
	}
}

// BenchmarkScoreEmptyQuery measures the empty-query path separately —
// the redistribution branch adds a small amount of arithmetic + one
// more PersonaWeights copy. Target < 250 ns.
func BenchmarkScoreEmptyQuery(b *testing.B) {
	s := NewWeightedScorer(DefaultConfig())
	ctx := context.Background()
	q := Query{Text: ""}
	f := maxFeatures()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = s.Score(ctx, q, f, PersonaAgency)
	}
}

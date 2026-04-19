package antigaming

import (
	"context"
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search/features"
)

// Pipeline runs the 5 rules deterministically, logs every firing, and
// surfaces the aggregate result.
func TestPipeline_Apply_CleanProfile(t *testing.T) {
	cfg := DefaultConfig()
	rec := &RecordingLogger{}
	p := NewPipeline(cfg, NoopLinkedReviewersDetector{}, rec)

	f := &features.Features{
		TextMatchScore:     0.9,
		RatingScoreDiverse: 0.7,
		AccountAgeBonus:    0.8,
		RawUniqueReviewers: 15,
	}
	original := *f

	raw := RawSignals{
		ProfileID:      "p1",
		Persona:        features.PersonaFreelance,
		Text:           "senior developer working across Paris and remote with 10 years experience",
		NowUnix:        1_700_000_000,
		AccountAgeDays: 400,
	}
	res := p.Apply(context.Background(), f, raw)

	assert.Empty(t, res.Penalties)
	assert.False(t, res.NewAccountCapped)
	assert.Empty(t, rec.Penalties)
	assert.Equal(t, original, *f) // nothing mutated
}

// Pipeline fires every applicable rule + accumulates penalties.
//
// Rules that mutate rating_score_diverse run in a specific order :
//   velocity (dampens) -> linked (dampens) -> reviewer_floor (assigns cap)
// Starting rating must stay high enough after the first two dampeners so
// the reviewer_floor has something to cap.
func TestPipeline_Apply_AllRulesFire(t *testing.T) {
	cfg := DefaultConfig()
	rec := &RecordingLogger{}

	// One of four reviewers flagged as linked -> fraction 0.25 ≤ 0.3, the
	// linked rule would NOT fire. We lift the linked count high enough that
	// dampening is moderate (keeps rating > cap so reviewer_floor fires).
	// 2/4 = 0.5 > 0.3, factor 0.5.
	p := NewPipeline(cfg, stubDetector{linked: 2}, rec)

	// Start rating very high so : 1.0 × velocity(0.7) × linked(0.5) = 0.35,
	// then reviewer_floor caps at 0.4 (0.35 < 0.4 so ACTUALLY floor
	// NO-ops). Need: 1.0 × 0.9 × 0.7 = 0.63 > 0.4 so floor caps.
	// Use velocity factor 0.9 (recent=6, cap=5, excess=1, total=10 -> 9/10).
	f := &features.Features{
		TextMatchScore:     0.9,
		RatingScoreDiverse: 1.0,
		AccountAgeBonus:    0.4,
		RawUniqueReviewers: 2, // below floor of 3
	}
	const now = 1_700_000_000
	// 6 timestamps in last 24h, cap 5 -> excess 1, total 10 -> factor 0.9.
	timestamps := make([]int64, 6)
	for i := range timestamps {
		timestamps[i] = now - 3600
	}
	raw := RawSignals{
		ProfileID:              "p1",
		Persona:                features.PersonaFreelance,
		Text:                   strings.Repeat("react ", 20),
		NowUnix:                now,
		RecentReviewTimestamps: timestamps,
		TotalReviewCount:       10,
		ReviewerIDs:            []string{"a", "b", "c", "d"}, // 4 reviewers
		AccountAgeDays:         3,
	}
	res := p.Apply(context.Background(), f, raw)

	// All five rules should have recorded a penalty.
	assert.True(t, res.NewAccountCapped, "new-account cap must be set")
	assert.Len(t, res.Penalties, 5, "all 5 rules should fire : %v", res.Penalties)

	ruleSet := make(map[Rule]struct{})
	for _, pen := range res.Penalties {
		ruleSet[pen.Rule] = struct{}{}
	}
	for _, r := range []Rule{
		RuleKeywordStuffing, RuleReviewVelocity,
		RuleLinkedAccounts, RuleReviewerFloor, RuleNewAccount,
	} {
		_, ok := ruleSet[r]
		assert.True(t, ok, "rule %s should have fired", r)
	}

	// Same rules logged to the logger.
	assert.Len(t, rec.Penalties, 5)
}

// Nil features -> safe no-op.
func TestPipeline_Apply_NilFeatures(t *testing.T) {
	p := NewPipeline(DefaultConfig(), nil, nil)
	res := p.Apply(context.Background(), nil, RawSignals{})
	assert.Empty(t, res.Penalties)
}

// Nil context accepted (falls back to context.Background).
func TestPipeline_Apply_NilContext(t *testing.T) {
	p := NewPipeline(DefaultConfig(), nil, nil)
	f := &features.Features{}
	// Compile-time: interfaces accept nil context ; runtime must not panic.
	res := p.Apply(nil, f, RawSignals{}) //nolint:staticcheck // test explicitly allows nil ctx
	assert.Empty(t, res.Penalties)
}

// Property test — the final Features state is stable under a second apply
// (idempotency on state, not on logs). Assignment-style rules (reviewer
// floor, new account) and multiplicative rules (stuffing, velocity, linked)
// all reach a fixed point after one pass.
func TestPipeline_Apply_IdempotentOnState(t *testing.T) {
	cfg := DefaultConfig()
	p := NewPipeline(cfg, NoopLinkedReviewersDetector{}, NoopLogger{})

	buildInputs := func() (*features.Features, RawSignals) {
		f := &features.Features{
			TextMatchScore:     0.5,
			RatingScoreDiverse: 0.8,
			AccountAgeBonus:    0.3,
			RawUniqueReviewers: 1,
		}
		raw := RawSignals{
			Text:           "normal about blurb content with plenty of variety across tokens",
			NowUnix:        1_700_000_000,
			AccountAgeDays: 2,
		}
		return f, raw
	}

	f1, raw := buildInputs()
	p.Apply(context.Background(), f1, raw)
	after1 := *f1

	p.Apply(context.Background(), f1, raw)
	after2 := *f1

	assert.Equal(t, after1, after2,
		"applying the pipeline twice must yield an identical Features state")
}

// Property test — feature invariants never break : after Pipeline.Apply,
// every positive feature stays in [0, 1].
func TestPipeline_Apply_PropertyBounds(t *testing.T) {
	cfg := DefaultConfig()
	p := NewPipeline(cfg, stubDetector{linked: 1}, NoopLogger{})
	rng := rand.New(rand.NewSource(17))

	for i := 0; i < 300; i++ {
		f := randomFeatures(rng)
		raw := randomRaw(rng)
		_ = p.Apply(context.Background(), f, raw)
		require.GreaterOrEqual(t, f.TextMatchScore, 0.0)
		require.LessOrEqual(t, f.TextMatchScore, 1.0)
		require.GreaterOrEqual(t, f.RatingScoreDiverse, 0.0)
		require.LessOrEqual(t, f.RatingScoreDiverse, 1.0)
		require.GreaterOrEqual(t, f.AccountAgeBonus, 0.0)
		require.LessOrEqual(t, f.AccountAgeBonus, 1.0)
	}
}

// Pipeline.Config returns a copy — mutating it cannot poison subsequent runs.
func TestPipeline_Config_ReturnsCopy(t *testing.T) {
	p := NewPipeline(DefaultConfig(), nil, nil)
	cfg := p.Config()
	cfg.StuffingPenalty = 0.999 // local mutation

	f := &features.Features{TextMatchScore: 0.8}
	raw := RawSignals{Text: strings.Repeat("react ", 20)}
	p.Apply(context.Background(), f, raw)
	// Default stuffing penalty is 0.5 -> 0.8 × 0.5 = 0.4 ; if the mutation
	// leaked, we'd see 0.8 × 0.999 = ~0.8.
	assert.InDelta(t, 0.4, f.TextMatchScore, 1e-9)
}

// Helper : random features + raw inputs for property tests. Values stay in
// realistic ranges so every rule gets exercised without NaN.
func randomFeatures(rng *rand.Rand) *features.Features {
	return &features.Features{
		TextMatchScore:     rng.Float64(),
		RatingScoreDiverse: rng.Float64(),
		AccountAgeBonus:    rng.Float64(),
		RawUniqueReviewers: rng.Intn(10),
	}
}

func randomRaw(rng *rand.Rand) RawSignals {
	texts := []string{
		"",
		"senior engineer with distributed systems background",
		strings.Repeat("react ", rng.Intn(30)),
		"flutter flutter flutter flutter flutter flutter flutter flutter flutter flutter flutter",
	}
	const now = 1_700_000_000
	nReviews := rng.Intn(15)
	ts := make([]int64, nReviews)
	for i := range ts {
		ts[i] = now - int64(rng.Intn(48*3600))
	}
	return RawSignals{
		Text:                   texts[rng.Intn(len(texts))],
		NowUnix:                now,
		RecentReviewTimestamps: ts,
		TotalReviewCount:       rng.Intn(50) + 1,
		ReviewerIDs:            []string{"r1", "r2", "r3"},
		AccountAgeDays:         rng.Intn(60),
	}
}

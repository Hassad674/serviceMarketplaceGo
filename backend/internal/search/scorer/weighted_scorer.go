package scorer

import (
	"context"
	"math"
)

// negativeSignalCap mirrors docs/ranking-v1.md §5.3: the dispute-driven
// penalty is already bounded to [0, 0.30] at the feature extractor.
// Applying clamp a second time here is a belt-and-braces guard: should
// the feature pipeline ever emit a value outside the contract, the
// final composite still lands in [0, 100] and no downstream consumer
// crashes. Exported for the test file to assert the bound explicitly.
const negativeSignalCap = 0.30

// WeightedScorer is the V1 composite-scoring implementation. It is a
// stateless struct (holds only the weight Config) so a single instance
// can be shared across goroutines without synchronisation. Score is
// pure arithmetic — no I/O, no allocation after the Breakdown map.
type WeightedScorer struct {
	cfg Config
}

// NewWeightedScorer panics if cfg fails validation. Callers are
// expected to have passed cfg through LoadConfigFromEnv which
// already validates. Using panic here (not returning an error) keeps
// the Reranker construction site in cmd/api/main.go free of error
// noise for what is a startup-only, deterministic-failure path.
func NewWeightedScorer(cfg Config) *WeightedScorer {
	if err := cfg.Validate(); err != nil {
		panic(err)
	}
	return &WeightedScorer{cfg: cfg}
}

// Score applies the per-persona weights to the 9 normalised features,
// subtracts the dispute penalty, scales to [0, 100], and returns a
// RankedScore with the full breakdown. The ctx parameter is part of
// the Reranker contract (future LTR scorers may need it for timing
// or cancellation) but is unused by V1 — arithmetic is fast enough.
//
// The output is clamped to [0, 100] as a defensive measure. Under
// normal conditions (all features in [0, 1], weights sum to 1,
// NegativeSignals in [0, 0.30]) the arithmetic already lands inside
// [0, 100] — the clamp only activates for malformed inputs.
func (s *WeightedScorer) Score(ctx context.Context, q Query, f Features, persona Persona) RankedScore {
	_ = ctx // reserved for LTR-scorer compatibility; see §9.3.

	weights := s.cfg.Select(persona)
	if isEmptyQuery(q) {
		weights = RedistributeForEmptyQuery(weights)
	}

	// Compute the nine contributions in one pass. The inline form
	// (rather than a closure) keeps the benchmark under 300 ns by
	// avoiding the closure's hidden allocation on recent Go versions.
	cTM := weights.TextMatch * f.TextMatchScore
	cSO := weights.SkillsOverlap * f.SkillsOverlapRatio
	cR := weights.Rating * f.RatingScoreDiverse
	cPW := weights.ProvenWork * f.ProvenWorkScore
	cRR := weights.ResponseRate * f.ResponseRate
	cVM := weights.VerifiedMature * f.IsVerifiedMature
	cC := weights.Completion * f.ProfileCompletion
	cLA := weights.LastActive * f.LastActiveDaysScore
	cAA := weights.AccountAge * f.AccountAgeBonus

	positive := cTM + cSO + cR + cPW + cRR + cVM + cC + cLA + cAA

	breakdown := map[string]float64{
		BreakdownTextMatch:      cTM,
		BreakdownSkillsOverlap:  cSO,
		BreakdownRating:         cR,
		BreakdownProvenWork:     cPW,
		BreakdownResponseRate:   cRR,
		BreakdownVerifiedMature: cVM,
		BreakdownCompletion:     cC,
		BreakdownLastActive:     cLA,
		BreakdownAccountAge:     cAA,
	}

	penalty := clampNegativeSignals(f.NegativeSignals)
	adjusted := positive * (1 - penalty)
	final := clamp01(adjusted) * 100

	return RankedScore{
		Base:      clamp01(positive),
		Adjusted:  clamp01(adjusted),
		Final:     final,
		Breakdown: breakdown,
	}
}

// clamp01 constrains v to [0, 1]. NaN collapses to 0 so a downstream
// consumer does not poison JSON logs or comparison operators.
func clamp01(v float64) float64 {
	if math.IsNaN(v) || v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// clampNegativeSignals constrains the penalty to [0, negativeSignalCap].
// The upstream extractor already applies the cap, so this is a defensive
// second check. NaN collapses to 0 so a malformed feature vector does
// not silently demote the whole persona to zero.
func clampNegativeSignals(v float64) float64 {
	if math.IsNaN(v) || v < 0 {
		return 0
	}
	if v > negativeSignalCap {
		return negativeSignalCap
	}
	return v
}

// Compile-time proof that *WeightedScorer satisfies the Reranker
// interface. If the interface or the struct drifts, the build fails
// here before a single test runs.
var _ Reranker = (*WeightedScorer)(nil)

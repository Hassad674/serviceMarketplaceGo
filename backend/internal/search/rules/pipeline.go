package rules

import (
	"context"
	"math/rand"
	"time"
)

// BusinessRules owns the deterministic knobs + random source used by
// the rules pipeline. Safe for concurrent use ONLY through Apply —
// the embedded *rand.Rand is not goroutine-safe on its own, so Apply
// serialises access through a cheap local clone (see newRun).
//
// Construct with NewBusinessRules; never zero-value.
type BusinessRules struct {
	cfg        Config
	seedSource func() int64
}

// NewBusinessRules returns a configured pipeline. Zero Config falls
// back to DefaultConfig so callers can opt into "use the defaults"
// without boilerplate.
//
// The seed source is a closure so prod can plug crypto/rand and
// tests can pin a fixed int64. When cfg.RandSeed != 0 that value is
// returned unchanged, which is the deterministic test path.
func NewBusinessRules(cfg Config) *BusinessRules {
	if cfg.TopN == 0 {
		cfg = DefaultConfig()
	}
	br := &BusinessRules{cfg: cfg}
	br.seedSource = func() int64 {
		if cfg.RandSeed != 0 {
			return cfg.RandSeed
		}
		return time.Now().UnixNano()
	}
	return br
}

// Config exposes the loaded knob set. Tests use it to assert defaults;
// ops dashboards eventually read it to display the live tuning state.
func (r *BusinessRules) Config() Config { return r.cfg }

// Apply runs the 5-stage pipeline (§8 of docs/ranking-v1.md) on the
// scored candidate list. Returns the re-ranked top-N. Never allocates
// new candidates, never duplicates existing ones.
//
// Apply is purposefully a value-in / value-out function: callers
// hand a ranked slice, receive a re-ranked slice. Any mutation
// internally stays on a local copy — the input slice is not
// modified past the len(cfg.TopN) boundary.
//
// Flow:
//  1. tierSort      — split into A (now/soon) and B (not_available).
//  2. randomise     — gaussian noise with rank-dependent sigma.
//  3. reSort        — within each tier on noise-adjusted score.
//  4. mergeTiers    — Tier A always precedes Tier B.
//  5. diversityPass — swap adjacents breaking 3+ same primary_skill.
//  6. injectRising  — slot rule (positions 5, 10, 15, 20).
//  7. applyFeatured — dormant V1 unless cfg.FeaturedEnabled.
//  8. truncate      — clip to cfg.TopN.
func (r *BusinessRules) Apply(
	ctx context.Context,
	candidates []Candidate,
	persona Persona,
) []Candidate {
	_ = ctx      // reserved for cancellation once Apply grows I/O helpers.
	_ = persona // persona is surfaced for future per-persona tuning (§13a).

	if len(candidates) == 0 {
		return nil
	}

	run := r.newRun()

	// Stage 1 + 2 + 3: tier sort + randomise + per-tier re-sort.
	tierA, tierB := splitTiers(candidates)
	knobs := noiseKnobs{
		coefficient:    r.cfg.NoiseCoefficient,
		top3Multiplier: r.cfg.NoiseTop3Multiplier,
		midMultiplier:  r.cfg.NoiseMidMultiplier,
		tailMultiplier: r.cfg.NoiseTailMultiplier,
	}
	randomiseWithKnobs(tierA, run, knobs)
	randomiseWithKnobs(tierB, run, knobs)
	sortByFinalDesc(tierA)
	sortByFinalDesc(tierB)

	// Stage 4: merge tiers — Tier A first.
	merged := make([]Candidate, 0, len(tierA)+len(tierB))
	merged = append(merged, tierA...)
	merged = append(merged, tierB...)

	// Stage 5: diversity pass over the top-N window.
	diversityPass(merged, r.cfg.TopN)

	// Stage 6: rising talent injection on the top-N window.
	// Requires the full candidate pool so we can look past TopN for
	// an eligible replacement.
	injectRising(merged, r.cfg)

	// Stage 7: featured override (dormant unless explicitly enabled).
	if r.cfg.FeaturedEnabled && r.cfg.FeaturedBoost > 0 {
		applyFeatured(merged, r.cfg.FeaturedBoost)
		// Featured override may disturb tier ordering; re-sort the
		// top window while preserving the tier partition.
		reSortRespectingTiers(merged, r.cfg.TopN)
	}

	// Stage 8: clip to cfg.TopN.
	if len(merged) > r.cfg.TopN {
		merged = merged[:r.cfg.TopN]
	}
	return merged
}

// pipelineRun is the per-Apply random source. Rolled locally so two
// concurrent Apply calls never share a *rand.Rand (which is not
// thread-safe).
type pipelineRun struct {
	rng *rand.Rand
}

func (r *BusinessRules) newRun() *pipelineRun {
	seed := r.seedSource()
	return &pipelineRun{rng: rand.New(rand.NewSource(seed))}
}

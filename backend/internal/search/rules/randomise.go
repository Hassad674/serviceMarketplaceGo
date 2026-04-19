package rules

// randomise.go implements §6.1 of docs/ranking-v1.md — gaussian
// noise applied to each candidate's final score with a rank-dependent
// sigma. Top-3 positions barely move (σ = 0.3 × NOISE_COEFFICIENT),
// mid-list (ranks 4-10) shuffle moderately, tail (rank 11+) rotates
// aggressively so every mid-tier profile gets a fair impression
// budget across page loads.
//
// Rationale: without noise, position 1 monopolises clicks, which
// reinforces its own ranking in a future LTR model (§9). Adding a
// small deterministic jitter also acts as a tiny anti-gaming tax —
// attackers cannot reliably A/B their tweaks against a fixed
// baseline.
//
// The noise is zero-mean so expected rank is preserved over many
// queries. For deterministic tests, the BusinessRules seed locks the
// RNG.

// randomiseWithKnobs applies gaussian noise in-place to the slice.
// It assumes the caller has already sorted by Score.Final, so index
// i is the current rank of candidates[i].
//
// Side-effect: candidates[i].Score.Final is overwritten with the
// noise-adjusted value (clamped to [0, 100]). The Breakdown map is
// left untouched — contributions are pre-noise by design.
func randomiseWithKnobs(candidates []Candidate, run *pipelineRun, knobs noiseKnobs) {
	for i := range candidates {
		sigma := noiseSigma(candidates[i].Score.Final, i+1, knobs)
		if sigma == 0 {
			continue
		}
		candidates[i].Score.Final = clampScore01(
			candidates[i].Score.Final + run.rng.NormFloat64()*sigma,
		)
	}
}

// noiseKnobs is the subset of Config that randomise needs. Extracted
// so tests can exercise the sigma formula without spinning up a full
// BusinessRules.
type noiseKnobs struct {
	coefficient    float64
	top3Multiplier float64
	midMultiplier  float64
	tailMultiplier float64
}

func defaultNoiseKnobs() noiseKnobs {
	d := DefaultConfig()
	return noiseKnobs{
		coefficient:    d.NoiseCoefficient,
		top3Multiplier: d.NoiseTop3Multiplier,
		midMultiplier:  d.NoiseMidMultiplier,
		tailMultiplier: d.NoiseTailMultiplier,
	}
}

// noiseSigma returns the standard deviation used to sample the
// gaussian noise for the candidate at the given 1-indexed rank
// position. Formula verbatim from §6.1:
//
//	σ = NOISE_COEFFICIENT × score × rank_multiplier(rank)
//
// with rank_multiplier:
//
//	rank ≤ 3    →  0.3
//	rank ≤ 10   →  0.8
//	else        →  1.5
//
// A zero or negative score returns zero sigma — we never add noise
// to a candidate whose score is already at the floor, which would
// push it into the negative range.
func noiseSigma(score float64, rank int, knobs noiseKnobs) float64 {
	if score <= 0 || rank <= 0 || knobs.coefficient == 0 {
		return 0
	}
	return knobs.coefficient * score * rankMultiplier(rank, knobs)
}

func rankMultiplier(rank int, knobs noiseKnobs) float64 {
	switch {
	case rank <= 3:
		return knobs.top3Multiplier
	case rank <= 10:
		return knobs.midMultiplier
	default:
		return knobs.tailMultiplier
	}
}

// clampScore01 bounds a score into [0, 100]. Used after adding
// gaussian noise so a very unlucky draw on a low-scored candidate
// cannot flip it negative (which downstream sort+truncate wouldn't
// mind, but downstream LTR logging would treat as drift).
func clampScore01(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

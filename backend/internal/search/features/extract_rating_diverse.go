package features

import "math"

// ExtractRatingDiverse computes the diverse-Bayesian rating score described
// in `docs/ranking-v1.md` §3.2-3.
//
// The formula combines three orthogonal signals :
//
//  1. Bayesian shrinkage : low-count averages are pulled toward the
//     marketplace mean so a profile with 1 × 5★ never beats a profile with
//     30 × 4.7★.
//  2. Reviewer diversity : (1 - max_reviewer_share) × unique_reviewers —
//     kills the "3 friends leave 10 reviews" attack.
//  3. Recency : mean of exp(-age_days / 365) across all reviews. Recent
//     reviews weigh more than two-year-old praise.
//
// For a cold-start profile (n == 0), a configurable floor (default 0.15) is
// returned so newcomers don't get ranked into the void.
//
// Implementation notes :
//   - rating_bayesian is divided by 5 to land in [0, 1].
//   - effective_count is normalised via log(1 + eff) / log(1 + cap).
//   - recency_factor is pre-computed at index time on the SearchDocument, so
//     this extractor just multiplies it in.
//
// Spec reference : §3.2-3 step 1..4.
func ExtractRatingDiverse(doc SearchDocumentLite, cfg Config) float64 {
	n := int(doc.RatingCount)
	if n <= 0 {
		return cfg.ColdStartFloor
	}

	bayes := bayesianAverage(doc.RatingAverage, n, cfg.BayesianPriorMean, cfg.BayesianPriorWeight)
	normalisedBayes := bayes / 5.0

	effectiveCount := effectiveReviewerCount(
		int(doc.UniqueReviewersCount),
		doc.MaxReviewerShare,
	)
	countComponent := logNormalise(effectiveCount, cfg.ReviewCountCap)

	recency := clamp01(doc.ReviewRecencyFactor)

	raw := normalisedBayes * countComponent * recency
	return clamp01(raw)
}

// bayesianAverage shrinks the observed mean `obs` (count `n`) toward the
// prior `priorMean` with weight `priorWeight`.
//
//	out = (priorWeight × priorMean + n × obs) / (priorWeight + n)
func bayesianAverage(obs float64, n int, priorMean, priorWeight float64) float64 {
	if priorWeight <= 0 {
		return obs
	}
	nf := float64(n)
	return (priorWeight*priorMean + nf*obs) / (priorWeight + nf)
}

// effectiveReviewerCount implements the diversity factor from §3.2-3 step 2.
//
//	effective_count = unique_reviewers × (1 - max_reviewer_share)
//
// max_reviewer_share is already bounded in [0, 1] by the indexer.
func effectiveReviewerCount(unique int, maxShare float64) float64 {
	if unique <= 0 {
		return 0
	}
	div := 1.0 - clamp01(maxShare)
	return float64(unique) * div
}

// logNormalise returns log(1+x) / log(1+cap), clamped to [0, 1]. Used by both
// rating_diverse (step 4) and proven_work_score (§3.2-4).
func logNormalise(x float64, cap int) float64 {
	if cap <= 0 {
		return 0
	}
	if x <= 0 {
		return 0
	}
	num := math.Log1p(x)
	den := math.Log1p(float64(cap))
	if den == 0 {
		return 0
	}
	return clamp01(num / den)
}

// clamp01 pins v inside [0, 1]. NaN is coerced to 0 (defensive, shouldn't
// happen in practice because the indexer validates numeric inputs).
func clamp01(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

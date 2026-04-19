package antigaming

import (
	"math"

	"marketplace-backend/internal/search/features"
)

// velocityRule implements `docs/ranking-v1.md` §7.2 — review velocity cap.
//
//	recent_reviews = reviews with created_at > now - 24h
//	if len(recent_reviews) > VelocityCap24h:
//	    excess = len(recent_reviews) - VelocityCap24h
//	    rating_score_diverse dampened by (n - excess) / n
//
// We do NOT re-run the Bayesian / diversity / recency formulas here —
// that would require the full review history. Instead we dampen the
// existing rating_score_diverse proportionally. The multiplier floors at
// 0 (saturation protection) and at most equals 1 (no-op).
//
// Cooldown (§7.2 : 14 days) is a higher-level concern handled by the
// admin dashboard ; for V1 the rule fires on every request where
// recent_reviews > cap. Persisted cooldown will be added when the
// dashboard ships.
func velocityRule(f *features.Features, raw RawSignals, cfg Config) *Penalty {
	const twentyFourHours = 24 * 3600
	if raw.NowUnix == 0 {
		// Without a reference time the rule cannot reason about the
		// 24h window ; stay safe and return no penalty.
		return nil
	}

	recent := 0
	for _, ts := range raw.RecentReviewTimestamps {
		if ts == 0 {
			continue
		}
		if raw.NowUnix-ts <= twentyFourHours {
			recent++
		}
	}

	if recent <= cfg.VelocityCap24h {
		return nil
	}
	excess := recent - cfg.VelocityCap24h

	// If the overall review count is unknown or not greater than the excess,
	// the safest thing is to zero the rating component — 100% of the
	// reviews came from the burst.
	n := raw.TotalReviewCount
	if n <= 0 || excess >= n {
		before := f.RatingScoreDiverse
		f.RatingScoreDiverse = 0
		return &Penalty{
			Rule:           RuleReviewVelocity,
			ProfileID:      raw.ProfileID,
			Persona:        raw.Persona,
			DetectionValue: float64(recent),
			Threshold:      float64(cfg.VelocityCap24h),
			PenaltyFactor:  0,
			BeforeValue:    before,
			AfterValue:     0,
		}
	}

	factor := float64(n-excess) / float64(n)
	// Pin in [0, 1] defensively.
	if math.IsNaN(factor) || factor < 0 {
		factor = 0
	} else if factor > 1 {
		factor = 1
	}

	before := f.RatingScoreDiverse
	f.RatingScoreDiverse *= factor
	return &Penalty{
		Rule:           RuleReviewVelocity,
		ProfileID:      raw.ProfileID,
		Persona:        raw.Persona,
		DetectionValue: float64(recent),
		Threshold:      float64(cfg.VelocityCap24h),
		PenaltyFactor:  factor,
		BeforeValue:    before,
		AfterValue:     f.RatingScoreDiverse,
	}
}

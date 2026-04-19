package antigaming

import "marketplace-backend/internal/search/features"

// reviewerFloorRule implements `docs/ranking-v1.md` §7.4 — unique reviewer
// floor.
//
// When the number of distinct reviewers is below the floor (default 3), cap
// rating_score_diverse at the `few_reviewer_cap` (default 0.4). The rule is
// only triggered for profiles that actually have some reviews — a cold-start
// profile with zero reviews already sits at the cold-start floor (0.15) via
// the extractor, no further cap needed.
//
// This is ADDITIONAL to the diversity_factor baked into rating_score_diverse
// (§3.2-3 step 2). The factor handles high-concentration cases ; this hard
// cap handles low-reviewer cases.
func reviewerFloorRule(f *features.Features, raw RawSignals, cfg Config) *Penalty {
	unique := f.RawUniqueReviewers
	if unique <= 0 || unique >= cfg.UniqueReviewerFloor {
		return nil
	}

	if f.RatingScoreDiverse <= cfg.FewReviewerCap {
		// Already below the cap, no penalty to apply but the rule still
		// technically ran — we do NOT emit a log event in the no-op case
		// to keep the signal clean (§7.6 : logs should only capture
		// actual penalties).
		return nil
	}

	before := f.RatingScoreDiverse
	f.RatingScoreDiverse = cfg.FewReviewerCap
	return &Penalty{
		Rule:           RuleReviewerFloor,
		ProfileID:      raw.ProfileID,
		Persona:        raw.Persona,
		DetectionValue: float64(unique),
		Threshold:      float64(cfg.UniqueReviewerFloor),
		PenaltyFactor:  cfg.FewReviewerCap,
		BeforeValue:    before,
		AfterValue:     cfg.FewReviewerCap,
	}
}

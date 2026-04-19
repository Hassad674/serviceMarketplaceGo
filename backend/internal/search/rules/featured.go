package rules

// featured.go implements §8 of docs/ranking-v1.md — the admin
// "featured" override. Dormant in V1 (Config.FeaturedEnabled defaults
// to false) but wired so a product decision to promote specific
// profiles can flip a single env var without a code change.
//
// The boost is multiplicative on Score.Final and capped by
// clampScore01 so it never pushes a candidate past 100. When
// FeaturedEnabled is true, the pipeline will re-sort the top window
// after this rule runs so the reshuffled candidates are surfaced
// immediately.
//
// Gameability: the is_featured flag lives in the admin panel, NOT
// on the user's own profile editor. Regular users cannot self-
// promote. Audit logging (handled outside this package) records
// every admin flip.

// applyFeatured multiplies Score.Final by (1 + boost) for every
// candidate flagged is_featured. Mutates the slice in place.
//
// Guards:
//   - boost ≤ 0 is a no-op (callers should have early-returned, but
//     we enforce it here as a belt-and-braces safety net).
//   - Candidates without is_featured are untouched.
//   - The post-boost value is clamped to [0, 100] so the display
//     score stays in the contract range even if ops misconfigures
//     the boost as 2.0 (200 %).
func applyFeatured(candidates []Candidate, boost float64) {
	if boost <= 0 {
		return
	}
	multiplier := 1.0 + boost
	for i := range candidates {
		if !candidates[i].IsFeatured {
			continue
		}
		candidates[i].Score.Final = clampScore01(
			candidates[i].Score.Final * multiplier,
		)
	}
}

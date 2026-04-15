package search

import (
	"math"
	"strings"
)

// ranking.go holds the pure helpers that convert raw signals (rating
// average + count, availability status, filled profile fields, etc.)
// into the numeric features used by the default Typesense sort_by
// formula. No I/O. No side effects. Every helper is table-driven
// tested so the ranking is fully reproducible.
//
// These formulas intentionally live here, not inside the indexer,
// because they are stable numerical contracts — changing one means
// every existing document needs a bulk reindex. Keeping them in a
// single file makes the diff obvious during code review.

// Availability priority values. Higher = shown earlier in the default
// sort. The values are stored on the document as an int32 so
// Typesense can `sort_by: availability_priority:desc` without any
// string → number conversion at query time.
const (
	// AvailabilityPriorityNow is the top bucket: actor is available
	// right now for a new engagement.
	AvailabilityPriorityNow int32 = 3
	// AvailabilityPrioritySoon covers actors who flagged themselves
	// as "available soon" (within the next few weeks).
	AvailabilityPrioritySoon int32 = 2
	// AvailabilityPriorityNot covers everything else — booked out,
	// on holiday, not taking clients. They still appear in results
	// but below the first two buckets.
	AvailabilityPriorityNot int32 = 1
)

// TopRatedMinAverage and TopRatedMinCount define the threshold for
// the "top rated" badge. Numbers chosen to match Malt's own badge
// (4.7+ average, 5+ reviews) so users have a familiar mental model.
const (
	TopRatedMinAverage = 4.7
	TopRatedMinCount   = 5
)

// AvailabilityPriority maps a free-form availability status string
// (as stored on the profile) to a numerical bucket. Unknown statuses
// fall back to "not" so the system never crashes on a typo — it
// simply demotes the actor in the default ranking.
func AvailabilityPriority(status string) int32 {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "now", "available", "available_now":
		return AvailabilityPriorityNow
	case "soon", "available_soon":
		return AvailabilityPrioritySoon
	default:
		return AvailabilityPriorityNot
	}
}

// BayesianRatingScore blends the rating average with the review count
// to prevent a 1-review 5-star profile from beating a 50-review 4.8-star
// profile. The formula is:
//
//	score = avg * log(1 + count)
//
// Why log(1+count) instead of count: reviews have diminishing returns
// beyond ~100, so we don't want a profile with 500 reviews to crush
// one with 50 just on volume. The logarithm softens the effect while
// still giving meaningful lift for early reviews.
//
// Edge cases:
//   - count <= 0 → score is 0 (unrated profiles sort at the bottom).
//   - avg <= 0 → score is 0 (catches bogus negative averages from
//     analytics bugs without crashing).
//   - count < 0 is clamped to 0 defensively.
func BayesianRatingScore(avg float64, count int) float64 {
	if count <= 0 || avg <= 0 {
		return 0
	}
	if math.IsNaN(avg) || math.IsInf(avg, 0) {
		return 0
	}
	return avg * math.Log1p(float64(count))
}

// IsTopRated reports whether the actor deserves the "top rated"
// badge. Requires BOTH a high average AND a minimum review volume
// so a freshly-created profile with one enthusiastic review does
// not get the badge.
func IsTopRated(avg float64, count int) bool {
	return avg >= TopRatedMinAverage && count >= TopRatedMinCount
}

// CompletionInput captures every profile signal that contributes to
// the 0-100 completion score. A zero-value struct scores 0; a fully-
// filled struct scores exactly 100. The weights are documented next
// to the struct so code review can sanity-check each field's impact.
type CompletionInput struct {
	HasPhoto         bool // +15 points
	HasAbout         bool // +15 points
	HasTitle         bool // +10 points
	HasVideo         bool // +10 points
	ExpertiseCount   int  // +10 if >=1 expertise domain
	SkillsCount      int  // +15 if >=5 skills, +10 if >=3, +5 if >=1
	HasPricing       bool // +10 points
	HasLocation      bool // +5 points (city + country)
	SocialLinksCount int  // +5 if >=1 social link
	LanguagesCount   int  // +5 if >=1 professional language
}

// ProfileCompletionScore computes the 0-100 completeness score.
//
// Weights:
//
//	has_photo          15
//	has_about          15
//	has_title          10
//	has_video          10
//	expertise_domains  10 (at least one)
//	skills             15 (tiered: 5/10/15 for 1+ / 3+ / 5+ skills)
//	has_pricing        10
//	has_location        5
//	social_links        5 (at least one)
//	languages           5 (at least one professional)
//	-------------------
//	total             100
//
// The tiered `skills` bucket rewards actors who invest time in
// curating a real skill list without making 1 skill feel useless.
// The tiers are exclusive — only one of them contributes per call.
//
// Clamping: the final score is clamped to [0, 100] in case weights
// are ever tuned upward without updating this function.
func ProfileCompletionScore(input CompletionInput) int {
	score := 0
	if input.HasPhoto {
		score += 15
	}
	if input.HasAbout {
		score += 15
	}
	if input.HasTitle {
		score += 10
	}
	if input.HasVideo {
		score += 10
	}
	if input.ExpertiseCount >= 1 {
		score += 10
	}
	score += skillsTier(input.SkillsCount)
	if input.HasPricing {
		score += 10
	}
	if input.HasLocation {
		score += 5
	}
	if input.SocialLinksCount >= 1 {
		score += 5
	}
	if input.LanguagesCount >= 1 {
		score += 5
	}
	return clampScore(score)
}

// skillsTier returns the skills component of the completion score.
// Extracted so the main function stays readable and so the tier
// boundaries can be unit-tested in isolation.
func skillsTier(count int) int {
	switch {
	case count >= 5:
		return 15
	case count >= 3:
		return 10
	case count >= 1:
		return 5
	}
	return 0
}

// clampScore bounds a raw score into the [0, 100] window. Defensive
// against future weight tuning that might accidentally push past 100.
func clampScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

// DefaultSortBy returns the canonical sort_by string used by the
// query path when no user-specified sort is provided.
//
// The formula blends BM25 text relevance, vector distance, and
// business signals in this order:
//
//  1. _text_match buckets of 10 keep visually-similar matches grouped
//     so the other fields can break ties cleanly.
//  2. _vector_distance:asc boosts semantically close documents.
//  3. availability_priority:desc pushes actors available NOW to the top.
//  4. rating_score:desc (Bayesian) prefers well-reviewed actors.
//  5. profile_completion_score:desc prefers complete profiles.
//  6. is_verified:desc prefers KYC-verified actors.
//  7. is_top_rated:desc prefers the top-rated badge holders.
//  8. last_active_at:desc prefers recently-active actors.
//
// Changing the order here MUST be followed by a full bulk reindex —
// even though the formula lives on the query side, reviewers need to
// re-tune the weights after any change.
func DefaultSortBy() string {
	return strings.Join([]string{
		"_text_match(buckets:10):desc",
		"_vector_distance:asc",
		"availability_priority:desc",
		"rating_score:desc",
		"profile_completion_score:desc",
		"is_verified:desc",
		"is_top_rated:desc",
		"last_active_at:desc",
	}, ",")
}

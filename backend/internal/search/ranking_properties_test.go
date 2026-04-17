package search

import (
	"math"
	"testing"
)

// ranking_properties_test.go locks in the monotonic properties the
// ranking formulas must hold. Regressions in these formulas silently
// misrank every search result — property tests catch the mistake
// before a reviewer.

// TestBayesianRatingScore_MonotonicInCount — when the average is held
// fixed (>0), adding more reviews MUST NOT lower the score. If this
// ever returns a lower value for a higher count, a profile with 500
// reviews could rank below the same profile with 50 reviews.
func TestBayesianRatingScore_MonotonicInCount(t *testing.T) {
	averages := []float64{1.0, 2.5, 3.7, 4.2, 4.9, 5.0}
	for _, avg := range averages {
		var prev float64
		for count := 1; count <= 500; count++ {
			got := BayesianRatingScore(avg, count)
			if got+1e-12 < prev {
				t.Fatalf("non-monotonic in count for avg=%g: count=%d got=%g prev=%g",
					avg, count, got, prev)
			}
			prev = got
		}
	}
}

// TestBayesianRatingScore_MonotonicInAverage — when the count is held
// fixed, raising the average MUST NOT lower the score.
func TestBayesianRatingScore_MonotonicInAverage(t *testing.T) {
	counts := []int{1, 5, 10, 50, 200}
	for _, count := range counts {
		var prev float64
		for avg := 0.1; avg <= 5.0; avg += 0.1 {
			got := BayesianRatingScore(avg, count)
			if got+1e-12 < prev {
				t.Fatalf("non-monotonic in avg for count=%d: avg=%g got=%g prev=%g",
					count, avg, got, prev)
			}
			prev = got
		}
	}
}

// TestBayesianRatingScore_EdgeCases pins the boundary returns the
// current implementation documents — important because upstream
// sorters rely on `0` meaning "unrated".
func TestBayesianRatingScore_EdgeCases(t *testing.T) {
	cases := map[string]struct {
		avg   float64
		count int
		want  float64
	}{
		"zero count":      {4.5, 0, 0},
		"negative count":  {4.5, -1, 0},
		"zero average":    {0, 10, 0},
		"negative avg":    {-1, 10, 0},
		"NaN avg":         {math.NaN(), 10, 0},
		"+Inf avg":        {math.Inf(1), 10, 0},
		"-Inf avg":        {math.Inf(-1), 10, 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := BayesianRatingScore(tc.avg, tc.count)
			if got != tc.want {
				t.Fatalf("BayesianRatingScore(%v,%d)=%v, want %v", tc.avg, tc.count, got, tc.want)
			}
		})
	}
}

// TestProfileCompletionScore_Clamped asserts the score is always
// within [0, 100]. A regression that over-weights one signal could
// push above 100 and break the frontend's progress bar math.
func TestProfileCompletionScore_Clamped(t *testing.T) {
	// All flags on + maxed counts — the max-possible input.
	input := CompletionInput{
		HasPhoto:         true,
		HasAbout:         true,
		HasTitle:         true,
		HasVideo:         true,
		ExpertiseCount:   99,
		SkillsCount:      99,
		HasPricing:       true,
		HasLocation:      true,
		SocialLinksCount: 99,
		LanguagesCount:   99,
	}
	got := ProfileCompletionScore(input)
	if got < 0 || got > 100 {
		t.Fatalf("ProfileCompletionScore out of range: %d", got)
	}

	// Zero-value input must return exactly 0.
	got0 := ProfileCompletionScore(CompletionInput{})
	if got0 != 0 {
		t.Fatalf("ProfileCompletionScore zero input = %d, want 0", got0)
	}
}

// TestProfileCompletionScore_WeightMonotonic — turning on any field
// MUST increase or hold the score; never decrease it.
func TestProfileCompletionScore_WeightMonotonic(t *testing.T) {
	base := ProfileCompletionScore(CompletionInput{})
	toggles := []func(*CompletionInput){
		func(i *CompletionInput) { i.HasPhoto = true },
		func(i *CompletionInput) { i.HasAbout = true },
		func(i *CompletionInput) { i.HasTitle = true },
		func(i *CompletionInput) { i.HasVideo = true },
		func(i *CompletionInput) { i.ExpertiseCount = 1 },
		func(i *CompletionInput) { i.SkillsCount = 1 },
		func(i *CompletionInput) { i.HasPricing = true },
		func(i *CompletionInput) { i.HasLocation = true },
		func(i *CompletionInput) { i.SocialLinksCount = 1 },
		func(i *CompletionInput) { i.LanguagesCount = 1 },
	}
	for i, toggle := range toggles {
		in := CompletionInput{}
		toggle(&in)
		got := ProfileCompletionScore(in)
		if got < base {
			t.Fatalf("toggle #%d decreased the score: base=%d got=%d", i, base, got)
		}
	}
}

// TestAvailabilityPriority_StableMapping catches regressions in the
// stringly-typed status → int32 mapping. Any change here requires a
// full bulk reindex — the test exists so reviewers cannot miss it.
func TestAvailabilityPriority_StableMapping(t *testing.T) {
	cases := map[string]int32{
		"now":             AvailabilityPriorityNow,
		"available":       AvailabilityPriorityNow,
		"available_now":   AvailabilityPriorityNow,
		"NOW":             AvailabilityPriorityNow,
		"  now  ":         AvailabilityPriorityNow,
		"soon":            AvailabilityPrioritySoon,
		"available_soon":  AvailabilityPrioritySoon,
		"not":             AvailabilityPriorityNot,
		"booked":          AvailabilityPriorityNot,
		"":                AvailabilityPriorityNot,
		"garbage-string":  AvailabilityPriorityNot,
	}
	for in, want := range cases {
		if got := AvailabilityPriority(in); got != want {
			t.Fatalf("AvailabilityPriority(%q)=%d, want %d", in, got, want)
		}
	}
}

// TestIsTopRated_ThresholdCorrect pins the 4.7 / 5 thresholds.
// Changing them would hide or reveal badges for existing profiles —
// worth making loud.
func TestIsTopRated_ThresholdCorrect(t *testing.T) {
	cases := []struct {
		avg   float64
		count int
		want  bool
	}{
		{4.69, 100, false}, // average just below
		{4.70, 5, true},    // exactly at both thresholds
		{4.70, 4, false},   // count just below
		{5.00, 500, true},  // ceiling case
		{0.00, 0, false},
	}
	for _, tc := range cases {
		if got := IsTopRated(tc.avg, tc.count); got != tc.want {
			t.Fatalf("IsTopRated(%g, %d)=%v, want %v", tc.avg, tc.count, got, tc.want)
		}
	}
}

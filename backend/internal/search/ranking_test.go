package search_test

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/search"
)

func TestAvailabilityPriority(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int32
	}{
		{"now", "now", search.AvailabilityPriorityNow},
		{"available_now", "available_now", search.AvailabilityPriorityNow},
		{"mixed case available", "Available", search.AvailabilityPriorityNow},
		{"soon", "soon", search.AvailabilityPrioritySoon},
		{"available_soon", "available_soon", search.AvailabilityPrioritySoon},
		{"padded soon", "  SOON  ", search.AvailabilityPrioritySoon},
		{"not available", "not_available", search.AvailabilityPriorityNot},
		{"empty", "", search.AvailabilityPriorityNot},
		{"bogus", "weird-value", search.AvailabilityPriorityNot},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, search.AvailabilityPriority(c.in))
		})
	}
}

func TestBayesianRatingScore(t *testing.T) {
	t.Run("zero count returns zero", func(t *testing.T) {
		assert.Equal(t, 0.0, search.BayesianRatingScore(4.8, 0))
	})

	t.Run("negative count returns zero", func(t *testing.T) {
		assert.Equal(t, 0.0, search.BayesianRatingScore(4.8, -3))
	})

	t.Run("zero average returns zero", func(t *testing.T) {
		assert.Equal(t, 0.0, search.BayesianRatingScore(0, 100))
	})

	t.Run("negative average returns zero", func(t *testing.T) {
		assert.Equal(t, 0.0, search.BayesianRatingScore(-1, 100))
	})

	t.Run("nan average returns zero", func(t *testing.T) {
		assert.Equal(t, 0.0, search.BayesianRatingScore(math.NaN(), 10))
	})

	t.Run("inf average returns zero", func(t *testing.T) {
		assert.Equal(t, 0.0, search.BayesianRatingScore(math.Inf(1), 10))
	})

	// The whole point of the Bayesian blend: a 50-review 4.8-star
	// profile must beat a 1-review 5.0-star profile.
	t.Run("volume beats 1-review 5-star", func(t *testing.T) {
		veteran := search.BayesianRatingScore(4.8, 50)
		newcomer := search.BayesianRatingScore(5.0, 1)
		assert.Greater(t, veteran, newcomer,
			"50-review 4.8 must rank above 1-review 5.0 — got %f vs %f",
			veteran, newcomer)
	})

	// But 500 reviews at 4.8 must not crush 50 reviews at 4.8 —
	// the log softens the volume effect.
	t.Run("log tames extreme volume", func(t *testing.T) {
		fifty := search.BayesianRatingScore(4.8, 50)
		fiveHundred := search.BayesianRatingScore(4.8, 500)
		ratio := fiveHundred / fifty
		assert.Less(t, ratio, 3.0,
			"500 reviews should not be >3x 50 reviews (got %f)", ratio)
	})

	t.Run("monotonic in count with fixed average", func(t *testing.T) {
		a := search.BayesianRatingScore(4.5, 1)
		b := search.BayesianRatingScore(4.5, 10)
		c := search.BayesianRatingScore(4.5, 100)
		assert.Less(t, a, b)
		assert.Less(t, b, c)
	})
}

func TestIsTopRated(t *testing.T) {
	cases := []struct {
		name  string
		avg   float64
		count int
		want  bool
	}{
		{"exactly on threshold", 4.7, 5, true},
		{"well above", 4.95, 120, true},
		{"high avg not enough reviews", 5.0, 4, false},
		{"enough reviews avg too low", 4.69, 50, false},
		{"zero reviews", 4.7, 0, false},
		{"zero everything", 0, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, search.IsTopRated(c.avg, c.count))
		})
	}
}

func TestProfileCompletionScore(t *testing.T) {
	t.Run("empty input scores 0", func(t *testing.T) {
		assert.Equal(t, 0, search.ProfileCompletionScore(search.CompletionInput{}))
	})

	t.Run("fully filled scores 100", func(t *testing.T) {
		score := search.ProfileCompletionScore(search.CompletionInput{
			HasPhoto:         true,
			HasAbout:         true,
			HasTitle:         true,
			HasVideo:         true,
			ExpertiseCount:   3,
			SkillsCount:      12,
			HasPricing:       true,
			HasLocation:      true,
			SocialLinksCount: 3,
			LanguagesCount:   2,
		})
		assert.Equal(t, 100, score)
	})

	t.Run("photo only", func(t *testing.T) {
		score := search.ProfileCompletionScore(search.CompletionInput{HasPhoto: true})
		assert.Equal(t, 15, score)
	})

	t.Run("skills tiers", func(t *testing.T) {
		cases := []struct {
			count int
			want  int
		}{
			{0, 0},
			{1, 5},
			{2, 5},
			{3, 10},
			{4, 10},
			{5, 15},
			{30, 15},
		}
		for _, c := range cases {
			got := search.ProfileCompletionScore(search.CompletionInput{SkillsCount: c.count})
			assert.Equal(t, c.want, got, "SkillsCount=%d", c.count)
		}
	})

	t.Run("score never exceeds 100 even with all signals", func(t *testing.T) {
		// Defensive: even a profile that somehow triggers every
		// condition must not exceed the clamp.
		score := search.ProfileCompletionScore(search.CompletionInput{
			HasPhoto:         true,
			HasAbout:         true,
			HasTitle:         true,
			HasVideo:         true,
			ExpertiseCount:   50,
			SkillsCount:      999,
			HasPricing:       true,
			HasLocation:      true,
			SocialLinksCount: 10,
			LanguagesCount:   20,
		})
		assert.LessOrEqual(t, score, 100)
	})
}

func TestDefaultSortBy(t *testing.T) {
	sortBy := search.DefaultSortBy()

	// Typesense 28.0 caps sort_by at three fields. The default
	// formula picks the three highest-impact signals and lets the
	// remaining quality signals (verified, top_rated, completion
	// score) influence ranking through the bayesian rating_score.
	// Phase 3 restored the vector-distance slot by swapping
	// availability_priority out of the default sort (availability
	// still surfaces via the facet filter).
	fragments := []string{
		"_text_match(buckets:10):desc",
		"_vector_distance:asc",
		"rating_score:desc",
	}

	last := -1
	for _, f := range fragments {
		idx := strings.Index(sortBy, f)
		assert.GreaterOrEqual(t, idx, 0, "missing fragment %q in sort_by", f)
		assert.Greater(t, idx, last, "fragment %q out of order in sort_by", f)
		last = idx
	}

	// And no more than 3 sort fields total — the cluster will
	// reject any larger set with a 400.
	require.Equal(t, 3, strings.Count(sortBy, ",")+1,
		"sort_by must contain exactly 3 sort fields (Typesense 28.0 cap)")
}

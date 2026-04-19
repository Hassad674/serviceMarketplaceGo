package features

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ExtractRatingDiverse combines Bayesian shrinkage, diversity, and recency
// with a cold-start floor for zero-review profiles.
func TestExtractRatingDiverse(t *testing.T) {
	cfg := DefaultConfig()

	t.Run("cold start returns floor", func(t *testing.T) {
		doc := SearchDocumentLite{}
		got := ExtractRatingDiverse(doc, cfg)
		assert.InDelta(t, cfg.ColdStartFloor, got, 1e-9)
	})

	t.Run("one mediocre review ≤ cold-start expectation", func(t *testing.T) {
		doc := SearchDocumentLite{
			RatingAverage:        3.0,
			RatingCount:          1,
			UniqueReviewersCount: 1,
			MaxReviewerShare:     1,
			ReviewRecencyFactor:  1,
		}
		got := ExtractRatingDiverse(doc, cfg)
		// Bayesian pulls toward 4.0 ; diversity factor is 0 here so result == 0.
		assert.InDelta(t, 0, got, 1e-9)
	})

	t.Run("max diversity + recency -> near bayesian/5", func(t *testing.T) {
		doc := SearchDocumentLite{
			RatingAverage:        5.0,
			RatingCount:          50,
			UniqueReviewersCount: 50,
			MaxReviewerShare:     1.0 / 50.0, // each reviewer gave exactly one review
			ReviewRecencyFactor:  1,
		}
		got := ExtractRatingDiverse(doc, cfg)
		// Bayesian = (8*4 + 50*5) / 58 = 282/58 ≈ 4.862, /5 ≈ 0.972
		// effective_count = 50 × (1 - 0.02) = 49 ; log(50) / log(51) ≈ 0.995
		// recency = 1
		want := 0.972 * 0.995
		assert.InDelta(t, want, got, 0.01)
	})

	t.Run("low diversity tanks the score", func(t *testing.T) {
		// 10 reviews, 8 from one person
		doc := SearchDocumentLite{
			RatingAverage:        5.0,
			RatingCount:          10,
			UniqueReviewersCount: 3,
			MaxReviewerShare:     0.8,
			ReviewRecencyFactor:  1,
		}
		got := ExtractRatingDiverse(doc, cfg)
		// effective_count = 3 × 0.2 = 0.6 -> small log
		assert.Less(t, got, 0.5)
	})

	t.Run("stale reviews drag down via recency", func(t *testing.T) {
		doc := SearchDocumentLite{
			RatingAverage:        4.8,
			RatingCount:          20,
			UniqueReviewersCount: 20,
			MaxReviewerShare:     0.05,
			ReviewRecencyFactor:  0.2,
		}
		got := ExtractRatingDiverse(doc, cfg)
		// recency of 0.2 roughly 1/5 of a fresh-review equivalent.
		assert.Less(t, got, 0.3)
	})

	t.Run("recency factor clamped above 1", func(t *testing.T) {
		doc := SearchDocumentLite{
			RatingAverage:        5.0,
			RatingCount:          1,
			UniqueReviewersCount: 1,
			MaxReviewerShare:     1.0 / 1.0,
			ReviewRecencyFactor:  9999, // nonsensical but we clamp
		}
		got := ExtractRatingDiverse(doc, cfg)
		// diversity factor = 0 -> result 0 regardless of recency clamp
		assert.Equal(t, 0.0, got)
	})

	t.Run("NaN recency treated as 0", func(t *testing.T) {
		doc := SearchDocumentLite{
			RatingAverage:        5.0,
			RatingCount:          30,
			UniqueReviewersCount: 30,
			MaxReviewerShare:     0.05,
			ReviewRecencyFactor:  math.NaN(),
		}
		got := ExtractRatingDiverse(doc, cfg)
		assert.InDelta(t, 0, got, 1e-9)
	})

	t.Run("value never exceeds 1", func(t *testing.T) {
		doc := SearchDocumentLite{
			RatingAverage:        5.0,
			RatingCount:          1000,
			UniqueReviewersCount: 1000,
			MaxReviewerShare:     0.001,
			ReviewRecencyFactor:  1,
		}
		got := ExtractRatingDiverse(doc, cfg)
		assert.LessOrEqual(t, got, 1.0)
	})

	t.Run("negative rating count returns floor", func(t *testing.T) {
		doc := SearchDocumentLite{RatingCount: -1}
		got := ExtractRatingDiverse(doc, cfg)
		assert.InDelta(t, cfg.ColdStartFloor, got, 1e-9)
	})
}

// bayesianAverage shrinks toward the prior with weight scaling with n.
func TestBayesianAverage(t *testing.T) {
	// 5-star average with 1 review + prior (8, 4.0) -> (32 + 5) / 9 = 4.111
	assert.InDelta(t, 4.111, bayesianAverage(5.0, 1, 4.0, 8), 0.001)
	// Zero observations -> returns prior mean (when caller's n=0; but this func
	// is invoked only n >= 1 by design)
	assert.InDelta(t, 4.0, bayesianAverage(4.0, 1, 4.0, 8), 0.001)
	// Large sample -> observed dominates
	// (8*4 + 1000*5) / 1008 = 5032/1008 ≈ 4.992
	assert.InDelta(t, 4.992, bayesianAverage(5.0, 1000, 4.0, 8), 0.01)
	// priorWeight <= 0 short-circuits
	assert.InDelta(t, 4.0, bayesianAverage(4.0, 10, 4.0, 0), 1e-9)
}

// logNormalise maps counts onto [0, 1] via log(1+x) / log(1+cap).
func TestLogNormalise(t *testing.T) {
	assert.InDelta(t, 0, logNormalise(0, 50), 1e-9)
	// log(1+50) / log(1+50) = 1
	assert.InDelta(t, 1, logNormalise(50, 50), 1e-9)
	// log(1+100) / log(1+50) > 1 → clamped to 1
	assert.InDelta(t, 1, logNormalise(100, 50), 1e-9)
	// cap <= 0 returns 0
	assert.Equal(t, 0.0, logNormalise(10, 0))
	// negative x returns 0
	assert.Equal(t, 0.0, logNormalise(-1, 50))
}

// effectiveReviewerCount is linear in unique and flips to 0 at max share.
func TestEffectiveReviewerCount(t *testing.T) {
	assert.InDelta(t, 8, effectiveReviewerCount(10, 0.2), 1e-9)
	assert.InDelta(t, 0, effectiveReviewerCount(0, 0), 1e-9)
	assert.InDelta(t, 0, effectiveReviewerCount(10, 1), 1e-9)
	// Clamp: negative share treated as 0 -> full count
	assert.InDelta(t, 10, effectiveReviewerCount(10, -0.5), 1e-9)
}

// clamp01 pins values in [0, 1], coerces NaN to 0.
func TestClamp01(t *testing.T) {
	tests := []struct {
		in, want float64
	}{
		{0, 0},
		{0.5, 0.5},
		{1, 1},
		{-1, 0},
		{1.2, 1},
		{math.NaN(), 0},
		{math.Inf(1), 1},
		{math.Inf(-1), 0},
	}
	for _, tt := range tests {
		assert.InDelta(t, tt.want, clamp01(tt.in), 1e-9)
	}
}

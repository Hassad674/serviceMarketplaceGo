package antigaming

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/search/features"
)

// velocityRule scales rating_score_diverse by (n-excess)/n when the 24h
// window exceeds the cap.
func TestVelocityRule(t *testing.T) {
	cfg := DefaultConfig()
	const now = 1_700_000_000
	const day = 86400

	t.Run("no recent reviews -> no penalty", func(t *testing.T) {
		f := &features.Features{RatingScoreDiverse: 0.8}
		raw := RawSignals{NowUnix: now}
		pen := velocityRule(f, raw, cfg)
		assert.Nil(t, pen)
		assert.InDelta(t, 0.8, f.RatingScoreDiverse, 1e-9)
	})

	t.Run("exactly at cap -> no penalty", func(t *testing.T) {
		f := &features.Features{RatingScoreDiverse: 0.8}
		timestamps := make([]int64, cfg.VelocityCap24h)
		for i := range timestamps {
			timestamps[i] = now - 3600 // 1h ago
		}
		raw := RawSignals{
			NowUnix:                now,
			RecentReviewTimestamps: timestamps,
			TotalReviewCount:       20,
		}
		pen := velocityRule(f, raw, cfg)
		assert.Nil(t, pen)
	})

	t.Run("above cap -> dampened", func(t *testing.T) {
		f := &features.Features{RatingScoreDiverse: 1.0}
		timestamps := make([]int64, 10) // 10 in 24h vs cap 5
		for i := range timestamps {
			timestamps[i] = now - 3600
		}
		raw := RawSignals{
			NowUnix:                now,
			RecentReviewTimestamps: timestamps,
			TotalReviewCount:       20,
		}
		pen := velocityRule(f, raw, cfg)
		assert.NotNil(t, pen)
		// excess = 10-5 = 5 ; factor = (20-5)/20 = 0.75
		assert.InDelta(t, 0.75, pen.PenaltyFactor, 1e-9)
		assert.InDelta(t, 0.75, f.RatingScoreDiverse, 1e-9)
	})

	t.Run("excess >= total -> zero rating", func(t *testing.T) {
		f := &features.Features{RatingScoreDiverse: 0.9}
		// cap=5, recent=20, total=5 -> excess 15 >= 5 -> zero
		timestamps := make([]int64, 20)
		for i := range timestamps {
			timestamps[i] = now - 3600
		}
		raw := RawSignals{
			NowUnix:                now,
			RecentReviewTimestamps: timestamps,
			TotalReviewCount:       5,
		}
		pen := velocityRule(f, raw, cfg)
		assert.NotNil(t, pen)
		assert.Equal(t, 0.0, pen.PenaltyFactor)
		assert.Equal(t, 0.0, f.RatingScoreDiverse)
	})

	t.Run("unknown total -> zero rating", func(t *testing.T) {
		f := &features.Features{RatingScoreDiverse: 0.9}
		timestamps := make([]int64, 10)
		for i := range timestamps {
			timestamps[i] = now - 3600
		}
		raw := RawSignals{
			NowUnix:                now,
			RecentReviewTimestamps: timestamps,
			TotalReviewCount:       0,
		}
		pen := velocityRule(f, raw, cfg)
		assert.NotNil(t, pen)
		assert.Equal(t, 0.0, f.RatingScoreDiverse)
	})

	t.Run("timestamps older than 24h filtered out", func(t *testing.T) {
		f := &features.Features{RatingScoreDiverse: 0.8}
		timestamps := []int64{
			now - 25*3600, // outside window
			now - 25*3600,
			now - 25*3600,
			now - 25*3600,
			now - 25*3600,
			now - 25*3600,
			now - 25*3600,
			now - 25*3600,
		}
		raw := RawSignals{
			NowUnix:                now,
			RecentReviewTimestamps: timestamps,
			TotalReviewCount:       20,
		}
		pen := velocityRule(f, raw, cfg)
		assert.Nil(t, pen)
	})

	t.Run("zero timestamps ignored", func(t *testing.T) {
		f := &features.Features{RatingScoreDiverse: 0.8}
		timestamps := []int64{0, 0, 0, 0, 0, 0}
		raw := RawSignals{
			NowUnix:                now,
			RecentReviewTimestamps: timestamps,
			TotalReviewCount:       20,
		}
		pen := velocityRule(f, raw, cfg)
		assert.Nil(t, pen)
	})

	t.Run("missing now unix -> safe no-op", func(t *testing.T) {
		f := &features.Features{RatingScoreDiverse: 0.8}
		timestamps := make([]int64, 10)
		for i := range timestamps {
			timestamps[i] = 1_000_000
		}
		raw := RawSignals{
			NowUnix:                0,
			RecentReviewTimestamps: timestamps,
			TotalReviewCount:       20,
		}
		pen := velocityRule(f, raw, cfg)
		assert.Nil(t, pen)
	})

	t.Run("penalty factor between 0 and 1 even with extreme inputs", func(t *testing.T) {
		f := &features.Features{RatingScoreDiverse: 0.5}
		timestamps := make([]int64, 1_000_000/day*day)
		_ = timestamps
		timestamps = make([]int64, 100)
		for i := range timestamps {
			timestamps[i] = now - 3600
		}
		raw := RawSignals{
			NowUnix:                now,
			RecentReviewTimestamps: timestamps,
			TotalReviewCount:       1000,
		}
		pen := velocityRule(f, raw, cfg)
		assert.NotNil(t, pen)
		assert.GreaterOrEqual(t, pen.PenaltyFactor, 0.0)
		assert.LessOrEqual(t, pen.PenaltyFactor, 1.0)
	})
}

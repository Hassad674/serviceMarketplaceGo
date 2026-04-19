package antigaming

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/search/features"
)

// stubDetector is a test double implementing LinkedReviewersDetector.
type stubDetector struct {
	linked int
	err    error
}

func (s stubDetector) LinkedCount(_ context.Context, _ []string) (int, error) {
	return s.linked, s.err
}

// NoopLinkedReviewersDetector always returns 0, so the rule never fires.
func TestNoopLinkedReviewersDetector(t *testing.T) {
	ctx := context.Background()
	det := NoopLinkedReviewersDetector{}
	n, err := det.LinkedCount(ctx, []string{"a", "b", "c"})
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
}

// linkedRule fires when linked_fraction > LinkedMaxFraction.
func TestLinkedRule(t *testing.T) {
	cfg := DefaultConfig() // LinkedMaxFraction = 0.3

	tests := []struct {
		name         string
		reviewerIDs  []string
		linked       int
		ratingBefore float64
		wantPen      bool
		wantAfter    float64
	}{
		{
			name:         "zero reviewers -> no penalty",
			reviewerIDs:  nil,
			linked:       0,
			ratingBefore: 0.8,
			wantPen:      false,
			wantAfter:    0.8,
		},
		{
			name:         "below fraction threshold -> no penalty",
			reviewerIDs:  []string{"a", "b", "c", "d", "e"},
			linked:       1, // 1/5 = 0.2
			ratingBefore: 0.8,
			wantPen:      false,
			wantAfter:    0.8,
		},
		{
			name:         "at exactly threshold -> no penalty",
			reviewerIDs:  []string{"a", "b", "c", "d", "e"},
			linked:       1, // 0.2 <= 0.3, no penalty
			ratingBefore: 0.8,
			wantPen:      false,
			wantAfter:    0.8,
		},
		{
			name:         "above threshold -> dampened",
			reviewerIDs:  []string{"a", "b", "c", "d"},
			linked:       2, // 2/4 = 0.5 > 0.3
			ratingBefore: 0.8,
			wantPen:      true,
			wantAfter:    0.4, // factor 0.5 * 0.8
		},
		{
			name:         "all linked -> zero",
			reviewerIDs:  []string{"a", "b", "c"},
			linked:       3, // 1.0
			ratingBefore: 0.9,
			wantPen:      true,
			wantAfter:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &features.Features{RatingScoreDiverse: tt.ratingBefore}
			raw := RawSignals{ReviewerIDs: tt.reviewerIDs}
			det := stubDetector{linked: tt.linked}
			pen, err := linkedRule(context.Background(), f, raw, cfg, det)
			assert.NoError(t, err)
			if tt.wantPen {
				assert.NotNil(t, pen)
				assert.Equal(t, RuleLinkedAccounts, pen.Rule)
			} else {
				assert.Nil(t, pen)
			}
			assert.InDelta(t, tt.wantAfter, f.RatingScoreDiverse, 1e-9)
		})
	}
}

// linkedRule returns the detector error + does NOT mutate features.
func TestLinkedRule_DetectorError(t *testing.T) {
	cfg := DefaultConfig()
	f := &features.Features{RatingScoreDiverse: 0.8}
	raw := RawSignals{ReviewerIDs: []string{"a", "b"}}
	det := stubDetector{err: errors.New("backend down")}
	pen, err := linkedRule(context.Background(), f, raw, cfg, det)
	assert.Error(t, err)
	assert.Nil(t, pen)
	assert.InDelta(t, 0.8, f.RatingScoreDiverse, 1e-9)
}

// Nil detector -> no-op.
func TestLinkedRule_NilDetector(t *testing.T) {
	cfg := DefaultConfig()
	f := &features.Features{RatingScoreDiverse: 0.8}
	raw := RawSignals{ReviewerIDs: []string{"a", "b"}}
	pen, err := linkedRule(context.Background(), f, raw, cfg, nil)
	assert.NoError(t, err)
	assert.Nil(t, pen)
}

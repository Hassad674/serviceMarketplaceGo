package antigaming

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/search/features"
)

// reviewerFloorRule caps rating_score_diverse at cfg.FewReviewerCap when the
// unique-reviewer count is below the floor.
func TestReviewerFloorRule(t *testing.T) {
	cfg := DefaultConfig() // floor 3, cap 0.4

	tests := []struct {
		name        string
		rating      float64
		uniqueRev   int
		wantPen     bool
		wantAfter   float64
	}{
		{
			name:      "5 reviewers -> no penalty",
			rating:    0.8,
			uniqueRev: 5,
			wantPen:   false,
			wantAfter: 0.8,
		},
		{
			name:      "exactly at floor (3) -> no penalty",
			rating:    0.8,
			uniqueRev: 3,
			wantPen:   false,
			wantAfter: 0.8,
		},
		{
			name:      "2 reviewers, rating 0.9 -> capped to 0.4",
			rating:    0.9,
			uniqueRev: 2,
			wantPen:   true,
			wantAfter: 0.4,
		},
		{
			name:      "1 reviewer, rating 0.5 -> capped to 0.4",
			rating:    0.5,
			uniqueRev: 1,
			wantPen:   true,
			wantAfter: 0.4,
		},
		{
			name:      "2 reviewers, rating 0.3 -> already below, no log",
			rating:    0.3,
			uniqueRev: 2,
			wantPen:   false,
			wantAfter: 0.3,
		},
		{
			name:      "zero reviewers (cold start) -> no penalty",
			rating:    0.15,
			uniqueRev: 0,
			wantPen:   false,
			wantAfter: 0.15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &features.Features{
				RatingScoreDiverse: tt.rating,
				RawUniqueReviewers: tt.uniqueRev,
			}
			raw := RawSignals{ProfileID: "p1", Persona: features.PersonaFreelance}
			pen := reviewerFloorRule(f, raw, cfg)
			if tt.wantPen {
				assert.NotNil(t, pen)
				assert.Equal(t, RuleReviewerFloor, pen.Rule)
			} else {
				assert.Nil(t, pen)
			}
			assert.InDelta(t, tt.wantAfter, f.RatingScoreDiverse, 1e-9)
		})
	}
}

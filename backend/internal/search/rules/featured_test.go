package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyFeatured_BoostsFlagged(t *testing.T) {
	in := []Candidate{
		{DocumentID: "a", IsFeatured: false, Score: Score{Final: 80}},
		{DocumentID: "b", IsFeatured: true, Score: Score{Final: 50}},
		{DocumentID: "c", IsFeatured: true, Score: Score{Final: 60}},
	}
	applyFeatured(in, 0.15)
	assert.InDelta(t, 80.0, in[0].Score.Final, 1e-9, "unflagged untouched")
	assert.InDelta(t, 57.5, in[1].Score.Final, 1e-9, "50 × 1.15 = 57.5")
	assert.InDelta(t, 69.0, in[2].Score.Final, 1e-9, "60 × 1.15 = 69")
}

func TestApplyFeatured_ZeroBoostNoop(t *testing.T) {
	in := []Candidate{
		{IsFeatured: true, Score: Score{Final: 40}},
	}
	applyFeatured(in, 0)
	assert.InDelta(t, 40.0, in[0].Score.Final, 1e-9)
}

func TestApplyFeatured_NegativeBoostNoop(t *testing.T) {
	in := []Candidate{
		{IsFeatured: true, Score: Score{Final: 40}},
	}
	applyFeatured(in, -0.5)
	assert.InDelta(t, 40.0, in[0].Score.Final, 1e-9,
		"negative boost ignored — enforced in config.validate too")
}

func TestApplyFeatured_ClampsAbove100(t *testing.T) {
	in := []Candidate{
		{IsFeatured: true, Score: Score{Final: 95}},
	}
	applyFeatured(in, 0.5) // 95 × 1.5 = 142.5 → clamped.
	assert.InDelta(t, 100.0, in[0].Score.Final, 1e-9)
}

func TestApplyFeatured_LeavesUnflaggedUntouched(t *testing.T) {
	in := []Candidate{
		{IsFeatured: false, Score: Score{Final: 99}},
		{IsFeatured: false, Score: Score{Final: 1}},
	}
	applyFeatured(in, 0.99)
	assert.InDelta(t, 99.0, in[0].Score.Final, 1e-9)
	assert.InDelta(t, 1.0, in[1].Score.Final, 1e-9)
}

func TestApplyFeatured_EmptySlice(t *testing.T) {
	var in []Candidate
	assert.NotPanics(t, func() { applyFeatured(in, 0.15) })
}

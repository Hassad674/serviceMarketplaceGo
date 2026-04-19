package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTierOf(t *testing.T) {
	cases := []struct {
		status string
		want   Tier
	}{
		{"available_now", TierA},
		{"now", TierA},
		{"available", TierA},
		{"AVAILABLE_SOON", TierA},
		{"soon", TierA},
		{"not_available", TierB},
		{"", TierB},
		{"paused", TierB},
		{"booked", TierB},
	}
	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			assert.Equal(t, tc.want, TierOf(tc.status))
		})
	}
}

func TestSplitTiers(t *testing.T) {
	input := []Candidate{
		{DocumentID: "1", AvailabilityStatus: "available_now"},
		{DocumentID: "2", AvailabilityStatus: "not_available"},
		{DocumentID: "3", AvailabilityStatus: "available_soon"},
		{DocumentID: "4", AvailabilityStatus: "paused"},
		{DocumentID: "5", AvailabilityStatus: "available_now"},
	}
	a, b := splitTiers(input)
	assert.Equal(t, []string{"1", "3", "5"}, idsOf(a))
	assert.Equal(t, []string{"2", "4"}, idsOf(b))
	// Original input must be untouched.
	assert.Equal(t, "available_now", input[0].AvailabilityStatus)
}

func TestSortByFinalDesc_Stable(t *testing.T) {
	in := []Candidate{
		{DocumentID: "a", Score: Score{Final: 70}},
		{DocumentID: "b", Score: Score{Final: 80}},
		{DocumentID: "c", Score: Score{Final: 80}}, // tie with b
		{DocumentID: "d", Score: Score{Final: 60}},
	}
	sortByFinalDesc(in)
	assert.Equal(t, []string{"b", "c", "a", "d"}, idsOf(in),
		"stable sort: tied scores keep pre-sort order")
}

func TestReSortRespectingTiers(t *testing.T) {
	in := []Candidate{
		{DocumentID: "tierB_high", AvailabilityStatus: "not_available", Score: Score{Final: 95}},
		{DocumentID: "tierA_low", AvailabilityStatus: "available_now", Score: Score{Final: 30}},
		{DocumentID: "tierA_high", AvailabilityStatus: "available_now", Score: Score{Final: 90}},
		{DocumentID: "tierB_low", AvailabilityStatus: "not_available", Score: Score{Final: 20}},
	}
	reSortRespectingTiers(in, 4)
	assert.Equal(t, []string{"tierA_high", "tierA_low", "tierB_high", "tierB_low"}, idsOf(in),
		"Tier A precedes Tier B, each block sorted by Final DESC")
}

func TestReSortRespectingTiers_TopNClamp(t *testing.T) {
	in := []Candidate{
		{DocumentID: "a", AvailabilityStatus: "not_available", Score: Score{Final: 50}},
		{DocumentID: "b", AvailabilityStatus: "available_now", Score: Score{Final: 40}},
		{DocumentID: "c", AvailabilityStatus: "available_now", Score: Score{Final: 30}},
	}
	// TopN larger than slice should be clamped silently.
	reSortRespectingTiers(in, 100)
	assert.Equal(t, []string{"b", "c", "a"}, idsOf(in))
}

func TestReSortRespectingTiers_EmptySlice(t *testing.T) {
	in := []Candidate{}
	reSortRespectingTiers(in, 20)
	assert.Empty(t, in)
}

// idsOf is a local helper that returns the DocumentID field of every
// candidate in order — keeps the assertions readable.
func idsOf(in []Candidate) []string {
	out := make([]string, len(in))
	for i, c := range in {
		out[i] = c.DocumentID
	}
	return out
}

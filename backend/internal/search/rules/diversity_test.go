package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiversityPass_BreaksThreeInARow(t *testing.T) {
	in := []Candidate{
		{DocumentID: "a", PrimarySkill: "React", AvailabilityStatus: "available_now", Score: Score{Final: 90}},
		{DocumentID: "b", PrimarySkill: "React", AvailabilityStatus: "available_now", Score: Score{Final: 80}},
		{DocumentID: "c", PrimarySkill: "React", AvailabilityStatus: "available_now", Score: Score{Final: 70}},
		{DocumentID: "d", PrimarySkill: "Go", AvailabilityStatus: "available_now", Score: Score{Final: 60}},
		{DocumentID: "e", PrimarySkill: "Python", AvailabilityStatus: "available_now", Score: Score{Final: 50}},
	}
	diversityPass(in, 5)
	// Position 2 (index 2) should now be Go or Python, not React.
	assert.NotEqual(t, "React", in[2].PrimarySkill,
		"third React in a row must be swapped out")
}

func TestDiversityPass_KeepsTwoInARow(t *testing.T) {
	in := []Candidate{
		{DocumentID: "a", PrimarySkill: "React", AvailabilityStatus: "available_now", Score: Score{Final: 90}},
		{DocumentID: "b", PrimarySkill: "React", AvailabilityStatus: "available_now", Score: Score{Final: 80}},
		{DocumentID: "c", PrimarySkill: "Go", AvailabilityStatus: "available_now", Score: Score{Final: 70}},
		{DocumentID: "d", PrimarySkill: "Python", AvailabilityStatus: "available_now", Score: Score{Final: 60}},
	}
	orig := cloneCandidates(in)
	diversityPass(in, 4)
	assert.Equal(t, idsOf(orig), idsOf(in),
		"2-in-a-row is acceptable, no swap needed")
}

func TestDiversityPass_NoSwapAvailable(t *testing.T) {
	// Everyone has the same primary skill — no swap partner exists.
	in := []Candidate{
		{DocumentID: "a", PrimarySkill: "React", AvailabilityStatus: "available_now", Score: Score{Final: 90}},
		{DocumentID: "b", PrimarySkill: "React", AvailabilityStatus: "available_now", Score: Score{Final: 80}},
		{DocumentID: "c", PrimarySkill: "React", AvailabilityStatus: "available_now", Score: Score{Final: 70}},
		{DocumentID: "d", PrimarySkill: "React", AvailabilityStatus: "available_now", Score: Score{Final: 60}},
	}
	orig := cloneCandidates(in)
	diversityPass(in, 4)
	assert.Equal(t, idsOf(orig), idsOf(in),
		"no alternative primary skill → keep the run (soft rule)")
}

func TestDiversityPass_EmptyPrimarySkillNeverMatches(t *testing.T) {
	in := []Candidate{
		{DocumentID: "a", PrimarySkill: "", AvailabilityStatus: "available_now", Score: Score{Final: 90}},
		{DocumentID: "b", PrimarySkill: "", AvailabilityStatus: "available_now", Score: Score{Final: 80}},
		{DocumentID: "c", PrimarySkill: "", AvailabilityStatus: "available_now", Score: Score{Final: 70}},
		{DocumentID: "d", PrimarySkill: "Go", AvailabilityStatus: "available_now", Score: Score{Final: 60}},
	}
	orig := cloneCandidates(in)
	diversityPass(in, 4)
	assert.Equal(t, idsOf(orig), idsOf(in),
		"empty primary skills never form a run")
}

func TestDiversityPass_RespectsTierBoundaries(t *testing.T) {
	// Position 2 is React tier A; swap partners on the tail are all tier B.
	// The swap must NOT happen because tier boundaries are hard.
	in := []Candidate{
		{DocumentID: "a", PrimarySkill: "React", AvailabilityStatus: "available_now", Score: Score{Final: 90}},
		{DocumentID: "b", PrimarySkill: "React", AvailabilityStatus: "available_now", Score: Score{Final: 80}},
		{DocumentID: "c", PrimarySkill: "React", AvailabilityStatus: "available_now", Score: Score{Final: 70}},
		{DocumentID: "d", PrimarySkill: "Go", AvailabilityStatus: "not_available", Score: Score{Final: 60}},
		{DocumentID: "e", PrimarySkill: "Python", AvailabilityStatus: "not_available", Score: Score{Final: 50}},
	}
	orig := cloneCandidates(in)
	diversityPass(in, 5)
	assert.Equal(t, idsOf(orig), idsOf(in),
		"diversity swap must stay inside the incumbent's tier")
}

func TestDiversityPass_CanonicalisesCase(t *testing.T) {
	in := []Candidate{
		{DocumentID: "a", PrimarySkill: "React", AvailabilityStatus: "available_now", Score: Score{Final: 90}},
		{DocumentID: "b", PrimarySkill: " react ", AvailabilityStatus: "available_now", Score: Score{Final: 80}},
		{DocumentID: "c", PrimarySkill: "REACT", AvailabilityStatus: "available_now", Score: Score{Final: 70}},
		{DocumentID: "d", PrimarySkill: "Go", AvailabilityStatus: "available_now", Score: Score{Final: 60}},
	}
	diversityPass(in, 4)
	assert.NotEqual(t, "c", in[2].DocumentID,
		"case + whitespace should canonicalise to the same skill")
}

func TestDiversityPass_BelowThreeIgnored(t *testing.T) {
	// A 2-element slice has no room for a run.
	in := []Candidate{
		{DocumentID: "a", PrimarySkill: "React"},
		{DocumentID: "b", PrimarySkill: "React"},
	}
	orig := cloneCandidates(in)
	diversityPass(in, 2)
	assert.Equal(t, idsOf(orig), idsOf(in))
}

func TestFormsRun(t *testing.T) {
	base := []Candidate{
		{PrimarySkill: "A"},
		{PrimarySkill: "A"},
		{PrimarySkill: "A"},
		{PrimarySkill: "B"},
		{PrimarySkill: "A"},
		{PrimarySkill: "A"},
	}
	assert.False(t, formsRun(base, 1), "i<2 cannot form a run")
	assert.True(t, formsRun(base, 2), "three A's in a row")
	assert.False(t, formsRun(base, 3), "A-A-B is not a run")
	assert.False(t, formsRun(base, 5), "B-A-A is not a run")
}

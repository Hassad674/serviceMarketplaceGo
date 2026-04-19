package rules

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBusinessRules_Apply_Empty(t *testing.T) {
	br := NewBusinessRules(testConfig(42))
	out := br.Apply(context.Background(), nil, PersonaFreelance)
	assert.Nil(t, out)
}

func TestBusinessRules_Apply_NoDuplicates(t *testing.T) {
	in := make20SyntheticCandidates()
	br := NewBusinessRules(testConfig(42))
	out := br.Apply(context.Background(), in, PersonaFreelance)

	assert.LessOrEqual(t, len(out), len(in))
	seen := map[string]bool{}
	for _, c := range out {
		assert.False(t, seen[c.DocumentID], "duplicate %s", c.DocumentID)
		seen[c.DocumentID] = true
	}
}

func TestBusinessRules_Apply_NeverInventsCandidates(t *testing.T) {
	in := make20SyntheticCandidates()
	allowed := map[string]bool{}
	for _, c := range in {
		allowed[c.DocumentID] = true
	}
	br := NewBusinessRules(testConfig(42))
	out := br.Apply(context.Background(), in, PersonaFreelance)

	for _, c := range out {
		assert.True(t, allowed[c.DocumentID], "invented candidate %s", c.DocumentID)
	}
}

func TestBusinessRules_Apply_TierA_WinsOverTierB(t *testing.T) {
	// A low-scored tier-A profile should still appear BEFORE the
	// highest-scored tier-B profile.
	in := []Candidate{
		{DocumentID: "veteranUnavailable", AvailabilityStatus: "not_available", Score: Score{Final: 99}},
		{DocumentID: "juniorAvailable", AvailabilityStatus: "available_now", Score: Score{Final: 40}},
	}
	br := NewBusinessRules(testConfig(42))
	out := br.Apply(context.Background(), in, PersonaFreelance)

	require.Len(t, out, 2)
	assert.Equal(t, "juniorAvailable", out[0].DocumentID,
		"any Tier A candidate must precede every Tier B candidate")
	assert.Equal(t, "veteranUnavailable", out[1].DocumentID)
}

func TestBusinessRules_Apply_DeterministicSeed(t *testing.T) {
	in := make20SyntheticCandidates()
	brA := NewBusinessRules(testConfig(42))
	brB := NewBusinessRules(testConfig(42))
	outA := brA.Apply(context.Background(), cloneCandidates(in), PersonaFreelance)
	outB := brB.Apply(context.Background(), cloneCandidates(in), PersonaFreelance)
	assert.Equal(t, idsOf(outA), idsOf(outB),
		"same seed + same input ⇒ same ordering")
}

func TestBusinessRules_Apply_DifferentSeedsDifferentOrders(t *testing.T) {
	// Build a tight cluster of candidates whose scores are within
	// the noise envelope (σ ~= 0.9 at rank 11). Gaps smaller than
	// ~1.5 points are reshuffable under different seeds.
	in := make([]Candidate, 20)
	for i := range in {
		in[i] = Candidate{
			DocumentID:         idOf(i),
			AvailabilityStatus: "available_now",
			PrimarySkill:       "React",
			Score:              Score{Final: 70 - float64(i)*0.3},
		}
	}

	cfg := testConfig(1)
	// Bump noise so the test is robust to PRNG luck on tiny gaps.
	cfg.NoiseCoefficient = 0.05
	brA := NewBusinessRules(cfg)
	cfgB := cfg
	cfgB.RandSeed = 10_000
	brB := NewBusinessRules(cfgB)
	outA := brA.Apply(context.Background(), cloneCandidates(in), PersonaFreelance)
	outB := brB.Apply(context.Background(), cloneCandidates(in), PersonaFreelance)
	differ := false
	for i := range outA {
		if i >= len(outB) {
			break
		}
		if outA[i].DocumentID != outB[i].DocumentID {
			differ = true
			break
		}
	}
	assert.True(t, differ, "different seeds should reshuffle at least once")
}

func TestBusinessRules_Apply_DiversityStress_AllSameSkill(t *testing.T) {
	// Everyone with the same primary skill — diversity rule has no
	// alternative and must leave the order alone beyond normal
	// rank-noise. Key invariant: no candidate lost.
	in := make([]Candidate, 20)
	for i := range in {
		in[i] = Candidate{
			DocumentID:         idOf(i),
			PrimarySkill:       "React",
			AvailabilityStatus: "available_now",
			Score:              Score{Final: float64(95 - i)},
		}
	}
	br := NewBusinessRules(testConfig(42))
	out := br.Apply(context.Background(), in, PersonaFreelance)
	assert.Len(t, out, 20)
}

func TestBusinessRules_Apply_NoRisingTalentFallback(t *testing.T) {
	// No eligible rising talent → slots 5/10/15/20 stay with their
	// veteran incumbents.
	in := make([]Candidate, 20)
	for i := range in {
		in[i] = Candidate{
			DocumentID:         idOf(i),
			AvailabilityStatus: "available_now",
			AccountAgeDays:     500,
			IsVerified:         true,
			Score:              Score{Final: float64(95 - i)},
		}
	}
	br := NewBusinessRules(testConfig(42))
	out := br.Apply(context.Background(), cloneCandidates(in), PersonaFreelance)
	// All 20 should be the exact input set.
	assert.ElementsMatch(t, idsOf(in), idsOf(out))
}

func TestBusinessRules_Apply_AllAvailableNow(t *testing.T) {
	in := make([]Candidate, 20)
	for i := range in {
		in[i] = Candidate{
			DocumentID:         idOf(i),
			AvailabilityStatus: "available_now",
			Score:              Score{Final: float64(90 - i)},
		}
	}
	br := NewBusinessRules(testConfig(42))
	out := br.Apply(context.Background(), in, PersonaFreelance)
	for _, c := range out {
		assert.Equal(t, TierA, TierOf(c.AvailabilityStatus))
	}
}

func TestBusinessRules_Apply_AllNotAvailable(t *testing.T) {
	in := make([]Candidate, 20)
	for i := range in {
		in[i] = Candidate{
			DocumentID:         idOf(i),
			AvailabilityStatus: "not_available",
			Score:              Score{Final: float64(90 - i)},
		}
	}
	br := NewBusinessRules(testConfig(42))
	out := br.Apply(context.Background(), in, PersonaFreelance)
	assert.Len(t, out, 20)
	for _, c := range out {
		assert.Equal(t, TierB, TierOf(c.AvailabilityStatus))
	}
}

func TestBusinessRules_Apply_FeaturedOverride(t *testing.T) {
	in := []Candidate{
		{DocumentID: "a", AvailabilityStatus: "available_now", Score: Score{Final: 85}},
		{DocumentID: "b", AvailabilityStatus: "available_now", Score: Score{Final: 80}, IsFeatured: true},
		{DocumentID: "c", AvailabilityStatus: "available_now", Score: Score{Final: 70}},
	}
	cfg := testConfig(42)
	cfg.FeaturedEnabled = true
	cfg.FeaturedBoost = 0.15
	br := NewBusinessRules(cfg)
	out := br.Apply(context.Background(), cloneCandidates(in), PersonaFreelance)
	// 'b' gets boosted to 80 × 1.15 = 92 which should promote it to
	// position 0.
	require.NotEmpty(t, out)
	assert.Equal(t, "b", out[0].DocumentID)
}

func TestBusinessRules_Apply_FeaturedDormantByDefault(t *testing.T) {
	in := []Candidate{
		{DocumentID: "a", AvailabilityStatus: "available_now", Score: Score{Final: 85}},
		{DocumentID: "b", AvailabilityStatus: "available_now", Score: Score{Final: 80}, IsFeatured: true},
	}
	br := NewBusinessRules(testConfig(42))
	out := br.Apply(context.Background(), cloneCandidates(in), PersonaFreelance)
	// Default config: FeaturedEnabled = false → 'a' still wins.
	require.NotEmpty(t, out)
	assert.Equal(t, "a", out[0].DocumentID)
}

func TestBusinessRules_Apply_ExtremeScoreSpread(t *testing.T) {
	// One candidate at 99, nineteen at 10. Noise never lets a 10
	// overtake the 99.
	in := make([]Candidate, 20)
	in[0] = Candidate{DocumentID: "star", AvailabilityStatus: "available_now", Score: Score{Final: 99}}
	for i := 1; i < 20; i++ {
		in[i] = Candidate{
			DocumentID:         idOf(i),
			AvailabilityStatus: "available_now",
			Score:              Score{Final: 10},
		}
	}
	br := NewBusinessRules(testConfig(42))
	out := br.Apply(context.Background(), in, PersonaFreelance)
	assert.Equal(t, "star", out[0].DocumentID,
		"extreme score spread → top-1 is unshakeable")
}

func TestBusinessRules_Apply_RisingTalentEligibilityNeverViolated(t *testing.T) {
	// Seed property test: whatever the config / input, any candidate
	// that ends up in a rising slot MUST be eligible.
	cfg := testConfig(42)
	in := make([]Candidate, 30)
	for i := range in {
		in[i] = Candidate{
			DocumentID:         idOf(i),
			AvailabilityStatus: "available_now",
			AccountAgeDays:     500,
			IsVerified:         true,
			Score:              Score{Final: float64(95 - i)},
		}
	}
	// Plant two eligible rising candidates deep in the pool.
	in[25].AccountAgeDays = 15
	in[25].Score.Final = 78
	in[27].AccountAgeDays = 40
	in[27].Score.Final = 72

	br := NewBusinessRules(cfg)
	out := br.Apply(context.Background(), in, PersonaFreelance)

	// Check property: for every rising slot (5, 10, 15, 20), IF the
	// candidate is NOT a veteran-incumbent (i.e. it was swapped in),
	// it must be eligible.
	median := medianFinal(in)
	for slot := cfg.RisingTalentSlotEvery; slot <= cfg.TopN && slot <= len(out); slot += cfg.RisingTalentSlotEvery {
		c := out[slot-1]
		if c.AccountAgeDays >= cfg.RisingTalentMaxAge {
			// Veteran incumbent — allowed, not a rising swap.
			continue
		}
		assert.True(t, isRisingEligible(c, cfg, median),
			"slot %d holds a rising candidate that is not eligible: %+v", slot, c)
	}
}

func TestBusinessRules_Apply_TruncatesToTopN(t *testing.T) {
	in := make([]Candidate, 50)
	for i := range in {
		in[i] = Candidate{
			DocumentID:         idOf(i),
			AvailabilityStatus: "available_now",
			Score:              Score{Final: float64(99 - i)},
		}
	}
	cfg := testConfig(42)
	cfg.TopN = 10
	br := NewBusinessRules(cfg)
	out := br.Apply(context.Background(), in, PersonaFreelance)
	assert.Len(t, out, 10)
}

func TestBusinessRules_Config(t *testing.T) {
	br := NewBusinessRules(testConfig(99))
	assert.Equal(t, int64(99), br.Config().RandSeed)
}

func TestBusinessRules_ZeroConfigFallsBackToDefaults(t *testing.T) {
	br := NewBusinessRules(Config{})
	assert.Equal(t, 20, br.Config().TopN,
		"zero Config should fall back to DefaultConfig")
}

func TestBusinessRules_NondeterministicSeed(t *testing.T) {
	// When RandSeed is 0, the seed source is time-based. We cannot
	// assert a specific output, but two freshly-built rules on
	// different wall clocks MUST yield different pipelineRun seeds.
	// Verify by running the pipeline through twice in quick
	// succession and checking at least one run produced a different
	// ordering from another.
	in := make([]Candidate, 20)
	for i := range in {
		in[i] = Candidate{
			DocumentID:         idOf(i),
			AvailabilityStatus: "available_now",
			PrimarySkill:       "React",
			Score:              Score{Final: 70 - float64(i)*0.1},
		}
	}
	cfg := DefaultConfig()
	cfg.NoiseCoefficient = 0.10 // enough noise to jiggle the tight cluster
	br := NewBusinessRules(cfg)
	firstRun := br.Apply(context.Background(), cloneCandidates(in), PersonaFreelance)

	// Force different wall-clock nanoseconds for the second seed.
	for attempts := 0; attempts < 50; attempts++ {
		secondRun := br.Apply(context.Background(), cloneCandidates(in), PersonaFreelance)
		for i := range firstRun {
			if firstRun[i].DocumentID != secondRun[i].DocumentID {
				return // success path
			}
		}
	}
	t.Fatal("nondeterministic seed never produced a different ordering across 50 runs")
}

func TestBusinessRules_Apply_EmptyBreakdownSurvives(t *testing.T) {
	in := []Candidate{
		{DocumentID: "solo", AvailabilityStatus: "available_now", Score: Score{Final: 75}},
	}
	br := NewBusinessRules(testConfig(7))
	out := br.Apply(context.Background(), in, PersonaFreelance)
	require.Len(t, out, 1)
	assert.Equal(t, "solo", out[0].DocumentID)
}

// testConfig returns a deterministic Config anchored on the given
// seed so tests can rely on reproducible pipeline output.
func testConfig(seed int64) Config {
	c := DefaultConfig()
	c.RandSeed = seed
	return c
}

// make20SyntheticCandidates returns a realistic 20-candidate mix
// with score spread, mixed availability, varied primary skills,
// and a sprinkling of rising-talent-eligible newcomers.
func make20SyntheticCandidates() []Candidate {
	skills := []string{"React", "React", "Go", "Python", "Design", "React", "Go", "Flutter", "Node", "Rails",
		"React", "React", "Python", "Design", "SEO", "React", "Vue", "Go", "Python", "React"}
	statuses := []string{
		"available_now", "available_now", "available_soon", "not_available", "available_now",
		"available_now", "available_soon", "available_now", "not_available", "available_now",
		"available_now", "available_now", "not_available", "available_soon", "available_now",
		"available_now", "not_available", "available_now", "available_soon", "not_available",
	}
	out := make([]Candidate, 20)
	for i := 0; i < 20; i++ {
		out[i] = Candidate{
			DocumentID:         fmt.Sprintf("doc-%02d", i),
			OrganizationID:     fmt.Sprintf("org-%02d", i),
			Persona:            PersonaFreelance,
			PrimarySkill:       skills[i],
			AvailabilityStatus: statuses[i],
			AccountAgeDays:     500 - i*5,
			IsVerified:         i%3 == 0,
			Score: Score{
				Final:     float64(95 - i*3),
				Base:      float64(95 - i*3),
				Adjusted:  float64(95 - i*3),
				Breakdown: map[string]float64{"text_match": 0.8, "rating": 0.7},
			},
		}
	}
	// Plant a few rising-talent eligibles.
	out[15].AccountAgeDays = 25
	out[15].IsVerified = true
	out[18].AccountAgeDays = 50
	out[18].IsVerified = true
	return out
}

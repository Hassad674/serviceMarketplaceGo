package rules

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoiseSigma_RankMultipliers(t *testing.T) {
	knobs := defaultNoiseKnobs()
	// §6.1 worked example: score = 80, coefficient = 0.006.
	// rank 1 → σ = 0.006 × 80 × 0.3 = 0.144.
	// rank 5 → σ = 0.006 × 80 × 0.8 = 0.384.
	// rank 15 → σ = 0.006 × 80 × 1.5 = 0.720.
	cases := []struct {
		rank  int
		want  float64
		label string
	}{
		{1, 0.144, "top-3 (rank 1)"},
		{3, 0.144, "top-3 (rank 3)"},
		{4, 0.384, "mid (rank 4)"},
		{10, 0.384, "mid (rank 10)"},
		{11, 0.720, "tail (rank 11)"},
		{50, 0.720, "tail (rank 50)"},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			got := noiseSigma(80, tc.rank, knobs)
			assert.InDelta(t, tc.want, got, 1e-9)
		})
	}
}

func TestNoiseSigma_ZeroScore(t *testing.T) {
	assert.Equal(t, 0.0, noiseSigma(0, 1, defaultNoiseKnobs()),
		"score=0 must not generate noise (would push the candidate negative)")
	assert.Equal(t, 0.0, noiseSigma(-1, 1, defaultNoiseKnobs()),
		"negative score also skipped")
}

func TestNoiseSigma_ZeroCoefficient(t *testing.T) {
	knobs := noiseKnobs{coefficient: 0, top3Multiplier: 0.3, midMultiplier: 0.8, tailMultiplier: 1.5}
	assert.Equal(t, 0.0, noiseSigma(80, 5, knobs),
		"coefficient=0 disables randomisation entirely")
}

func TestRandomiseWithKnobs_DeterministicSeed(t *testing.T) {
	original := buildScoredList([]float64{90, 80, 70, 60, 50, 40, 30, 20, 10, 5})

	run1 := &pipelineRun{rng: rand.New(rand.NewSource(42))}
	run2 := &pipelineRun{rng: rand.New(rand.NewSource(42))}
	a := cloneCandidates(original)
	b := cloneCandidates(original)
	randomiseWithKnobs(a, run1, defaultNoiseKnobs())
	randomiseWithKnobs(b, run2, defaultNoiseKnobs())

	for i := range a {
		assert.InDelta(t, a[i].Score.Final, b[i].Score.Final, 1e-12,
			"same seed ⇒ same sequence")
	}
}

func TestRandomiseWithKnobs_ProducesDifferentSequencesPerSeed(t *testing.T) {
	original := buildScoredList([]float64{90, 80, 70, 60, 50})

	runA := &pipelineRun{rng: rand.New(rand.NewSource(1))}
	runB := &pipelineRun{rng: rand.New(rand.NewSource(7))}
	a := cloneCandidates(original)
	b := cloneCandidates(original)
	randomiseWithKnobs(a, runA, defaultNoiseKnobs())
	randomiseWithKnobs(b, runB, defaultNoiseKnobs())

	diffs := 0
	for i := range a {
		if a[i].Score.Final != b[i].Score.Final {
			diffs++
		}
	}
	assert.Greater(t, diffs, 0, "different seeds should produce different noise")
}

func TestRandomiseWithKnobs_ClampsToValidRange(t *testing.T) {
	// Worst case: seed the RNG on a low-scored candidate with heavy
	// negative noise. Should never dip below 0 or above 100.
	list := buildScoredList([]float64{0.5, 0.5, 0.5, 0.5, 99.9, 99.9})
	run := &pipelineRun{rng: rand.New(rand.NewSource(1000))}
	// Extreme coefficient to force clamp exercise.
	knobs := noiseKnobs{coefficient: 5, top3Multiplier: 1, midMultiplier: 1, tailMultiplier: 1}
	randomiseWithKnobs(list, run, knobs)
	for i, c := range list {
		assert.GreaterOrEqual(t, c.Score.Final, 0.0, "candidate %d below zero", i)
		assert.LessOrEqual(t, c.Score.Final, 100.0, "candidate %d above 100", i)
	}
}

func TestClampScore01(t *testing.T) {
	assert.Equal(t, 0.0, clampScore01(-5))
	assert.Equal(t, 0.0, clampScore01(0))
	assert.Equal(t, 50.0, clampScore01(50))
	assert.Equal(t, 100.0, clampScore01(100))
	assert.Equal(t, 100.0, clampScore01(150))
}

// buildScoredList returns candidates with the given Score.Final values
// and synthetic unique DocumentIDs.
func buildScoredList(scores []float64) []Candidate {
	out := make([]Candidate, len(scores))
	for i, s := range scores {
		out[i] = Candidate{
			DocumentID: idOf(i),
			Score:      Score{Final: s, Breakdown: map[string]float64{}},
		}
	}
	return out
}

func cloneCandidates(in []Candidate) []Candidate {
	out := make([]Candidate, len(in))
	copy(out, in)
	return out
}

func idOf(i int) string {
	return "doc-" + itoa(i)
}

// itoa avoids pulling strconv for a 10-line helper.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	negative := i < 0
	if negative {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if negative {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

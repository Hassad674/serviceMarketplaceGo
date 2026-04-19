package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRisingEligible(t *testing.T) {
	cfg := DefaultConfig()
	median := 50.0
	cases := []struct {
		name string
		c    Candidate
		want bool
	}{
		{
			name: "too old",
			c:    Candidate{AccountAgeDays: 120, IsVerified: true, Score: Score{Final: 70}},
			want: false,
		},
		{
			name: "unverified",
			c:    Candidate{AccountAgeDays: 20, IsVerified: false, Score: Score{Final: 70}},
			want: false,
		},
		{
			name: "below median",
			c:    Candidate{AccountAgeDays: 20, IsVerified: true, Score: Score{Final: 40}},
			want: false,
		},
		{
			name: "eligible",
			c:    Candidate{AccountAgeDays: 20, IsVerified: true, Score: Score{Final: 70}},
			want: true,
		},
		{
			name: "exactly at median",
			c:    Candidate{AccountAgeDays: 30, IsVerified: true, Score: Score{Final: 50}},
			want: true,
		},
		{
			name: "zero age = brand new, skip",
			c:    Candidate{AccountAgeDays: 0, IsVerified: true, Score: Score{Final: 70}},
			want: false,
		},
		{
			name: "at max age boundary = not eligible (strict <)",
			c:    Candidate{AccountAgeDays: 60, IsVerified: true, Score: Score{Final: 70}},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isRisingEligible(tc.c, cfg, median))
		})
	}
}

func TestMedianFinal(t *testing.T) {
	cases := []struct {
		name   string
		scores []float64
		want   float64
	}{
		{"empty", nil, 0},
		{"single", []float64{80}, 80},
		{"odd count", []float64{10, 20, 30, 40, 50}, 30},
		{"even count", []float64{10, 20, 30, 40}, 25},
		{"unsorted", []float64{50, 10, 30, 40, 20}, 30},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := buildScoredList(tc.scores)
			assert.InDelta(t, tc.want, medianFinal(in), 1e-9)
		})
	}
}

func TestInjectRising_ReplacesEligibleIncumbent(t *testing.T) {
	cfg := DefaultConfig()
	// Top-20 filled with veterans; a strong rising talent sits at
	// position 21 (index 20).
	pool := make([]Candidate, 25)
	for i := 0; i < 25; i++ {
		pool[i] = Candidate{
			DocumentID:     idOf(i),
			Score:          Score{Final: float64(90 - i)},
			IsVerified:     true,
			AccountAgeDays: 500, // veteran
		}
	}
	// Position 5 incumbent (index 4) is a veteran at score 86.
	// Plant a rising talent at index 21 with a score close enough (84 ≥ 86−5=81).
	pool[21].AccountAgeDays = 20
	pool[21].Score.Final = 84

	injectRising(pool, cfg)
	assert.Equal(t, idOf(21), pool[4].DocumentID,
		"rising candidate should land in slot 5 (index 4)")
}

func TestInjectRising_RespectsDelta(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RisingTalentDelta = 2 // stricter
	pool := make([]Candidate, 25)
	for i := 0; i < 25; i++ {
		pool[i] = Candidate{
			DocumentID:     idOf(i),
			Score:          Score{Final: float64(90 - i)},
			IsVerified:     true,
			AccountAgeDays: 500,
		}
	}
	// Rising candidate at index 21 has score 82. Incumbent at
	// index 4 is 86. Cutoff = 86 − 2 = 84. 82 < 84 → no swap.
	pool[21].AccountAgeDays = 20
	pool[21].Score.Final = 82

	origID := pool[4].DocumentID
	injectRising(pool, cfg)
	assert.Equal(t, origID, pool[4].DocumentID,
		"rising candidate below delta cutoff must not replace the incumbent")
}

func TestInjectRising_SkipsIncumbentAlreadyRising(t *testing.T) {
	cfg := DefaultConfig()
	pool := make([]Candidate, 25)
	for i := 0; i < 25; i++ {
		pool[i] = Candidate{
			DocumentID:     idOf(i),
			Score:          Score{Final: float64(90 - i)},
			IsVerified:     true,
			AccountAgeDays: 500,
		}
	}
	// Slot 5 incumbent is already a rising talent.
	pool[4].AccountAgeDays = 25
	pool[4].Score.Final = 86 // above median
	// Plant a stronger candidate further down.
	pool[21].AccountAgeDays = 20
	pool[21].Score.Final = 87

	origSlot5 := pool[4].DocumentID
	injectRising(pool, cfg)
	assert.Equal(t, origSlot5, pool[4].DocumentID,
		"incumbent already rising → no swap")
}

func TestInjectRising_NoEligibleCandidate(t *testing.T) {
	cfg := DefaultConfig()
	pool := make([]Candidate, 25)
	for i := 0; i < 25; i++ {
		pool[i] = Candidate{
			DocumentID:     idOf(i),
			Score:          Score{Final: float64(90 - i)},
			IsVerified:     true,
			AccountAgeDays: 500, // all veterans
		}
	}
	orig := idsOf(pool)[:cfg.TopN]
	injectRising(pool, cfg)
	assert.Equal(t, orig, idsOf(pool)[:cfg.TopN],
		"no rising candidate → top-20 unchanged")
}

func TestInjectRising_UnverifiedRisingCandidateSkipped(t *testing.T) {
	cfg := DefaultConfig()
	pool := make([]Candidate, 25)
	for i := 0; i < 25; i++ {
		pool[i] = Candidate{
			DocumentID:     idOf(i),
			Score:          Score{Final: float64(90 - i)},
			IsVerified:     true,
			AccountAgeDays: 500,
		}
	}
	// New account but NOT verified.
	pool[21].AccountAgeDays = 20
	pool[21].Score.Final = 85
	pool[21].IsVerified = false

	origSlot5 := pool[4].DocumentID
	injectRising(pool, cfg)
	assert.Equal(t, origSlot5, pool[4].DocumentID,
		"unverified newcomer must not take a rising slot")
}

func TestInjectRising_NotEnoughCandidates(t *testing.T) {
	cfg := DefaultConfig()
	// Only 3 candidates — below RisingTalentSlotEvery (5).
	pool := buildScoredList([]float64{90, 80, 70})
	orig := cloneCandidates(pool)
	injectRising(pool, cfg)
	assert.Equal(t, idsOf(orig), idsOf(pool),
		"pool smaller than slot interval → no-op")
}

func TestInjectRising_ZeroSlotEvery(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RisingTalentSlotEvery = 0
	pool := make([]Candidate, 20)
	for i := range pool {
		pool[i] = Candidate{DocumentID: idOf(i), Score: Score{Final: float64(90 - i)}}
	}
	orig := cloneCandidates(pool)
	injectRising(pool, cfg)
	assert.Equal(t, idsOf(orig), idsOf(pool),
		"SlotEvery=0 must early-return without panicking")
}

func TestInjectRising_MultipleRisingCandidatesPicksHighestScore(t *testing.T) {
	cfg := DefaultConfig()
	pool := make([]Candidate, 25)
	for i := 0; i < 25; i++ {
		pool[i] = Candidate{
			DocumentID:     idOf(i),
			Score:          Score{Final: float64(90 - i)},
			IsVerified:     true,
			AccountAgeDays: 500,
		}
	}
	// Plant two rising candidates below the top-20 cut. The higher-
	// scored one must win slot 5.
	pool[21].AccountAgeDays = 20
	pool[21].Score.Final = 82
	pool[22].AccountAgeDays = 20
	pool[22].Score.Final = 85

	injectRising(pool, cfg)
	assert.Equal(t, idOf(22), pool[4].DocumentID,
		"among eligible rising candidates, the highest Final wins")
}

func TestInjectRising_RisingCandidatesInsideTopNSkipped(t *testing.T) {
	// A rising candidate already placed INSIDE the top-20 must not
	// be "moved" to a rising slot — that would be a no-op shuffle.
	cfg := DefaultConfig()
	pool := make([]Candidate, 25)
	for i := 0; i < 25; i++ {
		pool[i] = Candidate{
			DocumentID:     idOf(i),
			Score:          Score{Final: float64(90 - i)},
			IsVerified:     true,
			AccountAgeDays: 500,
		}
	}
	// Rising candidate is at index 7 (inside the top-20).
	pool[7].AccountAgeDays = 20
	origSlot10 := pool[9].DocumentID
	injectRising(pool, cfg)
	assert.Equal(t, origSlot10, pool[9].DocumentID,
		"rising candidate already in top-N must not be shuffled")
}

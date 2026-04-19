package scorer

import (
	"math"
	"testing"
)

// TestRedistribute_PreservesSum is the core invariant of §5.2: after
// redistributing TextMatch's weight across the remaining eight
// features, the total weight is still exactly 1.0 within
// floatTolerance. Runs across all three personas to prove the
// property holds independent of the specific weight table.
func TestRedistribute_PreservesSum(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		w    PersonaWeights
	}{
		{"freelance", DefaultFreelanceWeights()},
		{"agency", DefaultAgencyWeights()},
		{"referrer", DefaultReferrerWeights()},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			red := RedistributeForEmptyQuery(c.w)
			sum := red.Sum()
			if math.Abs(sum-1.0) > floatTolerance {
				t.Fatalf("redistributed %s sum = %.12f, want 1.0 (delta %.2e)",
					c.name, sum, math.Abs(sum-1.0))
			}
			// TextMatch must land at exactly 0 — that's the whole point.
			if red.TextMatch != 0 {
				t.Fatalf("redistributed TextMatch = %v, want 0", red.TextMatch)
			}
		})
	}
}

// TestRedistribute_Proportional asserts the relative weighting of the
// remaining eight features is preserved under the redistribution. For
// any two features a and b other than TextMatch, the ratio
// redistributed(a)/redistributed(b) must equal original(a)/original(b).
func TestRedistribute_Proportional(t *testing.T) {
	t.Parallel()

	orig := DefaultFreelanceWeights()
	red := RedistributeForEmptyQuery(orig)

	// Compare rating (0.20) vs proven_work (0.15). Original ratio 4:3.
	origRatio := orig.Rating / orig.ProvenWork
	redRatio := red.Rating / red.ProvenWork

	if math.Abs(origRatio-redRatio) > floatTolerance {
		t.Fatalf("proportions shifted: before %.6f, after %.6f", origRatio, redRatio)
	}

	// Also check skills (0.15) vs response (0.10) — ratio 3:2.
	origSkillsResp := orig.SkillsOverlap / orig.ResponseRate
	redSkillsResp := red.SkillsOverlap / red.ResponseRate
	if math.Abs(origSkillsResp-redSkillsResp) > floatTolerance {
		t.Fatalf("skills/response ratio shifted: before %.6f, after %.6f",
			origSkillsResp, redSkillsResp)
	}
}

// TestRedistribute_TextMatchZero_NoOp confirms the defensive branch:
// when TextMatch is already 0 the input is returned unchanged so the
// function is idempotent when reapplied.
func TestRedistribute_TextMatchZero_NoOp(t *testing.T) {
	t.Parallel()

	w := DefaultReferrerWeights()
	// Construct a weight table with TextMatch=0 (referrer-variant).
	zeroTM := w
	zeroTM.TextMatch = 0
	zeroTM.Rating += 0.20 // absorb the text_match slice into rating

	if math.Abs(zeroTM.Sum()-1.0) > floatTolerance {
		t.Fatalf("test fixture does not sum to 1: %v", zeroTM.Sum())
	}

	red := RedistributeForEmptyQuery(zeroTM)
	if red != zeroTM {
		t.Fatalf("redistribute on TextMatch=0 should be a no-op; got %#v", red)
	}
}

// TestRedistribute_DegenerateTextMatchOne covers the defensive branch
// where TextMatch alone consumes the entire weight. Should return the
// input untouched (real Config validation stops this at startup).
func TestRedistribute_DegenerateTextMatchOne(t *testing.T) {
	t.Parallel()

	deg := PersonaWeights{TextMatch: 1.0}
	red := RedistributeForEmptyQuery(deg)
	if red != deg {
		t.Fatalf("degenerate TextMatch=1 should be a no-op; got %#v", red)
	}
}

// TestRedistribute_Idempotent proves re-applying the redistribution to
// an already-redistributed weight table is a no-op (since TextMatch is
// already 0 after the first call). This is a useful safety property
// for the scorer hot path in case the redistribution is accidentally
// invoked twice.
func TestRedistribute_Idempotent(t *testing.T) {
	t.Parallel()

	once := RedistributeForEmptyQuery(DefaultFreelanceWeights())
	twice := RedistributeForEmptyQuery(once)
	if once != twice {
		t.Fatalf("redistribution not idempotent:\n once  = %#v\n twice = %#v",
			once, twice)
	}
}

// TestIsEmptyQuery covers the whitespace-trimming behaviour so the
// scorer does not treat a query like "   " as meaningful text-match
// input.
func TestIsEmptyQuery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want bool
	}{
		{"", true},
		{" ", true},
		{"\t\n  ", true},
		{"go", false},
		{" go", false},
		{"go ", false},
		{"  développeur React Paris senior  ", false},
	}

	for _, c := range cases {
		c := c
		t.Run(c.in, func(t *testing.T) {
			t.Parallel()
			got := isEmptyQuery(Query{Text: c.in})
			if got != c.want {
				t.Fatalf("isEmptyQuery(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

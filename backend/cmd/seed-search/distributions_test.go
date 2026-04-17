package main

import (
	"math/rand"
	"testing"
	"time"
)

// distributions_test.go exercises the deterministic pure helpers of
// the seed-search generator. Keeps the seed reproducible and locks
// the output ratios in — a regression here would make golden tests
// flake because the rating distribution drifted.

func TestRatingBucket_Distribution(t *testing.T) {
	// Over 10k draws, buckets should fall within ±3 ppts of brief.
	r := rand.New(rand.NewSource(42))
	unrated, high, mid, low := 0, 0, 0, 0
	const N = 10_000
	for i := 0; i < N; i++ {
		avg, count := ratingBucket(r)
		switch {
		case count == 0:
			unrated++
		case avg >= 4.0:
			high++
		case avg >= 3.0:
			mid++
		default:
			low++
		}
	}
	assertInRange(t, "unrated", unrated, 3700, 4300)
	assertInRange(t, "high", high, 3200, 3800)
	assertInRange(t, "mid", mid, 1700, 2300)
	assertInRange(t, "low", low, 300, 700)
}

func TestAvailabilityForRNG_Distribution(t *testing.T) {
	r := rand.New(rand.NewSource(1234))
	now, soon, not := 0, 0, 0
	const N = 10_000
	for i := 0; i < N; i++ {
		switch availabilityForRNG(r) {
		case "now":
			now++
		case "soon":
			soon++
		case "not":
			not++
		}
	}
	assertInRange(t, "now", now, 5700, 6300)
	assertInRange(t, "soon", soon, 2300, 2700)
	assertInRange(t, "not", not, 1300, 1700)
}

func TestPersonaCounts_DefaultSplit(t *testing.T) {
	c := personaCounts{}
	c.resolveDefaults(500)
	if c.freelance != 300 || c.agency != 120 || c.referrer != 80 {
		t.Fatalf("default split: got %+v, want {300,120,80}", c)
	}
	if c.total() != 500 {
		t.Fatalf("total=%d want 500", c.total())
	}
}

func TestPersonaCounts_OverrideKept(t *testing.T) {
	c := personaCounts{freelance: 100, agency: 50, referrer: 25}
	c.resolveDefaults(500)
	if c.freelance != 100 || c.agency != 50 || c.referrer != 25 {
		t.Fatalf("overrides got overwritten: %+v", c)
	}
}

func TestLastActiveAt_WithinLastYear(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	now := time.Now()
	cutoff := now.Add(-366 * 24 * time.Hour)
	for i := 0; i < 1000; i++ {
		got := lastActiveAt(r, now)
		if got.After(now) {
			t.Fatalf("last_active_at in future: %v", got)
		}
		if got.Before(cutoff) {
			t.Fatalf("last_active_at older than ceiling: %v", got)
		}
	}
}

func TestLanguagesForRNG_AlwaysIncludesFrOrEn(t *testing.T) {
	r := rand.New(rand.NewSource(13))
	for i := 0; i < 500; i++ {
		langs := languagesForRNG(r)
		hasFrOrEn := false
		for _, l := range langs {
			if l == "fr" || l == "en" {
				hasFrOrEn = true
				break
			}
		}
		if !hasFrOrEn {
			t.Fatalf("iteration %d returned %v — no fr/en", i, langs)
		}
	}
}

func TestDeterministicUUID_Stable(t *testing.T) {
	a := deterministicUUID("some-label")
	b := deterministicUUID("some-label")
	if a != b {
		t.Fatalf("deterministicUUID not stable: %v vs %v", a, b)
	}
	c := deterministicUUID("different-label")
	if a == c {
		t.Fatalf("deterministicUUID collision between distinct labels")
	}
}

func assertInRange(t *testing.T, name string, got, lo, hi int) {
	t.Helper()
	if got < lo || got > hi {
		t.Fatalf("%s=%d not in [%d,%d]", name, got, lo, hi)
	}
}

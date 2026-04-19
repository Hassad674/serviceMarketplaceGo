package rules

import "strings"

// diversity.go implements §6.5 of docs/ranking-v1.md — the soft
// diversity rule. Goal: prevent 10 consecutive React developers
// dominating the top of the grid. A 2-in-a-row is acceptable; a
// 3-in-a-row triggers a swap if a suitable alternative exists.
//
// The rule is purposely soft: if no alternative can break the run,
// we keep the run rather than forcing a lower-quality candidate
// upward. Diversity is never worth tanking relevance.
//
// Primary skill is the canonicalised first skill listed on the
// profile. It is surfaced on the Candidate struct by the caller so
// this package never has to parse free-text.

// diversityPass walks the top-N window looking for same-primary-skill
// runs of length ≥ 3. When a run is detected at position k, we try
// to swap candidates[k] with the best-scored candidate later in the
// slice whose primary skill differs from candidates[k-1] and
// candidates[k-2].
//
// topN bounds the window so we only break runs that are visible to
// the user. The tail is left untouched.
//
// Mutates the slice in place. Returns the (possibly) re-ordered top
// window length — useful for tests but not required by callers.
func diversityPass(candidates []Candidate, topN int) int {
	limit := topN
	if limit > len(candidates) {
		limit = len(candidates)
	}
	if limit < 3 {
		return limit
	}

	for i := 2; i < limit; i++ {
		if !formsRun(candidates, i) {
			continue
		}
		if swap := findDiversitySwap(candidates, i, limit); swap != -1 {
			candidates[i], candidates[swap] = candidates[swap], candidates[i]
		}
	}
	return limit
}

// formsRun reports whether candidates[i] extends a same-primary-skill
// run started at candidates[i-2] and candidates[i-1]. An empty
// primary skill is treated as "unknown" and never forms a run
// (otherwise two profiles with no skill listed would artificially
// count as matching).
func formsRun(candidates []Candidate, i int) bool {
	if i < 2 {
		return false
	}
	a := canonSkill(candidates[i-2].PrimarySkill)
	b := canonSkill(candidates[i-1].PrimarySkill)
	c := canonSkill(candidates[i].PrimarySkill)
	if a == "" || b == "" || c == "" {
		return false
	}
	return a == b && b == c
}

// findDiversitySwap scans candidates[i+1:limit] for the highest-
// scored candidate whose primary skill differs from both
// candidates[i-1] and candidates[i-2] AND which keeps the same tier
// as the incumbent at position i (we never swap across tiers —
// availability tiers are hard §6.4).
//
// Returns the index of the swap partner, or -1 when no suitable
// candidate is found. The "highest-scored" preference keeps the
// diversity pass as close to relevance-neutral as possible: we
// break runs with the best alternative available, not the first
// one we find.
func findDiversitySwap(candidates []Candidate, i, limit int) int {
	need := canonSkill(candidates[i-1].PrimarySkill)
	need2 := canonSkill(candidates[i-2].PrimarySkill)
	incumbentTier := TierOf(candidates[i].AvailabilityStatus)

	bestIdx := -1
	bestScore := -1.0
	for j := i + 1; j < limit; j++ {
		cand := candidates[j]
		candSkill := canonSkill(cand.PrimarySkill)
		if candSkill == "" || candSkill == need || candSkill == need2 {
			continue
		}
		if TierOf(cand.AvailabilityStatus) != incumbentTier {
			continue
		}
		if cand.Score.Final > bestScore {
			bestScore = cand.Score.Final
			bestIdx = j
		}
	}
	return bestIdx
}

// canonSkill normalises skill strings for equality checks. Case +
// whitespace are the only two normalisation axes — we explicitly do
// NOT collapse semantically-similar skills (React vs Next.js) because
// that is a product decision and deserves its own feature, not a
// buried helper here.
func canonSkill(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

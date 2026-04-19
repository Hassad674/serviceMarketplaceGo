package rules

import "sort"

// rising_talent.go implements §6.3 of docs/ranking-v1.md — the
// "rising talent" slot rule. Every RisingTalentSlotEvery positions
// (default 5) we attempt to replace the incumbent with a newcomer
// that:
//
//   - has AccountAgeDays < RisingTalentMaxAge (default 60)
//   - is verified (KYC)
//   - scored above the candidate-set median on Score.Final
//
// The incumbent is only replaced when the rising candidate's score
// is within RisingTalentDelta of the incumbent (default 5). This
// preserves "close enough" rather than "better than" — the rule
// trades a tiny relevance drop for a curation benefit.

// injectRising scans slots 5, 10, 15, 20 (or the config multiples)
// in the top-N window and swaps the incumbent with the best eligible
// rising candidate when the delta permits. Mutates the slice in
// place.
//
// The function never inserts a rising candidate that is already in
// the top-N window — that would be a no-op swap and would also
// bias the ranking toward whichever rising candidate happened to
// land near the top first.
func injectRising(candidates []Candidate, cfg Config) {
	if cfg.RisingTalentSlotEvery <= 0 {
		return
	}
	limit := cfg.TopN
	if limit > len(candidates) {
		limit = len(candidates)
	}
	if limit < cfg.RisingTalentSlotEvery {
		return
	}

	median := medianFinal(candidates)
	// Pre-compute the set of indices currently occupying the top-N
	// so we can exclude them as swap candidates.
	topSet := make(map[string]struct{}, limit)
	for i := 0; i < limit; i++ {
		topSet[candidates[i].DocumentID] = struct{}{}
	}

	for slot := cfg.RisingTalentSlotEvery; slot <= limit; slot += cfg.RisingTalentSlotEvery {
		slotIdx := slot - 1
		incumbent := candidates[slotIdx]
		if isRisingEligible(incumbent, cfg, median) {
			// Incumbent already IS a rising talent — nothing to do.
			continue
		}
		cutoff := incumbent.Score.Final - cfg.RisingTalentDelta
		swapIdx := findRisingCandidate(candidates, limit, cfg, median, cutoff, topSet)
		if swapIdx == -1 {
			continue
		}
		// Register the rising candidate as now occupying the slot so
		// later iterations don't double-use it. Drop the evicted
		// incumbent's DocumentID from the tracking set so downstream
		// slots could theoretically pick the old incumbent again —
		// unusual but not forbidden.
		delete(topSet, incumbent.DocumentID)
		topSet[candidates[swapIdx].DocumentID] = struct{}{}
		candidates[slotIdx], candidates[swapIdx] = candidates[swapIdx], candidates[slotIdx]
	}
}

// findRisingCandidate returns the index (within candidates[limit:])
// of the highest-scored rising-eligible candidate whose Final score
// is ≥ cutoff. Returns -1 when none exists.
//
// Scanning starts at `limit` because anyone already in the top-N is
// excluded — we want to PROMOTE, not reshuffle.
func findRisingCandidate(
	candidates []Candidate,
	limit int,
	cfg Config,
	median, cutoff float64,
	topSet map[string]struct{},
) int {
	bestIdx := -1
	var bestScore float64
	for j := limit; j < len(candidates); j++ {
		cand := candidates[j]
		if _, ok := topSet[cand.DocumentID]; ok {
			continue
		}
		if !isRisingEligible(cand, cfg, median) {
			continue
		}
		if cand.Score.Final < cutoff {
			continue
		}
		if bestIdx == -1 || cand.Score.Final > bestScore {
			bestIdx = j
			bestScore = cand.Score.Final
		}
	}
	return bestIdx
}

// isRisingEligible applies the three criteria from §6.3. Callers
// pass the cohort median so we don't recompute it per candidate.
func isRisingEligible(c Candidate, cfg Config, median float64) bool {
	if c.AccountAgeDays <= 0 || c.AccountAgeDays >= cfg.RisingTalentMaxAge {
		return false
	}
	if !c.IsVerified {
		return false
	}
	if c.Score.Final < median {
		return false
	}
	return true
}

// medianFinal computes the median of Score.Final across the whole
// candidate slice. Allocates a temp copy because sort.Slice mutates
// and we don't want to disturb the caller's ordering.
func medianFinal(candidates []Candidate) float64 {
	if len(candidates) == 0 {
		return 0
	}
	copied := make([]float64, len(candidates))
	for i, c := range candidates {
		copied[i] = c.Score.Final
	}
	sort.Float64s(copied)
	n := len(copied)
	if n%2 == 1 {
		return copied[n/2]
	}
	return (copied[n/2-1] + copied[n/2]) / 2
}

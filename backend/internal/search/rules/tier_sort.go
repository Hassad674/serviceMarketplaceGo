package rules

import (
	"sort"
	"strings"
)

// tier_sort.go implements §6.4 of docs/ranking-v1.md.
//
// Availability is tiered, not scored. Tier A (available_now or
// available_soon) is rendered first; Tier B (not_available) follows.
// Within each tier the composite score governs ordering. A tier-B
// candidate can NEVER outrank a tier-A candidate regardless of score.

// TierA and TierB are the two partitions surfaced to the caller.
type Tier int

const (
	TierA Tier = iota // available_now | available_soon
	TierB             // not_available or unknown
)

// TierOf inspects the candidate's availability status and returns
// its bucket. Unknown statuses default to TierB — consistent with
// the "never leak unknowns above available pros" principle from
// §6.4.
func TierOf(status string) Tier {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "available_now", "now", "available", "available_soon", "soon":
		return TierA
	default:
		return TierB
	}
}

// splitTiers partitions the candidate slice into two new slices,
// preserving the original ordering inside each tier. Original input
// is never mutated.
func splitTiers(in []Candidate) (tierA, tierB []Candidate) {
	tierA = make([]Candidate, 0, len(in))
	tierB = make([]Candidate, 0, len(in))
	for _, c := range in {
		if TierOf(c.AvailabilityStatus) == TierA {
			tierA = append(tierA, c)
		} else {
			tierB = append(tierB, c)
		}
	}
	return tierA, tierB
}

// sortByFinalDesc sorts candidates in place, highest Final first.
// Stable so equal-scored candidates keep their pre-sort order
// (important for deterministic tests).
func sortByFinalDesc(candidates []Candidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Score.Final > candidates[j].Score.Final
	})
}

// reSortRespectingTiers re-sorts the top window so Tier A always
// precedes Tier B AFTER a mutation (e.g. featured override) that may
// have disturbed the partition. Tie-breaker is Score.Final DESC.
//
// Only the first `topN` entries are sorted — the tail keeps its
// pre-existing order so we don't shuffle candidates that will be
// clipped anyway.
func reSortRespectingTiers(candidates []Candidate, topN int) {
	limit := topN
	if limit > len(candidates) {
		limit = len(candidates)
	}
	window := candidates[:limit]
	sort.SliceStable(window, func(i, j int) bool {
		ti, tj := TierOf(window[i].AvailabilityStatus), TierOf(window[j].AvailabilityStatus)
		if ti != tj {
			return ti < tj // TierA (0) before TierB (1)
		}
		return window[i].Score.Final > window[j].Score.Final
	})
}

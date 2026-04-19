package scorer

import "strings"

// RedistributeForEmptyQuery scales every weight EXCEPT TextMatch by
// 1 / (1 - TextMatch) and zeroes out TextMatch. This keeps the total at
// 1.0 while avoiding arbitrary choices about which feature should
// "inherit" the TextMatch slice when the user hasn't typed any query
// text. See docs/ranking-v1.md §5.2 for the formal definition.
//
// Invariant — the returned PersonaWeights.Sum() equals 1.0 within
// floatTolerance (enforced by the unit test TestRedistribute_PreservesSum).
//
// Edge cases :
//   - TextMatch == 0 → input is returned unchanged; no redistribution
//     is necessary.
//   - TextMatch == 1 → scaling factor is infinite; input is returned
//     unchanged and the empty-query case simply cannot happen with a
//     pathological weight table. This branch is defensive: a well-formed
//     Config validates away from this at startup.
func RedistributeForEmptyQuery(w PersonaWeights) PersonaWeights {
	missing := w.TextMatch
	if missing == 0 {
		return w
	}
	// Defensive: avoid division by zero when TextMatch is the entire
	// weight. Return input untouched; caller's Validate already
	// guarantees the 9 weights sum to 1 so this is a degenerate table.
	if 1-missing <= floatTolerance {
		return w
	}
	scale := 1 / (1 - missing)
	return PersonaWeights{
		TextMatch:      0,
		SkillsOverlap:  w.SkillsOverlap * scale,
		Rating:         w.Rating * scale,
		ProvenWork:     w.ProvenWork * scale,
		ResponseRate:   w.ResponseRate * scale,
		VerifiedMature: w.VerifiedMature * scale,
		Completion:     w.Completion * scale,
		LastActive:     w.LastActive * scale,
		AccountAge:     w.AccountAge * scale,
	}
}

// isEmptyQuery returns true if the query text is the empty string or
// contains only whitespace. Mirrors the retrieval-side "q=*" convention
// used when the user lands on a listing page without typing anything.
// We purposely do not call Typesense's own empty-query detector here —
// this decision is owned by the scorer.
func isEmptyQuery(q Query) bool {
	return strings.TrimSpace(q.Text) == ""
}

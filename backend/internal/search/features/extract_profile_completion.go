package features

// ExtractProfileCompletion maps the indexer-computed profile_completion_score
// (0-100) onto [0, 1], as described in the base of `docs/ranking-v1.md` §3.2-7.
//
// The entropy / junk-text penalty mentioned in the spec lives in the
// anti-gaming pipeline, not here. This extractor is intentionally boring so
// the penalty can layer on top without the two concerns getting tangled.
func ExtractProfileCompletion(doc SearchDocumentLite) float64 {
	score := float64(doc.ProfileCompletionScore) / 100.0
	return clamp01(score)
}

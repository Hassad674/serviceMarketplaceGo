package features

// ExtractTextMatch computes the text-match score described in
// `docs/ranking-v1.md` §3.2-1.
//
//	raw = min(10, typesense_text_match_bucket) / 10
//	text_match_score = raw
//
// The stuffing penalty (§3.2-1 + §7.1) is applied by the anti-gaming pipeline,
// not here — this extractor is intentionally side-effect free + does not know
// about anti-gaming. The Raw field on Features surfaces the original bucket
// so the downstream stuffing rule can decide whether to halve the score.
//
// Empty-query path : when Typesense runs `q=*` the bucket value we hand down
// is 0, which naturally produces a zero text-match. The scorer then
// redistributes the missing text-match weight across the remaining features
// (see §5.2).
func ExtractTextMatch(_ Query, doc SearchDocumentLite, _ Config) (score float64, rawBucket int) {
	rawBucket = doc.TextMatchBucket
	if rawBucket < 0 {
		rawBucket = 0
	}
	if rawBucket > 10 {
		rawBucket = 10
	}
	return float64(rawBucket) / 10.0, rawBucket
}

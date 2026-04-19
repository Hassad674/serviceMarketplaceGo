package features

// ExtractResponseRate passes through the response_rate signal documented in
// `docs/ranking-v1.md` §3.2-5.
//
// Indexer-computed value : fraction of incoming messages answered within 24h
// over the rolling 90-day window. The signal is already in [0, 1] ; this
// extractor defensively clamps + returns it as-is.
//
// Anti-gaming (response-rate cliff detection) is handled by the antigaming
// pipeline, not here.
func ExtractResponseRate(doc SearchDocumentLite) float64 {
	return clamp01(doc.ResponseRate)
}

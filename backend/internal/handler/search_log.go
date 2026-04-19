// search_log.go emits a single structured log line per successful
// `/api/v1/search` request. The line is the operator's primary
// signal for search traffic: one grep against the log stream tells
// you WHO queried WHAT, with WHICH filters, and HOW LONG it took.
//
// The log payload is built from a typed struct so the field list
// cannot drift silently — any new field must be added to SearchLog
// AND appear in the unit test's field-presence assertion. The log
// format is frozen by golden JSON tests in search_log_test.go.
//
// Design notes:
//
//   - Raw query text is truncated at 200 chars to cap log volume.
//     When truncation happens we set "truncated": true so log
//     consumers can filter out artificially-shortened queries.
//   - `user_id` is an empty string for anonymous requests rather
//     than `null` — slog emits empty strings as "" and drops keys
//     with nil, and we want the key to always be present for
//     consistent log-shape parsing.
//   - `cursor_active` is true when the client paginated into a
//     page > 1. First-page loads report false. Paired with
//     `results_count` an operator can detect pagination loops.
package handler

import (
	"log/slog"
)

// searchQueryLogMaxChars caps the logged query length. 200 chars is
// large enough to see real user intent (most search bars are 40-80
// chars) while keeping the log stream compact.
const searchQueryLogMaxChars = 200

// SearchLog is the typed payload written once per search request.
// Every field is required — an empty string or zero value is valid
// but the key must always be emitted so operators can rely on the
// shape.
type SearchLog struct {
	RequestID    string
	UserID       string
	Persona      string
	Query        string
	FilterBy     string
	SortBy       string
	ResultsCount int
	LatencyMs    int
	Hybrid       bool
	CursorActive bool
	Truncated    bool

	// Reranked is true when the Stage 2-5 ranking pipeline ran on
	// this request. False when no pipeline is wired (legacy path)
	// or when Typesense returned zero hits (nothing to rerank).
	Reranked bool

	// RerankDurationMs is the wall-clock time spent in the ranking
	// pipeline in milliseconds. Zero when Reranked = false.
	RerankDurationMs int

	// TopFinalScore is the Final score (0-100) of the top-ranked
	// candidate. Zero when no candidates were returned or the
	// pipeline did not run.
	TopFinalScore float64
}

// LogAttrs assembles the structured-log attribute list in a stable
// order. Factored out of emit so unit tests can pin the shape byte-
// for-byte via a bytes.Buffer handler.
//
// The three trailing ranking-* fields were added in phase 6F. They
// always appear so operators can filter "reranked=false" requests
// separately from legacy queries — every search.query line now
// carries the ranking signal shape regardless of whether the
// pipeline ran.
func (l SearchLog) LogAttrs() []slog.Attr {
	return []slog.Attr{
		slog.String("event", "search.query"),
		slog.String("request_id", l.RequestID),
		slog.String("user_id", l.UserID),
		slog.String("persona", l.Persona),
		slog.String("query", l.Query),
		slog.Bool("truncated", l.Truncated),
		slog.String("filter_by", l.FilterBy),
		slog.String("sort_by", l.SortBy),
		slog.Int("results_count", l.ResultsCount),
		slog.Int("latency_ms", l.LatencyMs),
		slog.Bool("hybrid", l.Hybrid),
		slog.Bool("cursor_active", l.CursorActive),
		slog.Bool("reranked", l.Reranked),
		slog.Int("rerank_duration_ms", l.RerankDurationMs),
		slog.Float64("top_final_score", l.TopFinalScore),
	}
}

// truncateQueryForLog applies the 200-char cap. Returns the possibly
// shortened string plus a bool signalling whether truncation
// occurred — callers copy that flag into SearchLog.Truncated so the
// log reader can filter them out.
//
// We count runes, not bytes, so multi-byte characters (French
// accents, emoji) count as a single character toward the cap.
func truncateQueryForLog(raw string) (string, bool) {
	runes := []rune(raw)
	if len(runes) <= searchQueryLogMaxChars {
		return raw, false
	}
	return string(runes[:searchQueryLogMaxChars]), true
}

// emitSearchLog writes the structured line at INFO. Accepts *slog.Logger
// so tests can inject a buffer-backed logger.
func emitSearchLog(logger *slog.Logger, payload SearchLog) {
	if logger == nil {
		logger = slog.Default()
	}
	attrs := payload.LogAttrs()
	// slog.LogAttrs bypasses the variadic slog.Any wrapping and
	// preserves attribute ordering across handlers.
	logger.LogAttrs(nil, slog.LevelInfo, "search.query", attrs...)
}

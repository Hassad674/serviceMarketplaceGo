// Package postgres slow_query.go — PERF-FINAL-B-04 / P10
//
// Lightweight query instrumentation. Three free helpers wrap the
// hot-path *sql.DB methods (QueryContext / QueryRowContext /
// ExecContext) with a Now()+Sub() pair and a slog emit when the
// duration crosses the configured thresholds.
//
// Thresholds match the CLAUDE.md spec:
//   - WARN  @  50ms
//   - ERROR @ 500ms
//
// Repos opt in by calling these helpers instead of `db.QueryContext`
// directly. Adoption is gradual — already-instrumented repos appear in
// the slow-query log as soon as they hit the threshold; un-migrated
// repos keep their pre-existing behaviour. The wrapper is deliberately
// not a method on a custom DB type so we avoid a 49-file mass
// refactor; the cost of opting in per repo is one import path change
// per call site.
package postgres

import (
	"context"
	"database/sql"
	"log/slog"
	"runtime"
	"strings"
	"time"

	"marketplace-backend/internal/handler/middleware"
)

// Slow query thresholds. Exposed as exported variables (rather than
// constants) so tests can lower them to simulate slow queries without
// having to wait real wall-clock time. Production code never mutates
// these.
var (
	// SlowQueryWarnThreshold is the floor above which a query emits a
	// WARN log line. CLAUDE.md mandates 50ms.
	SlowQueryWarnThreshold = 50 * time.Millisecond
	// SlowQueryErrorThreshold is the floor above which a query emits an
	// ERROR log line (and overrides the WARN level). CLAUDE.md mandates
	// 500ms.
	SlowQueryErrorThreshold = 500 * time.Millisecond
)

// querySnippetMaxBytes caps the inlined SQL fragment in log output so
// large queries (CTEs, multi-line INSERTs) do not flood the log
// pipeline. The cap is a byte count, not a rune count — at this scale
// the difference is rounding error and a byte cap is constant-time.
const querySnippetMaxBytes = 200

// Query is the timed equivalent of `db.QueryContext`. The signature
// mirrors `*sql.DB` exactly so adopting it in a repo is a one-line
// drop-in. The overhead vs. a direct call is a single `time.Now()`
// and a comparison — < 1µs per query on the hot path (verified by
// benchmark in slow_query_test.go).
func Query(ctx context.Context, db *sql.DB, query string, args ...any) (*sql.Rows, error) {
	started := time.Now()
	rows, err := db.QueryContext(ctx, query, args...)
	logSlowQuery(ctx, "Query", query, started, err)
	return rows, err
}

// QueryRow is the timed equivalent of `db.QueryRowContext`. We can't
// observe the err on the row scan from here (sql.Row defers that
// until the caller's Scan), so we only time the query dispatch.
func QueryRow(ctx context.Context, db *sql.DB, query string, args ...any) *sql.Row {
	started := time.Now()
	row := db.QueryRowContext(ctx, query, args...)
	// row dispatch alone is the timing target — Scan happens in the
	// caller's frame and is its own concern.
	logSlowQuery(ctx, "QueryRow", query, started, nil)
	return row
}

// Exec is the timed equivalent of `db.ExecContext`. Same drop-in
// contract as Query: identical signature, the only behaviour change
// is the slog emit on the slow path.
func Exec(ctx context.Context, db *sql.DB, query string, args ...any) (sql.Result, error) {
	started := time.Now()
	result, err := db.ExecContext(ctx, query, args...)
	logSlowQuery(ctx, "Exec", query, started, err)
	return result, err
}

// logSlowQuery emits a structured slog line when the elapsed time
// since started crosses one of the SlowQuery* thresholds. Below the
// WARN threshold the function is a single subtraction + comparison
// (~50ns) — designed to be cheap enough on the hot path that opting
// in everywhere is a sub-microsecond budget.
//
// Fields emitted (matching the brief):
//   - op          : "Query" | "QueryRow" | "Exec"
//   - duration_ms : milliseconds elapsed (rounded to nearest ms)
//   - query       : sanitized first 200 bytes of the SQL (no values)
//   - caller      : file:line of the call site (one frame up the
//                   stack from logSlowQuery's caller — i.e. the repo
//                   method that invoked Query/Exec/QueryRow)
//   - request_id  : the per-request UUID stamped by RequestID
//                   middleware (empty when the call originates outside
//                   an HTTP handler, e.g. background jobs)
//   - err         : error string when the underlying call failed
func logSlowQuery(ctx context.Context, op string, query string, started time.Time, callErr error) {
	elapsed := time.Since(started)
	if elapsed < SlowQueryWarnThreshold {
		return
	}

	level := slog.LevelWarn
	if elapsed >= SlowQueryErrorThreshold {
		level = slog.LevelError
	}

	attrs := []slog.Attr{
		slog.String("op", op),
		slog.Int64("duration_ms", elapsed.Milliseconds()),
		slog.String("query", sanitizeQuery(query)),
		slog.String("caller", callerFrame()),
		slog.String("request_id", middleware.GetRequestID(ctx)),
	}
	if callErr != nil {
		attrs = append(attrs, slog.String("err", callErr.Error()))
	}

	slog.LogAttrs(ctx, level, "slow query", attrs...)
}

// sanitizeQuery collapses whitespace and truncates the query so the
// log line stays bounded and grep-friendly. Parameter values never
// appear here (the helpers use parameterized queries — the args slice
// is intentionally not logged).
func sanitizeQuery(q string) string {
	q = strings.Join(strings.Fields(q), " ")
	if len(q) > querySnippetMaxBytes {
		return q[:querySnippetMaxBytes] + "..."
	}
	return q
}

// callerFrame walks two frames up the stack: skipping logSlowQuery
// itself, then skipping the Query/Exec/QueryRow helper, lands on the
// repo method that initiated the call. Returns "file:line" — empty
// string if the runtime can't recover the frame (extremely rare;
// guarded so a missing frame never panics).
func callerFrame() string {
	// Skip 2 frames: callerFrame + logSlowQuery + Query/Exec/QueryRow.
	const callerSkip = 3
	_, file, line, ok := runtime.Caller(callerSkip)
	if !ok {
		return ""
	}
	// Trim to the last path segment to keep the log line readable
	// without leaking the build host's GOPATH.
	if idx := strings.LastIndex(file, "/"); idx != -1 {
		file = file[idx+1:]
	}
	return file + ":" + itoa(line)
}

// itoa avoids strconv import here — stays under the crypto-style
// "small file, single concern" intent of the package and keeps the
// hot path's import surface minimal.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

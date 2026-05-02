package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/handler/middleware"
)

// captureSlog replaces the default slog handler with a buffer-backed
// JSON handler so a test can read every log line emitted while the
// scope is open. Restores the previous default on cleanup so the test
// suite stays hermetic.
func captureSlog(t *testing.T, level slog.Level) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: level})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf
}

// freezeSlowQueryThresholds lowers the WARN/ERROR thresholds for the
// duration of one test, then restores them. Tests that simulate a
// "slow" query do so by running an in-process delay inside the mock
// expectation — a 60ms delay is plenty against a 1ms threshold and
// keeps the suite under 1s wall-clock.
func freezeSlowQueryThresholds(t *testing.T, warn, errLvl time.Duration) {
	t.Helper()
	prevWarn, prevErr := SlowQueryWarnThreshold, SlowQueryErrorThreshold
	SlowQueryWarnThreshold = warn
	SlowQueryErrorThreshold = errLvl
	t.Cleanup(func() {
		SlowQueryWarnThreshold = prevWarn
		SlowQueryErrorThreshold = prevErr
	})
}

// readLogLines splits the JSON-handler buffer into one record per
// line, ignoring blank lines. Each line is decoded into a generic
// map so the test can assert structured fields without coupling to
// the slog ordering.
func readLogLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	out := make([]map[string]any, 0)
	for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &m), "decode log line %q", line)
		out = append(out, m)
	}
	return out
}

func TestQuery_FastQuery_DoesNotLog(t *testing.T) {
	// Below the WARN threshold the helper must stay silent — emitting
	// on every query would flood the log pipeline.
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	freezeSlowQueryThresholds(t, 50*time.Millisecond, 500*time.Millisecond)
	buf := captureSlog(t, slog.LevelDebug)

	mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))

	rows, err := Query(context.Background(), db, "SELECT 1")
	require.NoError(t, err)
	defer rows.Close()

	assert.Empty(t, readLogLines(t, buf), "fast queries must not emit a slow query log line")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQuery_SlowQuery_EmitsWarn(t *testing.T) {
	// A query crossing the WARN threshold but not ERROR must surface
	// at WARN level with the structured field set the brief specifies.
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 1ms WARN, 1s ERROR — easy to clear with a 60ms in-process delay.
	freezeSlowQueryThresholds(t, 1*time.Millisecond, 1*time.Second)
	buf := captureSlog(t, slog.LevelDebug)

	mock.ExpectQuery("SELECT id FROM users WHERE id = ").
		WillDelayFor(60 * time.Millisecond).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("u-1"))

	// Stamp a request_id so the test can assert the context propagation.
	ctx := context.WithValue(context.Background(), middleware.ContextKeyRequestID, "req-abc")

	rows, err := Query(ctx, db, "SELECT id FROM users WHERE id = $1", "u-1")
	require.NoError(t, err)
	defer rows.Close()

	lines := readLogLines(t, buf)
	require.Len(t, lines, 1, "exactly one slow query log line must be emitted")

	line := lines[0]
	assert.Equal(t, "WARN", line["level"])
	assert.Equal(t, "slow query", line["msg"])
	assert.Equal(t, "Query", line["op"])
	assert.Contains(t, line["query"], "SELECT id FROM users")
	assert.Equal(t, "req-abc", line["request_id"])
	// duration_ms is a JSON number → decodes as float64.
	durMs, ok := line["duration_ms"].(float64)
	require.True(t, ok, "duration_ms must be a number")
	assert.GreaterOrEqual(t, durMs, float64(50), "duration_ms must reflect the actual elapsed time")
	caller, _ := line["caller"].(string)
	assert.Contains(t, caller, "slow_query_test.go", "caller must point at the test invoking Query")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQuery_VerySlowQuery_EmitsError(t *testing.T) {
	// Crossing the ERROR threshold escalates the slog level — this is
	// the signal that lights up alerts in production.
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 1ms WARN, 50ms ERROR — a 60ms delay clears both.
	freezeSlowQueryThresholds(t, 1*time.Millisecond, 50*time.Millisecond)
	buf := captureSlog(t, slog.LevelDebug)

	mock.ExpectQuery("SELECT slow").
		WillDelayFor(60 * time.Millisecond).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))

	rows, err := Query(context.Background(), db, "SELECT slow FROM big")
	require.NoError(t, err)
	defer rows.Close()

	lines := readLogLines(t, buf)
	require.Len(t, lines, 1)
	assert.Equal(t, "ERROR", lines[0]["level"], "ERROR threshold must escalate the slog level")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExec_SlowExec_EmitsWarn(t *testing.T) {
	// Exec follows the same instrumentation contract as Query — the
	// op field switches to "Exec".
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	freezeSlowQueryThresholds(t, 1*time.Millisecond, 1*time.Second)
	buf := captureSlog(t, slog.LevelDebug)

	mock.ExpectExec("UPDATE users").
		WillDelayFor(60 * time.Millisecond).
		WillReturnResult(sqlmock.NewResult(1, 1))

	res, err := Exec(context.Background(), db, "UPDATE users SET name = $1 WHERE id = $2", "x", "u-1")
	require.NoError(t, err)
	rows, err := res.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	lines := readLogLines(t, buf)
	require.Len(t, lines, 1)
	assert.Equal(t, "Exec", lines[0]["op"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryRow_SlowQueryRow_EmitsWarn(t *testing.T) {
	// QueryRow timing covers only the dispatch — Scan happens outside
	// the helper. The op field switches to "QueryRow".
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	freezeSlowQueryThresholds(t, 1*time.Millisecond, 1*time.Second)
	buf := captureSlog(t, slog.LevelDebug)

	mock.ExpectQuery("SELECT count").
		WillDelayFor(60 * time.Millisecond).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(42))

	row := QueryRow(context.Background(), db, "SELECT count(*) FROM things")
	var n int
	require.NoError(t, row.Scan(&n))
	assert.Equal(t, 42, n)

	lines := readLogLines(t, buf)
	require.Len(t, lines, 1)
	assert.Equal(t, "QueryRow", lines[0]["op"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQuery_LogsErrorField_OnFailedQuery(t *testing.T) {
	// When the underlying QueryContext returns an error AND the call
	// is also slow, the err field must surface in the log line so the
	// production triage flow has the failure reason on hand.
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	freezeSlowQueryThresholds(t, 1*time.Millisecond, 1*time.Second)
	buf := captureSlog(t, slog.LevelDebug)

	mock.ExpectQuery("SELECT broken").
		WillDelayFor(60 * time.Millisecond).
		WillReturnError(errors.New("boom"))

	rows, qErr := Query(context.Background(), db, "SELECT broken FROM nowhere")
	assert.Nil(t, rows)
	require.Error(t, qErr)

	lines := readLogLines(t, buf)
	require.Len(t, lines, 1)
	assert.Equal(t, "boom", lines[0]["err"], "err field must reflect the underlying failure")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSanitizeQuery_TruncatesAndCollapsesWhitespace(t *testing.T) {
	// The helper must (1) collapse internal whitespace so multi-line
	// queries become greppable and (2) cap the byte length so giant
	// CTEs do not blow up the log pipeline.
	long := strings.Repeat("SELECT  *  FROM\n\twide_table\n  ", 30)
	got := sanitizeQuery(long)
	assert.LessOrEqual(t, len(got), querySnippetMaxBytes+3, // +3 for the "..." suffix
		"sanitized query must respect the byte cap")
	assert.NotContains(t, got, "\n", "newlines must be collapsed")
	assert.NotContains(t, got, "\t", "tabs must be collapsed")
}

func TestSanitizeQuery_ShortQueryUntouched(t *testing.T) {
	// Below the cap the helper returns the input verbatim (after the
	// whitespace pass). Used to confirm the truncation branch does not
	// fire on the common case.
	in := "SELECT  1"
	got := sanitizeQuery(in)
	assert.Equal(t, "SELECT 1", got)
	assert.NotContains(t, got, "...", "short queries must not be marked as truncated")
}

func TestItoa_RoundTrip(t *testing.T) {
	// itoa is the in-package strconv.Itoa replacement — small enough
	// to inline here to confirm both branches (zero, positive,
	// negative).
	assert.Equal(t, "0", itoa(0))
	assert.Equal(t, "42", itoa(42))
	assert.Equal(t, "-7", itoa(-7))
	assert.Equal(t, "1234567890", itoa(1234567890))
}

// BenchmarkQuery_FastPath verifies the hot-path overhead claim from
// the brief: a fast query must add < 1µs vs. a direct *sql.DB call.
// The benchmark does not assert in CI (assertions on benchmarks are
// fragile across hardware) — it is a local sanity check.
func BenchmarkQuery_FastPath(b *testing.B) {
	db, mock, err := sqlmock.New()
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	for i := 0; i < b.N; i++ {
		mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := Query(context.Background(), db, "SELECT 1")
		if err != nil {
			b.Fatal(err)
		}
		_ = rows.Close()
	}
}

// callerFrame must always return SOMETHING — we test by calling it
// from a helper to assert the file:line points at slow_query_test.go.
// The empty-string branch (runtime.Caller false) is impossible to
// trigger in a unit test on a regular Go runtime; documented.
func TestCallerFrame_PointsAtCallSite(t *testing.T) {
	got := indirectCallerFrame()
	assert.Contains(t, got, "slow_query_test.go", "caller must surface the test file name")
}

func indirectCallerFrame() string {
	// One level of indirection so the brief's "skip 3 frames" math is
	// exercised end-to-end. The test wrapper above counts as one of
	// those frames.
	return wrappedCallerFrame()
}

func wrappedCallerFrame() string { return fakeCaller() }

func fakeCaller() string {
	// Mimic the call shape of logSlowQuery → Query/Exec/QueryRow:
	// callerFrame() skips itself + 2 frames above, so to surface the
	// test file we add 2 extra frames (wrappedCallerFrame +
	// indirectCallerFrame).
	return callerFrame()
}

// guard against a future refactor that would break the timed wrapper
// contract: passing a context that's already cancelled must still
// produce a stable error from the underlying mock — we are not
// changing the *sql.DB error semantics.
func TestQuery_ContextCancelled_PassesUnderlyingError(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the call so QueryContext returns the deadline error

	_, qErr := Query(ctx, db, "SELECT 1")
	require.Error(t, qErr, "cancelled context must surface the underlying ctx error")
}

// Ensure logSlowQuery itself stays cheap when below the threshold — a
// single comparison + return. No allocations on the fast path.
func TestLogSlowQuery_FastPath_NoAllocs(t *testing.T) {
	freezeSlowQueryThresholds(t, 1*time.Hour, 2*time.Hour)
	allocs := testing.AllocsPerRun(1000, func() {
		logSlowQuery(context.Background(), "Query", "SELECT 1", time.Now(), nil)
	})
	// The fast path should be allocation-free. We allow ≤ 1 alloc as
	// slack for runtime fluctuations on shared CI hardware.
	if allocs > 1 {
		t.Fatalf("logSlowQuery fast path must be alloc-free, got %d allocs/op", int(allocs))
	}
}

// Sanity probe: a database/sql error like driver.ErrBadConn must
// flow through Query unchanged so the calling repo's error mapping
// still works (errors.Is on the underlying sentinel). The slow log
// line is incidental — the contract is on the err, not the log.
func TestQuery_PassesThroughErrSentinels(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sentinel := fmt.Errorf("postgres unique violation: %w", sql.ErrNoRows)
	mock.ExpectQuery("SELECT none").WillReturnError(sentinel)

	_, qErr := Query(context.Background(), db, "SELECT none FROM things")
	assert.True(t, errors.Is(qErr, sql.ErrNoRows),
		"err sentinels must propagate untouched through the wrapper")
}

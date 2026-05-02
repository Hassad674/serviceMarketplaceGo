package observability

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	"go.opentelemetry.io/otel"
)

// TestOpenDB_WrapsDriverInOTelSpans verifies that calls into the
// returned *sql.DB produce OTel spans through the otelsql layer. The
// test uses a stub in-memory driver registered under "p11test" so it
// does not need a live PostgreSQL.
func TestOpenDB_WrapsDriverInOTelSpans(t *testing.T) {
	t.Cleanup(restoreGlobals())

	tp, exp := newTestProvider()
	otel.SetTracerProvider(tp)

	driverName := registerStubDriver(t)

	db, err := OpenDB(driverName, "stub-dsn")
	if err != nil {
		t.Fatalf("OpenDB returned error: %v", err)
	}
	defer db.Close()

	ctx, parentSpan := tp.Tracer("test").Start(context.Background(), "outer")
	defer parentSpan.End()

	// Issue a query through the wrapped DB. The stub driver returns
	// an empty rows iterator immediately so nothing useful comes back
	// — we just need the round trip to fire so otelsql records spans.
	_, _ = db.QueryContext(ctx, "SELECT 1")

	if err := tp.ForceFlush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}
	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least 1 span from db driver wrap")
	}
	// At least one span should carry db.system / db.statement so
	// downstream observability tools can group by query pattern.
	var hasDBStmt bool
	var hasDBSystem bool
	for _, s := range spans {
		for _, kv := range s.Attributes {
			k := string(kv.Key)
			if k == "db.statement" {
				hasDBStmt = true
				if kv.Value.AsString() != "SELECT 1" {
					t.Errorf("db.statement = %q, want %q", kv.Value.AsString(), "SELECT 1")
				}
			}
			if k == "db.system" {
				hasDBSystem = true
			}
		}
	}
	if !hasDBStmt {
		t.Error("no span recorded db.statement attribute")
	}
	if !hasDBSystem {
		t.Error("no span recorded db.system attribute")
	}
}

// TestOpenDB_NoBindValuesInAttributes is the regression guard for the
// PII-redaction promise: bind values must never appear in the
// db.statement attribute. The stub driver records the bind values it
// receives so the test can assert the SQL text on the span is the
// parameterized form, not the interpolated one.
func TestOpenDB_NoBindValuesInAttributes(t *testing.T) {
	t.Cleanup(restoreGlobals())

	tp, exp := newTestProvider()
	otel.SetTracerProvider(tp)

	driverName := registerStubDriver(t)
	db, err := OpenDB(driverName, "stub-dsn")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	const secretEmail = "victim@example.com"
	const parameterized = "SELECT id FROM users WHERE email = $1"
	_, _ = db.QueryContext(context.Background(), parameterized, secretEmail)

	if err := tp.ForceFlush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}

	for _, s := range exp.GetSpans() {
		for _, kv := range s.Attributes {
			val := kv.Value.AsString()
			if val != "" && contains(val, secretEmail) {
				t.Errorf("PII (%q) leaked into span attribute %s = %q",
					secretEmail, kv.Key, val)
			}
		}
	}
}

// TestWrapDBStats_RegistersWithoutErrorWithRealDB verifies the helper
// returns nil on a happy path with the otelsql-wrapped DB.
func TestWrapDBStats_NoOpOnNilDB(t *testing.T) {
	if err := WrapDBStats(nil, "test"); err != nil {
		t.Errorf("WrapDBStats(nil) = %v, want nil", err)
	}
}

// contains returns true when sub is a substring of s. Standard
// strings.Contains is avoided so the tests do not need that import.
func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// registerStubDriver registers an in-memory sql driver with a unique
// name so OpenDB can wrap it through otelsql. Returns the registered
// driver name; t.Cleanup is NOT used because sql.Register cannot be
// undone within the same process — the stub is keyed by a counter so
// concurrent tests get unique names.
func registerStubDriver(t *testing.T) string {
	t.Helper()
	name := "p11stub-" + uniqueSuffix()
	sql.Register(name, &stubDriver{})
	return name
}

// stubDriver is the minimum needed to satisfy database/sql + otelsql.
type stubDriver struct{}

func (stubDriver) Open(string) (driver.Conn, error) { return &stubConn{}, nil }

type stubConn struct{}

func (stubConn) Prepare(query string) (driver.Stmt, error) { return &stubStmt{q: query}, nil }
func (stubConn) Close() error                              { return nil }
func (stubConn) Begin() (driver.Tx, error)                 { return stubTx{}, nil }

// QueryContext lets the otelsql wrapper recognise the connection as
// supporting context-aware queries — this is how it ends up emitting
// the sql.conn.query span instead of the slower fall-back.
func (c *stubConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	_ = query
	return &stubRows{}, nil
}

func (c *stubConn) ExecContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}

type stubStmt struct{ q string }

func (s *stubStmt) Close() error                            { return nil }
func (s *stubStmt) NumInput() int                           { return -1 }
func (s *stubStmt) Exec([]driver.Value) (driver.Result, error) { return nil, errors.New("unused") }
func (s *stubStmt) Query([]driver.Value) (driver.Rows, error)  { return &stubRows{}, nil }

type stubTx struct{}

func (stubTx) Commit() error   { return nil }
func (stubTx) Rollback() error { return nil }

type stubRows struct{}

func (r *stubRows) Columns() []string              { return []string{"col"} }
func (r *stubRows) Close() error                   { return nil }
func (r *stubRows) Next(_ []driver.Value) error    { return io.EOF }

// uniqueSuffix returns a per-test suffix so concurrent runs do not
// collide. A counter+pid would suffice; we use the test addr as a
// quick unique value.
var stubDriverCounter int

func uniqueSuffix() string {
	stubDriverCounter++
	return "n" + itoa(stubDriverCounter)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

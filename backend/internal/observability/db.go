package observability

import (
	"database/sql"
	"fmt"

	"github.com/XSAM/otelsql"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// OpenDB opens a *sql.DB through XSAM/otelsql so every Query / Exec /
// Prepare / Tx call is wrapped in an OTel span. When OTel is disabled
// the global tracer is the SDK no-op so the wrap reduces to a thin
// pass-through with no exporter dial and no span recording.
//
// Span attributes set by otelsql:
//   - db.system = "postgresql"
//   - db.statement = the SQL with placeholders ($1, $2, …)
//   - db.operation = SELECT / INSERT / UPDATE / DELETE / …
//
// Span attributes deliberately NOT set:
//   - bind variable values (PII / secrets)
//   - row payloads
//
// Tracking RowsNext events is disabled (one event per fetched row
// would explode trace size on hot list endpoints). Ping spans and
// connection-reset spans are suppressed for the same reason.
func OpenDB(driverName, dsn string) (*sql.DB, error) {
	db, err := otelsql.Open(driverName, dsn,
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
		otelsql.WithSpanOptions(otelsql.SpanOptions{
			// Ping, RowsNext, ConnPrepare, ConnReset, Connector — all
			// off by default in otelsql. Explicit here so a future
			// dependency bump cannot silently turn on the noisy spans
			// without our explicit opt-in.
			Ping:                 false,
			RowsNext:             false,
			OmitConnResetSession: true,
			OmitConnPrepare:      true,
			OmitRows:             true,
			OmitConnectorConnect: true,
			// db.statement is the parameterized SQL ($1, $2, ...) only.
			// Bind values are NEVER recorded by otelsql so no PII
			// leaks. We keep the statement on for query-pattern
			// debugging — the values that would be sensitive (emails,
			// tokens, password hashes) are sent as parameters and stay
			// out of the span.
			DisableQuery: false,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("otelsql open: %w", err)
	}
	return db, nil
}

// WrapDBStats registers the *sql.DB pool stats (open / idle / wait
// counts) with the global meter provider. Callers that have an open
// DB already (e.g. via postgres.NewConnection) use this to ship the
// pool metrics. Returns the registration handle so callers can
// unregister at shutdown if they need to.
//
// The error case (no meter provider configured) is non-fatal — the
// pool stats are nice-to-have, not required, so the caller should log
// + continue rather than fail boot.
func WrapDBStats(db *sql.DB, peerName string) error {
	if db == nil {
		return nil
	}
	attrs := []attribute.KeyValue{semconv.DBSystemPostgreSQL}
	if peerName != "" {
		attrs = append(attrs, attribute.String("peer.service", peerName))
	}
	_, err := otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(attrs...))
	if err != nil {
		return fmt.Errorf("register db stats metrics: %w", err)
	}
	return nil
}

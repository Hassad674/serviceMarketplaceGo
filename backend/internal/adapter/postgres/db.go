package postgres

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"marketplace-backend/internal/observability"
)

// NewConnection opens the application *sql.DB.
//
// The driver is wrapped through observability.OpenDB so every
// Query / Exec / Prepare / Tx call produces an OTel span when
// tracing is enabled. When OTel is disabled the wrap is essentially
// free — the global tracer is the SDK no-op, so the otelsql layer
// resolves to no recording and no allocations.
//
// PII safety: the wrapping records `db.statement` (the parameterized
// SQL with $1, $2, … placeholders) but never the bind values —
// passwords, emails, tokens, etc. always go through parameter slots
// and stay out of spans.
func NewConnection(databaseURL string) (*sql.DB, error) {
	db, err := observability.OpenDB("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

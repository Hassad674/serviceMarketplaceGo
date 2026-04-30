package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

// TxRunner is the postgres implementation of repository.TxRunner.
// It wraps a single *sql.DB pool and turns the
// "open tx → run user fn → commit/rollback" envelope into a one-call
// helper for application services.
//
// Used by the search outbox flow (BUG-05) to wire a profile mutation
// and the matching pending_events row into the same transaction so a
// crash, DB blip, or context cancel between the two writes can never
// leave Postgres and Typesense permanently out of sync.
type TxRunner struct {
	db *sql.DB
}

// NewTxRunner builds a TxRunner from the shared *sql.DB pool. The
// runner does not own the pool — callers continue to use the pool
// directly for any read/write that does not need cross-repository
// atomicity.
func NewTxRunner(db *sql.DB) *TxRunner {
	return &TxRunner{db: db}
}

// RunInTx opens a transaction with the default isolation level,
// invokes fn with the live *sql.Tx, and commits when fn returns nil.
// A non-nil fn error rolls back and is returned verbatim. Begin /
// commit failures are wrapped with operation context so the caller's
// errors.Is on its sentinel domain errors keeps working.
func (r *TxRunner) RunInTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	if fn == nil {
		return fmt.Errorf("tx runner: fn is required")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("tx runner: begin: %w", err)
	}

	// Rollback after commit is a no-op so it is always safe to defer.
	// We deliberately swallow Rollback errors — the caller has already
	// received the meaningful error from fn or Commit, and surfacing
	// rollback failures here would only obscure the real cause.
	defer func() { _ = tx.Rollback() }()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx runner: commit: %w", err)
	}
	return nil
}

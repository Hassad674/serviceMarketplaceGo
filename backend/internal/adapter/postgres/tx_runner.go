package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

// TxRunner is the postgres implementation of repository.TxRunner.
// It turns the "open tx → run user fn → commit/rollback" envelope
// into a one-call helper for application services.
//
// Used by the search outbox flow (BUG-05) to wire a profile mutation
// and the matching pending_events row into the same transaction so a
// crash, DB blip, or context cancel between the two writes can never
// leave Postgres and Typesense permanently out of sync.
//
// The runner can be built from either a single `*sql.DB` (legacy
// callers that don't need the two-pool routing) or a `*RoutedDB`
// (production wiring). When wrapped around a RoutedDB, every
// `BeginTx` is routed by `system.IsSystemActor(ctx)`:
//
//   - tagged ctx -> admin pool (BYPASSRLS)
//   - untagged ctx -> app pool (NOBYPASSRLS) — RLS fires normally
//
// The runner does not own its pools — callers continue to use the
// underlying *sql.DB directly for any read/write that does not need
// cross-repository atomicity.
type TxRunner struct {
	// db is the legacy single-pool handle. Set when the runner is
	// built via NewTxRunner. Mutually exclusive with `routed`.
	db *sql.DB
	// routed is the two-pool handle. Set when the runner is built
	// via NewRoutedTxRunner. Mutually exclusive with `db`.
	routed *RoutedDB
}

// NewTxRunner builds a TxRunner from a single *sql.DB pool. The
// runner does not own the pool. Used by unit tests and any wiring
// path that has not yet been migrated to the two-pool model.
func NewTxRunner(db *sql.DB) *TxRunner {
	return &TxRunner{db: db}
}

// NewRoutedTxRunner builds a TxRunner that routes BeginTx by context
// across the two pools held inside the supplied RoutedDB. The
// production wiring uses this constructor so the RLS-protected write
// paths run on the NOBYPASSRLS pool by default.
func NewRoutedTxRunner(r *RoutedDB) *TxRunner {
	return &TxRunner{routed: r}
}

// beginTx opens a transaction on the appropriate pool. When the
// runner was built from a RoutedDB, the routing is keyed on
// system.IsSystemActor(ctx). When built from a single *sql.DB, the
// pool is used unconditionally — backward compatible with every
// pre-rollout caller.
func (r *TxRunner) beginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if r.routed != nil {
		return r.routed.BeginTx(ctx, opts)
	}
	return r.db.BeginTx(ctx, opts)
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

	tx, err := r.beginTx(ctx, nil)
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

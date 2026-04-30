package repository

import (
	"context"
	"database/sql"
)

// TxRunner is the narrow port that lets an application service
// execute a unit-of-work that spans more than one repository call
// inside the same database transaction. Used by the search outbox
// pattern (BUG-05): the profile mutation MUST commit atomically
// with the pending_events row that triggers the Typesense reindex.
//
// Implementations are expected to wrap the function in a
// BeginTx → fn → Commit / Rollback envelope and surface any
// non-nil fn error as the transaction outcome.
//
// The *sql.Tx leaks into the port layer because a transaction is
// fundamentally a SQL concept — pretending it isn't would force
// either reflection or duplication of every repo method into
// "in-transaction" variants on a generic transport. Instead, the
// few repos that need to participate in a multi-step write expose
// `*Tx` variants of their write methods (e.g. UpdateCoreTx,
// ScheduleTx) that take the *sql.Tx alongside the usual context
// and arguments.
type TxRunner interface {
	// RunInTx opens a transaction on the underlying *sql.DB,
	// invokes fn with a *sql.Tx scoped to that transaction, and
	// commits when fn returns nil. A non-nil fn error rolls back
	// the transaction and is returned verbatim to the caller.
	RunInTx(ctx context.Context, fn func(tx *sql.Tx) error) error
}

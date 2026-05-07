package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"marketplace-backend/internal/system"
)

// RoutedDB is a context-aware wrapper around two `*sql.DB` pools:
//
//   - app    — the NOBYPASSRLS pool. Every user-facing repository call
//              flows through this connection, so the RLS policies
//              installed by migration 125 fire normally.
//   - admin  — the BYPASSRLS pool. System-actor paths (schedulers,
//              webhooks, GDPR purge, search indexer, admin overrides)
//              flow through this connection so they can read/write
//              across tenants without setting `app.current_org_id`.
//
// The wrapper picks the pool by inspecting the request context with
// `system.IsSystemActor(ctx)`:
//
//   - tagged ctx -> admin pool
//   - untagged ctx -> app pool
//
// The shape of the wrapper mirrors the subset of `*sql.DB` actually
// used by the postgres adapters in this package: QueryContext,
// QueryRowContext, ExecContext, BeginTx (the four methods every
// repository touches), plus PingContext / Close for lifecycle
// management. Anything else on `*sql.DB` is intentionally NOT exposed
// — repos that need an unusual pool primitive must reach into the
// underlying pool explicitly via `RoutedDB.AppPool()` /
// `RoutedDB.AdminPool()`.
//
// The wrapper does not own the pools — `wireInfrastructure` builds them
// once and passes them in. Closing the wrapper closes both pools (used
// by the graceful-shutdown path).
type RoutedDB struct {
	app   *sql.DB
	admin *sql.DB
}

// NewRoutedDB constructs a RoutedDB from two existing `*sql.DB` pools.
// Both pools must be non-nil — callers that don't want a separate
// admin pool pass the same `*sql.DB` for both arguments.
//
// The caller retains ownership of the pools' connection limits —
// RoutedDB only reads from them. A typical setup applies the standard
// `SetMaxOpenConns(50) / SetMaxIdleConns(25)` budget to each pool
// independently because the two pools authenticate as different roles
// and Postgres tracks per-role connection caps.
func NewRoutedDB(app, admin *sql.DB) (*RoutedDB, error) {
	if app == nil {
		return nil, fmt.Errorf("routed db: app pool is required")
	}
	if admin == nil {
		return nil, fmt.Errorf("routed db: admin pool is required")
	}
	return &RoutedDB{app: app, admin: admin}, nil
}

// AppPool returns the underlying NOBYPASSRLS pool. Used by the few
// adapters that must explicitly target the user-facing pool regardless
// of the caller's context (e.g. a defensive lookup that establishes the
// tenant context FOR a subsequent tenant tx — see ProposalRepository.
// resolveProposalOrgs for the canonical pattern).
//
// Most code paths should NOT call this helper — they should let
// RoutedDB pick the pool automatically.
func (r *RoutedDB) AppPool() *sql.DB { return r.app }

// AdminPool returns the underlying BYPASSRLS pool. Used by the
// pending-events worker and other infra paths that must skip the
// per-context routing because they own a long-lived background
// connection.
func (r *RoutedDB) AdminPool() *sql.DB { return r.admin }

// pickPool returns the pool RoutedDB should use for the given context.
// Centralised here so every public method shares the same routing
// logic and the unit tests can target it directly.
func (r *RoutedDB) pickPool(ctx context.Context) *sql.DB {
	if system.IsSystemActor(ctx) {
		return r.admin
	}
	return r.app
}

// QueryContext routes a query to the pool matching the context, then
// proxies to the underlying `*sql.DB.QueryContext`. The signature
// matches `*sql.DB` exactly so adapters can swap a `*sql.DB` for a
// `*RoutedDB` without further changes when the caller code uses
// embedded methods.
func (r *RoutedDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return r.pickPool(ctx).QueryContext(ctx, query, args...)
}

// QueryRowContext routes a single-row query to the pool matching the
// context. As with QueryContext, the signature mirrors `*sql.DB` so
// adapters can be wired with either type.
func (r *RoutedDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return r.pickPool(ctx).QueryRowContext(ctx, query, args...)
}

// ExecContext routes an exec (INSERT / UPDATE / DELETE) to the pool
// matching the context.
func (r *RoutedDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return r.pickPool(ctx).ExecContext(ctx, query, args...)
}

// BeginTx opens a transaction on the pool matching the context. The
// returned `*sql.Tx` is bound to that pool for its lifetime — the
// routing happens once at Begin time and no further switching can
// occur.
//
// Pass nil opts to use the default isolation level.
func (r *RoutedDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return r.pickPool(ctx).BeginTx(ctx, opts)
}

// PingContext checks both pools sequentially. The application
// `/ready` probe uses this so a misconfigured admin pool surfaces as
// 503 even when the app pool is healthy.
func (r *RoutedDB) PingContext(ctx context.Context) error {
	if err := r.app.PingContext(ctx); err != nil {
		return fmt.Errorf("routed db: app pool ping: %w", err)
	}
	if err := r.admin.PingContext(ctx); err != nil {
		return fmt.Errorf("routed db: admin pool ping: %w", err)
	}
	return nil
}

// Close releases both pools. Used by the graceful-shutdown path. After
// Close every method becomes invalid and returns the underlying
// `sql.ErrConnDone`.
func (r *RoutedDB) Close() error {
	var first error
	if err := r.app.Close(); err != nil {
		first = err
	}
	if err := r.admin.Close(); err != nil && first == nil {
		first = err
	}
	return first
}

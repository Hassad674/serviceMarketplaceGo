package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// rls.go — tenant-context plumbing for the PostgreSQL RLS policies
// installed by migration 125 (SEC-10).
//
// Every transaction that touches an RLS-protected table MUST set the
// tenant context before its first SELECT/UPDATE/DELETE. The two
// relevant settings are:
//
//   - app.current_org_id  — for the org-scoped tables: conversations,
//     messages, invoice, proposals, proposal_milestones, disputes,
//     payment_records.
//   - app.current_user_id — for the per-user tables: notifications,
//     audit_logs (per-actor). Also used by the conversations policy's
//     escape-hatch branch for solo-provider conversations that have
//     no organization_id.
//
// Both setters use SET LOCAL so the value is scoped to the current
// transaction only — when the tx commits or rolls back, the value is
// discarded. Critically, this means the setting can NEVER leak across
// requests sharing the same backend connection from the pool.
//
// If the application forgets to call these setters, Postgres treats
// the missing setting as NULL (because of the "true" arg in
// current_setting in the policy expressions), the USING expression
// evaluates to NULL, and the row is filtered out — a safe default.

// SetCurrentOrg sets the app.current_org_id session variable for the
// org-scoped RLS policies. Must be called inside an open transaction —
// SET LOCAL is only honored within a transaction.
//
// Pass uuid.Nil to clear the org context (e.g. for a request whose
// authenticated user is a solo provider with no organization). The
// setting in Postgres holds the literal string "00000000-..." in that
// case, which will never match a real org id and so denies access to
// every org-scoped row, falling back to the per-user escape hatches
// where they exist (currently the conversations policy).
func SetCurrentOrg(ctx context.Context, tx *sql.Tx, orgID uuid.UUID) error {
	if tx == nil {
		return fmt.Errorf("rls: tx is required")
	}
	// pq does not allow $1 placeholders for SET parameters because the
	// SET command parses its argument as an SQL identifier or literal
	// at parse time, before placeholder substitution. To stay safe, we
	// validate the input is a UUID (already typed as uuid.UUID at the
	// Go boundary, so no SQL-injection risk) and inline its canonical
	// string form. set_config(name, value, is_local) is the
	// runtime-callable equivalent of SET LOCAL that DOES accept
	// placeholders, so we use it instead.
	_, err := tx.ExecContext(ctx, "SELECT set_config('app.current_org_id', $1, true)", orgID.String())
	if err != nil {
		return fmt.Errorf("rls: set current org: %w", err)
	}
	return nil
}

// SetCurrentUser sets the app.current_user_id session variable for
// the per-user RLS policies (notifications, audit_logs, plus the
// conversations escape-hatch). Same SET LOCAL semantics as
// SetCurrentOrg — must be inside an open transaction.
func SetCurrentUser(ctx context.Context, tx *sql.Tx, userID uuid.UUID) error {
	if tx == nil {
		return fmt.Errorf("rls: tx is required")
	}
	_, err := tx.ExecContext(ctx, "SELECT set_config('app.current_user_id', $1, true)", userID.String())
	if err != nil {
		return fmt.Errorf("rls: set current user: %w", err)
	}
	return nil
}

// SetTenantContext is a convenience wrapper that sets BOTH the org
// and the user context in one call. The common case is a request
// from an authenticated user who is also a member of an org — both
// settings are needed because some policies (conversations) admit
// rows via either path.
//
// Pass uuid.Nil for either argument to skip that setter — useful when
// the user is a solo provider (no org) or when the call is from a
// background job (no user).
func SetTenantContext(ctx context.Context, tx *sql.Tx, orgID, userID uuid.UUID) error {
	if tx == nil {
		return fmt.Errorf("rls: tx is required")
	}
	if orgID != uuid.Nil {
		if err := SetCurrentOrg(ctx, tx, orgID); err != nil {
			return err
		}
	}
	if userID != uuid.Nil {
		if err := SetCurrentUser(ctx, tx, userID); err != nil {
			return err
		}
	}
	return nil
}

// RunInTxWithTenant is a tenant-aware variant of TxRunner.RunInTx.
// It opens a transaction, calls SetTenantContext with the supplied
// org/user ids, then invokes fn. The tenant context lives only for
// the lifetime of the transaction.
//
// This is the recommended entry point for any repository operation
// that touches an RLS-protected table. The non-tenant TxRunner.RunInTx
// remains for transactions that exclusively touch RLS-free tables
// (e.g. pending_events, search_queries, organizations).
func (r *TxRunner) RunInTxWithTenant(ctx context.Context, orgID, userID uuid.UUID, fn func(tx *sql.Tx) error) error {
	if fn == nil {
		return fmt.Errorf("tx runner: fn is required")
	}
	return r.RunInTx(ctx, func(tx *sql.Tx) error {
		if err := SetTenantContext(ctx, tx, orgID, userID); err != nil {
			return err
		}
		return fn(tx)
	})
}

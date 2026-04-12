package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/audit"
)

// AuditRepository persists audit log entries to the append-only
// audit_logs table. The interface is intentionally minimal: two
// methods for listing by dimension, and one for appending. Update
// and Delete do not exist by design — audit rows are immutable, and
// the database role used by the application should not have the
// privileges to modify them even if someone tried.
//
// Log is NEVER allowed to fail the caller's main operation. The app
// layer wraps every call in a goroutine or a deferred best-effort
// pattern so a broken DB connection does not block a successful
// business transaction. Audit completeness matters, but availability
// of the core flows matters more.
type AuditRepository interface {
	// Log appends a new entry. Returns an error on DB failures; the
	// caller is expected to log the error and continue rather than
	// surface it to the end user.
	Log(ctx context.Context, entry *audit.Entry) error

	// ListByResource returns the audit trail for a given resource,
	// cursor-paginated newest-first. Used by admin tooling and by
	// the role-permissions page to show the Owner a "recent changes"
	// preview before they save more edits.
	ListByResource(
		ctx context.Context,
		resourceType audit.ResourceType,
		resourceID uuid.UUID,
		cursor string,
		limit int,
	) ([]*audit.Entry, string, error)

	// ListByUser returns every audit row attributable to a user,
	// cursor-paginated newest-first. Used by admin investigations
	// when a compromised account needs to be traced through the
	// full history.
	ListByUser(
		ctx context.Context,
		userID uuid.UUID,
		cursor string,
		limit int,
	) ([]*audit.Entry, string, error)
}

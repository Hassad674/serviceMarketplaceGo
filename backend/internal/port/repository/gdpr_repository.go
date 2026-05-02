package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/gdpr"
)

// GDPRRepository persists and reads the data backing the right-to-erasure
// + right-to-export endpoints (P5).
//
// Three responsibilities, kept on a single interface because the
// methods consistently traverse multiple tables in service of the same
// flow — exporting a user, soft-deleting a user, hard-purging at T+30.
// Splitting them would force the service layer to carry three repos
// for very little segregation gain.
//
// Implementations MUST never mutate audit_logs other than via the
// AnonymizeAuditLogsForUser path: the audit table is append-only by
// policy and only this anonymization UPDATE is allowed (and only after
// a hard-purge).
type GDPRRepository interface {
	// LoadExport gathers every JSON section the export ZIP contains
	// for the given user. Returned slices are JSON-friendly maps so
	// the service can stream them to a ZIP without any further
	// transformation. An empty slice is acceptable for any section
	// other than Profile, which MUST contain at least one row.
	LoadExport(ctx context.Context, userID uuid.UUID) (*gdpr.Export, error)

	// SoftDelete marks the user as scheduled for deletion. Idempotent:
	// if deleted_at is already set, the call is a no-op and returns
	// the existing timestamp.
	SoftDelete(ctx context.Context, userID uuid.UUID, at time.Time) (time.Time, error)

	// CancelDeletion atomically clears deleted_at on the row and
	// returns whether a cancel actually happened (true when a row
	// transitioned from soft-deleted to active, false when the user
	// had no pending deletion).
	//
	// The implementation MUST use a single UPDATE ... WHERE
	// deleted_at IS NOT NULL so a concurrent purge cron tx that
	// already locked the row sees the cancel through SKIP LOCKED.
	CancelDeletion(ctx context.Context, userID uuid.UUID) (bool, error)

	// FindOwnedOrgsBlockingDeletion returns the orgs the user owns
	// that have at least one OTHER active member. Empty slice means
	// the user is free to be deleted. Each BlockedOrg includes a
	// short admin list so the frontend can suggest a transfer.
	FindOwnedOrgsBlockingDeletion(ctx context.Context, userID uuid.UUID) ([]gdpr.BlockedOrg, error)

	// ListPurgeable returns up to `limit` users whose deleted_at is
	// older than the cooldown window. The cron uses this to feed
	// PurgeUser one row at a time.
	//
	// SKIP LOCKED ensures concurrent worker instances never pick the
	// same row twice and a concurrent CancelDeletion is honored.
	ListPurgeable(ctx context.Context, before time.Time, limit int) ([]uuid.UUID, error)

	// PurgeUser hard-deletes the user, cascades through
	// org-shaped relationships per the migrations, and
	// anonymizes the user's rows in audit_logs.
	//
	// The whole operation MUST run inside a single tx with a
	// row-level FOR UPDATE SKIP LOCKED on users so a concurrent
	// CancelDeletion that landed between ListPurgeable and PurgeUser
	// is honored — implementations re-check deleted_at IS NOT NULL
	// AND deleted_at < before before issuing the DELETE.
	//
	// Returns ok=true when the row was actually purged, false when
	// the cancel won the race.
	PurgeUser(ctx context.Context, userID uuid.UUID, before time.Time, salt string) (bool, error)
}

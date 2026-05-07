package repository

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/receipt"
)

// ReceiptRepository is the read port the receipt feature uses. The
// adapter is responsible for both the SQL-level filter ("the caller is
// a party on the row") AND the deserialisation of the JSONB
// billing_snapshot column into the typed PartyBilling values.
//
// Errors:
//   - ErrNotFound (domain) when no row matches the id at all.
//   - ErrForbidden (domain) when the row exists but the caller's org
//     is not client / provider / referrer on it.
//
// The adapter MUST NOT collapse these two errors — the audit layer
// uses the difference to surface "row read by non-party" attempts as a
// stronger signal than plain 404s.
type ReceiptRepository interface {
	// ListForOrganization returns receipts where the caller's org is
	// either the client, the provider, or the referrer. Cursor-based
	// pagination, ordered by created_at DESC.
	ListForOrganization(
		ctx context.Context,
		orgID uuid.UUID,
		cursor string,
		limit int,
	) (rows []*receipt.Receipt, nextCursor string, err error)

	// GetForOrganization returns one receipt by id. Returns
	// receipt.ErrNotFound if the id does not exist; receipt.ErrForbidden
	// if it exists but the caller's org is not a party on it.
	GetForOrganization(
		ctx context.Context,
		receiptID uuid.UUID,
		orgID uuid.UUID,
	) (*receipt.Receipt, error)
}

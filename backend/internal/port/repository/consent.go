package repository

import (
	"context"

	"marketplace-backend/internal/domain/consent"
)

// ConsentLogRepository persists consent_log rows. Append-only by
// design — the table is the legal proof of consent so updates and
// deletes are not part of the contract.
type ConsentLogRepository interface {
	Create(ctx context.Context, entry *consent.Entry) error
}

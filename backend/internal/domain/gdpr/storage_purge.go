// Package gdpr contains GDPR-related domain entities, including the storage
// purge manifest used to track right-to-erasure object deletions in R2/MinIO.
package gdpr

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNoUserID indicates that a manifest is missing its user_id, which is
// required to associate the manifest with a deletion request.
var ErrNoUserID = errors.New("storage purge manifest: user_id is required")

// StoragePurgeManifest captures the result of attempting to delete every R2
// object key belonging to a user during account anonymization. It is persisted
// as compliance evidence: regulators may request proof that personal media
// (avatars, KYC scans, message attachments, portfolio assets, video pitches)
// was erased alongside the database anonymization.
type StoragePurgeManifest struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	OrganizationID *uuid.UUID
	Keys           []string // every key the scheduler attempted to delete
	PurgedKeys     []string // keys confirmed deleted by the storage provider
	FailedKeys     []string // keys that failed (transient or permanent errors)
	Errors         []string // human-readable error messages, aligned with FailedKeys
	PurgedAt       time.Time
}

// Validate enforces the minimum invariants required to persist a manifest.
// Returns ErrNoUserID when UserID is the zero UUID.
func (m StoragePurgeManifest) Validate() error {
	if m.UserID == uuid.Nil {
		return ErrNoUserID
	}
	return nil
}

// KeysCount returns the number of keys the manifest attempted to delete.
// Used by adapters to populate the keys_count column.
func (m StoragePurgeManifest) KeysCount() int {
	return len(m.Keys)
}

// HasFailures reports whether at least one key failed to delete.
func (m StoragePurgeManifest) HasFailures() bool {
	return len(m.FailedKeys) > 0
}

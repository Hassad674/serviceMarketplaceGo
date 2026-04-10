package organization

import (
	"time"

	"github.com/google/uuid"
)

// OrgType identifies the nature of the organization. Mirrors the user's
// marketplace role (agency or enterprise). Providers are solo and never
// have an organization.
type OrgType string

const (
	OrgTypeAgency     OrgType = "agency"
	OrgTypeEnterprise OrgType = "enterprise"
)

// IsValid reports whether the org type is a known value.
func (t OrgType) IsValid() bool {
	return t == OrgTypeAgency || t == OrgTypeEnterprise
}

// String implements fmt.Stringer.
func (t OrgType) String() string {
	return string(t)
}

// Organization represents the business entity (Acme Corp) as a first-class
// concept, distinct from the founder's user account.
//
// In V1, exactly one user holds the Owner role per organization. The
// OwnerUserID field is a denormalized cache of the current Owner for fast
// lookups (e.g. at JWT issuance); the source of truth is the row with
// role='owner' in organization_members, enforced unique by a partial index.
//
// The PendingTransfer* fields form a single consistent group: either all
// three are set (a transfer is in flight) or all three are nil. The DB
// CHECK constraint in migration 053 mirrors this invariant.
type Organization struct {
	ID          uuid.UUID
	OwnerUserID uuid.UUID
	Type        OrgType

	PendingTransferToUserID    *uuid.UUID
	PendingTransferInitiatedAt *time.Time
	PendingTransferExpiresAt   *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewOrganization creates a fresh organization for the given owner.
// The caller is responsible for persisting it AND for creating the
// matching organization_members row with role='owner' inside the same
// transaction, so the single-Owner invariant is maintained at all times.
//
// Providers cannot own an organization — the app layer must verify the
// user's marketplace role before calling this constructor. The check
// isn't duplicated here because it would require importing the user
// package, which would violate the domain isolation rule.
func NewOrganization(ownerUserID uuid.UUID, orgType OrgType) (*Organization, error) {
	if ownerUserID == uuid.Nil {
		return nil, ErrNameRequired // misuse — owner must exist
	}
	if !orgType.IsValid() {
		return nil, ErrInvalidOrgType
	}

	now := time.Now()
	return &Organization{
		ID:          uuid.New(),
		OwnerUserID: ownerUserID,
		Type:        orgType,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// IsTransferPending reports whether an ownership transfer has been
// initiated and is waiting for the target's acceptance.
func (o *Organization) IsTransferPending() bool {
	return o.PendingTransferToUserID != nil
}

// IsTransferExpired reports whether the pending transfer, if any, has
// passed its expiration window. Returns false if no transfer is pending.
func (o *Organization) IsTransferExpired() bool {
	if o.PendingTransferExpiresAt == nil {
		return false
	}
	return time.Now().After(*o.PendingTransferExpiresAt)
}

// InitiateTransfer records a pending ownership transfer targeting the
// given user. The target must be a valid member of the organization and
// must hold the Admin role — that check lives in the app layer because
// it requires a repository lookup.
//
// The transfer is valid for the provided duration (typically 7 days).
// Only one transfer may be pending at a time: if another is already in
// flight, this returns ErrTransferAlreadyPending.
//
// Self-transfer (owner transferring to themselves) is rejected.
func (o *Organization) InitiateTransfer(targetUserID uuid.UUID, duration time.Duration) error {
	if o.IsTransferPending() {
		return ErrTransferAlreadyPending
	}
	if targetUserID == uuid.Nil {
		return ErrTransferTargetInvalid
	}
	if targetUserID == o.OwnerUserID {
		return ErrCannotTransferToSelf
	}

	now := time.Now()
	expires := now.Add(duration)
	o.PendingTransferToUserID = &targetUserID
	o.PendingTransferInitiatedAt = &now
	o.PendingTransferExpiresAt = &expires
	o.UpdatedAt = now
	return nil
}

// CancelTransfer clears a pending transfer — either because the initiator
// withdrew it, or because the target declined. Safe to call when no
// transfer is pending (no-op), but the caller normally checks first so
// they can surface a "nothing to cancel" error to the user.
func (o *Organization) CancelTransfer() {
	if !o.IsTransferPending() {
		return
	}
	o.PendingTransferToUserID = nil
	o.PendingTransferInitiatedAt = nil
	o.PendingTransferExpiresAt = nil
	o.UpdatedAt = time.Now()
}

// CompleteTransfer finalizes the ownership change. It updates the cached
// OwnerUserID and clears the pending transfer state. The caller is
// responsible for also updating the organization_members rows
// (the old Owner becomes Admin, the new one becomes Owner) in the same
// transaction — this method only handles the organization's own fields.
//
// Returns ErrNoPendingTransfer if nothing is in flight, or
// ErrTransferExpired if the pending transfer has gone stale.
func (o *Organization) CompleteTransfer(accepterUserID uuid.UUID) error {
	if !o.IsTransferPending() {
		return ErrNoPendingTransfer
	}
	if o.IsTransferExpired() {
		return ErrTransferExpired
	}
	if *o.PendingTransferToUserID != accepterUserID {
		return ErrTransferTargetInvalid
	}

	o.OwnerUserID = accepterUserID
	o.PendingTransferToUserID = nil
	o.PendingTransferInitiatedAt = nil
	o.PendingTransferExpiresAt = nil
	o.UpdatedAt = time.Now()
	return nil
}

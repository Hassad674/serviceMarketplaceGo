package organization

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// OrgType identifies the nature of the organization. Three values exist
// post phase R1: agency and enterprise are created by a self-registering
// founder, and provider_personal is the auto-created org for every solo
// user (providers, admins) so that invited operators can join them under
// the same Stripe Dashboard semantics as companies.
type OrgType string

const (
	OrgTypeAgency           OrgType = "agency"
	OrgTypeEnterprise       OrgType = "enterprise"
	OrgTypeProviderPersonal OrgType = "provider_personal"
)

// IsValid reports whether the org type is a known value.
func (t OrgType) IsValid() bool {
	switch t {
	case OrgTypeAgency, OrgTypeEnterprise, OrgTypeProviderPersonal:
		return true
	}
	return false
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
	Name        string

	// Stripe Connect (moved from users in phase R5). The org is the
	// merchant of record: transfers, payouts and the KYC state all
	// live here so every operator of the team sees the same Stripe
	// Dashboard.
	StripeAccountID      *string
	StripeAccountCountry *string
	StripeLastState      []byte // jsonb raw — opaque at the domain level

	// KYC enforcement bookkeeping (migration 044 semantics, now
	// org-scoped).
	KYCFirstEarningAt        *time.Time
	KYCRestrictionNotifiedAt map[string]time.Time // tier → notified timestamp

	// AutoPayoutEnabledAt records when the org first successfully
	// completed a manual payout via /wallet/payout. From that point on
	// (provided KYC + billing are still complete), released milestones
	// auto-transfer instead of waiting on another explicit click. NULL
	// means consent has not been given yet — every transfer stays in
	// TransferPending until the provider clicks "Retirer" themselves.
	AutoPayoutEnabledAt *time.Time

	PendingTransferToUserID    *uuid.UUID
	PendingTransferInitiatedAt *time.Time
	PendingTransferExpiresAt   *time.Time

	// RoleOverrides is the per-organization customization of role
	// permissions on top of the static rolePermissions defaults. Owner
	// is never present here (Owner always has everything). Nil / empty
	// means "no customization — use defaults".
	//
	// Persisted as a JSONB column on the organizations table. See
	// migration 077_org_role_overrides.
	RoleOverrides RoleOverrides

	CreatedAt time.Time
	UpdatedAt time.Time
}

// SetRoleOverride sets (or clears) a single permission override for a
// role on this organization. Enforces the domain-level invariants that
// protect the permission model:
//
//   - The Owner role is never customized.
//   - Non-overridable permissions are never customized.
//   - Unknown roles and unknown permissions are rejected.
//
// To revert a permission to its default, pass granted=true and later
// remove the key via ClearRoleOverride — this method always stores an
// explicit grant/revoke.
//
// Callers that want to apply several overrides at once should batch
// them into a RoleOverrides map and pass it through ReplaceRoleOverrides
// so persistence happens in one write.
func (o *Organization) SetRoleOverride(role Role, perm Permission, granted bool) error {
	if role == RoleOwner {
		return ErrCannotOverrideOwner
	}
	if !role.IsValid() {
		return ErrInvalidRole
	}
	if _, known := permissionMetadataByKey[perm]; !known {
		return ErrUnknownPermission
	}
	if nonOverridablePermissions[perm] {
		return ErrPermissionNotOverridable
	}
	if o.RoleOverrides == nil {
		o.RoleOverrides = RoleOverrides{}
	}
	inner, ok := o.RoleOverrides[role]
	if !ok {
		inner = map[Permission]bool{}
		o.RoleOverrides[role] = inner
	}
	inner[perm] = granted
	o.UpdatedAt = time.Now()
	return nil
}

// ClearRoleOverride removes a single override from the organization,
// reverting that (role, perm) cell to its static default.
func (o *Organization) ClearRoleOverride(role Role, perm Permission) error {
	if role == RoleOwner {
		return ErrCannotOverrideOwner
	}
	if o.RoleOverrides == nil {
		return nil
	}
	inner, ok := o.RoleOverrides[role]
	if !ok {
		return nil
	}
	delete(inner, perm)
	if len(inner) == 0 {
		delete(o.RoleOverrides, role)
	}
	o.UpdatedAt = time.Now()
	return nil
}

// ReplaceRoleOverrides atomically replaces the overrides for a single
// role. Used by the role-permissions editor when the Owner saves the
// full matrix for one role at once — any cells the Owner did not
// override are removed (reverted to defaults) automatically.
//
// Validates the payload via ValidateRoleOverrides on a temporary map
// containing just the target role, so the existing validation logic
// catches any illegal combinations.
func (o *Organization) ReplaceRoleOverrides(role Role, perms map[Permission]bool) error {
	if role == RoleOwner {
		return ErrCannotOverrideOwner
	}
	if !role.IsValid() {
		return ErrInvalidRole
	}
	// Build a scratch RoleOverrides so ValidateRoleOverrides can check
	// every key in one pass.
	clone := make(map[Permission]bool, len(perms))
	for p, v := range perms {
		clone[p] = v
	}
	if err := ValidateRoleOverrides(RoleOverrides{role: clone}); err != nil {
		return err
	}
	if o.RoleOverrides == nil {
		o.RoleOverrides = RoleOverrides{}
	}
	if len(clone) == 0 {
		delete(o.RoleOverrides, role)
	} else {
		o.RoleOverrides[role] = clone
	}
	o.UpdatedAt = time.Now()
	return nil
}

// HasKYCCompleted returns true when a Stripe account exists for the org.
func (o *Organization) HasKYCCompleted() bool {
	return o.StripeAccountID != nil && *o.StripeAccountID != ""
}

// IsKYCBlocked returns true if the org has earned available funds, has
// NOT completed KYC, and 14 days have elapsed since the first earning.
// Mirrors the 14-day deadline enforced by the KYC scheduler.
func (o *Organization) IsKYCBlocked() bool {
	if o.HasKYCCompleted() {
		return false
	}
	if o.KYCFirstEarningAt == nil {
		return false
	}
	return time.Since(*o.KYCFirstEarningAt) >= 14*24*time.Hour
}

// HasAutoPayoutConsent reports whether the org has previously completed
// a manual payout, which we treat as the consent + the proof that
// Stripe payouts work for them. Subsequent milestone releases auto-
// transfer when this is true AND KYC + billing remain complete.
func (o *Organization) HasAutoPayoutConsent() bool {
	return o.AutoPayoutEnabledAt != nil
}

// MarkAutoPayoutEnabled stamps the consent timestamp the first time it
// is set. Subsequent calls are no-ops so we keep the original moment
// for audit. The caller is responsible for persisting the change.
func (o *Organization) MarkAutoPayoutEnabled(at time.Time) {
	if o.AutoPayoutEnabledAt != nil {
		return
	}
	o.AutoPayoutEnabledAt = &at
	o.UpdatedAt = time.Now()
}

// KYCDaysRemaining returns the number of days before restriction kicks
// in. -1 when not applicable (no earnings or KYC done), 0 when already
// restricted.
func (o *Organization) KYCDaysRemaining() int {
	if o.HasKYCCompleted() || o.KYCFirstEarningAt == nil {
		return -1
	}
	remaining := 14*24*time.Hour - time.Since(*o.KYCFirstEarningAt)
	if remaining <= 0 {
		return 0
	}
	return int(remaining.Hours() / 24)
}

// NewOrganization creates a fresh organization for the given owner.
// The caller is responsible for persisting it AND for creating the
// matching organization_members row with role='owner' inside the same
// transaction, so the single-Owner invariant is maintained at all times.
//
// Every user — Agency founder, Enterprise founder, Provider, admin — gets
// exactly one organization they own. Providers and admins receive a
// provider_personal org so that team invitations (operators) work the
// same way across all marketplace roles.
func NewOrganization(ownerUserID uuid.UUID, orgType OrgType, name string) (*Organization, error) {
	if ownerUserID == uuid.Nil {
		return nil, ErrNameRequired // misuse — owner must exist
	}
	if !orgType.IsValid() {
		return nil, ErrInvalidOrgType
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrNameRequired
	}

	now := time.Now()
	return &Organization{
		ID:          uuid.New(),
		OwnerUserID: ownerUserID,
		Type:        orgType,
		Name:        name,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Rename changes the organization's display name. Used by the team
// owner to turn the auto-generated personal name into a company name.
// Returns ErrNameRequired when the new name is blank.
func (o *Organization) Rename(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNameRequired
	}
	o.Name = name
	o.UpdatedAt = time.Now()
	return nil
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

package organization

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
)

// TransferTimeout is how long a pending ownership transfer stays valid
// before the target must act on it. Same 7-day window we use for
// invitations for consistency.
const TransferTimeout = 7 * 24 * time.Hour

// InitiateTransferOwnership starts the 2-step transfer flow. The actor
// must be the current Owner. The target must be an existing Admin of
// the org (a guardrail to prevent transferring to a cold account that
// may never claim the role).
//
// The org's pending_transfer_* fields are set and persisted. The target
// then has TransferTimeout to accept or decline. While pending:
//   - The actor can cancel the transfer
//   - The target can accept or decline
//   - Any other role changes on the target are blocked (they'd make
//     the transfer meaningless)
func (s *MembershipService) InitiateTransferOwnership(
	ctx context.Context,
	actorID, orgID, targetUserID uuid.UUID,
) (*organization.Organization, error) {
	if actorID == targetUserID {
		return nil, organization.ErrCannotTransferToSelf
	}

	actor, err := s.members.FindByOrgAndUser(ctx, orgID, actorID)
	if err != nil {
		return nil, mapNotMember(err)
	}
	if actor.Role != organization.RoleOwner {
		return nil, organization.ErrPermissionDenied
	}

	target, err := s.members.FindByOrgAndUser(ctx, orgID, targetUserID)
	if err != nil {
		return nil, organization.ErrTransferTargetInvalid
	}
	if target.Role != organization.RoleAdmin {
		// Transfer target must already be an Admin. This protects the
		// Owner from accidentally transferring to a Member or Viewer
		// who hasn't been vetted for the role.
		return nil, organization.ErrTransferTargetInvalid
	}

	org, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if err := org.InitiateTransfer(targetUserID, TransferTimeout); err != nil {
		return nil, err
	}
	if err := s.orgs.Update(ctx, org); err != nil {
		return nil, fmt.Errorf("initiate transfer: persist org: %w", err)
	}
	return org, nil
}

// CancelTransferOwnership clears a pending transfer on the org. Can be
// called by the current Owner (to retract) at any time before the
// target accepts. Idempotent: returns ErrNoPendingTransfer when nothing
// is pending so the caller can surface a clean UI message.
func (s *MembershipService) CancelTransferOwnership(
	ctx context.Context,
	actorID, orgID uuid.UUID,
) error {
	actor, err := s.members.FindByOrgAndUser(ctx, orgID, actorID)
	if err != nil {
		return mapNotMember(err)
	}
	if actor.Role != organization.RoleOwner {
		return organization.ErrPermissionDenied
	}

	org, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return err
	}
	if !org.IsTransferPending() {
		return organization.ErrNoPendingTransfer
	}

	org.CancelTransfer()
	if err := s.orgs.Update(ctx, org); err != nil {
		return fmt.Errorf("cancel transfer: persist org: %w", err)
	}
	return nil
}

// DeclineTransferOwnership is called by the PROPOSED new owner to
// refuse the transfer. Clears the pending_transfer_* fields on the org.
// The membership rows stay unchanged.
func (s *MembershipService) DeclineTransferOwnership(
	ctx context.Context,
	userID, orgID uuid.UUID,
) error {
	org, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return err
	}
	if !org.IsTransferPending() {
		return organization.ErrNoPendingTransfer
	}
	if org.PendingTransferToUserID == nil || *org.PendingTransferToUserID != userID {
		// Only the proposed target can decline.
		return organization.ErrPermissionDenied
	}

	org.CancelTransfer()
	if err := s.orgs.Update(ctx, org); err != nil {
		return fmt.Errorf("decline transfer: persist org: %w", err)
	}
	return nil
}

// AcceptTransferOwnership finalizes the transfer. Called by the
// PROPOSED new owner. Three things happen, ideally atomically:
//   1. The old Owner is demoted to Admin
//   2. The new Owner (caller) is promoted to Owner
//   3. The org's owner_user_id is updated and pending_transfer_* cleared
//
// Both users have their session_version bumped so any live token
// reflecting the old roles is invalidated on the next request.
//
// Note on atomicity: in V1 we perform these updates sequentially via
// the repository interface. A Tx-aware method on the org repository
// would make this fully atomic — deferred to a follow-up if we ever
// hit a race condition in practice.
func (s *MembershipService) AcceptTransferOwnership(
	ctx context.Context,
	userID, orgID uuid.UUID,
) (*organization.Organization, error) {
	org, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if !org.IsTransferPending() {
		return nil, organization.ErrNoPendingTransfer
	}
	if org.IsTransferExpired() {
		// Clear the stale transfer so subsequent reads stay clean.
		org.CancelTransfer()
		_ = s.orgs.Update(ctx, org)
		return nil, organization.ErrTransferExpired
	}
	if org.PendingTransferToUserID == nil || *org.PendingTransferToUserID != userID {
		return nil, organization.ErrPermissionDenied
	}

	oldOwnerID := org.OwnerUserID

	// Find both membership rows
	oldOwnerMember, err := s.members.FindByOrgAndUser(ctx, orgID, oldOwnerID)
	if err != nil {
		return nil, fmt.Errorf("accept transfer: find old owner: %w", err)
	}
	newOwnerMember, err := s.members.FindByOrgAndUser(ctx, orgID, userID)
	if err != nil {
		return nil, fmt.Errorf("accept transfer: find new owner: %w", err)
	}

	// Step 1: demote the old Owner to Admin FIRST so the partial unique
	// index idx_org_members_unique_owner doesn't fire when we promote
	// the new one.
	if err := oldOwnerMember.ChangeRole(organization.RoleAdmin); err != nil {
		return nil, err
	}
	if err := s.members.Update(ctx, oldOwnerMember); err != nil {
		return nil, fmt.Errorf("accept transfer: demote old owner: %w", err)
	}

	// Step 2: promote the new Owner
	if err := newOwnerMember.ChangeRole(organization.RoleOwner); err != nil {
		// Best-effort rollback of step 1 on failure
		_ = oldOwnerMember.ChangeRole(organization.RoleOwner)
		_ = s.members.Update(ctx, oldOwnerMember)
		return nil, err
	}
	if err := s.members.Update(ctx, newOwnerMember); err != nil {
		// Best-effort rollback
		_ = oldOwnerMember.ChangeRole(organization.RoleOwner)
		_ = s.members.Update(ctx, oldOwnerMember)
		return nil, fmt.Errorf("accept transfer: promote new owner: %w", err)
	}

	// Step 3: update the org's owner_user_id + clear pending transfer
	if err := org.CompleteTransfer(userID); err != nil {
		return nil, err
	}
	if err := s.orgs.Update(ctx, org); err != nil {
		return nil, fmt.Errorf("accept transfer: persist org: %w", err)
	}

	// Step 4: bump session_version on both users so stale tokens are
	// rejected on next use.
	if _, err := s.users.BumpSessionVersion(ctx, oldOwnerID); err != nil {
		_ = err
	}
	if _, err := s.users.BumpSessionVersion(ctx, userID); err != nil {
		_ = err
	}

	return org, nil
}

// GetPendingTransfer returns the org's current pending transfer, if
// any. Used by the admin panel and the targeted user's notification
// bell to surface "you have a pending ownership transfer" prompts.
func (s *MembershipService) GetPendingTransfer(
	ctx context.Context,
	orgID uuid.UUID,
) (*organization.Organization, error) {
	org, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if !org.IsTransferPending() {
		return nil, errors.New("no pending transfer")
	}
	return org, nil
}

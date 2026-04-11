package organization

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
)

// Admin-only override methods. These bypass the regular permission
// checks enforced by UpdateMemberRole / RemoveMember / transfer flow
// because the caller is a platform admin (verified at the HTTP layer
// via RequireAdmin middleware), not an org member.
//
// They still:
//   - persist via the same repositories
//   - bump session_version on affected users (immediate revocation)
//   - dispatch the same notifications through the notifier helper
//
// Non-goals:
//   - They do NOT identify the actor in the notification payload, so
//     the affected user sees the event as coming from "Someone" —
//     that is intentional: support cases shouldn't expose which
//     platform admin handled the intervention.

// ForceUpdateMemberRole changes a member's role without consulting the
// caller's permission set. Target must exist. Target cannot be the
// current Owner (use ForceTransferOwnership) and the new role cannot
// be Owner (same reason). Bumps the target's session_version and
// emits an org_member_role_changed notification.
func (s *MembershipService) ForceUpdateMemberRole(
	ctx context.Context,
	orgID, targetUserID uuid.UUID,
	newRole organization.Role,
) (*organization.Member, error) {
	if !newRole.IsValid() {
		return nil, organization.ErrInvalidRole
	}
	if newRole == organization.RoleOwner {
		return nil, organization.ErrCannotInviteAsOwner
	}

	target, err := s.members.FindByOrgAndUser(ctx, orgID, targetUserID)
	if err != nil {
		return nil, err
	}
	if target.IsOwner() {
		// Owners are never demoted by this endpoint. If the admin wants
		// to recover a locked org, they use ForceTransferOwnership which
		// atomically moves the ownership to a new user.
		return nil, organization.ErrPermissionDenied
	}

	oldRole := target.Role
	if err := target.ChangeRole(newRole); err != nil {
		return nil, err
	}
	if err := s.members.Update(ctx, target); err != nil {
		return nil, fmt.Errorf("force update member role: persist: %w", err)
	}

	if _, err := s.users.BumpSessionVersion(ctx, targetUserID); err != nil {
		return nil, fmt.Errorf("force update member role: bump session: %w", err)
	}

	// Emit the notification with a nil actor so the copy renders as
	// "Someone changed your role" — see the file-level comment for why
	// the admin identity stays hidden.
	org, _ := s.orgs.FindByID(ctx, orgID)
	notifyMemberRoleChanged(ctx, s.notifications, targetUserID, nil, org, oldRole, newRole)

	return target, nil
}

// ForceRemoveMember evicts a target from an organization without
// consulting the caller's role. Target must be a member. Target cannot
// be the Owner — platform admins must transfer ownership first.
//
// Operator accounts (account_type=operator) are deleted entirely,
// matching the behaviour of the user-driven RemoveMember flow.
func (s *MembershipService) ForceRemoveMember(
	ctx context.Context,
	orgID, targetUserID uuid.UUID,
) error {
	target, err := s.members.FindByOrgAndUser(ctx, orgID, targetUserID)
	if err != nil {
		return err
	}
	if target.IsOwner() {
		return organization.ErrOwnerCannotBeRemoved
	}

	if err := s.members.Delete(ctx, target.ID); err != nil {
		return fmt.Errorf("force remove member: delete membership: %w", err)
	}

	if _, err := s.users.BumpSessionVersion(ctx, targetUserID); err != nil {
		_ = err // non-fatal: stale token expires within 15min anyway
	}

	// Notify BEFORE deleting the operator user row so the notifications
	// row FK stays valid. Marketplace owners keep their user so the
	// ordering is not load-bearing for them.
	org, _ := s.orgs.FindByID(ctx, orgID)
	notifyMemberRemoved(ctx, s.notifications, targetUserID, nil, org)

	targetUser, err := s.users.GetByID(ctx, targetUserID)
	if err == nil && targetUser.AccountType == user.AccountTypeOperator {
		if err := s.users.Delete(ctx, targetUserID); err != nil {
			return fmt.Errorf("force remove member: delete operator user: %w", err)
		}
	}
	return nil
}

// ForceTransferOwnership moves an org's ownership from its current
// Owner to any existing member, regardless of that member's current
// role. Used by platform admins to recover a locked organization
// (e.g. Owner email compromised, no active Admin backup).
//
// Semantics are identical to the regular AcceptTransferOwnership
// flow:
//  1. Demote the current Owner to Admin
//  2. Promote the target to Owner (any starting role allowed)
//  3. Update the org's denormalized owner_user_id and clear any
//     pending transfer state
//  4. Bump session_version on both users
//  5. Notify the (former) Owner that they have been replaced
//
// Constraints:
//   - Target must be a member of the org
//   - Target cannot be the current Owner (nothing to do)
//   - No "must be Admin" guardrail — this is the escape hatch
func (s *MembershipService) ForceTransferOwnership(
	ctx context.Context,
	orgID, newOwnerUserID uuid.UUID,
) (*organization.Organization, error) {
	org, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	oldOwnerID := org.OwnerUserID
	if oldOwnerID == newOwnerUserID {
		return nil, errors.New("force transfer: target is already the owner")
	}

	newOwnerMember, err := s.members.FindByOrgAndUser(ctx, orgID, newOwnerUserID)
	if err != nil {
		// The target must already belong to the org. If the admin wants
		// to transfer to an outsider, they have to invite first.
		return nil, organization.ErrTransferTargetInvalid
	}

	oldOwnerMember, err := s.members.FindByOrgAndUser(ctx, orgID, oldOwnerID)
	if err != nil {
		return nil, fmt.Errorf("force transfer: find old owner: %w", err)
	}

	// Step 1: demote old Owner to Admin first, so the partial unique
	// index idx_org_members_unique_owner does not fire when the new
	// owner's role is updated.
	if err := oldOwnerMember.ChangeRole(organization.RoleAdmin); err != nil {
		return nil, err
	}
	if err := s.members.Update(ctx, oldOwnerMember); err != nil {
		return nil, fmt.Errorf("force transfer: demote old owner: %w", err)
	}

	// Step 2: promote the target. Any starting role is acceptable —
	// that's the whole point of this override.
	if err := newOwnerMember.ChangeRole(organization.RoleOwner); err != nil {
		// Best-effort rollback of step 1.
		_ = oldOwnerMember.ChangeRole(organization.RoleOwner)
		_ = s.members.Update(ctx, oldOwnerMember)
		return nil, err
	}
	if err := s.members.Update(ctx, newOwnerMember); err != nil {
		_ = oldOwnerMember.ChangeRole(organization.RoleOwner)
		_ = s.members.Update(ctx, oldOwnerMember)
		return nil, fmt.Errorf("force transfer: promote new owner: %w", err)
	}

	// Step 3: update the org row + clear any pending transfer that
	// might have been left over from a failed user-driven attempt.
	org.OwnerUserID = newOwnerUserID
	org.PendingTransferToUserID = nil
	org.PendingTransferInitiatedAt = nil
	org.PendingTransferExpiresAt = nil
	if err := s.orgs.Update(ctx, org); err != nil {
		return nil, fmt.Errorf("force transfer: persist org: %w", err)
	}

	// Step 4: bump both sessions.
	if _, err := s.users.BumpSessionVersion(ctx, oldOwnerID); err != nil {
		_ = err
	}
	if _, err := s.users.BumpSessionVersion(ctx, newOwnerUserID); err != nil {
		_ = err
	}

	// Step 5: notify the (former) Owner. The new Owner already knows
	// they asked for it (well — the admin asked for it), so we only
	// alert the demoted side.
	newOwner, _ := s.users.GetByID(ctx, newOwnerUserID)
	notifyTransferAccepted(ctx, s.notifications, oldOwnerID, newOwner, org)

	return org, nil
}

// ForceCancelInvitation is the admin-override version of
// InvitationService.CancelInvitation: it deletes a pending invitation
// without consulting the caller's role or verifying the org match.
// No notification is dispatched — the invitee has not been notified
// of the invitation creation in V1 either (only the email was sent),
// so there is nothing to correct in the notifications table.
func (s *InvitationService) ForceCancelInvitation(
	ctx context.Context,
	invitationID uuid.UUID,
) error {
	inv, err := s.invitations.FindByID(ctx, invitationID)
	if err != nil {
		return err
	}
	if inv.Status != organization.InvitationStatusPending {
		// Already accepted or already cancelled. Idempotent no-op.
		return nil
	}
	if err := s.invitations.Delete(ctx, invitationID); err != nil {
		return fmt.Errorf("force cancel invitation: %w", err)
	}
	return nil
}

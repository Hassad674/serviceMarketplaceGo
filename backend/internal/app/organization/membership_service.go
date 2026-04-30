package organization

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// MembershipService owns the day-to-day membership operations on an
// organization: promote/demote, update title, remove, self-leave, and
// the full ownership transfer flow.
//
// Every mutation that changes a user's effective permissions bumps the
// target's users.session_version so any in-flight JWT is immediately
// invalidated by the auth middleware on the next request — that is the
// "immediate revocation" guarantee we committed to.
//
// After each successful commit the service dispatches an in-app
// notification to the affected user via the NotificationSender port;
// dispatch is best-effort and cannot block the main flow (see
// notifier.go).
type MembershipService struct {
	orgs          repository.OrganizationRepository
	members       repository.OrganizationMemberRepository
	users         repository.UserRepository
	notifications service.NotificationSender // nil disables notifications
}

// MembershipServiceDeps groups the constructor arguments for NewMembershipService.
type MembershipServiceDeps struct {
	Orgs          repository.OrganizationRepository
	Members       repository.OrganizationMemberRepository
	Users         repository.UserRepository
	Notifications service.NotificationSender // optional
}

func NewMembershipService(deps MembershipServiceDeps) *MembershipService {
	return &MembershipService{
		orgs:          deps.Orgs,
		members:       deps.Members,
		users:         deps.Users,
		notifications: deps.Notifications,
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

// ListMembers returns the current members of an organization,
// cursor-paginated by joined_at DESC. Any authenticated member with
// team.view permission can call this (all four roles have view access).
func (s *MembershipService) ListMembers(
	ctx context.Context,
	actorID, orgID uuid.UUID,
	cursor string,
	limit int,
) ([]*organization.Member, string, error) {
	if err := s.requirePermission(ctx, actorID, orgID, organization.PermTeamView); err != nil {
		return nil, "", err
	}
	return s.members.List(ctx, repository.ListMembersParams{
		OrganizationID: orgID,
		Cursor:         cursor,
		Limit:          limit,
	})
}

// ---------------------------------------------------------------------------
// Update role
// ---------------------------------------------------------------------------

// UpdateMemberRole changes the role of an existing member.
//
// V1 rules enforced:
//   - Actor needs team.manage permission (Owner or Admin)
//   - Cannot promote to Owner (use transfer ownership flow instead)
//   - Cannot demote the current Owner (use transfer ownership flow)
//   - Actor cannot change their own role via this method (use the
//     self-demote / leave flows)
//
// On success, the target's session_version is bumped so any active
// JWT loses its authority immediately.
func (s *MembershipService) UpdateMemberRole(
	ctx context.Context,
	actorID, orgID, targetUserID uuid.UUID,
	newRole organization.Role,
) (*organization.Member, error) {
	if !newRole.IsValid() {
		return nil, organization.ErrInvalidRole
	}
	if newRole == organization.RoleOwner {
		return nil, organization.ErrCannotInviteAsOwner
	}
	if actorID == targetUserID {
		return nil, organization.ErrCannotChangeOwnRole
	}

	actor, err := s.members.FindByOrgAndUser(ctx, orgID, actorID)
	if err != nil {
		return nil, mapNotMember(err)
	}
	org, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("update member role: load org: %w", err)
	}
	if !organization.HasEffectivePermission(actor.Role, organization.PermTeamManage, org.RoleOverrides) {
		return nil, organization.ErrPermissionDenied
	}

	target, err := s.members.FindByOrgAndUser(ctx, orgID, targetUserID)
	if err != nil {
		return nil, err
	}
	if !actor.CanManageMember(target) {
		// Actor cannot touch an Owner and actor must be elevated.
		return nil, organization.ErrPermissionDenied
	}

	oldRole := target.Role
	if err := target.ChangeRole(newRole); err != nil {
		return nil, err
	}
	if err := s.members.Update(ctx, target); err != nil {
		return nil, fmt.Errorf("update member role: persist: %w", err)
	}

	// Only invalidate the target's session on demotion (fewer
	// permissions must take effect immediately). Promotions are
	// reflected on the next /me fetch without forcing a re-login.
	if organization.IsDemotion(oldRole, newRole) {
		if _, err := s.users.BumpSessionVersion(ctx, targetUserID); err != nil {
			return nil, fmt.Errorf("update member role: bump session: %w", err)
		}
	}

	// Notify the target their role changed. The org was already
	// loaded for the permission check above, so we reuse it rather
	// than fetching again. The actor is fetched best-effort for the
	// display name.
	actorUser, _ := s.users.GetByID(ctx, actorID)
	notifyMemberRoleChanged(ctx, s.notifications, targetUserID, actorUser, org, oldRole, newRole)

	return target, nil
}

// UpdateMemberTitle updates the free-text job title of a member.
// Permission: team.manage (Owner or Admin). Does NOT bump the session
// version because a title change has no effect on permissions.
func (s *MembershipService) UpdateMemberTitle(
	ctx context.Context,
	actorID, orgID, targetUserID uuid.UUID,
	newTitle string,
) (*organization.Member, error) {
	actor, err := s.members.FindByOrgAndUser(ctx, orgID, actorID)
	if err != nil {
		return nil, mapNotMember(err)
	}
	// Allow self-title-update freely; otherwise require team.manage.
	if actorID != targetUserID {
		org, err := s.orgs.FindByID(ctx, orgID)
		if err != nil {
			return nil, fmt.Errorf("update member title: load org: %w", err)
		}
		if !organization.HasEffectivePermission(actor.Role, organization.PermTeamManage, org.RoleOverrides) {
			return nil, organization.ErrPermissionDenied
		}
	}

	target, err := s.members.FindByOrgAndUser(ctx, orgID, targetUserID)
	if err != nil {
		return nil, err
	}
	if err := target.UpdateTitle(newTitle); err != nil {
		return nil, err
	}
	if err := s.members.Update(ctx, target); err != nil {
		return nil, fmt.Errorf("update member title: persist: %w", err)
	}

	// Skip notification when the user updated their own title — no one
	// else is involved so a notification would be noise.
	if actorID != targetUserID {
		actorUser, _ := s.users.GetByID(ctx, actorID)
		org, _ := s.orgs.FindByID(ctx, orgID)
		notifyMemberTitleChanged(ctx, s.notifications, targetUserID, actorUser, org, newTitle)
	}
	return target, nil
}

// ---------------------------------------------------------------------------
// Remove (evict) a member
// ---------------------------------------------------------------------------

// RemoveMember evicts a member from the organization.
//
// V1 rules enforced:
//   - Actor needs team.manage permission
//   - Cannot remove an Owner (ErrOwnerCannotBeRemoved)
//   - Actor cannot remove themselves (use LeaveOrganization)
//
// When the removed user is an operator (account_type=operator), the
// underlying user account is also deleted — operators have no purpose
// outside their org. When the removed user is a marketplace owner
// (freelance/agency/enterprise self-registered), only the membership
// is removed, preserving their independent account.
//
// The session_version is bumped before the delete so any live token
// the target holds is invalidated on the next request.
func (s *MembershipService) RemoveMember(
	ctx context.Context,
	actorID, orgID, targetUserID uuid.UUID,
) error {
	if actorID == targetUserID {
		return organization.ErrCannotRemoveSelf
	}

	actor, err := s.members.FindByOrgAndUser(ctx, orgID, actorID)
	if err != nil {
		return mapNotMember(err)
	}
	org, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("remove member: load org: %w", err)
	}
	if !organization.HasEffectivePermission(actor.Role, organization.PermTeamManage, org.RoleOverrides) {
		return organization.ErrPermissionDenied
	}

	target, err := s.members.FindByOrgAndUser(ctx, orgID, targetUserID)
	if err != nil {
		return err
	}
	if target.IsOwner() {
		return organization.ErrOwnerCannotBeRemoved
	}
	if !actor.CanManageMember(target) {
		return organization.ErrPermissionDenied
	}

	if err := s.members.Delete(ctx, target.ID); err != nil {
		return fmt.Errorf("remove member: delete membership: %w", err)
	}

	// Bump session version so any in-flight token is rejected on next use.
	// Don't fail the whole operation: the membership is already gone.
	// Worst case the target keeps a stale token until expiry (15 min).
	if _, err := s.users.BumpSessionVersion(ctx, targetUserID); err != nil {
		slog.Warn("remove member: bump session_version failed",
			"error", err, "user_id", targetUserID, "actor_id", actorID)
	}

	// Notify the target BEFORE the operator delete below, so the
	// notifications row has a valid user_id FK. Marketplace owners
	// keep their user row so the order doesn't strictly matter for
	// them, but we emit in the same spot for consistency.
	// Org was already loaded for the permission check above.
	actorUser, _ := s.users.GetByID(ctx, actorID)
	notifyMemberRemoved(ctx, s.notifications, targetUserID, actorUser, org)

	// Operator accounts are deleted entirely. Marketplace owners keep
	// their accounts since they have a life outside this org.
	targetUser, err := s.users.GetByID(ctx, targetUserID)
	if err == nil && targetUser.AccountType == user.AccountTypeOperator {
		if err := s.users.Delete(ctx, targetUserID); err != nil {
			return fmt.Errorf("remove member: delete operator user: %w", err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Self-leave
// ---------------------------------------------------------------------------

// LeaveOrganization removes the caller from the organization.
//
// V1 rules:
//   - The Owner cannot leave via this method — they must transfer
//     ownership first (ErrLastOwnerCannotLeave).
//   - If the caller is an operator, their user account is deleted on
//     leave (same logic as RemoveMember).
func (s *MembershipService) LeaveOrganization(
	ctx context.Context,
	userID, orgID uuid.UUID,
) error {
	member, err := s.members.FindByOrgAndUser(ctx, orgID, userID)
	if err != nil {
		return mapNotMember(err)
	}
	if member.IsOwner() {
		return organization.ErrLastOwnerCannotLeave
	}

	if err := s.members.Delete(ctx, member.ID); err != nil {
		return fmt.Errorf("leave organization: delete membership: %w", err)
	}

	if _, err := s.users.BumpSessionVersion(ctx, userID); err != nil {
		slog.Warn("leave organization: bump session_version failed",
			"error", err, "user_id", userID, "org_id", orgID)
	}

	// Notify the Owner that this user walked out. Lookup the leaver
	// user BEFORE any operator delete below so the display name is
	// still available. Lookup the org to resolve the Owner user id.
	leaver, _ := s.users.GetByID(ctx, userID)
	org, _ := s.orgs.FindByID(ctx, orgID)
	if org != nil && org.OwnerUserID != userID {
		notifyMemberLeft(ctx, s.notifications, org.OwnerUserID, leaver, org)
	}

	u, err := s.users.GetByID(ctx, userID)
	if err == nil && u.AccountType == user.AccountTypeOperator {
		if err := s.users.Delete(ctx, userID); err != nil {
			return fmt.Errorf("leave organization: delete operator user: %w", err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// requirePermission ensures the actor is a member of the org with the
// given permission. Returns ErrNotAMember or ErrPermissionDenied.
//
// Resolves the org's per-role overrides before evaluating the check so
// customized permissions take effect for every team operation.
func (s *MembershipService) requirePermission(
	ctx context.Context,
	actorID, orgID uuid.UUID,
	perm organization.Permission,
) error {
	member, err := s.members.FindByOrgAndUser(ctx, orgID, actorID)
	if err != nil {
		return mapNotMember(err)
	}
	org, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("permission check: load org: %w", err)
	}
	if !organization.HasEffectivePermission(member.Role, perm, org.RoleOverrides) {
		return organization.ErrPermissionDenied
	}
	return nil
}

// mapNotMember converts a generic ErrMemberNotFound into the more
// precise ErrNotAMember so the handler layer can distinguish "target
// doesn't exist" from "actor isn't authorized".
func mapNotMember(err error) error {
	if errors.Is(err, organization.ErrMemberNotFound) {
		return organization.ErrNotAMember
	}
	return err
}

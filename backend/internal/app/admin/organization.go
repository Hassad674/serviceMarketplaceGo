package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	organizationapp "marketplace-backend/internal/app/organization"
	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/port/repository"
)

// Admin-only organization / team management.
//
// These methods wrap the organization app services with the bypass
// semantics platform admins need (recovering locked orgs, removing
// abusive members, cancelling bad invitations). They all assume the
// caller has already been authorized by the admin middleware — there
// are no additional permission checks inside.

// AdminOrganizationDetail is the aggregate the admin UI needs when
// opening a user detail page for an Owner or an Operator. Built by
// GetUserOrganizationDetail as a single call so the handler does not
// fan out to N repositories.
type AdminOrganizationDetail struct {
	Organization       *organization.Organization
	Members            []*organization.Member
	PendingInvitations []*organization.Invitation
	// ViewingRole identifies the role of the user that was the
	// starting point of this lookup. For an Owner, ViewingRole is
	// "owner". For an Operator who belongs to the org, it is their
	// current role. The admin UI uses this to render a "you are
	// looking at this user" breadcrumb.
	ViewingRole organization.Role
}

// GetUserOrganizationDetail finds the organization the given user
// either owns or is a member of, and returns the full admin-view
// aggregate (members + pending invitations + transfer state, exposed
// via the embedded organization).
//
// Returns organization.ErrOrgNotFound when the user has no org at
// all (solo Provider, or unprovisioned user).
func (s *Service) GetUserOrganizationDetail(
	ctx context.Context,
	userID uuid.UUID,
) (*AdminOrganizationDetail, error) {
	if s.orgs == nil || s.orgMembers == nil || s.orgInvitations == nil {
		return nil, errors.New("admin organization feature not wired")
	}

	// Priority 1: is this user the Owner of an org? (Founders of an
	// agency / enterprise. Fast path via the unique owner_user_id
	// index.)
	org, err := s.orgs.FindByOwnerUserID(ctx, userID)
	if err != nil && !errors.Is(err, organization.ErrOrgNotFound) {
		return nil, fmt.Errorf("admin get user org: find by owner: %w", err)
	}

	var viewingRole organization.Role
	if org != nil {
		viewingRole = organization.RoleOwner
	} else {
		// Priority 2: is this user a member of someone else's org?
		// (Operators invited via team management.)
		member, err := s.orgMembers.FindUserPrimaryOrg(ctx, userID)
		if err != nil {
			if errors.Is(err, organization.ErrMemberNotFound) {
				return nil, organization.ErrOrgNotFound
			}
			return nil, fmt.Errorf("admin get user org: find member: %w", err)
		}
		org, err = s.orgs.FindByID(ctx, member.OrganizationID)
		if err != nil {
			return nil, fmt.Errorf("admin get user org: find org: %w", err)
		}
		viewingRole = member.Role
	}

	// Members: large list returned in a single shot. V1 caps org size
	// at ~100 members so one call with a generous limit is enough.
	// If we ever need strict pagination, swap in the cursor-based
	// path from the team management UI.
	members, _, err := s.orgMembers.List(ctx, repository.ListMembersParams{
		OrganizationID: org.ID,
		Limit:          200,
	})
	if err != nil {
		return nil, fmt.Errorf("admin get user org: list members: %w", err)
	}

	invitations, _, err := s.orgInvitations.List(ctx, repository.ListInvitationsParams{
		OrganizationID: org.ID,
		StatusFilter:   organization.InvitationStatusPending,
		Limit:          100,
	})
	if err != nil {
		return nil, fmt.Errorf("admin get user org: list invitations: %w", err)
	}

	return &AdminOrganizationDetail{
		Organization:       org,
		Members:            members,
		PendingInvitations: invitations,
		ViewingRole:        viewingRole,
	}, nil
}

// ForceTransferOwnership is a thin delegation to the MembershipService
// admin override. Exposed here so the admin handler does not need to
// know about the organization app package directly — it composes the
// admin.Service only.
//
// SEC-13: emits an audit row on success. The caller is the admin
// performing the override; the resource is the org whose ownership
// changed. The new owner's user_id is captured in metadata for
// forensic search ("which orgs did admin X take over and reassign?").
func (s *Service) ForceTransferOwnership(
	ctx context.Context,
	orgID, newOwnerUserID uuid.UUID,
) (*organization.Organization, error) {
	if s.membership == nil {
		return nil, errors.New("admin organization feature not wired")
	}
	org, err := s.membership.ForceTransferOwnership(ctx, orgID, newOwnerUserID)
	if err != nil {
		return nil, err
	}

	s.logAudit(ctx, audit.NewEntryInput{
		Action:       audit.ActionAdminForceTransfer,
		ResourceType: audit.ResourceTypeOrganization,
		ResourceID:   &orgID,
		Metadata: map[string]any{
			"new_owner_user_id": newOwnerUserID.String(),
		},
	})
	return org, nil
}

// ForceUpdateMemberRole delegates to the override method.
func (s *Service) ForceUpdateMemberRole(
	ctx context.Context,
	orgID, targetUserID uuid.UUID,
	newRole organization.Role,
) (*organization.Member, error) {
	if s.membership == nil {
		return nil, errors.New("admin organization feature not wired")
	}
	return s.membership.ForceUpdateMemberRole(ctx, orgID, targetUserID, newRole)
}

// ForceRemoveMember delegates to the override method.
func (s *Service) ForceRemoveMember(
	ctx context.Context,
	orgID, targetUserID uuid.UUID,
) error {
	if s.membership == nil {
		return errors.New("admin organization feature not wired")
	}
	return s.membership.ForceRemoveMember(ctx, orgID, targetUserID)
}

// ForceCancelInvitation delegates to the invitation service override.
func (s *Service) ForceCancelInvitation(
	ctx context.Context,
	invitationID uuid.UUID,
) error {
	if s.invitation == nil {
		return errors.New("admin organization feature not wired")
	}
	return s.invitation.ForceCancelInvitation(ctx, invitationID)
}

// Package-level satisfaction check: the organization app services
// referenced via interface fields here live in a sibling package.
// Importing them directly in service.go would create a mutual-import
// hazard, so we keep the reference as a typed field pointer and let
// the caller (cmd/api/main.go) wire it at startup.
var _ = (*organizationapp.MembershipService)(nil)
var _ = (*organizationapp.InvitationService)(nil)

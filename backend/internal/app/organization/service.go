package organization

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// Service is the application layer for organization management.
//
// In Phase 1 it covers the bare minimum required to wire the auth flow:
// create an organization for a new Agency/Enterprise user, resolve a
// user's org context at JWT issuance, and compute effective permissions.
//
// Team management (invite, promote, demote, remove, transfer ownership)
// lives on top of this service and is added in Phase 2+.
type Service struct {
	orgs        repository.OrganizationRepository
	members     repository.OrganizationMemberRepository
	invitations repository.OrganizationInvitationRepository
}

// NewService wires the org service. All three repositories are required
// even in Phase 1 — invitations is injected now so the wiring never
// changes as we add capabilities.
func NewService(
	orgs repository.OrganizationRepository,
	members repository.OrganizationMemberRepository,
	invitations repository.OrganizationInvitationRepository,
) *Service {
	return &Service{
		orgs:        orgs,
		members:     members,
		invitations: invitations,
	}
}

// Context captures everything the auth flow needs about a user's org
// membership to populate the JWT and /me responses.
//
// When the user does not belong to an organization (typical Provider
// case), the Organization field is nil.
type Context struct {
	Organization *organization.Organization
	Member       *organization.Member
	Permissions  []organization.Permission
}

// CreateForOwner provisions a brand new organization owned by the given
// user, creating both the org row and the Owner membership in a single
// DB transaction.
//
// Only users with marketplace role Agency or Enterprise can own an org.
// Providers are solo and calling this for them is a logic error — it
// returns ErrProviderCannotOwnOrg, which the auth service branches on
// to decide whether to create an org at registration time.
//
// The user's OwnerUserID on the returned Organization matches u.ID.
// The caller should reuse the returned Context when issuing the user's
// first access token so the JWT carries the fresh org_id and org_role.
func (s *Service) CreateForOwner(ctx context.Context, u *user.User) (*Context, error) {
	if u == nil || u.ID == uuid.Nil {
		return nil, fmt.Errorf("create organization for owner: %w", organization.ErrOrgNotFound)
	}

	orgType, err := orgTypeFromMarketplaceRole(u.Role)
	if err != nil {
		return nil, fmt.Errorf("create organization for owner: %w", err)
	}

	// Construct the domain entities first — validation happens here, so
	// a bad input fails before we touch the database.
	org, err := organization.NewOrganization(u.ID, orgType)
	if err != nil {
		return nil, fmt.Errorf("create organization for owner: build org: %w", err)
	}

	displayName := firstNonEmpty(u.DisplayName, u.FirstName+" "+u.LastName)
	ownerMember, err := organization.NewMember(org.ID, u.ID, organization.RoleOwner, "")
	if err != nil {
		return nil, fmt.Errorf("create organization for owner: build owner member: %w", err)
	}
	_ = displayName // reserved for future use (e.g. default title)

	// Persist atomically.
	if err := s.orgs.CreateWithOwnerMembership(ctx, org, ownerMember); err != nil {
		return nil, fmt.Errorf("create organization for owner: persist: %w", err)
	}

	return &Context{
		Organization: org,
		Member:       ownerMember,
		Permissions:  organization.PermissionsFor(organization.RoleOwner),
	}, nil
}

// ResolveContext returns the org context for a user, or nil when the
// user does not belong to any organization. Never returns an error when
// the user is simply solo — only when the DB lookups fail for other
// reasons.
//
// Used by the auth flow at login/refresh time to build the access token
// claims and by /me to populate the response envelope.
func (s *Service) ResolveContext(ctx context.Context, userID uuid.UUID) (*Context, error) {
	member, err := s.members.FindUserPrimaryOrg(ctx, userID)
	if err != nil {
		if errors.Is(err, organization.ErrMemberNotFound) {
			return nil, nil // solo user, no org
		}
		return nil, fmt.Errorf("resolve org context: find member: %w", err)
	}

	org, err := s.orgs.FindByID(ctx, member.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("resolve org context: find org: %w", err)
	}

	return &Context{
		Organization: org,
		Member:       member,
		Permissions:  organization.PermissionsFor(member.Role),
	}, nil
}

// HasPermission reports whether the user has the given permission in
// their (optionally specified) organization. Delegates to the domain
// permission map — the service layer never hard-codes role checks.
//
// Returns false when the user has no org (solo Provider) regardless of
// the permission argument.
func (s *Service) HasPermission(ctx context.Context, userID uuid.UUID, perm organization.Permission) (bool, error) {
	orgCtx, err := s.ResolveContext(ctx, userID)
	if err != nil {
		return false, err
	}
	if orgCtx == nil {
		return false, nil
	}
	return organization.HasPermission(orgCtx.Member.Role, perm), nil
}

// orgTypeFromMarketplaceRole maps the user's marketplace role to the
// corresponding organization type. Only Agency and Enterprise users can
// own an organization in V1.
func orgTypeFromMarketplaceRole(role user.Role) (organization.OrgType, error) {
	switch role {
	case user.RoleAgency:
		return organization.OrgTypeAgency, nil
	case user.RoleEnterprise:
		return organization.OrgTypeEnterprise, nil
	case user.RoleProvider:
		return "", organization.ErrProviderCannotOwnOrg
	default:
		return "", organization.ErrInvalidOrgType
	}
}

// firstNonEmpty returns the first argument that is not blank.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

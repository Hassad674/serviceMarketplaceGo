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
	// orgs reuses the package-local orgReaderWriter composite — the
	// service reads the org row in ResolveContextForUser and creates a
	// fresh org + owner membership atomically in CreateForOwner.
	orgs        orgReaderWriter
	members     repository.OrganizationMemberRepository
	invitations repository.OrganizationInvitationRepository
}

// NewService wires the org service. All three repositories are required
// even in Phase 1 — invitations is injected now so the wiring never
// changes as we add capabilities.
func NewService(
	orgs orgReaderWriter,
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
// Every user gets an organization at registration — agencies and
// enterprises get a company org, providers get a provider_personal org
// so that invitations and shared-account operators work identically
// across all marketplace roles (Stripe Dashboard semantics).
//
// The default org name is the user's display name (falling back to
// first+last). The owner can rename it from the team settings UI.
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

	defaultName := firstNonEmpty(u.DisplayName, u.FirstName+" "+u.LastName, u.Email)

	// Construct the domain entities first — validation happens here, so
	// a bad input fails before we touch the database.
	org, err := organization.NewOrganization(u.ID, orgType, defaultName)
	if err != nil {
		return nil, fmt.Errorf("create organization for owner: build org: %w", err)
	}

	ownerMember, err := organization.NewMember(org.ID, u.ID, organization.RoleOwner, "")
	if err != nil {
		return nil, fmt.Errorf("create organization for owner: build owner member: %w", err)
	}

	// Persist atomically.
	if err := s.orgs.CreateWithOwnerMembership(ctx, org, ownerMember); err != nil {
		return nil, fmt.Errorf("create organization for owner: persist: %w", err)
	}

	// Owner is immune to overrides by design — EffectivePermissionsFor
	// with a nil overrides map is equivalent to the hardcoded Owner set.
	// We still go through EffectivePermissionsFor for symmetry with the
	// login/refresh path, which is where the override resolution
	// actually matters.
	return &Context{
		Organization: org,
		Member:       ownerMember,
		Permissions:  organization.EffectivePermissionsFor(organization.RoleOwner, org.RoleOverrides),
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

	// Load the org's permission overrides and resolve the effective
	// permission set. Every downstream consumer (JWT claims, /me,
	// middleware fast-path) reads this list, so the override resolution
	// happens exactly once per login/refresh — zero per-request cost.
	return &Context{
		Organization: org,
		Member:       member,
		Permissions:  organization.EffectivePermissionsFor(member.Role, org.RoleOverrides),
	}, nil
}

// HasPermission reports whether the user has the given permission in
// their (optionally specified) organization. Resolves the org's
// customized role overrides, so this check honors "Admin can now
// withdraw" style customizations transparently.
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
	if orgCtx.Organization == nil {
		return false, nil
	}
	return organization.HasEffectivePermission(
		orgCtx.Member.Role,
		perm,
		orgCtx.Organization.RoleOverrides,
	), nil
}

// orgTypeFromMarketplaceRole maps the user's marketplace role to the
// corresponding organization type. Providers get a provider_personal
// org since phase R1 — the Stripe Dashboard model requires every user
// to act through an org so invited operators can join.
func orgTypeFromMarketplaceRole(role user.Role) (organization.OrgType, error) {
	switch role {
	case user.RoleAgency:
		return organization.OrgTypeAgency, nil
	case user.RoleEnterprise:
		return organization.OrgTypeEnterprise, nil
	case user.RoleProvider:
		return organization.OrgTypeProviderPersonal, nil
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

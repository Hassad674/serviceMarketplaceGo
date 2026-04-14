package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/port/repository"
)

// profilePricingOrgInfoResolverAdapter bridges the profile pricing
// service's OrgInfoResolver contract to the existing organization
// and user repositories. The profile pricing package refuses to
// import domain/organization or domain/user directly
// (feature-independence invariant), so the thin shim lives here in
// main.go where cross-feature wiring is allowed — the one place in
// the codebase where separate bounded contexts may meet.
//
// The resolver looks up BOTH the org type AND the owner user's
// referrer_enabled flag, because IsKindAllowedForOrg needs both
// inputs to correctly gate pricing writes for provider_personal
// orgs with / without the apporteur cap on.
type profilePricingOrgInfoResolverAdapter struct {
	orgs  repository.OrganizationRepository
	users repository.UserRepository
}

// newProfilePricingOrgInfoResolverAdapter returns a resolver ready
// to be passed into profilepricingapp.NewService.
func newProfilePricingOrgInfoResolverAdapter(
	orgs repository.OrganizationRepository,
	users repository.UserRepository,
) *profilePricingOrgInfoResolverAdapter {
	return &profilePricingOrgInfoResolverAdapter{orgs: orgs, users: users}
}

// GetOrgInfo implements profilepricingapp.OrgInfoResolver. Returns
// the organization's type string ("agency", "provider_personal",
// "enterprise") plus the owner user's referrer_enabled flag. Any
// failure to resolve either record surfaces as a wrapped error so
// the service layer can log it with context.
func (a *profilePricingOrgInfoResolverAdapter) GetOrgInfo(ctx context.Context, orgID uuid.UUID) (string, bool, error) {
	org, err := a.orgs.FindByID(ctx, orgID)
	if err != nil {
		return "", false, fmt.Errorf("resolve org: %w", err)
	}
	if org == nil {
		return "", false, errors.New("organization not found")
	}

	owner, err := a.users.GetByID(ctx, org.OwnerUserID)
	if err != nil {
		return "", false, fmt.Errorf("resolve org owner: %w", err)
	}
	if owner == nil {
		// Owner is a required FK on organizations — missing here
		// means a data-integrity bug, not a legitimate "no owner"
		// state. Surfacing it explicitly lets ops notice fast.
		return "", false, errors.New("organization owner user not found")
	}

	return string(org.Type), owner.ReferrerEnabled, nil
}

package main

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/port/repository"
)

// orgOverridesAdapter bridges the concrete OrganizationRepository to
// the narrow middleware.OrgOverridesResolver contract so the auth
// middleware can compute live permissions on every request without
// importing the full organization repo surface.
//
// The repository does not expose a dedicated GetRoleOverrides method
// (overrides travel as a JSONB column on the org row, read whenever
// the org is loaded), so this adapter calls FindByID and projects
// just the field we need. If org-level caching gets added later, this
// is the single place to wire it — the middleware does not need to
// know.
type orgOverridesAdapter struct {
	repo repository.OrganizationRepository
}

// GetRoleOverrides returns the live role_overrides JSONB snapshot for
// the given org. A nil map is a valid return value (brand-new orgs
// have no overrides and EffectivePermissionsFor treats nil as
// "defaults only"), so callers must not conflate "no overrides" with
// "lookup failed".
func (a orgOverridesAdapter) GetRoleOverrides(
	ctx context.Context,
	orgID uuid.UUID,
) (organization.RoleOverrides, error) {
	org, err := a.repo.FindByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return org.RoleOverrides, nil
}

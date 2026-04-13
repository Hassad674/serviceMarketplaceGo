package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	domainskill "marketplace-backend/internal/domain/skill"
	"marketplace-backend/internal/port/repository"
)

// orgTypeResolverAdapter bridges the skill service's OrgTypeResolver
// contract to the existing OrganizationRepository. The skill package
// refuses to import domain/organization directly (feature-independence
// invariant), so the thin shim lives here in main.go where cross-feature
// wiring is allowed — this is the only place in the codebase where
// separate bounded contexts may meet.
//
// The resolver is stateless beyond the repository pointer, so a single
// instance is shared across all skill service calls.
type orgTypeResolverAdapter struct {
	repo repository.OrganizationRepository
}

// newOrgTypeResolverAdapter returns a resolver ready to be passed into
// skillapp.NewService.
func newOrgTypeResolverAdapter(repo repository.OrganizationRepository) *orgTypeResolverAdapter {
	return &orgTypeResolverAdapter{repo: repo}
}

// GetOrgType implements skillapp.OrgTypeResolver. Returns the
// organization's type string ("agency", "provider_personal",
// "enterprise") or a wrapped error if the organization cannot be
// resolved.
func (a *orgTypeResolverAdapter) GetOrgType(ctx context.Context, orgID uuid.UUID) (domainskill.OrgType, error) {
	org, err := a.repo.FindByID(ctx, orgID)
	if err != nil {
		return "", fmt.Errorf("resolve org type: %w", err)
	}
	if org == nil {
		return "", errors.New("organization not found")
	}
	return domainskill.OrgType(org.Type), nil
}

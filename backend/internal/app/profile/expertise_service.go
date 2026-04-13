package profileapp

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/expertise"
	"marketplace-backend/internal/port/repository"
)

// ExpertiseService owns the use cases attached to an organization's
// declared expertise domains. It lives in the profile application
// package because expertise is part of the org's public profile —
// co-located with social links, portfolio, and the main profile
// service for the same reason.
//
// Dependencies: the expertise repository (persistence) and the
// organization repository (to resolve the org type for the per-type
// maximum). Both are interfaces from port/, so the service is fully
// testable with mocks.
type ExpertiseService struct {
	expertise     repository.ExpertiseRepository
	organizations repository.OrganizationRepository
}

// NewExpertiseService wires a new expertise service. It takes the
// repository interfaces directly — no service struct — so the
// dependency graph at wiring time stays flat and obvious.
func NewExpertiseService(
	expertiseRepo repository.ExpertiseRepository,
	orgRepo repository.OrganizationRepository,
) *ExpertiseService {
	return &ExpertiseService{
		expertise:     expertiseRepo,
		organizations: orgRepo,
	}
}

// ListByOrganization returns the ordered list of expertise keys for
// the given organization. Always returns a non-nil slice so the HTTP
// response carries "[]" instead of "null" when nothing is declared.
func (s *ExpertiseService) ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]string, error) {
	keys, err := s.expertise.ListByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list expertise: %w", err)
	}
	if keys == nil {
		keys = []string{}
	}
	return keys, nil
}

// SetExpertise replaces the organization's expertise list atomically.
// Validation order (each step short-circuits on failure):
//
//  1. Resolve the org to check it exists and to read its type.
//  2. Reject enterprise orgs — the feature is forbidden for clients.
//  3. Reject unknown domain keys.
//  4. Reject duplicates in the incoming slice.
//  5. Reject counts above the per-org-type maximum.
//  6. Delegate to the repository, which performs the transactional
//     DELETE + INSERT.
//
// The returned slice is the normalized list (same keys, same order,
// with a non-nil empty slice when the caller cleared the list).
func (s *ExpertiseService) SetExpertise(
	ctx context.Context,
	orgID uuid.UUID,
	domainKeys []string,
) ([]string, error) {
	org, err := s.organizations.FindByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("set expertise: resolve org: %w", err)
	}

	orgType := expertise.OrgType(org.Type)
	if !expertise.IsFeatureEnabled(orgType) {
		return nil, expertise.ErrForbiddenOrgType
	}

	if err := validateExpertisePayload(domainKeys, orgType); err != nil {
		return nil, err
	}

	// Pre-allocate a fresh slice so the caller's input array cannot
	// alias the persisted copy held by the repository mock in tests,
	// and so the returned slice is non-nil even when empty.
	normalized := make([]string, len(domainKeys))
	copy(normalized, domainKeys)

	if err := s.expertise.Replace(ctx, orgID, normalized); err != nil {
		return nil, fmt.Errorf("set expertise: persist: %w", err)
	}
	return normalized, nil
}

// validateExpertisePayload enforces the four validation rules that do
// not require a database round-trip. Extracted so SetExpertise reads
// as a linear pipeline and to keep individual function bodies short.
func validateExpertisePayload(keys []string, orgType expertise.OrgType) error {
	max := expertise.MaxForOrgType(orgType)
	if len(keys) > max {
		return expertise.ErrOverMax
	}
	seen := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if !expertise.IsValidKey(k) {
			return expertise.ErrUnknownKey
		}
		if _, dup := seen[k]; dup {
			return expertise.ErrDuplicate
		}
		seen[k] = struct{}{}
	}
	return nil
}

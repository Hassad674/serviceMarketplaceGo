package profileapp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/expertise"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
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
// testable with mocks. The cacheInvalidator is optional: when wired
// (production), a successful SetExpertise flushes the cached read
// path so the next list reflects reality immediately.
type ExpertiseService struct {
	expertise repository.ExpertiseRepository
	// organizations is narrowed to OrganizationReader — the service
	// only resolves the org by id to read its type before SetExpertise
	// gates the request.
	organizations    repository.OrganizationReader
	cacheInvalidator portservice.CacheInvalidatorByOrgID
}

// NewExpertiseService wires a new expertise service. It takes the
// repository interfaces directly — no service struct — so the
// dependency graph at wiring time stays flat and obvious.
func NewExpertiseService(
	expertiseRepo repository.ExpertiseRepository,
	orgRepo repository.OrganizationReader,
) *ExpertiseService {
	return &ExpertiseService{
		expertise:     expertiseRepo,
		organizations: orgRepo,
	}
}

// WithCacheInvalidator attaches the optional read-cache invalidator
// fired after every successful SetExpertise. Returns the same
// service for fluent wiring in main.go. Passing nil is allowed
// (tests, search engine disabled) and disables invalidation —
// callers will simply wait for the TTL to age out, which is still
// correct, only slower.
func (s *ExpertiseService) WithCacheInvalidator(inv portservice.CacheInvalidatorByOrgID) *ExpertiseService {
	if s == nil {
		return nil
	}
	s.cacheInvalidator = inv
	return s
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

	// Cache invalidation order: DB write succeeds → cache delete.
	// The reverse order (cache delete first, then DB write) opens a
	// split-brain window where a concurrent reader can re-populate
	// the cache from the OLD DB row before the new one commits.
	// Best-effort: a failed Del logs but does not unwind the
	// successful persist; the next read will simply hit the stale
	// entry until the TTL ages out, which converges to correctness.
	if s.cacheInvalidator != nil {
		if invErr := s.cacheInvalidator.Invalidate(ctx, orgID); invErr != nil {
			slog.Warn("set expertise: cache invalidation failed",
				"org_id", orgID, "error", invErr)
		}
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

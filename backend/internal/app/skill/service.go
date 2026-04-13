// Package skill is the application service layer for the hybrid skills
// catalog and per-organization skill attachments. It orchestrates the
// two port interfaces (SkillCatalogRepository and ProfileSkillRepository)
// plus a local OrgTypeResolver to enforce role-based limits when an
// organization edits its declared skills.
//
// The service deliberately imports only stdlib, uuid, the skill and
// expertise domain packages, and port/repository. Zero adapter imports,
// zero cross-feature imports — keeping the hexagonal feature-
// independence invariant intact.
package skill

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/expertise"
	domainskill "marketplace-backend/internal/domain/skill"
	"marketplace-backend/internal/port/repository"
)

// OrgTypeResolver is a thin read-only dependency the skill service
// needs to look up an organization's type (agency, provider_personal,
// enterprise) so it can enforce per-type limits on ReplaceProfileSkills.
//
// Defined locally in this package — rather than in port/repository —
// because it is a one-method collaboration scoped to this feature.
// Keeping it here preserves the invariant that skill does not import
// the organization package, even at the interface level: the wiring
// layer (cmd/api/main.go) is free to supply any adapter that matches
// this signature.
type OrgTypeResolver interface {
	GetOrgType(ctx context.Context, orgID uuid.UUID) (string, error)
}

// Service orchestrates the skill use cases: catalog browsing,
// autocomplete, profile skill replacement with limits enforcement,
// and free-form user skill creation.
type Service struct {
	catalog  repository.SkillCatalogRepository
	profiles repository.ProfileSkillRepository
	orgs     OrgTypeResolver
}

// NewService wires the skill service with its dependencies. All three
// parameters are required — the service has no optional collaborators
// and no sane default for any of them.
func NewService(
	catalog repository.SkillCatalogRepository,
	profiles repository.ProfileSkillRepository,
	orgs OrgTypeResolver,
) *Service {
	return &Service{catalog: catalog, profiles: profiles, orgs: orgs}
}

// ---- Catalog read path (public reads, no auth needed) ----

// GetCuratedForExpertise returns curated skills for the browse-by-
// expertise panel. The expertise key must be a canonical value from
// the frozen catalog, otherwise ErrInvalidExpertiseKey is returned
// without touching the repository. Limit is clamped to [1, 100] with
// a default of 50 when the caller passes 0 or a negative value.
func (s *Service) GetCuratedForExpertise(
	ctx context.Context,
	expertiseKey string,
	limit int,
) ([]*domainskill.CatalogEntry, error) {
	if !expertise.IsValidKey(expertiseKey) {
		return nil, domainskill.ErrInvalidExpertiseKey
	}
	limit = clampLimit(limit, 50, 1, 100)
	return s.catalog.ListCuratedByExpertise(ctx, expertiseKey, limit)
}

// CountCuratedForExpertise returns the count of curated skills in a
// given expertise domain. Used by UI badges like "142 skills in
// Development" without paying for a full list fetch.
func (s *Service) CountCuratedForExpertise(ctx context.Context, expertiseKey string) (int, error) {
	if !expertise.IsValidKey(expertiseKey) {
		return 0, domainskill.ErrInvalidExpertiseKey
	}
	return s.catalog.CountCuratedByExpertise(ctx, expertiseKey)
}

// Autocomplete returns skills matching a query string. The raw query
// is normalized via the domain helper so the comparison is case-
// insensitive and whitespace-tolerant. An empty (or whitespace-only)
// query returns (nil, nil) — not an error — so the UI can simply
// render an empty dropdown when the input field is cleared. The
// limit is clamped to [1, 50] with a default of 20.
func (s *Service) Autocomplete(
	ctx context.Context,
	rawQuery string,
	limit int,
) ([]*domainskill.CatalogEntry, error) {
	normalized := domainskill.NormalizeSkillText(rawQuery)
	if normalized == "" {
		return nil, nil
	}
	limit = clampLimit(limit, 20, 1, 50)
	return s.catalog.SearchAutocomplete(ctx, normalized, limit)
}

// ---- Profile skills (authenticated) ----

// GetProfileSkills returns the organization's declared skills in the
// order the org chose to display them. The repository returns an
// empty (non-nil) slice when nothing is declared.
func (s *Service) GetProfileSkills(
	ctx context.Context,
	orgID uuid.UUID,
) ([]*domainskill.ProfileSkill, error) {
	return s.profiles.ListByOrgID(ctx, orgID)
}

// ReplaceProfileSkillsInput is the payload for ReplaceProfileSkills.
// SkillTexts is the ordered list of skill_text values (not yet
// normalized — the service handles that). Position 0 = first.
type ReplaceProfileSkillsInput struct {
	OrganizationID uuid.UUID
	SkillTexts     []string
}

// ReplaceProfileSkills atomically replaces the organization's skills
// with the given list.
//
// Validation order (each step short-circuits on failure):
//
//  1. Resolve the organization type via the OrgTypeResolver.
//  2. Reject when the feature is disabled for that org type
//     (ErrSkillsDisabledForOrgType).
//  3. Normalize each skill_text and drop exact duplicates while
//     preserving first-occurrence order.
//  4. Reject when the deduped list exceeds MaxSkillsForOrgType
//     (ErrTooManySkills).
//  5. Verify every skill exists in the catalog via FindByText.
//     Unknown skills return ErrSkillNotFound, wrapped with the
//     offending text so the caller can surface it to the user.
//
// Validation passing, the method delegates to ProfileSkillRepository.
// ReplaceForOrg which performs the transactional DELETE + INSERT.
//
// Caller responsibility: authentication and ownership checks happen
// at the handler layer — this method trusts that the caller has
// already verified the user is entitled to write the target org.
//
// NOTE: usage_count is NOT updated here. Doing it correctly requires
// diffing the old and new lists (increment for additions, decrement
// for removals) and coordinating with the catalog repo. That diff
// logic is deferred to a follow-up — v1 accepts slightly stale
// counters in exchange for simpler code.
func (s *Service) ReplaceProfileSkills(ctx context.Context, input ReplaceProfileSkillsInput) error {
	orgType, err := s.orgs.GetOrgType(ctx, input.OrganizationID)
	if err != nil {
		return fmt.Errorf("replace profile skills: resolve org type: %w", err)
	}
	if !domainskill.IsSkillsFeatureEnabled(orgType) {
		return domainskill.ErrSkillsDisabledForOrgType
	}

	normalized := normalizeAndDedupe(input.SkillTexts)
	if len(normalized) > domainskill.MaxSkillsForOrgType(orgType) {
		return domainskill.ErrTooManySkills
	}

	profileSkills, err := s.buildProfileSkills(ctx, input.OrganizationID, normalized)
	if err != nil {
		return err
	}

	if err := s.profiles.ReplaceForOrg(ctx, input.OrganizationID, profileSkills); err != nil {
		return fmt.Errorf("replace profile skills: persist: %w", err)
	}
	return nil
}

// buildProfileSkills materializes the ProfileSkill slice to persist
// after verifying every entry exists in the catalog. Kept separate
// from ReplaceProfileSkills so the parent function stays short and
// the catalog-lookup loop is easy to read in isolation.
func (s *Service) buildProfileSkills(
	ctx context.Context,
	orgID uuid.UUID,
	normalized []string,
) ([]*domainskill.ProfileSkill, error) {
	profileSkills := make([]*domainskill.ProfileSkill, 0, len(normalized))
	for i, text := range normalized {
		entry, err := s.catalog.FindByText(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("replace profile skills: find %q: %w", text, err)
		}
		if entry == nil {
			return nil, fmt.Errorf("%w: %q", domainskill.ErrSkillNotFound, text)
		}
		profileSkills = append(profileSkills, &domainskill.ProfileSkill{
			OrganizationID: orgID,
			SkillText:      text,
			Position:       i,
		})
	}
	return profileSkills, nil
}

// ---- User-created skill path (authenticated, free-form creation) ----

// CreateUserSkillInput is the payload for CreateUserSkill. The
// ExpertiseKeys slice is typically auto-inherited from the creator's
// organization expertise domains at the handler layer — the service
// still filters defensively in case an unknown key sneaks through.
type CreateUserSkillInput struct {
	DisplayText   string
	ExpertiseKeys []string
}

// CreateUserSkill inserts a new non-curated skill into the catalog,
// or returns the existing entry if the normalized form collides with
// an already-present skill (curated or not). This is the free-form
// "Create '<your term>'" path surfaced by the autocomplete dropdown
// when no match is found.
//
// Validation:
//
//  1. Filter expertise keys against expertise.IsValidKey; invalid keys
//     are silently dropped (defense-in-depth — the caller is expected
//     to have prefiltered them).
//  2. Build a candidate CatalogEntry via domainskill.NewCatalogEntry,
//     which normalizes skill_text and rejects empty display text.
//  3. Look up the normalized text: if a row already exists, return it
//     untouched. Users get a consistent, canonical entry regardless of
//     the casing they typed, and we never overwrite curated metadata.
//  4. Otherwise Upsert the new entry and re-fetch to pick up the DB
//     defaults (created_at / updated_at).
func (s *Service) CreateUserSkill(
	ctx context.Context,
	input CreateUserSkillInput,
) (*domainskill.CatalogEntry, error) {
	validKeys := filterValidExpertiseKeys(input.ExpertiseKeys)

	entry, err := domainskill.NewCatalogEntry(input.DisplayText, input.DisplayText, validKeys, false)
	if err != nil {
		return nil, fmt.Errorf("create user skill: %w", err)
	}

	existing, err := s.catalog.FindByText(ctx, entry.SkillText)
	if err != nil {
		return nil, fmt.Errorf("create user skill: find existing: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	if err := s.catalog.Upsert(ctx, entry); err != nil {
		return nil, fmt.Errorf("create user skill: upsert: %w", err)
	}
	// Re-fetch to pick up DB-assigned timestamps (created_at/updated_at).
	created, err := s.catalog.FindByText(ctx, entry.SkillText)
	if err != nil {
		return nil, fmt.Errorf("create user skill: refetch: %w", err)
	}
	return created, nil
}

// ---- unexported helpers ----

// clampLimit returns def when limit <= 0, otherwise clamps to [min, max].
// All the read endpoints share the same idiom, so this avoids three
// near-identical blocks across the public methods above.
func clampLimit(limit, def, min, max int) int {
	if limit <= 0 {
		return def
	}
	if limit < min {
		return min
	}
	if limit > max {
		return max
	}
	return limit
}

// normalizeAndDedupe normalizes each entry via domainskill.NormalizeSkillText
// and drops subsequent duplicates, preserving first-occurrence order.
// Empty (post-normalization) strings are dropped entirely — the service
// treats them as a no-op rather than a validation error so a client
// sending a trailing empty row in a form doesn't get rejected.
func normalizeAndDedupe(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, raw := range in {
		normalized := domainskill.NormalizeSkillText(raw)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

// filterValidExpertiseKeys drops keys not present in the expertise
// catalog, preserving order and deduplicating. Used by CreateUserSkill
// as a defensive filter — invalid keys are silently dropped rather
// than rejected so a user typing a new skill never fails just because
// an upstream call forgot to prefilter its inherited keys.
func filterValidExpertiseKeys(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, key := range in {
		if !expertise.IsValidKey(key) {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

// Package skill wires the skill domain to the outside world.
//
// The service composes two repositories (SkillCatalogRepository and
// ProfileSkillRepository) and one thin OrgTypeResolver dependency used
// to enforce the per-org-type limits (agency 40, provider_personal 25,
// enterprise 0). Expertise key validation happens here — domain/skill
// intentionally does not import the expertise package, so this service
// is the only place that knows about both.
//
// Method signatures stay small and rectangular so the HTTP handler can
// call them directly without further translation, and so unit tests can
// mock the service via a narrow interface living in the handler package.
package skill

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/expertise"
	domainskill "marketplace-backend/internal/domain/skill"
	"marketplace-backend/internal/port/repository"
)

// OrgTypeResolver is the only external dependency the skill feature
// needs beyond its own repositories: a way to resolve an organization
// id to its type string ("agency", "provider_personal", "enterprise").
//
// Defined locally to keep the skill package free of any cross-feature
// import — the wiring layer in cmd/api/main.go is expected to provide
// a small adapter around the real organization repository.
type OrgTypeResolver interface {
	GetOrgType(ctx context.Context, orgID uuid.UUID) (domainskill.OrgType, error)
}

// Service is the application layer entry point for the skill feature.
type Service struct {
	catalog  repository.SkillCatalogRepository
	profiles repository.ProfileSkillRepository
	orgTypes OrgTypeResolver
}

// NewService constructs a skill service with explicit dependencies.
// Every field is required in production; tests that only exercise a
// subset of methods may pass nil for the unused collaborators.
func NewService(
	catalog repository.SkillCatalogRepository,
	profiles repository.ProfileSkillRepository,
	orgTypes OrgTypeResolver,
) *Service {
	return &Service{
		catalog:  catalog,
		profiles: profiles,
		orgTypes: orgTypes,
	}
}

// GetCuratedForExpertise returns the curated catalog entries tagged
// with the given expertise key, sorted by usage_count desc. The key
// must belong to the frozen expertise catalog — unknown keys return
// domainskill.ErrInvalidExpertiseKey.
func (s *Service) GetCuratedForExpertise(
	ctx context.Context,
	expertiseKey string,
	limit int,
) ([]*domainskill.CatalogEntry, error) {
	if !expertise.IsValidKey(expertiseKey) {
		return nil, domainskill.ErrInvalidExpertiseKey
	}
	if limit <= 0 {
		limit = 50
	}
	entries, err := s.catalog.ListCuratedByExpertise(ctx, expertiseKey, limit)
	if err != nil {
		return nil, fmt.Errorf("list curated skills: %w", err)
	}
	return entries, nil
}

// CountCuratedForExpertise returns the total number of curated entries
// tagged with the given expertise key.
func (s *Service) CountCuratedForExpertise(
	ctx context.Context,
	expertiseKey string,
) (int, error) {
	if !expertise.IsValidKey(expertiseKey) {
		return 0, domainskill.ErrInvalidExpertiseKey
	}
	n, err := s.catalog.CountCuratedByExpertise(ctx, expertiseKey)
	if err != nil {
		return 0, fmt.Errorf("count curated skills: %w", err)
	}
	return n, nil
}

// Autocomplete returns catalog entries matching the normalized query
// prefix. An empty query returns an empty (non-nil) slice rather than
// an error — the frontend autocomplete widget renders "no results" in
// that case and the handler marshals the result to the JSON array [].
func (s *Service) Autocomplete(
	ctx context.Context,
	rawQuery string,
	limit int,
) ([]*domainskill.CatalogEntry, error) {
	normalized := domainskill.NormalizeSkillText(rawQuery)
	if normalized == "" {
		return []*domainskill.CatalogEntry{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	entries, err := s.catalog.SearchAutocomplete(ctx, normalized, limit)
	if err != nil {
		return nil, fmt.Errorf("search skills: %w", err)
	}
	return entries, nil
}

// GetProfileSkills returns the ordered list of skills declared on the
// organization's public profile. Always returns a non-nil slice.
func (s *Service) GetProfileSkills(
	ctx context.Context,
	orgID uuid.UUID,
) ([]*domainskill.ProfileSkill, error) {
	skills, err := s.profiles.ListByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list profile skills: %w", err)
	}
	if skills == nil {
		skills = []*domainskill.ProfileSkill{}
	}
	return skills, nil
}

// ReplaceProfileSkillsInput carries the normalized payload for
// ReplaceProfileSkills. Position is derived from slice index — callers
// pass the skills in the desired display order.
type ReplaceProfileSkillsInput struct {
	OrganizationID uuid.UUID
	SkillTexts     []string
}

// ReplaceProfileSkills atomically swaps the organization's full skill
// list. Enforces the per-org-type limit (0 = feature disabled) and
// rejects duplicates, empty texts, and skills that are not present in
// the catalog. See domain/skill/errors.go for the full error mapping.
func (s *Service) ReplaceProfileSkills(
	ctx context.Context,
	in ReplaceProfileSkillsInput,
) error {
	orgType, err := s.resolveOrgType(ctx, in.OrganizationID)
	if err != nil {
		return err
	}

	if !domainskill.IsSkillsFeatureEnabled(orgType) {
		return domainskill.ErrSkillsDisabledForOrgType
	}

	normalized, err := s.normalizeAndValidate(in.SkillTexts, orgType)
	if err != nil {
		return err
	}

	// Verify every skill already exists in the catalog. Unknown skills
	// here mean the client sent a stale cache entry or tried to bypass
	// the POST /skills flow — both are client errors.
	if err := s.ensureCatalogEntriesExist(ctx, normalized); err != nil {
		return err
	}

	profileSkills := make([]*domainskill.ProfileSkill, len(normalized))
	for i, text := range normalized {
		profileSkills[i] = &domainskill.ProfileSkill{
			OrganizationID: in.OrganizationID,
			SkillText:      text,
			Position:       i,
		}
	}
	if err := s.profiles.ReplaceForOrg(ctx, in.OrganizationID, profileSkills); err != nil {
		return fmt.Errorf("replace profile skills: %w", err)
	}
	return nil
}

// CreateUserSkillInput carries the payload for the free-form "Create X"
// path in the autocomplete dropdown. ExpertiseKeys may be nil — in
// which case the entry is created with an empty expertise_keys array
// and will not appear in any curated browse panel.
type CreateUserSkillInput struct {
	DisplayText   string
	ExpertiseKeys []string
}

// CreateUserSkill upserts a new user-contributed catalog entry. The
// display text is normalized to derive the skill_text primary key. If
// the entry already exists, the existing row is returned unchanged.
func (s *Service) CreateUserSkill(
	ctx context.Context,
	in CreateUserSkillInput,
) (*domainskill.CatalogEntry, error) {
	// Validate expertise keys up front — importing expertise here is
	// allowed (this is the app layer, not the domain layer).
	for _, key := range in.ExpertiseKeys {
		if !expertise.IsValidKey(key) {
			return nil, domainskill.ErrInvalidExpertiseKey
		}
	}

	entry, err := domainskill.NewCatalogEntry(in.DisplayText, in.DisplayText, in.ExpertiseKeys, false)
	if err != nil {
		return nil, err
	}

	// Short-circuit: if the skill already exists, return it as-is so
	// the client can proceed with attaching it to the profile. The
	// upsert below would also work, but explicit is clearer.
	existing, err := s.catalog.FindByText(ctx, entry.SkillText)
	if err != nil {
		return nil, fmt.Errorf("lookup existing skill: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	if err := s.catalog.Upsert(ctx, entry); err != nil {
		return nil, fmt.Errorf("upsert skill: %w", err)
	}
	return entry, nil
}

// resolveOrgType maps an org id to its type string. Missing orgs and
// transport errors bubble up with operation context wrapped in.
func (s *Service) resolveOrgType(
	ctx context.Context,
	orgID uuid.UUID,
) (domainskill.OrgType, error) {
	if s.orgTypes == nil {
		return "", errors.New("org type resolver not configured")
	}
	orgType, err := s.orgTypes.GetOrgType(ctx, orgID)
	if err != nil {
		return "", fmt.Errorf("resolve org type: %w", err)
	}
	return orgType, nil
}

// normalizeAndValidate applies normalization, duplicate detection, and
// the per-org-type cap. Extracted so ReplaceProfileSkills stays linear.
func (s *Service) normalizeAndValidate(
	raw []string,
	orgType domainskill.OrgType,
) ([]string, error) {
	out := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, text := range raw {
		normalized := domainskill.NormalizeSkillText(text)
		if normalized == "" {
			return nil, domainskill.ErrInvalidSkillText
		}
		if _, dup := seen[normalized]; dup {
			return nil, domainskill.ErrDuplicateSkill
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) > domainskill.MaxSkillsForOrgType(orgType) {
		return nil, domainskill.ErrTooManySkills
	}
	return out, nil
}

// ensureCatalogEntriesExist verifies every normalized skill text resolves
// to an existing catalog row. Returns ErrSkillNotFound on the first miss.
func (s *Service) ensureCatalogEntriesExist(
	ctx context.Context,
	texts []string,
) error {
	for _, text := range texts {
		entry, err := s.catalog.FindByText(ctx, text)
		if err != nil {
			return fmt.Errorf("lookup catalog skill: %w", err)
		}
		if entry == nil {
			return domainskill.ErrSkillNotFound
		}
	}
	return nil
}

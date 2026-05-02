// Package referrerprofile is the application service layer for the
// referrer ("apporteur d'affaires") persona of a provider_personal
// organization. Mirrors the shape of the freelanceprofile service
// with one behavioural difference: the read path auto-creates an
// empty row on first access so a user who just toggled
// referrer_enabled on the day of the split sees a clean blank
// profile instead of a 404.
//
// No cross-feature imports. Only depends on the referrerprofile and
// profile domain packages (the latter for the shared AvailabilityStatus
// enum).
package referrerprofile

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/domain/referrerprofile"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/search"
)

// SearchIndexPublisher is the narrow port the service uses to
// trigger a Typesense reindex after a mutation. Optional — nil
// is accepted so the search engine can be disabled without
// touching this service.
type SearchIndexPublisher interface {
	PublishReindex(ctx context.Context, orgID uuid.UUID, persona search.Persona) error
}

// Service orchestrates the referrer profile use cases.
type Service struct {
	profiles    repository.ReferrerProfileRepository
	searchIndex SearchIndexPublisher

	// Reputation aggregate deps — optional. Wired via
	// WithReputationDeps, consumed by GetReferrerReputation. Nil is
	// an accepted state: the reputation endpoint returns an empty
	// aggregate in that case, keeping the profile feature removable.
	referrals repository.ReferralRepository
	// proposals is narrowed to ProposalReader — the reputation
	// aggregate only batch-loads proposals by id (GetByIDs).
	proposals repository.ProposalReader
	reviews   repository.ReviewRepository
	users     repository.UserBatchReader
}

// NewService wires the referrer profile service with its single
// dependency.
func NewService(profiles repository.ReferrerProfileRepository) *Service {
	return &Service{profiles: profiles}
}

// WithSearchIndexPublisher attaches a Typesense publisher that
// emits a `search.reindex` event on every mutation. Mirrors the
// freelance profile service — a builder method keeps NewService's
// signature stable.
func (s *Service) WithSearchIndexPublisher(publisher SearchIndexPublisher) *Service {
	if s == nil {
		return nil
	}
	clone := *s
	clone.searchIndex = publisher
	return &clone
}

// publishReindex is the best-effort wrapper that swallows the
// error (logged only) so a degraded search engine cannot block a
// profile update.
func (s *Service) publishReindex(ctx context.Context, orgID uuid.UUID) {
	if s.searchIndex == nil {
		return
	}
	if err := s.searchIndex.PublishReindex(ctx, orgID, search.PersonaReferrer); err != nil {
		slog.Warn("referrer profile: search reindex publish failed",
			"org_id", orgID, "error", err)
	}
}

// GetByOrgID returns the hydrated referrer profile view for the
// given org. The repository lazily creates a default row when
// none exists, so this method never surfaces
// referrerprofile.ErrProfileNotFound — callers can rely on a
// non-nil view when the call succeeds.
func (s *Service) GetByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.ReferrerProfileView, error) {
	view, err := s.profiles.GetOrCreateByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get referrer profile: %w", err)
	}
	return view, nil
}

// GetReferrerProfileIDByOrgID resolves the surrogate profile ID
// from an organization ID. Used by the pricing handler — mirrors
// GetFreelanceProfileIDByOrgID on the freelance side. Lazily
// creates a default referrer profile if the row does not yet
// exist, so the pricing handler can write a row for a fresh
// provider without a separate "initialize profile" call.
func (s *Service) GetReferrerProfileIDByOrgID(ctx context.Context, orgID uuid.UUID) (uuid.UUID, error) {
	view, err := s.profiles.GetOrCreateByOrgID(ctx, orgID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve referrer profile id: %w", err)
	}
	return view.Profile.ID, nil
}

// UpdateCoreInput groups the core text edits (title, about, video).
type UpdateCoreInput struct {
	Title    string
	About    string
	VideoURL string
}

// UpdateCore writes the triplet atomically and returns the
// refreshed view. Whitespace is trimmed.
func (s *Service) UpdateCore(ctx context.Context, orgID uuid.UUID, input UpdateCoreInput) (*repository.ReferrerProfileView, error) {
	// Ensure the row exists first — the user may reach the edit
	// endpoint without having performed a read. GetOrCreate is
	// cheap and guarantees the subsequent UpdateCore hits a row.
	if _, err := s.profiles.GetOrCreateByOrgID(ctx, orgID); err != nil {
		return nil, fmt.Errorf("update referrer profile core: ensure row: %w", err)
	}

	title := strings.TrimSpace(input.Title)
	about := strings.TrimSpace(input.About)
	videoURL := strings.TrimSpace(input.VideoURL)

	if err := s.profiles.UpdateCore(ctx, orgID, title, about, videoURL); err != nil {
		return nil, fmt.Errorf("update referrer profile core: %w", err)
	}
	s.publishReindex(ctx, orgID)
	return s.GetByOrgID(ctx, orgID)
}

// UpdateAvailability writes a single availability value. Validates
// the raw string via profile.ParseAvailabilityStatus.
func (s *Service) UpdateAvailability(ctx context.Context, orgID uuid.UUID, raw string) (*repository.ReferrerProfileView, error) {
	status, err := profile.ParseAvailabilityStatus(raw)
	if err != nil {
		return nil, fmt.Errorf("update referrer profile availability: %w", err)
	}
	if _, err := s.profiles.GetOrCreateByOrgID(ctx, orgID); err != nil {
		return nil, fmt.Errorf("update referrer profile availability: ensure row: %w", err)
	}
	if err := s.profiles.UpdateAvailability(ctx, orgID, status); err != nil {
		return nil, fmt.Errorf("update referrer profile availability: %w", err)
	}
	s.publishReindex(ctx, orgID)
	return s.GetByOrgID(ctx, orgID)
}

// UpdateExpertise replaces the referrer expertise list atomically.
// Normalization (trim, dedup) mirrors the freelance service so both
// personas carry a canonical shape.
func (s *Service) UpdateExpertise(ctx context.Context, orgID uuid.UUID, domains []string) (*repository.ReferrerProfileView, error) {
	if _, err := s.profiles.GetOrCreateByOrgID(ctx, orgID); err != nil {
		return nil, fmt.Errorf("update referrer profile expertise: ensure row: %w", err)
	}
	normalized := normalizeExpertise(domains)
	if err := s.profiles.UpdateExpertiseDomains(ctx, orgID, normalized); err != nil {
		return nil, fmt.Errorf("update referrer profile expertise: %w", err)
	}
	s.publishReindex(ctx, orgID)
	return s.GetByOrgID(ctx, orgID)
}

// normalizeExpertise trims, dedups, and drops empty strings.
// Duplicated in both persona services rather than extracted because
// the two services must stay fully independent at the package level
// — if a future refactor needs a shared helper, the rule of three
// kicks in only after a third call site appears.
func normalizeExpertise(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, v := range in {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

// Compile-time check that the referrerprofile.ErrProfileNotFound
// sentinel still exists, so a rename in the domain package fails
// this file rather than silently drifting.
var _ = referrerprofile.ErrProfileNotFound

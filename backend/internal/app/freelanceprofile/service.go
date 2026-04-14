// Package freelanceprofile is the application service layer for
// the freelance persona of a provider_personal organization. It is
// an intentionally thin orchestrator over the
// repository.FreelanceProfileRepository port: most of the
// validation lives in the domain layer and most of the persistence
// lives in the adapter layer.
//
// No cross-feature imports — the service only depends on the
// freelanceprofile and profile domain packages (the latter for the
// shared AvailabilityStatus enum). It does not know anything about
// the referrer persona, organizations, expertise catalog, or skills.
package freelanceprofile

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/freelanceprofile"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/port/repository"
)

// Service orchestrates the freelance profile use cases: read,
// update core, update availability, update expertise.
type Service struct {
	profiles repository.FreelanceProfileRepository
}

// NewService wires the freelance profile service with its single
// dependency. The repository handle is required — there is no sane
// default.
func NewService(profiles repository.FreelanceProfileRepository) *Service {
	return &Service{profiles: profiles}
}

// GetByOrgID returns the hydrated freelance profile view (persona
// columns + shared block) for the given org. Used by the OWNER
// side (authenticated self-read and every mutation) — lazily
// creates a default row when none exists so new provider_personal
// accounts registered after the split migration transparently get
// one on first access instead of hitting a 404.
func (s *Service) GetByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error) {
	view, err := s.profiles.GetOrCreateByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get freelance profile: %w", err)
	}
	return view, nil
}

// GetPublicByOrgID is the read path for the public /freelance-profiles/{id}
// endpoint. Strict: returns freelanceprofile.ErrProfileNotFound when
// the row does not exist instead of lazily creating one. Viewing
// someone else's profile must never mutate its storage.
func (s *Service) GetPublicByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error) {
	view, err := s.profiles.GetByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get public freelance profile: %w", err)
	}
	return view, nil
}

// GetFreelanceProfileIDByOrgID resolves the surrogate profile ID
// from an organization ID. Used by the pricing handler, which
// receives an org ID from the JWT context but needs a profile ID
// to hit the freelance_pricing table. Exposed as a dedicated
// method (rather than making the caller do a full GetByOrgID and
// extract .Profile.ID) so the handler side stays agnostic of the
// repository.FreelanceProfileView shape. Uses the lazy path so a
// provider_personal owner who opens the pricing editor before
// their profile row exists still gets a clean response.
func (s *Service) GetFreelanceProfileIDByOrgID(ctx context.Context, orgID uuid.UUID) (uuid.UUID, error) {
	view, err := s.profiles.GetOrCreateByOrgID(ctx, orgID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve freelance profile id: %w", err)
	}
	return view.Profile.ID, nil
}

// UpdateCoreInput groups the core text edits. Kept as a struct so
// the service signature stays under the 4-parameter cap and gives
// the handler a stable payload type.
type UpdateCoreInput struct {
	Title    string
	About    string
	VideoURL string
}

// UpdateCore writes the title / about / video_url triplet and
// returns the refreshed view. Whitespace is trimmed around every
// string so stray newlines from a client paste do not survive the
// round-trip.
func (s *Service) UpdateCore(ctx context.Context, orgID uuid.UUID, input UpdateCoreInput) (*repository.FreelanceProfileView, error) {
	title := strings.TrimSpace(input.Title)
	about := strings.TrimSpace(input.About)
	videoURL := strings.TrimSpace(input.VideoURL)

	if err := s.profiles.UpdateCore(ctx, orgID, title, about, videoURL); err != nil {
		return nil, fmt.Errorf("update freelance profile core: %w", err)
	}
	return s.GetByOrgID(ctx, orgID)
}

// UpdateAvailability writes a single availability value. The
// input is validated via profile.ParseAvailabilityStatus — an
// empty or unknown string surfaces as profile.ErrInvalidAvailabilityStatus.
func (s *Service) UpdateAvailability(ctx context.Context, orgID uuid.UUID, raw string) (*repository.FreelanceProfileView, error) {
	status, err := profile.ParseAvailabilityStatus(raw)
	if err != nil {
		return nil, fmt.Errorf("update freelance profile availability: %w", err)
	}
	if err := s.profiles.UpdateAvailability(ctx, orgID, status); err != nil {
		return nil, fmt.Errorf("update freelance profile availability: %w", err)
	}
	return s.GetByOrgID(ctx, orgID)
}

// UpdateExpertise replaces the freelance expertise list atomically.
// Normalization (trim, dedup) is applied here so the repository
// writes a canonical shape. The domain-level expertise catalog is
// NOT enforced at this layer — that is the job of the dedicated
// expertise service when the handler wires it in. This keeps
// freelanceprofile independent of domain/expertise.
func (s *Service) UpdateExpertise(ctx context.Context, orgID uuid.UUID, domains []string) (*repository.FreelanceProfileView, error) {
	normalized := normalizeExpertise(domains)
	if err := s.profiles.UpdateExpertiseDomains(ctx, orgID, normalized); err != nil {
		return nil, fmt.Errorf("update freelance profile expertise: %w", err)
	}
	return s.GetByOrgID(ctx, orgID)
}

// normalizeExpertise trims whitespace, drops empty strings, and
// deduplicates preserving first-occurrence order. Nil input
// yields an empty (non-nil) slice so downstream serialization
// yields `[]` rather than null.
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

// Ensure the service returns domain-level errors verbatim so the
// handler layer can errors.Is against both freelanceprofile and
// profile sentinel values. Compile-time check that the imported
// ErrProfileNotFound still exists, so a rename in the domain
// package fails this file rather than silently drifting.
var _ = freelanceprofile.ErrProfileNotFound

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
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/freelanceprofile"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/search"
)

// SearchIndexPublisher is the narrow port the service uses to
// trigger a Typesense reindex after a profile mutation. Optional —
// a nil publisher is accepted as a no-op so the search engine can
// be disabled (or removed entirely) without breaking the freelance
// profile flow. Defined locally to keep this package free of
// cross-feature imports: it doesn't know about search.Publisher,
// only about the method it calls.
//
// The Tx variant is used by the outbox pattern (BUG-05): the
// pending_events INSERT must commit alongside the profile UPDATE
// so a DB blip between the two writes can never produce a
// permanently-stale Typesense index. The legacy hors-tx
// PublishReindex is kept for non-mutation paths only.
type SearchIndexPublisher interface {
	PublishReindex(ctx context.Context, orgID uuid.UUID, persona search.Persona) error
	PublishReindexTx(ctx context.Context, tx *sql.Tx, orgID uuid.UUID, persona search.Persona) error
}

// Service orchestrates the freelance profile use cases: read,
// update core, update availability, update expertise.
type Service struct {
	profiles    repository.FreelanceProfileRepository
	searchIndex SearchIndexPublisher

	// txRunner is required for the outbox-aware mutation flow
	// (BUG-05). When set, every UpdateXxx method begins a
	// transaction, calls the tx-bound repo method, schedules the
	// search.reindex pending event in the same tx, and commits.
	// When nil, the mutations fall back to the pre-outbox
	// behaviour (separate writes, hors-tx schedule) — this is
	// needed for tests that drive the service without a database.
	txRunner repository.TxRunner
}

// NewService wires the freelance profile service with its single
// dependency. The repository handle is required — there is no sane
// default.
func NewService(profiles repository.FreelanceProfileRepository) *Service {
	return &Service{profiles: profiles}
}

// WithSearchIndexPublisher returns a copy of the service with a
// Typesense publisher attached. Every subsequent mutation will
// emit a `search.reindex` event in the SAME transaction as the
// repository UPDATE — see the doc on Service.txRunner — which
// guarantees Postgres and Typesense never permanently drift apart.
//
// Using a builder method instead of a bigger constructor keeps
// the NewService signature stable for the 6+ call sites that
// already exist.
func (s *Service) WithSearchIndexPublisher(publisher SearchIndexPublisher) *Service {
	if s == nil {
		return nil
	}
	clone := *s
	clone.searchIndex = publisher
	return &clone
}

// WithTxRunner attaches the database transaction runner that wires
// the outbox pattern (BUG-05). Once set, every mutation runs
// repo.UpdateXxxTx and publisher.PublishReindexTx inside the same
// *sql.Tx — so a Schedule failure rolls back the profile UPDATE
// (and vice versa).
func (s *Service) WithTxRunner(runner repository.TxRunner) *Service {
	if s == nil {
		return nil
	}
	clone := *s
	clone.txRunner = runner
	return &clone
}

// publishReindex is the best-effort wrapper around the legacy
// hors-tx publisher path. Kept for backwards compatibility with
// non-mutation flows (none today on this service) and for the
// fallback when no TxRunner is wired (test setup). Failures are
// logged because we are explicitly outside the atomic boundary.
func (s *Service) publishReindex(ctx context.Context, orgID uuid.UUID) {
	if s.searchIndex == nil {
		return
	}
	if err := s.searchIndex.PublishReindex(ctx, orgID, search.PersonaFreelance); err != nil {
		slog.Warn("freelance profile: search reindex publish failed",
			"org_id", orgID, "error", err)
	}
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
//
// When a TxRunner is wired, the UPDATE and the matching
// search.reindex pending event commit in a single transaction so
// the search index can never drift permanently from Postgres
// (BUG-05). When the TxRunner is absent (some test setups), the
// pre-outbox behaviour is preserved: separate writes, best-effort
// schedule.
func (s *Service) UpdateCore(ctx context.Context, orgID uuid.UUID, input UpdateCoreInput) (*repository.FreelanceProfileView, error) {
	title := strings.TrimSpace(input.Title)
	about := strings.TrimSpace(input.About)
	videoURL := strings.TrimSpace(input.VideoURL)

	if s.txRunner != nil && s.searchIndex != nil {
		if err := s.txRunner.RunInTx(ctx, func(tx *sql.Tx) error {
			if err := s.profiles.UpdateCoreTx(ctx, tx, orgID, title, about, videoURL); err != nil {
				return err
			}
			return s.searchIndex.PublishReindexTx(ctx, tx, orgID, search.PersonaFreelance)
		}); err != nil {
			return nil, fmt.Errorf("update freelance profile core: %w", err)
		}
		return s.GetByOrgID(ctx, orgID)
	}

	if err := s.profiles.UpdateCore(ctx, orgID, title, about, videoURL); err != nil {
		return nil, fmt.Errorf("update freelance profile core: %w", err)
	}
	s.publishReindex(ctx, orgID)
	return s.GetByOrgID(ctx, orgID)
}

// UpdateAvailability writes a single availability value. The
// input is validated via profile.ParseAvailabilityStatus — an
// empty or unknown string surfaces as profile.ErrInvalidAvailabilityStatus.
//
// Outbox-aware: see UpdateCore for the rationale.
func (s *Service) UpdateAvailability(ctx context.Context, orgID uuid.UUID, raw string) (*repository.FreelanceProfileView, error) {
	status, err := profile.ParseAvailabilityStatus(raw)
	if err != nil {
		return nil, fmt.Errorf("update freelance profile availability: %w", err)
	}

	if s.txRunner != nil && s.searchIndex != nil {
		if err := s.txRunner.RunInTx(ctx, func(tx *sql.Tx) error {
			if err := s.profiles.UpdateAvailabilityTx(ctx, tx, orgID, status); err != nil {
				return err
			}
			return s.searchIndex.PublishReindexTx(ctx, tx, orgID, search.PersonaFreelance)
		}); err != nil {
			return nil, fmt.Errorf("update freelance profile availability: %w", err)
		}
		return s.GetByOrgID(ctx, orgID)
	}

	if err := s.profiles.UpdateAvailability(ctx, orgID, status); err != nil {
		return nil, fmt.Errorf("update freelance profile availability: %w", err)
	}
	s.publishReindex(ctx, orgID)
	return s.GetByOrgID(ctx, orgID)
}

// UpdateExpertise replaces the freelance expertise list atomically.
// Normalization (trim, dedup) is applied here so the repository
// writes a canonical shape. The domain-level expertise catalog is
// NOT enforced at this layer — that is the job of the dedicated
// expertise service when the handler wires it in. This keeps
// freelanceprofile independent of domain/expertise.
//
// Outbox-aware: see UpdateCore for the rationale.
func (s *Service) UpdateExpertise(ctx context.Context, orgID uuid.UUID, domains []string) (*repository.FreelanceProfileView, error) {
	normalized := normalizeExpertise(domains)

	if s.txRunner != nil && s.searchIndex != nil {
		if err := s.txRunner.RunInTx(ctx, func(tx *sql.Tx) error {
			if err := s.profiles.UpdateExpertiseDomainsTx(ctx, tx, orgID, normalized); err != nil {
				return err
			}
			return s.searchIndex.PublishReindexTx(ctx, tx, orgID, search.PersonaFreelance)
		}); err != nil {
			return nil, fmt.Errorf("update freelance profile expertise: %w", err)
		}
		return s.GetByOrgID(ctx, orgID)
	}

	if err := s.profiles.UpdateExpertiseDomains(ctx, orgID, normalized); err != nil {
		return nil, fmt.Errorf("update freelance profile expertise: %w", err)
	}
	s.publishReindex(ctx, orgID)
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

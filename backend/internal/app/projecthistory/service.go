// Package projecthistory orchestrates reading a provider's completed missions
// together with their reviews. It is the single point where the proposal and
// review features meet for this read-only view, preserving the feature
// isolation principle (neither repo knows about the other).
package projecthistory

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	proposaldomain "marketplace-backend/internal/domain/proposal"
	reviewdomain "marketplace-backend/internal/domain/review"
	"marketplace-backend/internal/port/repository"
)

// Currency is hardcoded to EUR for v1.
const Currency = "EUR"

// Entry is one completed mission enriched with its optional review.
type Entry struct {
	ProposalID  uuid.UUID
	Title       string // empty when the client opted out via the review form
	Amount      int64
	Currency    string
	CompletedAt time.Time
	Review      *reviewdomain.Review // nil when the client has not yet reviewed
}

// ServiceDeps groups the repositories needed by the service.
//
// Proposals is narrowed to ProposalReader — the public project history
// only ever lists completed missions; it never mutates a proposal.
type ServiceDeps struct {
	Proposals repository.ProposalReader
	Reviews   repository.ReviewRepository
}

// Service orchestrates the project history read.
type Service struct {
	proposals repository.ProposalReader
	reviews   repository.ReviewRepository
}

// NewService creates a new project history service.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		proposals: deps.Proposals,
		reviews:   deps.Reviews,
	}
}

// ListByOrganization returns the completed missions of the given
// organization (provider side) with the associated (optional) review
// joined in-memory. Used to render an org's public project history.
func (s *Service) ListByOrganization(ctx context.Context, orgID uuid.UUID, cursor string, limit int) ([]Entry, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	proposals, nextCursor, err := s.proposals.ListCompletedByOrganization(ctx, orgID, cursor, limit)
	if err != nil {
		return nil, "", fmt.Errorf("list completed proposals: %w", err)
	}
	if len(proposals) == 0 {
		return []Entry{}, "", nil
	}

	ids := make([]uuid.UUID, len(proposals))
	for i, p := range proposals {
		ids[i] = p.ID
	}

	// Only the client→provider review belongs on the provider's public
	// project history. The provider→client side is filtered out at the
	// SQL level so the map keyed by proposal_id cannot collide on
	// proposals that have both sides submitted.
	reviews, err := s.reviews.GetByProposalIDs(ctx, ids, reviewdomain.SideClientToProvider)
	if err != nil {
		return nil, "", fmt.Errorf("get reviews by proposal ids: %w", err)
	}

	entries := make([]Entry, 0, len(proposals))
	for _, p := range proposals {
		entry := entryFromProposal(p)
		if rv, ok := reviews[p.ID]; ok {
			entry.Review = rv
			if !rv.TitleVisible {
				entry.Title = ""
			}
		}
		entries = append(entries, entry)
	}

	return entries, nextCursor, nil
}

// ListByClientOrganization is the client-side mirror of
// ListByOrganization: it returns completed deals where the given
// organization is the CLIENT (not the provider), together with the
// provider→client review attached to each one. Used by the public
// client profile's project-history section. Bounded fetch (limit
// capped at 100) — no cursor because the surface shows a small
// window.
func (s *Service) ListByClientOrganization(ctx context.Context, orgID uuid.UUID, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	proposals, err := s.proposals.ListCompletedByClientOrganization(ctx, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("list completed by client organization: %w", err)
	}
	if len(proposals) == 0 {
		return []Entry{}, nil
	}

	ids := make([]uuid.UUID, len(proposals))
	for i, p := range proposals {
		ids[i] = p.ID
	}

	// The provider→client review is the one that belongs on the
	// client's public project history — symmetric counterpart of the
	// client→provider side used by ListByOrganization. Filter at the
	// SQL level so the map keyed by proposal_id cannot collide.
	reviews, err := s.reviews.GetByProposalIDs(ctx, ids, reviewdomain.SideProviderToClient)
	if err != nil {
		return nil, fmt.Errorf("get reviews by proposal ids: %w", err)
	}

	entries := make([]Entry, 0, len(proposals))
	for _, p := range proposals {
		entry := entryFromProposal(p)
		if rv, ok := reviews[p.ID]; ok {
			entry.Review = rv
			if !rv.TitleVisible {
				entry.Title = ""
			}
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func entryFromProposal(p *proposaldomain.Proposal) Entry {
	var completedAt time.Time
	if p.CompletedAt != nil {
		completedAt = *p.CompletedAt
	}
	return Entry{
		ProposalID:  p.ID,
		Title:       p.Title,
		Amount:      p.Amount,
		Currency:    Currency,
		CompletedAt: completedAt,
	}
}

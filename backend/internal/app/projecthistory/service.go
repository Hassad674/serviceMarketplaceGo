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
type ServiceDeps struct {
	Proposals repository.ProposalRepository
	Reviews   repository.ReviewRepository
}

// Service orchestrates the project history read.
type Service struct {
	proposals repository.ProposalRepository
	reviews   repository.ReviewRepository
}

// NewService creates a new project history service.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		proposals: deps.Proposals,
		reviews:   deps.Reviews,
	}
}

// ListByProvider returns the completed missions of a provider with the
// associated (optional) review joined in-memory.
func (s *Service) ListByProvider(ctx context.Context, providerID uuid.UUID, cursor string, limit int) ([]Entry, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	proposals, nextCursor, err := s.proposals.ListCompletedByProvider(ctx, providerID, cursor, limit)
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

	reviews, err := s.reviews.GetByProposalIDs(ctx, ids)
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

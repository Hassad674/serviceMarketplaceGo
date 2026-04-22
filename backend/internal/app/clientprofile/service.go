// Package clientprofile orchestrates the public read of an org's
// client-facing profile: the client description, the aggregate client
// stats (total spent, review count, average rating, projects
// completed as client) and the recent project history on the client
// side. It is the symmetric counterpart to projecthistory (which
// handles the provider side) and lives in its own package so a
// consumer that only needs the client profile view does not drag in
// the provider aggregates.
//
// Reviews are intentionally not materialized as a top-level list on
// the public aggregate — the review attached to each completed deal
// is surfaced inline on project_history, mirroring the provider
// profile's single-section UX.
package clientprofile

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/profile"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/port/repository"
)

// Currency is hardcoded to EUR for v1, matching projecthistory.
const Currency = "EUR"

// DefaultProjectHistoryLimit is the cap on project-history rows
// returned by the public client profile. Capped small because the
// endpoint is public and must not scan unbounded result sets.
const DefaultProjectHistoryLimit = 20

// ProjectHistoryEntry is one completed deal where the org was the
// client, with the provider's identity resolved so the public page
// can render the counterparty.
type ProjectHistoryEntry struct {
	ProposalID  uuid.UUID
	Title       string
	Amount      int64
	Currency    string
	CompletedAt time.Time
	Provider    *profile.PublicProfile // may be nil when the provider org is gone
}

// PublicClientProfile is the aggregated shape surfaced by the public
// /api/v1/clients/{orgId} endpoint. Every field is pre-allocated (no
// nil slices) so the JSON shape stays stable across requests — a
// brand-new org with no deals still serializes as { …,
// "project_history": [] }.
type PublicClientProfile struct {
	OrganizationID            uuid.UUID
	Type                      string
	CompanyName               string
	AvatarURL                 string
	ClientDescription         string
	TotalSpent                int64
	ReviewCount               int
	AverageRating             float64
	ProjectsCompletedAsClient int
	ProjectHistory            []ProjectHistoryEntry
}

// ServiceDeps groups the repositories the service orchestrates. Every
// dependency is a port interface so tests can inject minimal mocks.
type ServiceDeps struct {
	Organizations repository.OrganizationRepository
	Profiles      repository.ProfileRepository
	Proposals     repository.ProposalRepository
	Reviews       repository.ReviewRepository
}

// Service owns the public client profile read use case.
type Service struct {
	organizations repository.OrganizationRepository
	profiles      repository.ProfileRepository
	proposals     repository.ProposalRepository
	reviews       repository.ReviewRepository
}

// NewService wires the service with its dependencies.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		organizations: deps.Organizations,
		profiles:      deps.Profiles,
		proposals:     deps.Proposals,
		reviews:       deps.Reviews,
	}
}

// GetPublicClientProfile composes the full public client profile
// payload for the given org. Flow:
//
//  1. Resolve the organization (identity + type). A missing org
//     surfaces profile.ErrProfileNotFound so the handler returns 404
//     without leaking the distinction between "org exists but has no
//     profile" and "org does not exist".
//  2. Reject provider_personal orgs with the same ErrProfileNotFound.
//     v1 exposes the client profile only to agency and enterprise.
//     Hiding the 403 behind a 404 prevents a probe from discovering
//     which orgs are freelancers; extending the feature to
//     provider_personal later is a one-line flip in isClientProfileExposed.
//  3. Fetch the profile row for the client_description + photo fallback.
//     A missing profile (legacy pre-Tier 1 org) is acceptable — defaults
//     are applied so the payload is still well-formed.
//  4. Aggregate the stats (total spent, review count, average rating,
//     projects completed) in parallel-friendly sequential calls. Each
//     is bounded and hits a dedicated index.
//  5. Load the project history capped at DefaultProjectHistoryLimit.
//     Reviews are not fetched as a top-level list — the frontend
//     renders one unified "Completed projects" section and reads the
//     review inline from project_history. GetClientAverageRating still
//     runs because its count + average feed the header stats block.
func (s *Service) GetPublicClientProfile(ctx context.Context, orgID uuid.UUID) (*PublicClientProfile, error) {
	org, err := s.organizations.FindByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get public client profile: resolve org: %w", err)
	}
	if !isClientProfileExposed(org.Type) {
		// Deliberately return ErrProfileNotFound — same opaque shape the
		// handler already maps to 404. See doc comment.
		return nil, profile.ErrProfileNotFound
	}

	desc, avatar := s.loadClientProfileText(ctx, orgID)
	totalSpent, err := s.proposals.SumPaidByClientOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get public client profile: sum paid: %w", err)
	}
	avgRating, err := s.reviews.GetClientAverageRating(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get public client profile: avg rating: %w", err)
	}

	history, err := s.loadProjectHistory(ctx, orgID)
	if err != nil {
		return nil, err
	}

	return &PublicClientProfile{
		OrganizationID:            org.ID,
		Type:                      string(org.Type),
		CompanyName:               org.Name,
		AvatarURL:                 avatar,
		ClientDescription:         desc,
		TotalSpent:                totalSpent,
		ReviewCount:               avgRating.Count,
		AverageRating:             avgRating.Average,
		ProjectsCompletedAsClient: len(history),
		ProjectHistory:            history,
	}, nil
}

// ClientStats is the computed-aggregate slice of the client profile:
// total spent, provider→client review count + average rating, and the
// number of projects completed as client. Used by the private
// /api/v1/profile endpoint to decorate the owner's response with the
// same numbers the public /api/v1/clients/{orgId} endpoint surfaces —
// the owner sees their own live stats without a second round-trip.
type ClientStats struct {
	TotalSpent                int64
	ReviewCount               int
	AverageRating             float64
	ProjectsCompletedAsClient int
}

// GetStats returns the authenticated owner's client-side stats. Safe
// to call for any org type — provider_personal orgs simply get zeros
// across the board (no projects completed as client, no reviews
// received as client). The handler decides whether to surface the
// block based on org type; the service stays neutral so the
// computation can be reused later when v1's org-type gating relaxes.
func (s *Service) GetStats(ctx context.Context, orgID uuid.UUID) (*ClientStats, error) {
	totalSpent, err := s.proposals.SumPaidByClientOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get client stats: sum paid: %w", err)
	}
	avg, err := s.reviews.GetClientAverageRating(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get client stats: avg rating: %w", err)
	}
	completed, err := s.proposals.ListCompletedByClientOrganization(ctx, orgID, DefaultProjectHistoryLimit)
	if err != nil {
		return nil, fmt.Errorf("get client stats: list completed: %w", err)
	}
	return &ClientStats{
		TotalSpent:                totalSpent,
		ReviewCount:               avg.Count,
		AverageRating:             avg.Average,
		ProjectsCompletedAsClient: len(completed),
	}, nil
}

// loadClientProfileText fetches the client_description and falls back
// to the empty string / legacy photo if the profile row is missing.
// A missing profile is not fatal — the public client page still
// renders.
func (s *Service) loadClientProfileText(ctx context.Context, orgID uuid.UUID) (description string, avatarURL string) {
	p, err := s.profiles.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return "", ""
	}
	return p.ClientDescription, p.PhotoURL
}

// loadProjectHistory fetches the most recent completed deals where
// the org was the client and enriches each with the provider's org
// public profile for counterparty display. The provider lookup is
// batched via ProfileRepository.OrgProfilesByUserIDs so we never
// trigger an N+1 across the result set.
func (s *Service) loadProjectHistory(ctx context.Context, orgID uuid.UUID) ([]ProjectHistoryEntry, error) {
	proposals, err := s.proposals.ListCompletedByClientOrganization(ctx, orgID, DefaultProjectHistoryLimit)
	if err != nil {
		return nil, fmt.Errorf("get public client profile: list completed: %w", err)
	}
	if len(proposals) == 0 {
		return []ProjectHistoryEntry{}, nil
	}

	providerIDs := make([]uuid.UUID, 0, len(proposals))
	for _, p := range proposals {
		providerIDs = append(providerIDs, p.ProviderID)
	}
	providerByUser, err := s.profiles.OrgProfilesByUserIDs(ctx, providerIDs)
	if err != nil {
		// Degrade gracefully — the deal rows are still useful without the
		// counterparty decoration.
		providerByUser = map[uuid.UUID]*profile.PublicProfile{}
	}

	entries := make([]ProjectHistoryEntry, 0, len(proposals))
	for _, p := range proposals {
		entries = append(entries, entryFromProposal(p, providerByUser[p.ProviderID]))
	}
	return entries, nil
}

// entryFromProposal projects a domain proposal + an (optional) provider
// public profile onto the ProjectHistoryEntry shape. Kept as a pure
// function so the mapping can be unit-tested in isolation.
func entryFromProposal(p *proposaldomain.Proposal, provider *profile.PublicProfile) ProjectHistoryEntry {
	var completedAt time.Time
	if p.CompletedAt != nil {
		completedAt = *p.CompletedAt
	}
	return ProjectHistoryEntry{
		ProposalID:  p.ID,
		Title:       p.Title,
		Amount:      p.Amount,
		Currency:    Currency,
		CompletedAt: completedAt,
		Provider:    provider,
	}
}

// isClientProfileExposed is the single source of truth for which org
// types expose a public client profile. v1 restricts the feature to
// agency + enterprise; provider_personal flips to true here when the
// feature is opened up.
func isClientProfileExposed(t organization.OrgType) bool {
	switch t {
	case organization.OrgTypeAgency, organization.OrgTypeEnterprise:
		return true
	}
	return false
}

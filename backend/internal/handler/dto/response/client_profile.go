package response

import (
	"time"

	"marketplace-backend/internal/app/clientprofile"
)

// ClientProjectHistoryProvider is the compact counterparty descriptor
// embedded in each client-side project history entry. Surfaces only
// the public identity bits (org id + display name + avatar) — private
// profile fields never leak onto this surface.
type ClientProjectHistoryProvider struct {
	OrganizationID string `json:"organization_id"`
	DisplayName    string `json:"display_name"`
	AvatarURL      string `json:"avatar_url"`
}

// ClientProjectHistoryEntry is the public API shape of one completed
// deal where the org was the client. Title is empty when the provider
// did not elect to share it in their review.
type ClientProjectHistoryEntry struct {
	ProposalID  string                        `json:"proposal_id"`
	Title       string                        `json:"title"`
	Amount      int64                         `json:"amount"`
	CompletedAt string                        `json:"completed_at"`
	Provider    *ClientProjectHistoryProvider `json:"provider"`
}

// PublicClientProfileResponse is the public read shape for the
// /api/v1/clients/{orgId} endpoint. Every slice and every nullable is
// explicitly modeled so the JSON shape is stable across requests.
//
// Reviews are NOT surfaced as a top-level list — the review attached
// to each completed deal travels inline on project_history[].review,
// and the frontend renders one unified "Completed projects" section
// (mirroring the provider profile). ReviewCount + AverageRating stay
// at the top level because they feed the header stats block.
type PublicClientProfileResponse struct {
	OrganizationID            string                      `json:"organization_id"`
	Type                      string                      `json:"type"`
	CompanyName               string                      `json:"company_name"`
	AvatarURL                 string                      `json:"avatar_url"`
	ClientDescription         string                      `json:"client_description"`
	TotalSpent                int64                       `json:"total_spent"`
	ReviewCount               int                         `json:"review_count"`
	AverageRating             float64                     `json:"average_rating"`
	ProjectsCompletedAsClient int                         `json:"projects_completed_as_client"`
	ProjectHistory            []ClientProjectHistoryEntry `json:"project_history"`
}

// NewPublicClientProfileResponse builds the public response envelope
// from the app-layer aggregate. Every slice is pre-allocated (non-nil
// even when empty) so the JSON output does not alternate between []
// and null across requests.
func NewPublicClientProfileResponse(p *clientprofile.PublicClientProfile) PublicClientProfileResponse {
	history := make([]ClientProjectHistoryEntry, 0, len(p.ProjectHistory))
	for _, e := range p.ProjectHistory {
		history = append(history, newClientProjectHistoryEntry(e))
	}
	return PublicClientProfileResponse{
		OrganizationID:            p.OrganizationID.String(),
		Type:                      p.Type,
		CompanyName:               p.CompanyName,
		AvatarURL:                 p.AvatarURL,
		ClientDescription:         p.ClientDescription,
		TotalSpent:                p.TotalSpent,
		ReviewCount:               p.ReviewCount,
		AverageRating:             p.AverageRating,
		ProjectsCompletedAsClient: p.ProjectsCompletedAsClient,
		ProjectHistory:            history,
	}
}

// newClientProjectHistoryEntry maps one app-layer entry to the API
// shape. Nil provider stays nil so the frontend can render a generic
// placeholder for deleted counterparties without a defensive
// guard on other fields.
func newClientProjectHistoryEntry(e clientprofile.ProjectHistoryEntry) ClientProjectHistoryEntry {
	entry := ClientProjectHistoryEntry{
		ProposalID:  e.ProposalID.String(),
		Title:       e.Title,
		Amount:      e.Amount,
		CompletedAt: e.CompletedAt.Format(time.RFC3339),
	}
	if e.Provider != nil {
		entry.Provider = &ClientProjectHistoryProvider{
			OrganizationID: e.Provider.OrganizationID.String(),
			DisplayName:    e.Provider.Name,
			AvatarURL:      e.Provider.PhotoURL,
		}
	}
	return entry
}

// NewProfileClientSection maps the client-stats aggregate to the DTO
// used inline by GET /api/v1/profile. Nil input returns nil so the
// caller can safely chain into ProfileResponse.WithClientSection.
func NewProfileClientSection(stats *clientprofile.ClientStats) *ProfileClientSection {
	if stats == nil {
		return nil
	}
	return &ProfileClientSection{
		TotalSpent:                stats.TotalSpent,
		ReviewCount:               stats.ReviewCount,
		AverageRating:             stats.AverageRating,
		ProjectsCompletedAsClient: stats.ProjectsCompletedAsClient,
	}
}

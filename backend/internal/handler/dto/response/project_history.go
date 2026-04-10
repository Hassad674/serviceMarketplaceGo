package response

import (
	"time"

	"marketplace-backend/internal/app/projecthistory"
)

// ProjectHistoryEntry is the API representation of one completed mission
// with its optional review. Title is empty when the client opted out of
// sharing the mission title in the review form.
type ProjectHistoryEntry struct {
	ProposalID  string          `json:"proposal_id"`
	Title       string          `json:"title"`
	Amount      int64           `json:"amount"`
	Currency    string          `json:"currency"`
	CompletedAt string          `json:"completed_at"`
	Review      *ReviewResponse `json:"review"`
}

// ProjectHistoryListResponse is the paginated response envelope.
type ProjectHistoryListResponse struct {
	Data       []ProjectHistoryEntry `json:"data"`
	NextCursor string                `json:"next_cursor"`
	HasMore    bool                  `json:"has_more"`
}

// NewProjectHistoryEntry maps a service Entry to an API entry.
func NewProjectHistoryEntry(e projecthistory.Entry) ProjectHistoryEntry {
	entry := ProjectHistoryEntry{
		ProposalID:  e.ProposalID.String(),
		Title:       e.Title,
		Amount:      e.Amount,
		Currency:    e.Currency,
		CompletedAt: e.CompletedAt.Format(time.RFC3339),
	}
	if e.Review != nil {
		resp := ReviewFromDomain(e.Review)
		entry.Review = &resp
	}
	return entry
}

// NewProjectHistoryListResponse builds the paginated envelope.
func NewProjectHistoryListResponse(entries []projecthistory.Entry, nextCursor string) ProjectHistoryListResponse {
	data := make([]ProjectHistoryEntry, 0, len(entries))
	for _, e := range entries {
		data = append(data, NewProjectHistoryEntry(e))
	}
	return ProjectHistoryListResponse{
		Data:       data,
		NextCursor: nextCursor,
		HasMore:    nextCursor != "",
	}
}

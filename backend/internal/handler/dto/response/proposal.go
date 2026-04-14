package response

import (
	"time"

	"marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/domain/proposal"
)

type ProposalResponse struct {
	ID                       string              `json:"id"`
	ConversationID           string              `json:"conversation_id"`
	SenderID                 string              `json:"sender_id"`
	RecipientID              string              `json:"recipient_id"`
	Title                    string              `json:"title"`
	Description              string              `json:"description"`
	Amount                   int64               `json:"amount"`
	Deadline                 *string             `json:"deadline"`
	Status                   string              `json:"status"`
	ParentID                 *string             `json:"parent_id"`
	Version                  int                 `json:"version"`
	ClientID                 string              `json:"client_id"`
	ProviderID               string              `json:"provider_id"`
	ClientName               string              `json:"client_name"`
	ProviderName             string              `json:"provider_name"`
	ActiveDisputeID          *string             `json:"active_dispute_id"`
	LastDisputeID            *string             `json:"last_dispute_id"`
	Documents                []DocumentResponse  `json:"documents"`
	PaymentMode              string              `json:"payment_mode"`
	Milestones               []MilestoneResponse `json:"milestones"`
	CurrentMilestoneSequence *int                `json:"current_milestone_sequence,omitempty"`
	AcceptedAt               *string             `json:"accepted_at,omitempty"`
	PaidAt                   *string             `json:"paid_at,omitempty"`
	CreatedAt                string              `json:"created_at"`
}

// MilestoneResponse is the per-milestone payload inside a ProposalResponse.
// Status is the milestone-level enum (pending_funding, funded, submitted,
// approved, released, disputed, cancelled, refunded), distinct from the
// proposal-level macro status.
type MilestoneResponse struct {
	ID          string  `json:"id"`
	Sequence    int     `json:"sequence"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Amount      int64   `json:"amount"`
	Deadline    *string `json:"deadline,omitempty"`
	Status      string  `json:"status"`
	Version     int     `json:"version"`
	FundedAt    *string `json:"funded_at,omitempty"`
	SubmittedAt *string `json:"submitted_at,omitempty"`
	ApprovedAt  *string `json:"approved_at,omitempty"`
	ReleasedAt  *string `json:"released_at,omitempty"`
	DisputedAt  *string `json:"disputed_at,omitempty"`
	CancelledAt *string `json:"cancelled_at,omitempty"`
}

type DocumentResponse struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}

type ProjectListResponse struct {
	Data       []ProposalResponse `json:"data"`
	NextCursor string             `json:"next_cursor"`
	HasMore    bool               `json:"has_more"`
}

func NewProposalResponse(p *proposal.Proposal, docs []*proposal.ProposalDocument) ProposalResponse {
	return NewProposalResponseWithNames(p, docs, nil, "", "")
}

// NewProposalResponseWithMilestones is the phase-5 enriched factory used
// by the milestone-aware GET handlers. It materialises the milestones
// slice + computes the current_milestone_sequence so the frontend can
// render the tracker and surface the right CTA without a second round trip.
func NewProposalResponseWithMilestones(p *proposal.Proposal, docs []*proposal.ProposalDocument, milestones []*milestone.Milestone) ProposalResponse {
	return NewProposalResponseWithNames(p, docs, milestones, "", "")
}

func NewProposalResponseWithNames(p *proposal.Proposal, docs []*proposal.ProposalDocument, milestones []*milestone.Milestone, clientName, providerName string) ProposalResponse {
	resp := ProposalResponse{
		ID:             p.ID.String(),
		ConversationID: p.ConversationID.String(),
		SenderID:       p.SenderID.String(),
		RecipientID:    p.RecipientID.String(),
		Title:          p.Title,
		Description:    p.Description,
		Amount:         p.Amount,
		Status:         string(p.Status),
		Version:        p.Version,
		ClientID:       p.ClientID.String(),
		ProviderID:     p.ProviderID.String(),
		ClientName:     clientName,
		ProviderName:   providerName,
		Documents:      NewDocumentListResponse(docs),
		PaymentMode:    paymentModeOf(milestones),
		Milestones:     NewMilestoneListResponse(milestones),
		CreatedAt:      p.CreatedAt.Format(time.RFC3339),
	}

	if seq := currentMilestoneSequence(milestones); seq != nil {
		resp.CurrentMilestoneSequence = seq
	}

	if p.Deadline != nil {
		d := p.Deadline.Format(time.RFC3339)
		resp.Deadline = &d
	}
	if p.ParentID != nil {
		s := p.ParentID.String()
		resp.ParentID = &s
	}
	if p.AcceptedAt != nil {
		t := p.AcceptedAt.Format(time.RFC3339)
		resp.AcceptedAt = &t
	}
	if p.ActiveDisputeID != nil {
		s := p.ActiveDisputeID.String()
		resp.ActiveDisputeID = &s
	}
	if p.LastDisputeID != nil {
		s := p.LastDisputeID.String()
		resp.LastDisputeID = &s
	}
	if p.PaidAt != nil {
		t := p.PaidAt.Format(time.RFC3339)
		resp.PaidAt = &t
	}

	return resp
}

// paymentModeOf derives the UX hint from the milestone count. Single-
// milestone proposals are always rendered as "one-time" by the frontend
// regardless of how the proposal was originally created.
func paymentModeOf(milestones []*milestone.Milestone) string {
	if len(milestones) <= 1 {
		return "one_time"
	}
	return "milestone"
}

// currentMilestoneSequence walks the milestones and returns the lowest
// non-terminal sequence — the one whose CTA appears in the UI. Returns
// nil when every milestone is terminal (the proposal is fully done).
func currentMilestoneSequence(milestones []*milestone.Milestone) *int {
	current := milestone.FindCurrentActive(milestones)
	if current == nil {
		return nil
	}
	seq := current.Sequence
	return &seq
}

// NewMilestoneResponse converts a single domain milestone into its DTO
// representation, rendering optional timestamps as RFC3339 strings.
func NewMilestoneResponse(m *milestone.Milestone) MilestoneResponse {
	resp := MilestoneResponse{
		ID:          m.ID.String(),
		Sequence:    m.Sequence,
		Title:       m.Title,
		Description: m.Description,
		Amount:      m.Amount,
		Status:      string(m.Status),
		Version:     m.Version,
	}
	if m.Deadline != nil {
		d := m.Deadline.Format(time.RFC3339)
		resp.Deadline = &d
	}
	if m.FundedAt != nil {
		t := m.FundedAt.Format(time.RFC3339)
		resp.FundedAt = &t
	}
	if m.SubmittedAt != nil {
		t := m.SubmittedAt.Format(time.RFC3339)
		resp.SubmittedAt = &t
	}
	if m.ApprovedAt != nil {
		t := m.ApprovedAt.Format(time.RFC3339)
		resp.ApprovedAt = &t
	}
	if m.ReleasedAt != nil {
		t := m.ReleasedAt.Format(time.RFC3339)
		resp.ReleasedAt = &t
	}
	if m.DisputedAt != nil {
		t := m.DisputedAt.Format(time.RFC3339)
		resp.DisputedAt = &t
	}
	if m.CancelledAt != nil {
		t := m.CancelledAt.Format(time.RFC3339)
		resp.CancelledAt = &t
	}
	return resp
}

// NewMilestoneListResponse maps a slice of milestones to its DTO form.
// Returns an empty slice (never nil) so JSON callers always see [].
func NewMilestoneListResponse(milestones []*milestone.Milestone) []MilestoneResponse {
	out := make([]MilestoneResponse, 0, len(milestones))
	for _, m := range milestones {
		out = append(out, NewMilestoneResponse(m))
	}
	return out
}

func NewDocumentResponse(d *proposal.ProposalDocument) DocumentResponse {
	return DocumentResponse{
		ID:       d.ID.String(),
		Filename: d.Filename,
		URL:      d.URL,
		Size:     d.Size,
		MimeType: d.MimeType,
	}
}

func NewDocumentListResponse(docs []*proposal.ProposalDocument) []DocumentResponse {
	result := make([]DocumentResponse, len(docs))
	for i, d := range docs {
		result[i] = NewDocumentResponse(d)
	}
	return result
}

func NewProjectListResponse(proposals []*proposal.Proposal, nextCursor string) ProjectListResponse {
	data := make([]ProposalResponse, len(proposals))
	for i, p := range proposals {
		data[i] = NewProposalResponse(p, nil)
	}
	return ProjectListResponse{
		Data:       data,
		NextCursor: nextCursor,
		HasMore:    nextCursor != "",
	}
}

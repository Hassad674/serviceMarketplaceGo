package response

import (
	"time"

	"marketplace-backend/internal/domain/proposal"
)

type ProposalResponse struct {
	ID             string             `json:"id"`
	ConversationID string             `json:"conversation_id"`
	SenderID       string             `json:"sender_id"`
	RecipientID    string             `json:"recipient_id"`
	Title          string             `json:"title"`
	Description    string             `json:"description"`
	Amount         int64              `json:"amount"`
	Deadline       *string            `json:"deadline"`
	Status         string             `json:"status"`
	ParentID       *string            `json:"parent_id"`
	Version        int                `json:"version"`
	ClientID       string             `json:"client_id"`
	ProviderID     string             `json:"provider_id"`
	ClientName     string             `json:"client_name"`
	ProviderName   string             `json:"provider_name"`
	Documents      []DocumentResponse `json:"documents"`
	AcceptedAt     *string            `json:"accepted_at,omitempty"`
	PaidAt         *string            `json:"paid_at,omitempty"`
	CreatedAt      string             `json:"created_at"`
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
	return NewProposalResponseWithNames(p, docs, "", "")
}

func NewProposalResponseWithNames(p *proposal.Proposal, docs []*proposal.ProposalDocument, clientName, providerName string) ProposalResponse {
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
		CreatedAt:      p.CreatedAt.Format(time.RFC3339),
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
	if p.PaidAt != nil {
		t := p.PaidAt.Format(time.RFC3339)
		resp.PaidAt = &t
	}

	return resp
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

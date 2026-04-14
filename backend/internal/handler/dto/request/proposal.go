package request

// CreateProposalRequest is the payload accepted by POST /api/v1/proposals.
//
// Two modes coexist (phase 4 unified pipeline):
//
//  1. One-time mode (default, backward compat with every existing client):
//     omit Milestones, send Amount. The backend synthesises a single
//     milestone covering the full Amount.
//
//  2. Milestone mode: send PaymentMode="milestone" and a non-empty
//     Milestones slice. The backend uses milestones[] verbatim and
//     ignores Amount (the canonical total is the sum of milestones).
//
// Either way, the server-side state machine is identical — only the
// frontend rendering differs.
type CreateProposalRequest struct {
	RecipientID    string                  `json:"recipient_id"`
	ConversationID string                  `json:"conversation_id"`
	Title          string                  `json:"title"`
	Description    string                  `json:"description"`
	Amount         int64                   `json:"amount"`
	Deadline       string                  `json:"deadline"`
	Documents      []DocumentInput         `json:"documents"`
	PaymentMode    string                  `json:"payment_mode,omitempty"`
	Milestones     []MilestoneInputRequest `json:"milestones,omitempty"`
}

type DocumentInput struct {
	Filename string `json:"filename"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}

// MilestoneInputRequest is the per-milestone payload inside a milestone-
// mode CreateProposalRequest. Sequences must be consecutive starting at 1
// and the count is capped at 20 (enforced by the domain layer).
type MilestoneInputRequest struct {
	Sequence    int    `json:"sequence"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Amount      int64  `json:"amount"`
	Deadline    string `json:"deadline,omitempty"`
}

type ModifyProposalRequest struct {
	Title       string                  `json:"title"`
	Description string                  `json:"description"`
	Amount      int64                   `json:"amount"`
	Deadline    string                  `json:"deadline"`
	Documents   []DocumentInput         `json:"documents"`
	PaymentMode string                  `json:"payment_mode,omitempty"`
	Milestones  []MilestoneInputRequest `json:"milestones,omitempty"`
}

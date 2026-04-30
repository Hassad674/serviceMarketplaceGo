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
	RecipientID    string                  `json:"recipient_id" validate:"required,uuid"`
	ConversationID string                  `json:"conversation_id" validate:"required,uuid"`
	Title          string                  `json:"title" validate:"required,min=1,max=200"`
	Description    string                  `json:"description" validate:"required,min=1,max=5000"`
	Amount         int64                   `json:"amount" validate:"gte=0,lte=999999999"`
	Deadline       string                  `json:"deadline" validate:"omitempty,max=64"`
	Documents      []DocumentInput         `json:"documents" validate:"omitempty,max=20,dive"`
	PaymentMode    string                  `json:"payment_mode,omitempty" validate:"omitempty,oneof=one_time milestone"`
	Milestones     []MilestoneInputRequest `json:"milestones,omitempty" validate:"omitempty,max=20,dive"`
}

type DocumentInput struct {
	Filename string `json:"filename" validate:"required,min=1,max=255"`
	URL      string `json:"url" validate:"required,url,max=2048"`
	Size     int64  `json:"size" validate:"gte=0,lte=104857600"` // 100 MB cap
	MimeType string `json:"mime_type" validate:"required,min=1,max=128"`
}

// MilestoneInputRequest is the per-milestone payload inside a milestone-
// mode CreateProposalRequest. Sequences must be consecutive starting at 1
// and the count is capped at 20 (enforced by the domain layer).
type MilestoneInputRequest struct {
	Sequence    int    `json:"sequence" validate:"gte=1,lte=20"`
	Title       string `json:"title" validate:"required,min=1,max=200"`
	Description string `json:"description" validate:"omitempty,max=5000"`
	Amount      int64  `json:"amount" validate:"gte=0,lte=999999999"`
	Deadline    string `json:"deadline,omitempty" validate:"omitempty,max=64"`
}

// ModifyProposalRequest is the body of PATCH /api/v1/proposals/{id}.
type ModifyProposalRequest struct {
	Title       string                  `json:"title" validate:"required,min=1,max=200"`
	Description string                  `json:"description" validate:"required,min=1,max=5000"`
	Amount      int64                   `json:"amount" validate:"gte=0,lte=999999999"`
	Deadline    string                  `json:"deadline" validate:"omitempty,max=64"`
	Documents   []DocumentInput         `json:"documents" validate:"omitempty,max=20,dive"`
	PaymentMode string                  `json:"payment_mode,omitempty" validate:"omitempty,oneof=one_time milestone"`
	Milestones  []MilestoneInputRequest `json:"milestones,omitempty" validate:"omitempty,max=20,dive"`
}

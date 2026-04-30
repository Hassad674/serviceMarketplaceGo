package request

// OpenDisputeRequest is the body of POST /api/v1/disputes.
type OpenDisputeRequest struct {
	ProposalID      string              `json:"proposal_id" validate:"required,uuid"`
	Reason          string              `json:"reason" validate:"required,min=1,max=200"`
	Description     string              `json:"description" validate:"required,min=1,max=5000"`
	MessageToParty  string              `json:"message_to_party" validate:"omitempty,max=5000"`
	RequestedAmount int64               `json:"requested_amount" validate:"gte=0,lte=999999999"`
	Attachments     []DocumentInputItem `json:"attachments" validate:"omitempty,max=20,dive"`
}

// DocumentInputItem is the per-attachment payload for disputes and
// other features carrying user-uploaded documents.
type DocumentInputItem struct {
	Filename string `json:"filename" validate:"required,min=1,max=255"`
	URL      string `json:"url" validate:"required,url,max=2048"`
	Size     int64  `json:"size" validate:"gte=0,lte=104857600"` // 100 MB cap
	MimeType string `json:"mime_type" validate:"required,min=1,max=128"`
}

// CounterProposeRequest is the body of POST /api/v1/disputes/{id}/counter.
type CounterProposeRequest struct {
	AmountClient   int64               `json:"amount_client" validate:"gte=0,lte=999999999"`
	AmountProvider int64               `json:"amount_provider" validate:"gte=0,lte=999999999"`
	Message        string              `json:"message" validate:"omitempty,max=5000"`
	Attachments    []DocumentInputItem `json:"attachments" validate:"omitempty,max=20,dive"`
}

// RespondToCounterRequest is the body of POST /api/v1/disputes/{id}/respond.
type RespondToCounterRequest struct {
	Accept bool `json:"accept"`
}

// RespondToCancellationRequest is the body of
// POST /api/v1/disputes/{id}/cancel-respond.
type RespondToCancellationRequest struct {
	Accept bool `json:"accept"`
}

// AdminResolveDisputeRequest is the body of admin
// POST /api/v1/admin/disputes/{id}/resolve.
type AdminResolveDisputeRequest struct {
	AmountClient   int64  `json:"amount_client" validate:"gte=0,lte=999999999"`
	AmountProvider int64  `json:"amount_provider" validate:"gte=0,lte=999999999"`
	Note           string `json:"note" validate:"omitempty,max=5000"`
}

// AskAIDisputeRequest is the body of the admin AI chat endpoint. The
// chat history is loaded from the database by the backend (not from the
// request body) so admins cannot tamper with the context the AI sees and
// so multiple admins on the same dispute share state automatically.
type AskAIDisputeRequest struct {
	Question string `json:"question" validate:"required,min=1,max=5000"`
}

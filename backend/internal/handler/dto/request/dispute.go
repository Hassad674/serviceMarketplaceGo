package request

type OpenDisputeRequest struct {
	ProposalID      string              `json:"proposal_id"`
	Reason          string              `json:"reason"`
	Description     string              `json:"description"`
	MessageToParty  string              `json:"message_to_party"`
	RequestedAmount int64               `json:"requested_amount"`
	Attachments     []DocumentInputItem `json:"attachments"`
}

type DocumentInputItem struct {
	Filename string `json:"filename"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}

type CounterProposeRequest struct {
	AmountClient   int64               `json:"amount_client"`
	AmountProvider int64               `json:"amount_provider"`
	Message        string              `json:"message"`
	Attachments    []DocumentInputItem `json:"attachments"`
}

type RespondToCounterRequest struct {
	Accept bool `json:"accept"`
}

type RespondToCancellationRequest struct {
	Accept bool `json:"accept"`
}

type AdminResolveDisputeRequest struct {
	AmountClient   int64  `json:"amount_client"`
	AmountProvider int64  `json:"amount_provider"`
	Note           string `json:"note"`
}

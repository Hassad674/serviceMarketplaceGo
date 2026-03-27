package request

type CreateProposalRequest struct {
	RecipientID    string          `json:"recipient_id"`
	ConversationID string          `json:"conversation_id"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Amount         int64           `json:"amount"`
	Deadline       string          `json:"deadline"`
	Documents      []DocumentInput `json:"documents"`
}

type DocumentInput struct {
	Filename string `json:"filename"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}

type ModifyProposalRequest struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Amount      int64           `json:"amount"`
	Deadline    string          `json:"deadline"`
	Documents   []DocumentInput `json:"documents"`
}

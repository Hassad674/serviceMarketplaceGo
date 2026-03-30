package request

// CreateReportRequest is the payload for POST /api/v1/reports.
type CreateReportRequest struct {
	TargetType     string `json:"target_type"`
	TargetID       string `json:"target_id"`
	ConversationID string `json:"conversation_id,omitempty"`
	Reason         string `json:"reason"`
	Description    string `json:"description,omitempty"`
}

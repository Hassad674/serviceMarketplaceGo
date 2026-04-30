package request

// CreateReportRequest is the payload for POST /api/v1/reports.
type CreateReportRequest struct {
	TargetType     string `json:"target_type" validate:"required,min=1,max=50"`
	TargetID       string `json:"target_id" validate:"required,uuid"`
	ConversationID string `json:"conversation_id,omitempty" validate:"omitempty,uuid"`
	Reason         string `json:"reason" validate:"required,min=1,max=200"`
	Description    string `json:"description,omitempty" validate:"omitempty,max=5000"`
}

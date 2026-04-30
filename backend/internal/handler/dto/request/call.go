package request

// InitiateCallRequest is the body of POST /api/v1/calls.
type InitiateCallRequest struct {
	ConversationID string `json:"conversation_id" validate:"required,uuid"`
	RecipientID    string `json:"recipient_id" validate:"required,uuid"`
	Type           string `json:"type" validate:"required,oneof=audio video"`
}

// EndCallRequest is the body of POST /api/v1/calls/{id}/end. Duration
// is in seconds.
type EndCallRequest struct {
	Duration int `json:"duration" validate:"gte=0,lte=86400"`
}

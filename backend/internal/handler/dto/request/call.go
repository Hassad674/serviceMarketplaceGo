package request

type InitiateCallRequest struct {
	ConversationID string `json:"conversation_id"`
	RecipientID    string `json:"recipient_id"`
	Type           string `json:"type"`
}

type EndCallRequest struct {
	Duration int `json:"duration"`
}

package request

import "encoding/json"

type StartConversationRequest struct {
	RecipientID string          `json:"recipient_id"`
	Content     string          `json:"content"`
	Type        string          `json:"type"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

type SendMessageRequest struct {
	Content  string          `json:"content"`
	Type     string          `json:"type"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type MarkAsReadRequest struct {
	Seq int `json:"seq"`
}

type EditMessageRequest struct {
	Content string `json:"content"`
}

type PresignedURLRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

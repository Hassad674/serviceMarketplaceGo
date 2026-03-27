package request

import "encoding/json"

type StartConversationRequest struct {
	RecipientID string          `json:"recipient_id"`
	Content     string          `json:"content"`
	Type        string          `json:"type"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

type SendMessageRequest struct {
	Content   string          `json:"content"`
	Type      string          `json:"type"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	ReplyToID string          `json:"reply_to_id,omitempty"`
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
	MimeType    string `json:"mime_type"`
}

// ResolvedContentType returns content_type if set, falling back to mime_type.
func (r PresignedURLRequest) ResolvedContentType() string {
	if r.ContentType != "" {
		return r.ContentType
	}
	return r.MimeType
}

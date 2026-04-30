package request

import "encoding/json"

// StartConversationRequest is the body of POST /api/v1/conversations.
type StartConversationRequest struct {
	RecipientOrgID string          `json:"recipient_org_id" validate:"required,uuid"`
	Content        string          `json:"content" validate:"required,min=1,max=10000"`
	Type           string          `json:"type" validate:"omitempty,max=50"`
	Metadata       json.RawMessage `json:"metadata,omitempty" validate:"omitempty"`
}

// SendMessageRequest is the body of POST /api/v1/conversations/{id}/messages.
type SendMessageRequest struct {
	Content   string          `json:"content" validate:"required,min=1,max=10000"`
	Type      string          `json:"type" validate:"omitempty,max=50"`
	Metadata  json.RawMessage `json:"metadata,omitempty" validate:"omitempty"`
	ReplyToID string          `json:"reply_to_id,omitempty" validate:"omitempty,uuid"`
}

// MarkAsReadRequest is the body of POST /api/v1/conversations/{id}/read.
type MarkAsReadRequest struct {
	Seq int `json:"seq" validate:"gte=0"`
}

// EditMessageRequest is the body of PATCH /api/v1/messages/{id}.
type EditMessageRequest struct {
	Content string `json:"content" validate:"required,min=1,max=10000"`
}

// PresignedURLRequest is the body of POST /api/v1/messages/presigned-url.
type PresignedURLRequest struct {
	Filename    string `json:"filename" validate:"required,min=1,max=255"`
	ContentType string `json:"content_type" validate:"omitempty,max=128"`
	MimeType    string `json:"mime_type" validate:"omitempty,max=128"`
}

// ResolvedContentType returns content_type if set, falling back to mime_type.
func (r PresignedURLRequest) ResolvedContentType() string {
	if r.ContentType != "" {
		return r.ContentType
	}
	return r.MimeType
}

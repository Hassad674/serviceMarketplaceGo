package response

import (
	"encoding/json"
	"time"

	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/port/repository"
)

type MessageResponse struct {
	ID             string           `json:"id"`
	ConversationID string           `json:"conversation_id"`
	SenderID       string           `json:"sender_id"`
	Content        string           `json:"content"`
	Type           string           `json:"type"`
	Metadata       json.RawMessage  `json:"metadata"`
	ReplyTo        *ReplyToResponse `json:"reply_to,omitempty"`
	Seq            int              `json:"seq"`
	Status         string           `json:"status"`
	EditedAt       *string          `json:"edited_at,omitempty"`
	DeletedAt      *string          `json:"deleted_at,omitempty"`
	CreatedAt      string           `json:"created_at"`
}

// ReplyToResponse is the lightweight preview of the original message.
type ReplyToResponse struct {
	ID       string `json:"id"`
	SenderID string `json:"sender_id"`
	Content  string `json:"content"`
	Type     string `json:"type"`
}

// ConversationResponse describes a conversation from the caller org's
// perspective. The "other" side is the organization on the other end
// of the conversation (Stripe Dashboard semantics).
type ConversationResponse struct {
	ConversationID string  `json:"id"`
	OtherOrgID     string  `json:"other_org_id"`
	OtherOrgName   string  `json:"other_org_name"`
	OtherOrgType   string  `json:"other_org_type"`
	OtherPhotoURL  string  `json:"other_photo_url"`
	LastMessage    *string `json:"last_message"`
	LastMessageAt  *string `json:"last_message_at,omitempty"`
	LastMessageSeq int     `json:"last_message_seq"`
	UnreadCount    int     `json:"unread_count"`
	Online         bool    `json:"online"`
}

type StartConversationResponse struct {
	ConversationID string          `json:"conversation_id"`
	Message        MessageResponse `json:"message"`
}

type PresignedURLResponse struct {
	UploadURL string `json:"upload_url"`
	FileKey   string `json:"file_key"`
	PublicURL string `json:"public_url"`
}

type UnreadCountResponse struct {
	Count int `json:"count"`
}

func NewMessageResponse(m *message.Message) MessageResponse {
	metadata := json.RawMessage("null")
	if len(m.Metadata) > 0 {
		metadata = m.Metadata
	}

	resp := MessageResponse{
		ID:             m.ID.String(),
		ConversationID: m.ConversationID.String(),
		SenderID:       m.SenderID.String(),
		Content:        m.Content,
		Type:           string(m.Type),
		Metadata:       metadata,
		Seq:            m.Seq,
		Status:         string(m.Status),
		CreatedAt:      m.CreatedAt.Format(time.RFC3339),
	}

	if m.ReplyPreview != nil {
		resp.ReplyTo = &ReplyToResponse{
			ID:       m.ReplyPreview.ID.String(),
			SenderID: m.ReplyPreview.SenderID.String(),
			Content:  m.ReplyPreview.Content,
			Type:     string(m.ReplyPreview.Type),
		}
	}

	if m.EditedAt != nil {
		t := m.EditedAt.Format(time.RFC3339)
		resp.EditedAt = &t
	}
	if m.DeletedAt != nil {
		t := m.DeletedAt.Format(time.RFC3339)
		resp.DeletedAt = &t
	}

	return resp
}

func NewMessageListResponse(msgs []*message.Message) []MessageResponse {
	result := make([]MessageResponse, len(msgs))
	for i, m := range msgs {
		result[i] = NewMessageResponse(m)
	}
	return result
}

func NewConversationResponse(s repository.ConversationSummary) ConversationResponse {
	resp := ConversationResponse{
		ConversationID: s.ConversationID.String(),
		OtherOrgID:     s.OtherOrgID.String(),
		OtherOrgName:   s.OtherOrgName,
		OtherOrgType:   s.OtherOrgType,
		OtherPhotoURL:  s.OtherPhotoURL,
		LastMessage:    s.LastMessage,
		LastMessageSeq: s.LastMessageSeq,
		UnreadCount:    s.UnreadCount,
		Online:         s.Online,
	}

	if s.LastMessageAt != nil {
		t := s.LastMessageAt.Format(time.RFC3339)
		resp.LastMessageAt = &t
	}

	return resp
}

func NewConversationListResponse(summaries []repository.ConversationSummary) []ConversationResponse {
	result := make([]ConversationResponse, len(summaries))
	for i, s := range summaries {
		result[i] = NewConversationResponse(s)
	}
	return result
}

package response

import (
	"time"

	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/port/repository"
)

type MessageResponse struct {
	ID             string  `json:"id"`
	ConversationID string  `json:"conversation_id"`
	SenderID       string  `json:"sender_id"`
	Content        string  `json:"content"`
	Type           string  `json:"type"`
	Seq            int     `json:"seq"`
	Status         string  `json:"status"`
	EditedAt       *string `json:"edited_at,omitempty"`
	DeletedAt      *string `json:"deleted_at,omitempty"`
	CreatedAt      string  `json:"created_at"`
}

type ConversationResponse struct {
	ConversationID string  `json:"conversation_id"`
	OtherUserID    string  `json:"other_user_id"`
	OtherUserName  string  `json:"other_user_name"`
	OtherUserRole  string  `json:"other_user_role"`
	OtherPhotoURL  string  `json:"other_photo_url"`
	LastMessage    *string `json:"last_message"`
	LastMessageAt  *string `json:"last_message_at,omitempty"`
	LastMessageSeq int     `json:"last_message_seq"`
	UnreadCount    int     `json:"unread_count"`
}

type StartConversationResponse struct {
	ConversationID string          `json:"conversation_id"`
	Message        MessageResponse `json:"message"`
}

type PresignedURLResponse struct {
	UploadURL string `json:"upload_url"`
	PublicURL string `json:"public_url"`
}

type UnreadCountResponse struct {
	Count int `json:"count"`
}

func NewMessageResponse(m *message.Message) MessageResponse {
	resp := MessageResponse{
		ID:             m.ID.String(),
		ConversationID: m.ConversationID.String(),
		SenderID:       m.SenderID.String(),
		Content:        m.Content,
		Type:           string(m.Type),
		Seq:            m.Seq,
		Status:         string(m.Status),
		CreatedAt:      m.CreatedAt.Format(time.RFC3339),
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
		OtherUserID:    s.OtherUserID.String(),
		OtherUserName:  s.OtherUserName,
		OtherUserRole:  s.OtherUserRole,
		OtherPhotoURL:  s.OtherPhotoURL,
		LastMessage:    s.LastMessage,
		LastMessageSeq: s.LastMessageSeq,
		UnreadCount:    s.UnreadCount,
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

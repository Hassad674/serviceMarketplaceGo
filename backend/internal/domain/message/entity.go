package message

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type MessageType string

const (
	MessageTypeText MessageType = "text"
	MessageTypeFile MessageType = "file"
)

func (mt MessageType) IsValid() bool {
	switch mt {
	case MessageTypeText, MessageTypeFile:
		return true
	}
	return false
}

type MessageStatus string

const (
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
)

func (ms MessageStatus) IsValid() bool {
	switch ms {
	case MessageStatusSent, MessageStatusDelivered, MessageStatusRead:
		return true
	}
	return false
}

const MaxContentLength = 5000

type Conversation struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewConversation() *Conversation {
	now := time.Now()
	return &Conversation{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

type Participant struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	UnreadCount    int
	LastReadSeq    int
	JoinedAt       time.Time
}

type Message struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	Content        string
	Type           MessageType
	Metadata       json.RawMessage
	Seq            int
	Status         MessageStatus
	EditedAt       *time.Time
	DeletedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type FileMetadata struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}

func NewMessage(
	conversationID, senderID uuid.UUID,
	content string,
	msgType MessageType,
	metadata json.RawMessage,
	seq int,
) (*Message, error) {
	if !msgType.IsValid() {
		return nil, ErrInvalidMessageType
	}

	if msgType == MessageTypeText && content == "" {
		return nil, ErrEmptyContent
	}

	if len(content) > MaxContentLength {
		return nil, ErrContentTooLong
	}

	now := time.Now()
	return &Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        content,
		Type:           msgType,
		Metadata:       metadata,
		Seq:            seq,
		Status:         MessageStatusSent,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (m *Message) Edit(content string) {
	now := time.Now()
	m.Content = content
	m.EditedAt = &now
	m.UpdatedAt = now
}

func (m *Message) SoftDelete() {
	now := time.Now()
	m.Content = ""
	m.DeletedAt = &now
	m.UpdatedAt = now
}

func (m *Message) MarkDelivered() {
	m.Status = MessageStatusDelivered
	m.UpdatedAt = time.Now()
}

func (m *Message) MarkRead() {
	m.Status = MessageStatusRead
	m.UpdatedAt = time.Now()
}

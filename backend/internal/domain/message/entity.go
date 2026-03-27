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

	// Proposal message types — carry data in metadata, not content.
	MessageTypeProposalSent             MessageType = "proposal_sent"
	MessageTypeProposalAccepted         MessageType = "proposal_accepted"
	MessageTypeProposalDeclined         MessageType = "proposal_declined"
	MessageTypeProposalModified         MessageType = "proposal_modified"
	MessageTypeProposalPaid             MessageType = "proposal_paid"
	MessageTypeProposalPaymentRequested MessageType = "proposal_payment_requested"
)

// IsProposalType returns true if the message type is a proposal event type.
func (mt MessageType) IsProposalType() bool {
	switch mt {
	case MessageTypeProposalSent, MessageTypeProposalAccepted,
		MessageTypeProposalDeclined, MessageTypeProposalModified,
		MessageTypeProposalPaid, MessageTypeProposalPaymentRequested:
		return true
	}
	return false
}

func (mt MessageType) IsValid() bool {
	switch mt {
	case MessageTypeText, MessageTypeFile,
		MessageTypeProposalSent, MessageTypeProposalAccepted,
		MessageTypeProposalDeclined, MessageTypeProposalModified,
		MessageTypeProposalPaid, MessageTypeProposalPaymentRequested:
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

	// Proposal message types carry data in metadata, content is optional.
	// File messages also allow empty content (already handled above).

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

package message

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type MessageType string

const (
	MessageTypeText  MessageType = "text"
	MessageTypeFile  MessageType = "file"
	MessageTypeVoice MessageType = "voice"

	// Proposal message types — carry data in metadata, not content.
	MessageTypeProposalSent                MessageType = "proposal_sent"
	MessageTypeProposalAccepted            MessageType = "proposal_accepted"
	MessageTypeProposalDeclined            MessageType = "proposal_declined"
	MessageTypeProposalModified            MessageType = "proposal_modified"
	MessageTypeProposalPaid                MessageType = "proposal_paid"
	MessageTypeProposalPaymentRequested    MessageType = "proposal_payment_requested"
	MessageTypeProposalCompletionRequested MessageType = "proposal_completion_requested"
	MessageTypeProposalCompleted           MessageType = "proposal_completed"
	MessageTypeProposalCompletionRejected  MessageType = "proposal_completion_rejected"

	// Review message types
	MessageTypeEvaluationRequest MessageType = "evaluation_request"

	// Call message types
	MessageTypeCallEnded  MessageType = "call_ended"
	MessageTypeCallMissed MessageType = "call_missed"

	// Dispute message types
	MessageTypeDisputeOpened                 MessageType = "dispute_opened"
	MessageTypeDisputeCounterProposal        MessageType = "dispute_counter_proposal"
	MessageTypeDisputeCounterAccepted        MessageType = "dispute_counter_accepted"
	MessageTypeDisputeCounterRejected        MessageType = "dispute_counter_rejected"
	MessageTypeDisputeEscalated              MessageType = "dispute_escalated"
	MessageTypeDisputeResolved               MessageType = "dispute_resolved"
	MessageTypeDisputeCancelled              MessageType = "dispute_cancelled"
	MessageTypeDisputeAutoResolved           MessageType = "dispute_auto_resolved"
	MessageTypeDisputeCancellationRequested  MessageType = "dispute_cancellation_requested"
	MessageTypeDisputeCancellationRefused    MessageType = "dispute_cancellation_refused"

	// Referral (apport d'affaires) — the only system message the referral
	// feature posts inside the provider↔client conversation, when a client
	// accepts the introduction. Commission events (paid/clawed back) are
	// notification-only and never appear in the conversation: the chat
	// stays strictly 1:1 between the working parties (B2B confidentiality).
	MessageTypeReferralIntroActivated MessageType = "referral_intro_activated"
)

// IsProposalType returns true if the message type is a proposal event type.
func (mt MessageType) IsProposalType() bool {
	switch mt {
	case MessageTypeProposalSent, MessageTypeProposalAccepted,
		MessageTypeProposalDeclined, MessageTypeProposalModified,
		MessageTypeProposalPaid, MessageTypeProposalPaymentRequested,
		MessageTypeProposalCompletionRequested, MessageTypeProposalCompleted,
		MessageTypeProposalCompletionRejected:
		return true
	}
	return false
}

// IsDisputeType returns true if the message type is a dispute event type.
func (mt MessageType) IsDisputeType() bool {
	switch mt {
	case MessageTypeDisputeOpened, MessageTypeDisputeCounterProposal,
		MessageTypeDisputeCounterAccepted, MessageTypeDisputeCounterRejected,
		MessageTypeDisputeEscalated, MessageTypeDisputeResolved,
		MessageTypeDisputeCancelled, MessageTypeDisputeAutoResolved,
		MessageTypeDisputeCancellationRequested, MessageTypeDisputeCancellationRefused:
		return true
	}
	return false
}

func (mt MessageType) IsValid() bool {
	switch mt {
	case MessageTypeText, MessageTypeFile, MessageTypeVoice,
		MessageTypeProposalSent, MessageTypeProposalAccepted,
		MessageTypeProposalDeclined, MessageTypeProposalModified,
		MessageTypeProposalPaid, MessageTypeProposalPaymentRequested,
		MessageTypeProposalCompletionRequested, MessageTypeProposalCompleted,
		MessageTypeProposalCompletionRejected,
		MessageTypeEvaluationRequest,
		MessageTypeCallEnded, MessageTypeCallMissed,
		MessageTypeDisputeOpened, MessageTypeDisputeCounterProposal,
		MessageTypeDisputeCounterAccepted, MessageTypeDisputeCounterRejected,
		MessageTypeDisputeEscalated, MessageTypeDisputeResolved,
		MessageTypeDisputeCancelled, MessageTypeDisputeAutoResolved,
		MessageTypeDisputeCancellationRequested, MessageTypeDisputeCancellationRefused:
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
	ReplyToID      *uuid.UUID
	ReplyPreview   *ReplyPreview // populated on read, nil if no reply
	Seq            int
	Status         MessageStatus
	EditedAt       *time.Time
	DeletedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ReplyPreview is a lightweight snapshot of a replied-to message,
// resolved by the adapter layer and attached for API responses.
type ReplyPreview struct {
	ID       uuid.UUID
	SenderID uuid.UUID
	Content  string
	Type     MessageType
}

// TruncateContent shortens text to maxLen runes, appending "..." if truncated.
func TruncateContent(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

type FileMetadata struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}

type VoiceMetadata struct {
	URL      string  `json:"url"`
	Duration float64 `json:"duration"`
	Size     int64   `json:"size"`
	MimeType string  `json:"mime_type"`
}

// NewMessageInput groups parameters for NewMessage to stay within the 4-param limit.
type NewMessageInput struct {
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	Content        string
	Type           MessageType
	Metadata       json.RawMessage
	ReplyToID      *uuid.UUID
	Seq            int
}

func NewMessage(in NewMessageInput) (*Message, error) {
	if !in.Type.IsValid() {
		return nil, ErrInvalidMessageType
	}

	if in.Type == MessageTypeText && in.Content == "" {
		return nil, ErrEmptyContent
	}

	// Proposal message types carry data in metadata, content is optional.
	// File and voice messages also allow empty content (already handled above).

	if len(in.Content) > MaxContentLength {
		return nil, ErrContentTooLong
	}

	now := time.Now()
	return &Message{
		ID:             uuid.New(),
		ConversationID: in.ConversationID,
		SenderID:       in.SenderID,
		Content:        in.Content,
		Type:           in.Type,
		Metadata:       in.Metadata,
		ReplyToID:      in.ReplyToID,
		Seq:            in.Seq,
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

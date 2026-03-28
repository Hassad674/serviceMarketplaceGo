package ws

import "github.com/google/uuid"

// Inbound message types (client -> server)
const (
	TypeHeartbeat = "heartbeat"
	TypeTyping    = "typing"
	TypeAck       = "ack"
	TypeSync      = "sync"
)

// Outbound message types (server -> client)
const (
	TypeNewMessage   = "new_message"
	TypeTypingEvent  = "typing"
	TypeStatusUpdate = "status_update"
	TypeUnreadCount  = "unread_count"
	TypePresence     = "presence"
	TypePong         = "pong"
	TypeSyncResult   = "sync_result"
	TypeError        = "error"
	TypeCallEvent    = "call_event"
	TypeNotification = "notification"
)

// StreamEvent represents a broadcast event received from the pub/sub layer.
// This is a local copy to avoid importing the redis adapter package directly.
type StreamEvent struct {
	Type         string
	RecipientIDs string
	Payload      string
	SourceID     string
}

// Envelope is the standard message format for WebSocket communication.
type Envelope struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

// InboundMessage represents a parsed message from the client.
type InboundMessage struct {
	Type           string         `json:"type"`
	ConversationID string         `json:"conversation_id,omitempty"`
	MessageID      string         `json:"message_id,omitempty"`
	SinceSeq       int            `json:"since_seq,omitempty"`
	Conversations  map[string]int `json:"conversations,omitempty"` // Multi-conversation sync: map[conversationID]sinceSeq
	UserID         uuid.UUID      `json:"-"`
}

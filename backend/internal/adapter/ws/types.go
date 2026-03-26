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
	TypeTypingEvent  = "typing_event"
	TypeStatusUpdate = "status_update"
	TypeUnreadCount  = "unread_count"
	TypePong         = "pong"
	TypeSyncResult   = "sync_result"
	TypeError        = "error"
)

// Envelope is the standard message format for WebSocket communication.
type Envelope struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

// InboundMessage represents a parsed message from the client.
type InboundMessage struct {
	Type           string    `json:"type"`
	ConversationID string    `json:"conversation_id,omitempty"`
	MessageID      string    `json:"message_id,omitempty"`
	SinceSeq       int       `json:"since_seq,omitempty"`
	UserID         uuid.UUID `json:"-"`
}

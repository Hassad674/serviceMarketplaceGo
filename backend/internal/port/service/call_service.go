package service

import (
	"context"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/call"
)

// LiveKitService generates tokens and manages rooms on the LiveKit server.
type LiveKitService interface {
	CreateRoom(ctx context.Context, roomName string) error
	GenerateToken(roomName, identity, displayName string) (string, error)
	DeleteRoom(ctx context.Context, roomName string) error
}

// CallStateService stores active calls in Redis (no DB persistence).
type CallStateService interface {
	SaveActiveCall(ctx context.Context, c *call.Call) error
	GetActiveCall(ctx context.Context, callID uuid.UUID) (*call.Call, error)
	GetActiveCallByUser(ctx context.Context, userID uuid.UUID) (*call.Call, error)
	RemoveActiveCall(ctx context.Context, callID uuid.UUID) error
}

// CallBroadcaster sends call signaling events to WebSocket clients.
type CallBroadcaster interface {
	BroadcastCallEvent(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error
}

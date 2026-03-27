package call

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	calldomain "marketplace-backend/internal/domain/call"
	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

type ServiceDeps struct {
	LiveKit     service.LiveKitService
	CallState   service.CallStateService
	Presence    service.PresenceService
	Broadcaster service.CallBroadcaster
	Messages    service.MessageSender
	Users       repository.UserRepository
}

type Service struct {
	livekit     service.LiveKitService
	callState   service.CallStateService
	presence    service.PresenceService
	broadcaster service.CallBroadcaster
	messages    service.MessageSender
	users       repository.UserRepository
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		livekit:     deps.LiveKit,
		callState:   deps.CallState,
		presence:    deps.Presence,
		broadcaster: deps.Broadcaster,
		messages:    deps.Messages,
		users:       deps.Users,
	}
}

type InitiateInput struct {
	ConversationID uuid.UUID
	InitiatorID    uuid.UUID
	RecipientID    uuid.UUID
	Type           calldomain.Type
}

type InitiateResult struct {
	CallID         uuid.UUID `json:"call_id"`
	RoomName       string    `json:"room_name"`
	InitiatorToken string    `json:"token"`
}

func (s *Service) Initiate(ctx context.Context, input InitiateInput) (*InitiateResult, error) {
	// Check initiator is not already in a call
	if _, err := s.callState.GetActiveCallByUser(ctx, input.InitiatorID); err == nil {
		return nil, calldomain.ErrUserBusy
	}

	// Check recipient is not already in a call
	if _, err := s.callState.GetActiveCallByUser(ctx, input.RecipientID); err == nil {
		return nil, calldomain.ErrUserBusy
	}

	// Check recipient is online
	online, err := s.presence.IsOnline(ctx, input.RecipientID)
	if err != nil {
		return nil, fmt.Errorf("initiate call: check presence: %w", err)
	}
	if !online {
		return nil, calldomain.ErrRecipientOffline
	}

	// Create domain entity
	c, err := calldomain.New(input.ConversationID, input.InitiatorID, input.RecipientID, input.Type)
	if err != nil {
		return nil, fmt.Errorf("initiate call: %w", err)
	}

	// Create LiveKit room
	if err := s.livekit.CreateRoom(ctx, c.RoomName); err != nil {
		return nil, fmt.Errorf("initiate call: create room: %w", err)
	}

	// Generate token for initiator
	initiatorName := s.resolveDisplayName(ctx, input.InitiatorID)
	token, err := s.livekit.GenerateToken(c.RoomName, input.InitiatorID.String(), initiatorName)
	if err != nil {
		return nil, fmt.Errorf("initiate call: generate token: %w", err)
	}

	// Save in Redis
	if err := s.callState.SaveActiveCall(ctx, c); err != nil {
		return nil, fmt.Errorf("initiate call: save state: %w", err)
	}

	// Broadcast incoming call to recipient
	s.broadcastCallSignal(ctx, "call_incoming", c, []uuid.UUID{input.RecipientID})

	return &InitiateResult{
		CallID:         c.ID,
		RoomName:       c.RoomName,
		InitiatorToken: token,
	}, nil
}

type AcceptResult struct {
	Token    string `json:"token"`
	RoomName string `json:"room_name"`
}

func (s *Service) Accept(ctx context.Context, callID, userID uuid.UUID) (*AcceptResult, error) {
	c, err := s.callState.GetActiveCall(ctx, callID)
	if err != nil {
		return nil, fmt.Errorf("accept call: %w", err)
	}

	if c.RecipientID != userID {
		return nil, calldomain.ErrNotParticipant
	}

	if err := c.Accept(); err != nil {
		return nil, fmt.Errorf("accept call: %w", err)
	}

	// Update state
	if err := s.callState.SaveActiveCall(ctx, c); err != nil {
		return nil, fmt.Errorf("accept call: save state: %w", err)
	}

	// Generate token for recipient
	recipientName := s.resolveDisplayName(ctx, userID)
	token, err := s.livekit.GenerateToken(c.RoomName, userID.String(), recipientName)
	if err != nil {
		return nil, fmt.Errorf("accept call: generate token: %w", err)
	}

	// Notify initiator
	s.broadcastCallSignal(ctx, "call_accepted", c, []uuid.UUID{c.InitiatorID})

	return &AcceptResult{
		Token:    token,
		RoomName: c.RoomName,
	}, nil
}

func (s *Service) Decline(ctx context.Context, callID, userID uuid.UUID) error {
	c, err := s.callState.GetActiveCall(ctx, callID)
	if err != nil {
		return fmt.Errorf("decline call: %w", err)
	}

	if c.RecipientID != userID && c.InitiatorID != userID {
		return calldomain.ErrNotParticipant
	}

	if err := c.Decline(); err != nil {
		return fmt.Errorf("decline call: %w", err)
	}

	// Notify the other party
	otherID := s.otherParty(c, userID)
	s.broadcastCallSignal(ctx, "call_declined", c, []uuid.UUID{otherID})

	// Cleanup
	_ = s.callState.RemoveActiveCall(ctx, callID)
	_ = s.livekit.DeleteRoom(ctx, c.RoomName)

	return nil
}

type EndInput struct {
	CallID   uuid.UUID
	UserID   uuid.UUID
	Duration int
}

func (s *Service) End(ctx context.Context, input EndInput) error {
	c, err := s.callState.GetActiveCall(ctx, input.CallID)
	if err != nil {
		return fmt.Errorf("end call: %w", err)
	}

	if c.InitiatorID != input.UserID && c.RecipientID != input.UserID {
		return calldomain.ErrNotParticipant
	}

	if err := c.End(input.Duration); err != nil {
		return fmt.Errorf("end call: %w", err)
	}

	// Notify the other party
	otherID := s.otherParty(c, input.UserID)
	s.broadcastCallSignal(ctx, "call_ended", c, []uuid.UUID{otherID})

	// Send system message in chat
	s.sendCallSystemMessage(ctx, c)

	// Cleanup
	_ = s.callState.RemoveActiveCall(ctx, input.CallID)
	_ = s.livekit.DeleteRoom(ctx, c.RoomName)

	return nil
}

func (s *Service) otherParty(c *calldomain.Call, userID uuid.UUID) uuid.UUID {
	if c.InitiatorID == userID {
		return c.RecipientID
	}
	return c.InitiatorID
}

func (s *Service) resolveDisplayName(ctx context.Context, userID uuid.UUID) string {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return "User"
	}
	return u.DisplayName
}

func (s *Service) broadcastCallSignal(ctx context.Context, eventType string, c *calldomain.Call, recipientIDs []uuid.UUID) {
	initiatorName := s.resolveDisplayName(ctx, c.InitiatorID)
	recipientName := s.resolveDisplayName(ctx, c.RecipientID)

	payload, err := json.Marshal(map[string]string{
		"event":           eventType,
		"call_id":         c.ID.String(),
		"conversation_id": c.ConversationID.String(),
		"initiator_id":    c.InitiatorID.String(),
		"recipient_id":    c.RecipientID.String(),
		"call_type":       string(c.Type),
		"initiator_name":  initiatorName,
		"recipient_name":  recipientName,
	})
	if err != nil {
		return
	}
	_ = s.broadcaster.BroadcastCallEvent(ctx, recipientIDs, payload)
}

func (s *Service) sendCallSystemMessage(ctx context.Context, c *calldomain.Call) {
	var content string
	var msgType message.MessageType

	if c.Duration > 0 {
		durationStr := formatDuration(c.Duration)
		content = fmt.Sprintf("Audio call - %s", durationStr)
		msgType = message.MessageTypeCallEnded
	} else {
		content = "Missed call"
		msgType = message.MessageTypeCallMissed
	}

	metadata, _ := json.Marshal(map[string]any{
		"call_id":  c.ID.String(),
		"duration": c.Duration,
		"type":     string(c.Type),
	})

	_ = s.messages.SendSystemMessage(ctx, service.SystemMessageInput{
		ConversationID: c.ConversationID,
		SenderID:       c.InitiatorID,
		Content:        content,
		Type:           string(msgType),
		Metadata:       metadata,
	})
}

func formatDuration(seconds int) string {
	m := seconds / 60
	s := seconds % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

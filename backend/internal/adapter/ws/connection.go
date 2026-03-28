package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/port/service"
)

var errUnauthorizedWS = errors.New("websocket authentication failed")

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	heartbeatTick  = 30 * time.Second
	maxMessageSize = 4096
	sendBufferSize = 64
)

type ConnDeps struct {
	Hub              *Hub
	MessagingSvc     service.MessagingQuerier
	TokenSvc         service.TokenService
	SessionSvc       service.SessionService
	PresenceSvc      service.PresenceService
	Broadcaster      service.MessageBroadcaster
	AllowedWSOrigins []string
}

// ServeWS returns an HTTP handler that upgrades connections to WebSocket.
func ServeWS(deps ConnDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := authenticateWS(r, deps.TokenSvc, deps.SessionSvc)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: deps.AllowedWSOrigins,
		})
		if err != nil {
			slog.Error("websocket accept failed", "error", err)
			return
		}

		client := &Client{
			UserID: userID,
			Send:   make(chan []byte, sendBufferSize),
			hub:    deps.Hub,
		}

		deps.Hub.register <- client

		_ = deps.PresenceSvc.SetOnline(r.Context(), userID)
		go broadcastPresenceChange(userID, true, deps)

		go writePump(conn, client)
		readPump(conn, client, deps)
	}
}

func authenticateWS(r *http.Request, tokenSvc service.TokenService, sessionSvc service.SessionService) (uuid.UUID, error) {
	// Strategy 1: Session cookie (web, same-origin)
	if cookie, err := r.Cookie("session_id"); err == nil && cookie.Value != "" {
		session, err := sessionSvc.Get(r.Context(), cookie.Value)
		if err == nil {
			return session.UserID, nil
		}
	}

	// Strategy 2: Session ID as query param (web, cross-origin production)
	if sid := r.URL.Query().Get("session_id"); sid != "" {
		session, err := sessionSvc.Get(r.Context(), sid)
		if err == nil {
			return session.UserID, nil
		}
	}

	// Strategy 3: JWT token as query param (mobile)
	token := r.URL.Query().Get("token")
	if token != "" {
		claims, err := tokenSvc.ValidateAccessToken(token)
		if err == nil {
			return claims.UserID, nil
		}
	}

	return uuid.UUID{}, errUnauthorizedWS
}

func readPump(conn *websocket.Conn, client *Client, deps ConnDeps) {
	defer func() {
		// Check connection count BEFORE unregistering so we can detect
		// if this was the last connection for the user.
		isLast := deps.Hub.ConnectionCount(client.UserID) <= 1

		deps.Hub.unregister <- client
		conn.Close(websocket.StatusNormalClosure, "")

		if isLast {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = deps.PresenceSvc.SetOffline(ctx, client.UserID)
			go broadcastPresenceChange(client.UserID, false, deps)
		}
	}()

	conn.SetReadLimit(maxMessageSize)

	for {
		readCtx, readCancel := context.WithTimeout(context.Background(), pongWait)
		_, data, err := conn.Read(readCtx)
		readCancel()

		if err != nil {
			return
		}

		handleInboundMessage(client, data, deps)
	}
}

func handleInboundMessage(client *Client, data []byte, deps ConnDeps) {
	var msg InboundMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		sendError(client, "invalid message format")
		return
	}

	msg.UserID = client.UserID

	switch msg.Type {
	case TypeHeartbeat:
		handleHeartbeat(client, deps.PresenceSvc)
	case TypeTyping:
		handleTyping(client, msg, deps)
	case TypeAck:
		handleAck(client, msg, deps)
	case TypeSync:
		handleSync(client, msg, deps)
	default:
		sendError(client, "unknown message type")
	}
}

func handleHeartbeat(client *Client, presenceSvc service.PresenceService) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = presenceSvc.SetOnline(ctx, client.UserID)

	pong, err := json.Marshal(Envelope{Type: TypePong})
	if err != nil {
		slog.Error("failed to marshal pong", "error", err)
		return
	}

	select {
	case client.Send <- pong:
	default:
		slog.Warn("send buffer full on heartbeat pong", "user_id", client.UserID)
	}
}

func handleTyping(client *Client, msg InboundMessage, deps ConnDeps) {
	convID, err := uuid.Parse(msg.ConversationID)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	participantIDs, err := deps.MessagingSvc.GetParticipantIDs(ctx, convID)
	if err != nil {
		return
	}

	payload, err := json.Marshal(map[string]string{
		"conversation_id": convID.String(),
		"user_id":         client.UserID.String(),
	})
	if err != nil {
		slog.Error("failed to marshal typing event", "error", err)
		return
	}

	_ = deps.Hub.broadcastToOthers(client.UserID, participantIDs, TypeTypingEvent, payload)
}

func handleAck(client *Client, msg InboundMessage, deps ConnDeps) {
	messageID, err := uuid.Parse(msg.MessageID)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = deps.MessagingSvc.DeliverMessage(ctx, messageID, client.UserID)
}

func handleSync(client *Client, msg InboundMessage, deps ConnDeps) {
	// Multi-conversation sync: iterate over Conversations map
	if len(msg.Conversations) > 0 {
		for convIDStr, sinceSeq := range msg.Conversations {
			syncSingleConversation(client, convIDStr, sinceSeq, deps)
		}
		return
	}

	// Backward compat: single conversation sync
	if msg.ConversationID != "" {
		syncSingleConversation(client, msg.ConversationID, msg.SinceSeq, deps)
	}
}

func syncSingleConversation(client *Client, convIDStr string, sinceSeq int, deps ConnDeps) {
	convID, err := uuid.Parse(convIDStr)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	messages, err := deps.MessagingSvc.GetMessagesSinceSeq(ctx, client.UserID, convID, sinceSeq)
	if err != nil {
		return
	}

	// Convert domain messages to snake_case maps matching the client-side
	// Message type. Domain structs have no json tags and would serialize
	// with PascalCase field names, breaking frontend parsing.
	serialized := make([]map[string]any, len(messages))
	for i, msg := range messages {
		serialized[i] = marshalMessageForSync(msg)
	}

	envelope, err := json.Marshal(Envelope{
		Type: TypeSyncResult,
		Payload: map[string]any{
			"conversation_id": convID.String(),
			"messages":        serialized,
		},
	})
	if err != nil {
		slog.Error("failed to marshal sync result", "error", err)
		return
	}

	client.Send <- envelope
}

// marshalMessageForSync converts a domain Message to a JSON-friendly map
// with snake_case keys matching the client-side Message type.
// This mirrors the format used by the new_message WS broadcast and the
// REST API responses, ensuring sync results are parseable by frontends.
func marshalMessageForSync(msg *message.Message) map[string]any {
	metadata := json.RawMessage("null")
	if len(msg.Metadata) > 0 {
		metadata = msg.Metadata
	}

	result := map[string]any{
		"id":              msg.ID.String(),
		"conversation_id": msg.ConversationID.String(),
		"sender_id":       msg.SenderID.String(),
		"content":         msg.Content,
		"type":            string(msg.Type),
		"metadata":        metadata,
		"reply_to":        nil,
		"seq":             msg.Seq,
		"status":          string(msg.Status),
		"edited_at":       nil,
		"deleted_at":      nil,
		"created_at":      msg.CreatedAt.Format(time.RFC3339),
	}

	if msg.ReplyPreview != nil {
		result["reply_to"] = map[string]any{
			"id":        msg.ReplyPreview.ID.String(),
			"sender_id": msg.ReplyPreview.SenderID.String(),
			"content":   msg.ReplyPreview.Content,
			"type":      string(msg.ReplyPreview.Type),
		}
	}

	if msg.EditedAt != nil {
		result["edited_at"] = msg.EditedAt.Format(time.RFC3339)
	}
	if msg.DeletedAt != nil {
		result["deleted_at"] = msg.DeletedAt.Format(time.RFC3339)
	}

	return result
}

func sendError(client *Client, errMsg string) {
	data, err := json.Marshal(Envelope{
		Type:    TypeError,
		Payload: map[string]string{"message": errMsg},
	})
	if err != nil {
		slog.Error("failed to marshal error envelope", "error", err)
		return
	}
	client.Send <- data
}

func writePump(conn *websocket.Conn, client *Client) {
	ticker := time.NewTicker(heartbeatTick)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-client.Send:
			if !ok {
				conn.Close(websocket.StatusNormalClosure, "")
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			err := conn.Write(ctx, websocket.MessageText, msg)
			cancel()

			if err != nil {
				return
			}

		case <-ticker.C:
			// Ticker is kept for potential future server-initiated pings.
			// Presence refresh is handled by client heartbeats via handleHeartbeat.
		}
	}
}

// broadcastPresenceChange notifies all contacts of a user's online/offline status.
func broadcastPresenceChange(userID uuid.UUID, online bool, deps ConnDeps) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	contactIDs, err := deps.MessagingSvc.GetContactIDs(ctx, userID)
	if err != nil || len(contactIDs) == 0 {
		return
	}

	payload, err := json.Marshal(map[string]any{
		"user_id": userID.String(),
		"online":  online,
	})
	if err != nil {
		slog.Error("failed to marshal presence change", "error", err)
		return
	}

	if err := deps.Broadcaster.BroadcastPresence(ctx, contactIDs, payload); err != nil {
		slog.Error("broadcast presence change failed", "error", err, "user_id", userID)
	}
}

func (h *Hub) broadcastToOthers(senderID uuid.UUID, participantIDs []uuid.UUID, eventType string, payload []byte) error {
	envelope, err := json.Marshal(Envelope{
		Type:    eventType,
		Payload: json.RawMessage(payload),
	})
	if err != nil {
		slog.Error("failed to marshal broadcast envelope", "error", err, "event_type", eventType)
		return err
	}

	for _, id := range participantIDs {
		if id == senderID {
			continue
		}
		h.SendToUser(id, envelope)
	}

	return nil
}

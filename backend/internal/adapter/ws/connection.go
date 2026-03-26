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

	"marketplace-backend/internal/app/messaging"
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
	Hub         *Hub
	MessagingSvc *messaging.Service
	TokenSvc     service.TokenService
	SessionSvc   service.SessionService
	PresenceSvc  service.PresenceService
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
			InsecureSkipVerify: true, // CORS handled by middleware
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

		go writePump(conn, client, deps.PresenceSvc)
		readPump(conn, client, deps)
	}
}

func authenticateWS(r *http.Request, tokenSvc service.TokenService, sessionSvc service.SessionService) (uuid.UUID, error) {
	// Strategy 1: Session cookie (web)
	if cookie, err := r.Cookie("session_id"); err == nil && cookie.Value != "" {
		session, err := sessionSvc.Get(r.Context(), cookie.Value)
		if err == nil {
			return session.UserID, nil
		}
	}

	// Strategy 2: Query param token (mobile)
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
		deps.Hub.unregister <- client
		conn.Close(websocket.StatusNormalClosure, "")
	}()

	conn.SetReadLimit(maxMessageSize)

	for {
		_, data, err := conn.Read(context.Background())
		if err != nil {
			if websocket.CloseStatus(err) != -1 {
				slog.Debug("websocket closed", "user_id", client.UserID)
			}
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

	pong, _ := json.Marshal(Envelope{Type: TypePong})
	client.Send <- pong
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

	payload, _ := json.Marshal(map[string]string{
		"conversation_id": convID.String(),
		"user_id":         client.UserID.String(),
	})

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
	convID, err := uuid.Parse(msg.ConversationID)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	messages, err := deps.MessagingSvc.GetMessagesSinceSeq(ctx, client.UserID, convID, msg.SinceSeq)
	if err != nil {
		return
	}

	envelope, _ := json.Marshal(Envelope{
		Type: TypeSyncResult,
		Payload: map[string]any{
			"conversation_id": convID.String(),
			"messages":        messages,
		},
	})

	client.Send <- envelope
}

func sendError(client *Client, errMsg string) {
	data, _ := json.Marshal(Envelope{
		Type:    TypeError,
		Payload: map[string]string{"message": errMsg},
	})
	client.Send <- data
}

func writePump(conn *websocket.Conn, client *Client, presenceSvc service.PresenceService) {
	ticker := time.NewTicker(heartbeatTick)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				conn.Close(websocket.StatusNormalClosure, "")
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			err := conn.Write(ctx, websocket.MessageText, message)
			cancel()

			if err != nil {
				return
			}

		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = presenceSvc.SetOnline(ctx, client.UserID)
			cancel()
		}
	}
}

func (h *Hub) broadcastToOthers(senderID uuid.UUID, participantIDs []uuid.UUID, eventType string, payload []byte) error {
	envelope, _ := json.Marshal(Envelope{
		Type:    eventType,
		Payload: json.RawMessage(payload),
	})

	for _, id := range participantIDs {
		if id == senderID {
			continue
		}
		h.SendToUser(id, envelope)
	}

	return nil
}

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
			conn:   conn,
		}

		// Register synchronously so the user's connection is observable
		// the instant SetOnline / broadcastPresenceChange fires. The
		// historical async send-on-channel left a window where a sibling
		// goroutine could miss the new client and double-broadcast a
		// stale "offline" state (BUG-07).
		deps.Hub.Register(client)

		_ = deps.PresenceSvc.SetOnline(r.Context(), userID)
		// Goroutines below outlive the HTTP handler — using
		// context.WithoutCancel so trace/baggage survives but
		// request cancellation does not propagate. Otherwise the
		// handler returning would tear down the WS connection.
		// gosec G118: explicit detached context replaces context.Background().
		bgCtx := context.WithoutCancel(r.Context())
		go broadcastPresenceChange(bgCtx, userID, true, deps)

		// writePump and readPump intentionally outlive the HTTP
		// handler — the WS connection persists for as long as the
		// client is online. They derive their own timeout contexts
		// internally (writeWait, pongWait) so request-scoped
		// cancellation does not apply. gosec G118 suppression is
		// scoped per call below.
		go writePump(conn, client) // #nosec G118 -- WS goroutine outlives the handler intentionally
		readPump(conn, client, deps)
	}
}

// authenticateWS authenticates a WebSocket upgrade request through one
// of two short-lived credentials:
//
//  1. session_id cookie (web, same-origin) — validated against Redis
//     session store.
//  2. ws_token query param (web cross-origin AND mobile, since SEC-15)
//     — validated against the single-use WS-token store.
//
// SEC-15: the legacy "?token=<JWT>" strategy was removed in Phase 1.
// Logging the long-lived JWT in proxies / access logs gave any
// log-aggregator-with-a-bug a free credentials capture. Mobile must
// now POST /api/v1/auth/ws-token with its Bearer token and connect
// using the returned single-use ticket — the same flow web has been
// using since SEC-15 shipped on web.
func authenticateWS(r *http.Request, _ service.TokenService, sessionSvc service.SessionService) (uuid.UUID, error) {
	if cookie, err := r.Cookie("session_id"); err == nil && cookie.Value != "" {
		session, err := sessionSvc.Get(r.Context(), cookie.Value)
		if err == nil {
			return session.UserID, nil
		}
	}

	if wsToken := r.URL.Query().Get("ws_token"); wsToken != "" {
		userID, err := sessionSvc.ValidateWSToken(r.Context(), wsToken)
		if err == nil {
			return userID, nil
		}
	}

	return uuid.UUID{}, errUnauthorizedWS
}

func readPump(conn *websocket.Conn, client *Client, deps ConnDeps) {
	defer func() {
		// Closes BUG-07: take the unregister decision under the hub
		// mutex so concurrent disconnects on the same user observe a
		// single authoritative wasLast=true signal. The previous code
		// computed `Hub.ConnectionCount() <= 1` BEFORE the unregister,
		// allowing two parallel disconnects to each broadcast an
		// offline event, or — worse — a fresh connection to register
		// between the count read and the unregister, leaving the user
		// erroneously marked offline while still online elsewhere.
		wasLast := deps.Hub.Unregister(client)
		if err := conn.Close(websocket.StatusNormalClosure, ""); err != nil {
			// A close error here is expected when the peer has already
			// disconnected (broken pipe / connection reset). We log at
			// DEBUG-equivalent WARN to avoid log spam while still
			// surfacing unexpected failure modes during incidents.
			slog.Warn("ws: close after read pump failed",
				"error", err, "user_id", client.UserID)
		}

		if wasLast {
			// readPump runs in its own goroutine that outlives the
			// HTTP handler — there is no request context to detach
			// from at this point. Use context.Background() with a
			// hard timeout for the offline propagation; gosec G118
			// is acceptable here because the parent goroutine has
			// already lost its request scope by definition.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // #nosec G118 -- detached after request lifetime
			defer cancel()
			_ = deps.PresenceSvc.SetOffline(ctx, client.UserID)
			// The presence-change goroutine inherits the timeout
			// context's lifetime but spawns its own detached one
			// internally (5s budget) — see broadcastPresenceChange.
			go broadcastPresenceChange(context.Background(), client.UserID, false, deps) // #nosec G118 -- detached after request lifetime
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

	sendOrDrop(client, pong, TypePong)
}

// sendOrDrop is the SINGLE blessed way to push a payload onto a
// Client.Send channel from within the readPump path. It performs a
// non-blocking send so a slow / wedged writePump can never block the
// readPump on a full buffer.
//
// Closes BUG-06: previously syncSingleConversation and sendError sent
// synchronously (`client.Send <- envelope`). When the writePump is slow
// — most commonly a mobile client backgrounded over a flaky link — the
// 64-slot Send buffer fills up and the readPump blocks until the
// pongWait timeout (60s) tears down the entire connection. During
// that 60s the goroutine is wedged, presence is incorrect, and the
// Client object leaks until the conn finally fails. The select-default
// pattern matches the existing Hub.SendToUser drop policy and keeps
// the readPump responsive: a dropped envelope is recoverable (the
// client can re-sync), a wedged readPump is not.
//
// envelopeKind is added to the structured log for triage — operators
// can grep for which message types are being dropped under load.
func sendOrDrop(client *Client, payload []byte, envelopeKind string) {
	select {
	case client.Send <- payload:
	default:
		slog.Warn("ws send buffer full, dropping",
			"client_user_id", client.UserID,
			"envelope_type", envelopeKind,
			"buffer_size", sendBufferSize,
		)
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

	sendOrDrop(client, envelope, TypeSyncResult)
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
	sendOrDrop(client, data, TypeError)
}

func writePump(conn *websocket.Conn, client *Client) {
	ticker := time.NewTicker(heartbeatTick)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-client.Send:
			if !ok {
				if err := conn.Close(websocket.StatusNormalClosure, ""); err != nil {
					// Caller-driven close: typically benign (socket
					// already gone). WARN so we still capture
					// unexpected close failures without flooding INFO.
					slog.Warn("ws: close on send-channel close failed",
						"error", err, "user_id", client.UserID)
				}
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

// broadcastPresenceChange notifies all contacts of a user's online/offline
// status. The supplied parent ctx is the request's detached
// (WithoutCancel) context — trace/baggage is preserved, request
// cancellation is not. We add a 5 second timeout so a slow downstream
// dependency cannot leak the goroutine.
//
// gosec G118: parent context comes from the caller (request-scoped
// + WithoutCancel), not context.Background().
func broadcastPresenceChange(parent context.Context, userID uuid.UUID, online bool, deps ConnDeps) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
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

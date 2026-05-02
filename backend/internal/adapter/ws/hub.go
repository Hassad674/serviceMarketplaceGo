package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

type Client struct {
	UserID uuid.UUID
	Send   chan []byte
	hub    *Hub
	// conn is the underlying WebSocket connection. Optional —
	// non-nil only on real ServeWS-issued clients. Tests that
	// fabricate a Client (send_or_drop_test.go, hub_test.go) leave
	// it nil; GracefulShutdown is robust to that.
	conn *websocket.Conn
}

type Hub struct {
	clients    map[uuid.UUID]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]bool),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
	}
}

// Run drains the legacy register/unregister channels. Synchronous callers
// should prefer Register / Unregister which return immediately under the
// hub mutex (the channel-driven path discards the wasLast signal so it
// cannot be used for race-free presence tracking — see BUG-07).
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			h.addClient(client)
		case client := <-h.unregister:
			h.removeClient(client)
		}
	}
}

// Register adds a client to the Hub synchronously, ensuring the Hub map
// is updated before the caller proceeds. Closes a subtle window in
// BUG-07 where a Send could be issued before the channel-buffered
// register operation completed.
func (h *Hub) Register(client *Client) {
	h.addClient(client)
}

// Unregister removes a client and reports whether this was the user's
// final connection — see removeClient for the wasLast contract. Always
// prefer Unregister over the unregister channel: the channel discards
// wasLast and reintroduces the BUG-07 race.
func (h *Hub) Unregister(client *Client) (wasLast bool) {
	return h.removeClient(client)
}

func (h *Hub) addClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[client.UserID] == nil {
		h.clients[client.UserID] = make(map[*Client]bool)
	}
	h.clients[client.UserID][client] = true
}

// removeClient unregisters a single Client from the Hub.
//
// It returns wasLast=true ONLY when this call removed the user's final
// active connection — i.e. the user has zero remaining connections after
// the operation completes AND the client we just removed was indeed
// registered. The decision is taken under the same mutex that mutates
// the connection map, so concurrent disconnects on the same userID are
// linearised: only ONE caller observes wasLast=true.
//
// Closes BUG-07 (WS isLast race on presence offline). Previously the
// readPump computed `Hub.ConnectionCount(userID) <= 1` BEFORE sending the
// client to the unregister channel — between the read and the
// unregister, another goroutine could mutate the map, and two parallel
// disconnects on the same userID would both observe `<= 1` and both
// broadcast a presence-offline event. The new contract returns the
// authoritative wasLast under the lock so callers have a definitive
// signal.
func (h *Hub) removeClient(client *Client) (wasLast bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conns, ok := h.clients[client.UserID]
	if !ok {
		return false
	}
	if _, exists := conns[client]; !exists {
		return false
	}

	delete(conns, client)
	close(client.Send)

	if len(conns) == 0 {
		delete(h.clients, client.UserID)
		return true
	}
	return false
}

// ConnectionCount returns the number of active WS connections for a user.
func (h *Hub) ConnectionCount(userID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients[userID])
}

func (h *Hub) SendToUser(userID uuid.UUID, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.clients[userID]
	if !ok {
		return
	}

	for client := range clients {
		select {
		case client.Send <- payload:
		default:
			slog.Warn("client send buffer full, dropping message",
				"user_id", userID.String())
		}
	}
}

func (h *Hub) SendToUsers(userIDs []uuid.UUID, payload []byte, excludeUserID uuid.UUID) {
	for _, id := range userIDs {
		if id == excludeUserID {
			continue
		}
		h.SendToUser(id, payload)
	}
}

// GracefulShutdown closes every active WebSocket connection with the
// 1001 "Going Away" status frame so clients can re-connect to the
// next instance instead of timing out on a dropped TCP connection.
//
// The function is bounded by ctx — if the context expires before all
// conns are closed (e.g. a wedged client) the remaining conns are
// abandoned and the function returns. This is the WS-shutdown sub-
// budget of runServer's 3-step graceful shutdown (commit P11 #5).
//
// Closing semantics:
//   - For each tracked conn, write a 1001 close frame with
//     "server shutting down" reason. Failures are logged at WARN
//     and ignored — a client whose socket is already half-dead will
//     fail the close, which is fine.
//   - The Send channel is NOT closed here. The writePump goroutine
//     observes the close-frame write and returns naturally; closing
//     the channel from two paths would race.
//   - Bookkeeping (h.clients map, presence) is left to the readPump
//     defer chain — once the client closes its side of the conn,
//     the existing Unregister path runs.
func (h *Hub) GracefulShutdown(ctx context.Context) error {
	h.mu.RLock()
	conns := make([]*websocket.Conn, 0)
	users := make([]uuid.UUID, 0)
	for userID, set := range h.clients {
		for client := range set {
			if client.conn != nil {
				conns = append(conns, client.conn)
				users = append(users, userID)
			}
		}
	}
	h.mu.RUnlock()

	slog.Info("ws: graceful shutdown starting", "active_connections", len(conns))

	const closeReason = "server shutting down"
	for i, conn := range conns {
		if ctx.Err() != nil {
			slog.Warn("ws: graceful shutdown timed out — abandoning remaining conns",
				"closed", i, "remaining", len(conns)-i)
			return ctx.Err()
		}
		// Per-conn deadline so a single slow client cannot consume
		// the whole budget. 500ms is plenty for a TCP write of a
		// close frame; anything slower is a stalled link we are
		// happy to abandon. coder/websocket's Close blocks on the
		// write until the deadline fires, then returns.
		_, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		err := conn.Close(websocket.StatusGoingAway, closeReason)
		cancel()
		if err != nil {
			slog.Warn("ws: graceful close failed",
				"user_id", users[i], "error", err)
		}
	}

	slog.Info("ws: graceful shutdown complete", "closed_connections", len(conns))
	return nil
}

// HandleStreamEvent dispatches Redis stream events to local WebSocket clients.
func (h *Hub) HandleStreamEvent(event StreamEvent) {
	var recipientIDs []uuid.UUID
	if err := json.Unmarshal([]byte(event.RecipientIDs), &recipientIDs); err != nil {
		slog.Error("failed to unmarshal recipient ids", "error", err)
		return
	}


	envelope := Envelope{
		Type:    event.Type,
		Payload: json.RawMessage(event.Payload),
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		slog.Error("failed to marshal stream event", "error", err)
		return
	}

	for _, id := range recipientIDs {
		h.SendToUser(id, data)
	}
}

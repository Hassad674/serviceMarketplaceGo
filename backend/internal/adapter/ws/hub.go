package ws

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

type Client struct {
	UserID uuid.UUID
	Send   chan []byte
	hub    *Hub
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

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.addClient(client)
		case client := <-h.unregister:
			h.removeClient(client)
		}
	}
}

func (h *Hub) addClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[client.UserID] == nil {
		h.clients[client.UserID] = make(map[*Client]bool)
	}
	h.clients[client.UserID][client] = true
}

func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.clients[client.UserID]; ok {
		if _, exists := conns[client]; exists {
			delete(conns, client)
			close(client.Send)
			if len(conns) == 0 {
				delete(h.clients, client.UserID)
			}
		}
	}
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
			// Client buffer full, skip
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

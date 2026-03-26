package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHub() *Hub {
	return NewHub()
}

func newTestClient(hub *Hub, userID uuid.UUID) *Client {
	return &Client{
		UserID: userID,
		Send:   make(chan []byte, sendBufferSize),
		hub:    hub,
	}
}

// --- Register / Unregister ---

func TestHub_AddClient(t *testing.T) {
	hub := newTestHub()
	userID := uuid.New()
	client := newTestClient(hub, userID)

	hub.addClient(client)

	assert.Equal(t, 1, hub.ConnectionCount(userID))
}

func TestHub_AddClient_MultipleConnections(t *testing.T) {
	hub := newTestHub()
	userID := uuid.New()
	client1 := newTestClient(hub, userID)
	client2 := newTestClient(hub, userID)

	hub.addClient(client1)
	hub.addClient(client2)

	assert.Equal(t, 2, hub.ConnectionCount(userID))
}

func TestHub_RemoveClient(t *testing.T) {
	hub := newTestHub()
	userID := uuid.New()
	client := newTestClient(hub, userID)

	hub.addClient(client)
	assert.Equal(t, 1, hub.ConnectionCount(userID))

	hub.removeClient(client)
	assert.Equal(t, 0, hub.ConnectionCount(userID))
}

func TestHub_RemoveClient_ClosesChannel(t *testing.T) {
	hub := newTestHub()
	userID := uuid.New()
	client := newTestClient(hub, userID)

	hub.addClient(client)
	hub.removeClient(client)

	// Channel should be closed — reading from it should return immediately
	_, ok := <-client.Send
	assert.False(t, ok, "Send channel should be closed after unregister")
}

func TestHub_RemoveClient_OnlyRemovesTarget(t *testing.T) {
	hub := newTestHub()
	userID := uuid.New()
	client1 := newTestClient(hub, userID)
	client2 := newTestClient(hub, userID)

	hub.addClient(client1)
	hub.addClient(client2)
	assert.Equal(t, 2, hub.ConnectionCount(userID))

	hub.removeClient(client1)
	assert.Equal(t, 1, hub.ConnectionCount(userID))
}

func TestHub_RemoveClient_DifferentUsers(t *testing.T) {
	hub := newTestHub()
	userA := uuid.New()
	userB := uuid.New()
	clientA := newTestClient(hub, userA)
	clientB := newTestClient(hub, userB)

	hub.addClient(clientA)
	hub.addClient(clientB)

	hub.removeClient(clientA)
	assert.Equal(t, 0, hub.ConnectionCount(userA))
	assert.Equal(t, 1, hub.ConnectionCount(userB))
}

func TestHub_RemoveClient_NotRegistered(t *testing.T) {
	hub := newTestHub()
	client := newTestClient(hub, uuid.New())

	// Should not panic when removing a client that was never registered
	hub.removeClient(client)
}

// --- ConnectionCount ---

func TestHub_ConnectionCount_NoClients(t *testing.T) {
	hub := newTestHub()

	assert.Equal(t, 0, hub.ConnectionCount(uuid.New()))
}

// --- SendToUser ---

func TestHub_SendToUser_ClientExists(t *testing.T) {
	hub := newTestHub()
	userID := uuid.New()
	client := newTestClient(hub, userID)

	hub.addClient(client)

	payload := []byte(`{"type":"test","payload":"hello"}`)
	hub.SendToUser(userID, payload)

	select {
	case received := <-client.Send:
		assert.Equal(t, payload, received)
	case <-time.After(time.Second):
		t.Fatal("expected message in send channel")
	}
}

func TestHub_SendToUser_ClientDoesNotExist(t *testing.T) {
	hub := newTestHub()

	// Should not panic when sending to a non-existent user
	hub.SendToUser(uuid.New(), []byte(`{"type":"test"}`))
}

func TestHub_SendToUser_MultipleConnections(t *testing.T) {
	hub := newTestHub()
	userID := uuid.New()
	client1 := newTestClient(hub, userID)
	client2 := newTestClient(hub, userID)

	hub.addClient(client1)
	hub.addClient(client2)

	payload := []byte(`{"type":"new_message"}`)
	hub.SendToUser(userID, payload)

	// Both clients should receive the message
	select {
	case received := <-client1.Send:
		assert.Equal(t, payload, received)
	case <-time.After(time.Second):
		t.Fatal("client1 should have received the message")
	}

	select {
	case received := <-client2.Send:
		assert.Equal(t, payload, received)
	case <-time.After(time.Second):
		t.Fatal("client2 should have received the message")
	}
}

func TestHub_SendToUser_BufferFull(t *testing.T) {
	hub := newTestHub()
	userID := uuid.New()
	// Create client with a buffer of 1
	client := &Client{
		UserID: userID,
		Send:   make(chan []byte, 1),
		hub:    hub,
	}

	hub.addClient(client)

	// Fill the buffer
	hub.SendToUser(userID, []byte("msg1"))
	// This should be dropped (buffer full) without blocking
	hub.SendToUser(userID, []byte("msg2"))

	select {
	case received := <-client.Send:
		assert.Equal(t, []byte("msg1"), received)
	case <-time.After(time.Second):
		t.Fatal("expected first message")
	}
}

// --- SendToUsers ---

func TestHub_SendToUsers_ExcludesSender(t *testing.T) {
	hub := newTestHub()
	senderID := uuid.New()
	recipientID := uuid.New()

	senderClient := newTestClient(hub, senderID)
	recipientClient := newTestClient(hub, recipientID)

	hub.addClient(senderClient)
	hub.addClient(recipientClient)

	payload := []byte(`{"type":"new_message"}`)
	hub.SendToUsers([]uuid.UUID{senderID, recipientID}, payload, senderID)

	// Recipient should receive
	select {
	case received := <-recipientClient.Send:
		assert.Equal(t, payload, received)
	case <-time.After(time.Second):
		t.Fatal("recipient should have received the message")
	}

	// Sender should NOT receive
	select {
	case <-senderClient.Send:
		t.Fatal("sender should not have received the message")
	case <-time.After(50 * time.Millisecond):
		// Expected: no message for sender
	}
}

func TestHub_SendToUsers_AllRecipients(t *testing.T) {
	hub := newTestHub()
	senderID := uuid.New()
	recipientA := uuid.New()
	recipientB := uuid.New()

	clientA := newTestClient(hub, recipientA)
	clientB := newTestClient(hub, recipientB)

	hub.addClient(clientA)
	hub.addClient(clientB)

	payload := []byte(`{"type":"typing"}`)
	hub.SendToUsers([]uuid.UUID{senderID, recipientA, recipientB}, payload, senderID)

	select {
	case received := <-clientA.Send:
		assert.Equal(t, payload, received)
	case <-time.After(time.Second):
		t.Fatal("clientA should have received")
	}

	select {
	case received := <-clientB.Send:
		assert.Equal(t, payload, received)
	case <-time.After(time.Second):
		t.Fatal("clientB should have received")
	}
}

// --- HandleStreamEvent ---

func TestHub_HandleStreamEvent_RoutesToRecipients(t *testing.T) {
	hub := newTestHub()
	recipientID := uuid.New()
	client := newTestClient(hub, recipientID)
	hub.addClient(client)

	recipientIDs, _ := json.Marshal([]uuid.UUID{recipientID})
	payload := `{"content":"hello"}`

	event := StreamEvent{
		Type:         TypeNewMessage,
		RecipientIDs: string(recipientIDs),
		Payload:      payload,
		SourceID:     "server-1",
	}

	hub.HandleStreamEvent(event)

	select {
	case received := <-client.Send:
		var envelope Envelope
		err := json.Unmarshal(received, &envelope)
		require.NoError(t, err)
		assert.Equal(t, TypeNewMessage, envelope.Type)
	case <-time.After(time.Second):
		t.Fatal("client should have received the stream event")
	}
}

func TestHub_HandleStreamEvent_InvalidRecipientIDs(t *testing.T) {
	hub := newTestHub()

	event := StreamEvent{
		Type:         TypeNewMessage,
		RecipientIDs: "invalid-json",
		Payload:      `{}`,
		SourceID:     "server-1",
	}

	// Should not panic
	hub.HandleStreamEvent(event)
}

func TestHub_HandleStreamEvent_NoRecipients(t *testing.T) {
	hub := newTestHub()

	recipientIDs, _ := json.Marshal([]uuid.UUID{})
	event := StreamEvent{
		Type:         TypeTypingEvent,
		RecipientIDs: string(recipientIDs),
		Payload:      `{}`,
		SourceID:     "server-1",
	}

	// Should not panic on empty recipients
	hub.HandleStreamEvent(event)
}

func TestHub_HandleStreamEvent_MultipleRecipients(t *testing.T) {
	hub := newTestHub()
	userA := uuid.New()
	userB := uuid.New()
	clientA := newTestClient(hub, userA)
	clientB := newTestClient(hub, userB)
	hub.addClient(clientA)
	hub.addClient(clientB)

	recipientIDs, _ := json.Marshal([]uuid.UUID{userA, userB})
	event := StreamEvent{
		Type:         TypePresence,
		RecipientIDs: string(recipientIDs),
		Payload:      `{"user_id":"x","online":true}`,
		SourceID:     "server-1",
	}

	hub.HandleStreamEvent(event)

	for _, client := range []*Client{clientA, clientB} {
		select {
		case received := <-client.Send:
			var envelope Envelope
			err := json.Unmarshal(received, &envelope)
			require.NoError(t, err)
			assert.Equal(t, TypePresence, envelope.Type)
		case <-time.After(time.Second):
			t.Fatal("both clients should have received the event")
		}
	}
}

// --- broadcastToOthers ---

func TestHub_BroadcastToOthers_ExcludesSender(t *testing.T) {
	hub := newTestHub()
	senderID := uuid.New()
	otherID := uuid.New()

	senderClient := newTestClient(hub, senderID)
	otherClient := newTestClient(hub, otherID)
	hub.addClient(senderClient)
	hub.addClient(otherClient)

	payload := []byte(`{"conversation_id":"conv-1","user_id":"sender"}`)
	err := hub.broadcastToOthers(senderID, []uuid.UUID{senderID, otherID}, TypeTypingEvent, payload)
	require.NoError(t, err)

	// Other should receive
	select {
	case received := <-otherClient.Send:
		var envelope Envelope
		err := json.Unmarshal(received, &envelope)
		require.NoError(t, err)
		assert.Equal(t, TypeTypingEvent, envelope.Type)
	case <-time.After(time.Second):
		t.Fatal("other client should have received")
	}

	// Sender should NOT receive
	select {
	case <-senderClient.Send:
		t.Fatal("sender should not receive their own broadcast")
	case <-time.After(50 * time.Millisecond):
		// Expected
	}
}

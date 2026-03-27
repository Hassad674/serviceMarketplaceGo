package call

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	calldomain "marketplace-backend/internal/domain/call"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/service"
)

// --- Mocks ---

type mockLiveKit struct {
	createRoomCalled bool
	deleteRoomCalled bool
	tokenGenerated   bool
}

func (m *mockLiveKit) CreateRoom(_ context.Context, _ string) error {
	m.createRoomCalled = true
	return nil
}

func (m *mockLiveKit) GenerateToken(_, _, _ string) (string, error) {
	m.tokenGenerated = true
	return "mock-livekit-token", nil
}

func (m *mockLiveKit) DeleteRoom(_ context.Context, _ string) error {
	m.deleteRoomCalled = true
	return nil
}

type mockCallState struct {
	calls map[uuid.UUID]*calldomain.Call
	users map[uuid.UUID]uuid.UUID
}

func newMockCallState() *mockCallState {
	return &mockCallState{
		calls: make(map[uuid.UUID]*calldomain.Call),
		users: make(map[uuid.UUID]uuid.UUID),
	}
}

func (m *mockCallState) SaveActiveCall(_ context.Context, c *calldomain.Call) error {
	m.calls[c.ID] = c
	m.users[c.InitiatorID] = c.ID
	m.users[c.RecipientID] = c.ID
	return nil
}

func (m *mockCallState) GetActiveCall(_ context.Context, id uuid.UUID) (*calldomain.Call, error) {
	c, ok := m.calls[id]
	if !ok {
		return nil, calldomain.ErrCallNotFound
	}
	return c, nil
}

func (m *mockCallState) GetActiveCallByUser(_ context.Context, userID uuid.UUID) (*calldomain.Call, error) {
	callID, ok := m.users[userID]
	if !ok {
		return nil, calldomain.ErrCallNotFound
	}
	return m.GetActiveCall(context.Background(), callID)
}

func (m *mockCallState) RemoveActiveCall(_ context.Context, id uuid.UUID) error {
	c, ok := m.calls[id]
	if ok {
		delete(m.users, c.InitiatorID)
		delete(m.users, c.RecipientID)
		delete(m.calls, id)
	}
	return nil
}

type mockPresence struct {
	online map[uuid.UUID]bool
}

func newMockPresence() *mockPresence {
	return &mockPresence{online: make(map[uuid.UUID]bool)}
}

func (m *mockPresence) SetOnline(_ context.Context, id uuid.UUID) error {
	m.online[id] = true
	return nil
}
func (m *mockPresence) SetOffline(_ context.Context, id uuid.UUID) error {
	m.online[id] = false
	return nil
}
func (m *mockPresence) IsOnline(_ context.Context, id uuid.UUID) (bool, error) {
	return m.online[id], nil
}
func (m *mockPresence) BulkIsOnline(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	res := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		res[id] = m.online[id]
	}
	return res, nil
}

type mockBroadcaster struct {
	events []broadcastEvent
}

type broadcastEvent struct {
	recipientIDs []uuid.UUID
	payload      []byte
}

func (m *mockBroadcaster) BroadcastCallEvent(_ context.Context, ids []uuid.UUID, payload []byte) error {
	m.events = append(m.events, broadcastEvent{recipientIDs: ids, payload: payload})
	return nil
}

type mockMessageSender struct {
	sent []service.SystemMessageInput
}

func (m *mockMessageSender) SendSystemMessage(_ context.Context, input service.SystemMessageInput) error {
	m.sent = append(m.sent, input)
	return nil
}

type mockUserRepo struct {
	users map[uuid.UUID]*user.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[uuid.UUID]*user.User)}
}

func (m *mockUserRepo) Create(_ context.Context, u *user.User) error {
	m.users[u.ID] = u
	return nil
}
func (m *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*user.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, user.ErrUserNotFound
	}
	return u, nil
}
func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (m *mockUserRepo) Update(_ context.Context, _ *user.User) error  { return nil }
func (m *mockUserRepo) Delete(_ context.Context, _ uuid.UUID) error   { return nil }
func (m *mockUserRepo) ExistsByEmail(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// --- Helper ---

func setupService() (*Service, *mockLiveKit, *mockCallState, *mockPresence, *mockBroadcaster, *mockMessageSender) {
	lk := &mockLiveKit{}
	cs := newMockCallState()
	pr := newMockPresence()
	br := &mockBroadcaster{}
	ms := &mockMessageSender{}
	ur := newMockUserRepo()

	svc := NewService(ServiceDeps{
		LiveKit:     lk,
		CallState:   cs,
		Presence:    pr,
		Broadcaster: br,
		Messages:    ms,
		Users:       ur,
	})
	return svc, lk, cs, pr, br, ms
}

// --- Tests ---

func TestInitiate_HappyPath(t *testing.T) {
	svc, lk, _, pr, br, _ := setupService()
	initiatorID := uuid.New()
	recipientID := uuid.New()
	pr.online[recipientID] = true

	result, err := svc.Initiate(context.Background(), InitiateInput{
		ConversationID: uuid.New(),
		InitiatorID:    initiatorID,
		RecipientID:    recipientID,
		Type:           calldomain.TypeAudio,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.CallID)
	assert.NotEmpty(t, result.RoomName)
	assert.Equal(t, "mock-livekit-token", result.InitiatorToken)
	assert.True(t, lk.createRoomCalled)
	assert.True(t, lk.tokenGenerated)
	require.Len(t, br.events, 1)
	assert.Equal(t, []uuid.UUID{recipientID}, br.events[0].recipientIDs)
}

func TestInitiate_RecipientOffline(t *testing.T) {
	svc, _, _, _, _, _ := setupService()
	_, err := svc.Initiate(context.Background(), InitiateInput{
		ConversationID: uuid.New(),
		InitiatorID:    uuid.New(),
		RecipientID:    uuid.New(),
		Type:           calldomain.TypeAudio,
	})
	assert.ErrorIs(t, err, calldomain.ErrRecipientOffline)
}

func TestInitiate_InitiatorBusy(t *testing.T) {
	svc, _, cs, pr, _, _ := setupService()
	initiatorID := uuid.New()
	recipientID := uuid.New()
	pr.online[recipientID] = true

	// Simulate initiator already in a call
	existingCall := &calldomain.Call{
		ID:          uuid.New(),
		InitiatorID: initiatorID,
		RecipientID: uuid.New(),
	}
	_ = cs.SaveActiveCall(context.Background(), existingCall)

	_, err := svc.Initiate(context.Background(), InitiateInput{
		ConversationID: uuid.New(),
		InitiatorID:    initiatorID,
		RecipientID:    recipientID,
		Type:           calldomain.TypeAudio,
	})
	assert.ErrorIs(t, err, calldomain.ErrUserBusy)
}

func TestAccept_HappyPath(t *testing.T) {
	svc, lk, cs, pr, br, _ := setupService()
	initiatorID := uuid.New()
	recipientID := uuid.New()
	pr.online[recipientID] = true

	result, _ := svc.Initiate(context.Background(), InitiateInput{
		ConversationID: uuid.New(),
		InitiatorID:    initiatorID,
		RecipientID:    recipientID,
		Type:           calldomain.TypeAudio,
	})

	lk.tokenGenerated = false
	acceptResult, err := svc.Accept(context.Background(), result.CallID, recipientID)

	require.NoError(t, err)
	assert.Equal(t, "mock-livekit-token", acceptResult.Token)
	assert.True(t, lk.tokenGenerated)

	// Check call state is now active
	c, _ := cs.GetActiveCall(context.Background(), result.CallID)
	assert.Equal(t, calldomain.StatusActive, c.Status)

	// Check broadcast to initiator
	require.Len(t, br.events, 2)
	assert.Equal(t, []uuid.UUID{initiatorID}, br.events[1].recipientIDs)
}

func TestAccept_WrongUser(t *testing.T) {
	svc, _, _, pr, _, _ := setupService()
	recipientID := uuid.New()
	pr.online[recipientID] = true

	result, _ := svc.Initiate(context.Background(), InitiateInput{
		ConversationID: uuid.New(),
		InitiatorID:    uuid.New(),
		RecipientID:    recipientID,
		Type:           calldomain.TypeAudio,
	})

	_, err := svc.Accept(context.Background(), result.CallID, uuid.New())
	assert.ErrorIs(t, err, calldomain.ErrNotParticipant)
}

func TestDecline_HappyPath(t *testing.T) {
	svc, lk, cs, pr, br, _ := setupService()
	initiatorID := uuid.New()
	recipientID := uuid.New()
	pr.online[recipientID] = true

	result, _ := svc.Initiate(context.Background(), InitiateInput{
		ConversationID: uuid.New(),
		InitiatorID:    initiatorID,
		RecipientID:    recipientID,
		Type:           calldomain.TypeAudio,
	})

	err := svc.Decline(context.Background(), result.CallID, recipientID)
	require.NoError(t, err)
	assert.True(t, lk.deleteRoomCalled)

	// Call should be removed
	_, err = cs.GetActiveCall(context.Background(), result.CallID)
	assert.ErrorIs(t, err, calldomain.ErrCallNotFound)

	// Broadcast decline to initiator
	require.Len(t, br.events, 2)
	assert.Equal(t, []uuid.UUID{initiatorID}, br.events[1].recipientIDs)
}

func TestEnd_HappyPath(t *testing.T) {
	svc, lk, cs, pr, _, ms := setupService()
	initiatorID := uuid.New()
	recipientID := uuid.New()
	pr.online[recipientID] = true

	result, _ := svc.Initiate(context.Background(), InitiateInput{
		ConversationID: uuid.New(),
		InitiatorID:    initiatorID,
		RecipientID:    recipientID,
		Type:           calldomain.TypeAudio,
	})
	_, _ = svc.Accept(context.Background(), result.CallID, recipientID)

	err := svc.End(context.Background(), EndInput{
		CallID:   result.CallID,
		UserID:   initiatorID,
		Duration: 120,
	})
	require.NoError(t, err)
	assert.True(t, lk.deleteRoomCalled)

	// Call removed
	_, err = cs.GetActiveCall(context.Background(), result.CallID)
	assert.ErrorIs(t, err, calldomain.ErrCallNotFound)

	// System message sent
	require.Len(t, ms.sent, 1)
	assert.Contains(t, ms.sent[0].Content, "2:00")
}

func TestEnd_MissedCall_ZeroDuration(t *testing.T) {
	svc, _, _, pr, _, ms := setupService()
	initiatorID := uuid.New()
	recipientID := uuid.New()
	pr.online[recipientID] = true

	result, _ := svc.Initiate(context.Background(), InitiateInput{
		ConversationID: uuid.New(),
		InitiatorID:    initiatorID,
		RecipientID:    recipientID,
		Type:           calldomain.TypeAudio,
	})

	// End while still ringing (not accepted) with duration 0
	err := svc.End(context.Background(), EndInput{
		CallID:   result.CallID,
		UserID:   initiatorID,
		Duration: 0,
	})
	require.NoError(t, err)

	// System message should say "Missed call", not "Audio call - 0:00"
	require.Len(t, ms.sent, 1)
	assert.Equal(t, "Missed call", ms.sent[0].Content)
	assert.Equal(t, "call_missed", ms.sent[0].Type)
}

func TestEnd_NotParticipant(t *testing.T) {
	svc, _, _, pr, _, _ := setupService()
	recipientID := uuid.New()
	pr.online[recipientID] = true

	result, _ := svc.Initiate(context.Background(), InitiateInput{
		ConversationID: uuid.New(),
		InitiatorID:    uuid.New(),
		RecipientID:    recipientID,
		Type:           calldomain.TypeAudio,
	})
	_, _ = svc.Accept(context.Background(), result.CallID, recipientID)

	err := svc.End(context.Background(), EndInput{
		CallID:   result.CallID,
		UserID:   uuid.New(), // stranger
		Duration: 10,
	})
	assert.ErrorIs(t, err, calldomain.ErrNotParticipant)
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{0, "0:00"},
		{5, "0:05"},
		{60, "1:00"},
		{120, "2:00"},
		{3661, "61:01"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, formatDuration(tt.seconds))
	}
}

func TestBroadcastCallSignal_PayloadStructure(t *testing.T) {
	svc, _, _, pr, br, _ := setupService()
	recipientID := uuid.New()
	pr.online[recipientID] = true

	result, _ := svc.Initiate(context.Background(), InitiateInput{
		ConversationID: uuid.New(),
		InitiatorID:    uuid.New(),
		RecipientID:    recipientID,
		Type:           calldomain.TypeAudio,
	})
	require.NotNil(t, result)
	require.Len(t, br.events, 1)

	var payload map[string]string
	err := json.Unmarshal(br.events[0].payload, &payload)
	require.NoError(t, err)
	assert.Equal(t, "call_incoming", payload["event"])
	assert.Equal(t, "audio", payload["call_type"])
	assert.NotEmpty(t, payload["call_id"])
	// Verify caller name fields are present (resolve to "User" since mock has no users)
	assert.Equal(t, "User", payload["initiator_name"])
	assert.Equal(t, "User", payload["recipient_name"])
}

package ws

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// presenceSnapshotEnvelope is the shape we expect on the wire when
// the new TypePresenceSnapshot frame is emitted.
type presenceSnapshotEnvelope struct {
	Type    string `json:"type"`
	Payload struct {
		OnlineUserIDs []string `json:"online_user_ids"`
	} `json:"payload"`
}

// captureSnapshot reads one frame from the client's Send channel with
// a short deadline. The test fails if no frame arrives.
func captureSnapshot(t *testing.T, client *Client) presenceSnapshotEnvelope {
	t.Helper()
	select {
	case data := <-client.Send:
		var env presenceSnapshotEnvelope
		require.NoError(t, json.Unmarshal(data, &env), "snapshot frame must be valid JSON")
		return env
	case <-time.After(2 * time.Second):
		t.Fatal("expected a presence_snapshot frame on the Send channel")
		return presenceSnapshotEnvelope{}
	}
}

// TestSendPresenceSnapshot_SendsFrameWithOnlinePartners — table-driven
// scenarios covering the primary path: a freshly connected user must
// receive a snapshot listing the conversation partners that are
// currently online.
func TestSendPresenceSnapshot_SendsFrameWithOnlinePartners(t *testing.T) {
	bID := uuid.New()
	cID := uuid.New()
	dID := uuid.New()

	tests := []struct {
		name           string
		contactIDs     []uuid.UUID
		onlineMap      map[uuid.UUID]bool
		wantOnlineUIDs []string
	}{
		{
			name:           "single partner online",
			contactIDs:     []uuid.UUID{bID},
			onlineMap:      map[uuid.UUID]bool{bID: true},
			wantOnlineUIDs: []string{bID.String()},
		},
		{
			name:           "mixed online and offline partners",
			contactIDs:     []uuid.UUID{bID, cID, dID},
			onlineMap:      map[uuid.UUID]bool{bID: true, cID: false, dID: true},
			wantOnlineUIDs: []string{bID.String(), dID.String()},
		},
		{
			name:           "all partners offline",
			contactIDs:     []uuid.UUID{bID, cID},
			onlineMap:      map[uuid.UUID]bool{bID: false, cID: false},
			wantOnlineUIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newUser := uuid.New()
			client := &Client{UserID: newUser, Send: make(chan []byte, 4)}

			mq := &fakeMessagingQuerier{
				getContactIDs: func(_ context.Context, uid uuid.UUID) ([]uuid.UUID, error) {
					assert.Equal(t, newUser, uid)
					return tt.contactIDs, nil
				},
			}
			pres := &fakePresenceSvcMap{online: tt.onlineMap}
			deps := ConnDeps{MessagingSvc: mq, PresenceSvc: pres}

			sendPresenceSnapshot(context.Background(), client, deps)

			env := captureSnapshot(t, client)
			assert.Equal(t, TypePresenceSnapshot, env.Type)
			assert.ElementsMatch(t, tt.wantOnlineUIDs, env.Payload.OnlineUserIDs)
			assert.Equal(t, int32(1), atomic.LoadInt32(&pres.bulkCalls),
				"BulkIsOnline must be called once (batched, no N+1)")
		})
	}
}

// TestSendPresenceSnapshot_ScopedToConversationPartners ensures the
// snapshot for a user with NO conversations is empty — no leak of
// unrelated online users. This is the privacy invariant.
func TestSendPresenceSnapshot_ScopedToConversationPartners(t *testing.T) {
	newUser := uuid.New()
	client := &Client{UserID: newUser, Send: make(chan []byte, 4)}

	mq := &fakeMessagingQuerier{
		getContactIDs: func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
			return []uuid.UUID{}, nil // no conversations
		},
	}
	pres := &fakePresenceSvcMap{online: map[uuid.UUID]bool{uuid.New(): true}}
	deps := ConnDeps{MessagingSvc: mq, PresenceSvc: pres}

	sendPresenceSnapshot(context.Background(), client, deps)

	env := captureSnapshot(t, client)
	assert.Equal(t, TypePresenceSnapshot, env.Type)
	assert.Empty(t, env.Payload.OnlineUserIDs,
		"user with no conversations must NOT receive any online ids — privacy invariant")
	assert.Equal(t, int32(0), atomic.LoadInt32(&pres.bulkCalls),
		"BulkIsOnline must be skipped when there are no partners — no useless Redis call")
}

// TestSendPresenceSnapshot_BatchedQuery asserts BulkIsOnline is called
// EXACTLY ONCE — not once per partner. This pins the N+1 invariant.
func TestSendPresenceSnapshot_BatchedQuery(t *testing.T) {
	newUser := uuid.New()
	client := &Client{UserID: newUser, Send: make(chan []byte, 4)}

	partners := make([]uuid.UUID, 50)
	online := map[uuid.UUID]bool{}
	for i := range partners {
		partners[i] = uuid.New()
		online[partners[i]] = i%2 == 0
	}

	mq := &fakeMessagingQuerier{
		getContactIDs: func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
			return partners, nil
		},
	}
	pres := &fakePresenceSvcMap{online: online}
	deps := ConnDeps{MessagingSvc: mq, PresenceSvc: pres}

	sendPresenceSnapshot(context.Background(), client, deps)

	env := captureSnapshot(t, client)
	assert.Equal(t, TypePresenceSnapshot, env.Type)
	assert.Len(t, env.Payload.OnlineUserIDs, 25,
		"exactly half of the 50 partners should be reported online")
	assert.Equal(t, int32(1), atomic.LoadInt32(&pres.bulkCalls),
		"50 partners must be checked in a single BulkIsOnline call — N+1 invariant")
	assert.Equal(t, int32(50), atomic.LoadInt32(&pres.lastBulkSize),
		"the single BulkIsOnline call must batch all 50 ids")
}

// TestSendPresenceSnapshot_ContactIDsErrorIsSilent — a downstream
// failure must not leak as an error frame, must not crash the
// goroutine, and must not push anything to the client. The frontend
// safety-net (refetch on open) handles the missed snapshot.
func TestSendPresenceSnapshot_ContactIDsErrorIsSilent(t *testing.T) {
	client := &Client{UserID: uuid.New(), Send: make(chan []byte, 4)}

	mq := &fakeMessagingQuerier{
		getContactIDs: func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
			return nil, assertErr("contact fetch failed")
		},
	}
	pres := &fakePresenceSvcMap{}
	deps := ConnDeps{MessagingSvc: mq, PresenceSvc: pres}

	sendPresenceSnapshot(context.Background(), client, deps)

	select {
	case <-client.Send:
		t.Fatal("expected no frame when GetContactIDs fails — must fail silently")
	case <-time.After(50 * time.Millisecond):
		// Expected.
	}
}

// TestSendPresenceSnapshot_BulkIsOnlineErrorIsSilent — second failure
// surface: contacts fetch succeeds but Redis bulk check fails. The
// snapshot is dropped, no frame is sent.
func TestSendPresenceSnapshot_BulkIsOnlineErrorIsSilent(t *testing.T) {
	client := &Client{UserID: uuid.New(), Send: make(chan []byte, 4)}

	mq := &fakeMessagingQuerier{
		getContactIDs: func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
			return []uuid.UUID{uuid.New()}, nil
		},
	}
	pres := &fakePresenceSvcMap{bulkErr: assertErr("redis down")}
	deps := ConnDeps{MessagingSvc: mq, PresenceSvc: pres}

	sendPresenceSnapshot(context.Background(), client, deps)

	select {
	case <-client.Send:
		t.Fatal("expected no frame when BulkIsOnline fails — must fail silently")
	case <-time.After(50 * time.Millisecond):
		// Expected.
	}
}

// --- helpers ---

// fakePresenceSvcMap is a richer presence service stub that lets tests
// supply a fixed online map AND inspect call counts to assert the
// batched-query invariant.
type fakePresenceSvcMap struct {
	online       map[uuid.UUID]bool
	bulkCalls    int32
	lastBulkSize int32
	bulkErr      error
}

func (f *fakePresenceSvcMap) SetOnline(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (f *fakePresenceSvcMap) SetOffline(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (f *fakePresenceSvcMap) IsOnline(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (f *fakePresenceSvcMap) BulkIsOnline(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	atomic.AddInt32(&f.bulkCalls, 1)
	atomic.StoreInt32(&f.lastBulkSize, int32(len(ids)))
	if f.bulkErr != nil {
		return nil, f.bulkErr
	}
	out := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		out[id] = f.online[id]
	}
	return out, nil
}

// assertErr is a tiny error helper to avoid pulling errors.New everywhere.
type assertErr string

func (e assertErr) Error() string { return string(e) }

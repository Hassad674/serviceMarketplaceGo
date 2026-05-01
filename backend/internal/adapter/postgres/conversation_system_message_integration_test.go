package postgres_test

// Integration test for the system-actor message persistence path.
// Reproduces the production bug where messages.sender_id NOT NULL +
// FK on users(id) silently dropped every uuid.Nil-sender message
// emitted by runEndOfProjectEffects, leaving completed missions with
// no proposal_completed / evaluation_request card.
//
// Gated behind MARKETPLACE_TEST_DATABASE_URL — auto-skips when unset.
// Migration 128 must be applied on the target DB.

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/message"
)

// insertConversationWithParticipants creates a fresh conversation row
// with two real-user participants. The cleanup hook tears the
// conversation back down once the test exits.
func insertConversationWithParticipants(t *testing.T, db *sql.DB, userA, userB uuid.UUID) uuid.UUID {
	t.Helper()
	convID := uuid.New()
	_, err := db.Exec(`INSERT INTO conversations (id) VALUES ($1)`, convID)
	require.NoError(t, err, "insert conversation")
	_, err = db.Exec(`
		INSERT INTO conversation_participants (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3)`, convID, userA, userB)
	require.NoError(t, err, "insert participants")

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM messages WHERE conversation_id = $1`, convID)
		_, _ = db.Exec(`DELETE FROM conversation_participants WHERE conversation_id = $1`, convID)
		_, _ = db.Exec(`DELETE FROM conversations WHERE id = $1`, convID)
	})
	return convID
}

// TestCreateMessage_SystemActor_PersistsAsNULL drives the real DB
// adapter end-to-end and asserts that a uuid.Nil-sender message
// (proposal_completed, evaluation_request, milestone_auto_approved,
// proposal_auto_closed, dispute_auto_resolved) is actually persisted
// with sender_id = NULL.
//
// Before migration 130 this test would fail with a NOT NULL violation
// at insert time — exactly the production failure mode that hid
// completed-mission cards from the conversation.
func TestCreateMessage_SystemActor_PersistsAsNULL(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewConversationRepository(db)

	userA := insertTestUser(t, db)
	userB := insertTestUser(t, db)
	convID := insertConversationWithParticipants(t, db, userA, userB)

	systemTypes := []message.MessageType{
		message.MessageTypeProposalCompleted,
		message.MessageTypeEvaluationRequest,
	}
	for _, msgType := range systemTypes {
		t.Run(string(msgType), func(t *testing.T) {
			msg, err := message.NewMessage(message.NewMessageInput{
				ConversationID: convID,
				SenderID:       uuid.Nil, // system actor
				Type:           msgType,
				Metadata:       []byte(`{"proposal_id": "00000000-0000-0000-0000-000000000001"}`),
			})
			require.NoError(t, err)

			require.NoError(t, repo.CreateMessage(context.Background(), msg),
				"system-actor send must persist (migration 130 makes sender_id nullable)")

			var senderID *uuid.UUID
			var storedType string
			err = db.QueryRow(
				`SELECT sender_id, msg_type FROM messages WHERE id = $1`, msg.ID,
			).Scan(&senderID, &storedType)
			require.NoError(t, err)
			assert.Nil(t, senderID, "sender_id must be SQL NULL for system-actor messages")
			assert.Equal(t, string(msgType), storedType, "message type must round-trip")

			// Round-trip through the read path: NULL must come back as
			// uuid.Nil so the rest of the app can keep its "system
			// actor" branch on the sentinel.
			read, err := repo.GetMessage(context.Background(), msg.ID)
			require.NoError(t, err)
			assert.Equal(t, uuid.Nil, read.SenderID, "read path must surface uuid.Nil sentinel")
			assert.Equal(t, msgType, read.Type)
		})
	}
}

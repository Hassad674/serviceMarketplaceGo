package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/message"
)

// ---------------------------------------------------------------------------
// Bug fix coverage: messages.sender_id must accept SQL NULL for
// system-actor sends.
//
// The bug: runEndOfProjectEffects (and the scheduler) emits
// proposal_completed / evaluation_request / milestone_auto_approved /
// proposal_auto_closed system messages with SenderID = uuid.Nil. Before
// migration 128 the sender_id column was NOT NULL with a FK on
// users(id), so the row insert failed (FK violation when bound as the
// zero UUID, NOT-NULL violation when bound as SQL NULL). The proposal
// service ignored the error from SendSystemMessage so the message
// silently dropped — neither the conversation card nor the review
// modal trigger ever appeared, blocking the entire post-mission review
// flow.
//
// These tests pin two invariants:
//
//  1. senderForInsert maps uuid.Nil → nil (driver-level SQL NULL),
//     never the zero UUID byte string.
//  2. CreateMessage binds NULL when SenderID is uuid.Nil, so the
//     adapter relies on the column being nullable (post-migration 128)
//     to persist the row.
// ---------------------------------------------------------------------------

func TestSenderForInsert_NilSentinelMapsToNULL(t *testing.T) {
	got := senderForInsert(uuid.Nil)
	assert.Nil(t, got, "uuid.Nil must map to driver NULL — the FK on users(id) cannot resolve the zero UUID and silently rolls back the insert")
}

func TestSenderForInsert_RealUserPassesThrough(t *testing.T) {
	id := uuid.New()
	got := senderForInsert(id)
	assert.Equal(t, id, got, "real user ids must round-trip unchanged")
}

func TestSenderForRead_NULLMapsToNilSentinel(t *testing.T) {
	got := senderForRead(nil)
	assert.Equal(t, uuid.Nil, got, "SQL NULL must round-trip to uuid.Nil sentinel — the rest of the app branches on uuid.Nil to recognise system actors")
}

func TestCreateMessage_SystemActorBindsNULL(t *testing.T) {
	// Drives the full CreateMessage flow with sqlmock and asserts
	// that the INSERT into messages binds nil (SQL NULL) for the
	// sender_id column when SenderID is uuid.Nil. Failing to bind
	// NULL is exactly what dropped end-of-project system messages
	// (FK violation on the zero UUID).
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewConversationRepository(db)
	convID := uuid.New()
	now := time.Now().UTC()

	msg, mErr := message.NewMessage(message.NewMessageInput{
		ConversationID: convID,
		SenderID:       uuid.Nil,
		Content:        "",
		Type:           message.MessageTypeProposalCompleted,
	})
	require.NoError(t, mErr)
	msg.CreatedAt = now
	msg.UpdatedAt = now

	mock.ExpectBegin()
	mock.ExpectExec(`SELECT id FROM conversations WHERE id = \$1 FOR UPDATE`).
		WithArgs(convID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT COALESCE\(MAX\(seq\), 0\) \+ 1 FROM messages WHERE conversation_id = \$1`).
		WithArgs(convID).
		WillReturnRows(sqlmock.NewRows([]string{"next"}).AddRow(42))
	// $3 must bind to nil — NOT to the zero UUID string. sqlmock
	// only treats `nil` as matching SQL NULL; if the adapter ever
	// degrades to binding the zero UUID, the matcher fails the test
	// loudly instead of silently passing.
	mock.ExpectExec(`INSERT INTO messages \(id, conversation_id, sender_id, content, msg_type, metadata, reply_to_id, seq, status, created_at, updated_at\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7, \$8, \$9, \$10, \$11\)`).
		WithArgs(
			msg.ID, convID, nil, "", "proposal_completed",
			[]byte(nil), sql.NullString{}, 42, "sent", now, now,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE conversations SET updated_at = \$2 WHERE id = \$1`).
		WithArgs(convID, now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.CreateMessage(context.Background(), msg))
	require.NoError(t, mock.ExpectationsWereMet(),
		"sender_id MUST bind as NULL for system-actor sends — otherwise the FK on users(id) silently drops the message")
}

func TestCreateMessage_RealSenderBindsUUID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewConversationRepository(db)
	convID := uuid.New()
	senderID := uuid.New()
	now := time.Now().UTC()

	msg, mErr := message.NewMessage(message.NewMessageInput{
		ConversationID: convID,
		SenderID:       senderID,
		Content:        "hello",
		Type:           message.MessageTypeText,
	})
	require.NoError(t, mErr)
	msg.CreatedAt = now
	msg.UpdatedAt = now

	mock.ExpectBegin()
	mock.ExpectExec(`SELECT id FROM conversations WHERE id = \$1 FOR UPDATE`).
		WithArgs(convID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT COALESCE\(MAX\(seq\), 0\) \+ 1 FROM messages WHERE conversation_id = \$1`).
		WithArgs(convID).
		WillReturnRows(sqlmock.NewRows([]string{"next"}).AddRow(7))
	mock.ExpectExec(`INSERT INTO messages \(id, conversation_id, sender_id, content, msg_type, metadata, reply_to_id, seq, status, created_at, updated_at\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7, \$8, \$9, \$10, \$11\)`).
		WithArgs(
			msg.ID, convID, senderID, "hello", "text",
			[]byte(nil), sql.NullString{}, 7, "sent", now, now,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE conversations SET updated_at = \$2 WHERE id = \$1`).
		WithArgs(convID, now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.CreateMessage(context.Background(), msg))
	require.NoError(t, mock.ExpectationsWereMet())
}


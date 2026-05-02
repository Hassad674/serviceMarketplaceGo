package postgres

import (
	"context"
	"database/sql"
	"strings"
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
// migration 130 the sender_id column was NOT NULL with a FK on
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
//     adapter relies on the column being nullable (post-migration 130)
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
	// P6: createMessageInTx now also denormalizes onto conversations.
	// last_message_sender_id MUST bind as NULL for system-actor sends —
	// same NULL contract as messages.sender_id.
	mock.ExpectExec(`UPDATE conversations\s+SET updated_at\s+= \$2,\s+last_message_seq\s+= \$3,\s+last_message_content_preview = LEFT\(\$4, 100\),\s+last_message_at\s+= \$2,\s+last_message_sender_id\s+= \$5\s+WHERE id = \$1`).
		WithArgs(convID, now, 42, "", nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	// Pass uuid.Nil for the tenant args — the repo built without a
	// txRunner falls through to createMessageLegacy where the tenant
	// values are unused (SET LOCAL is a no-op outside the RLS path).
	require.NoError(t, repo.CreateMessage(context.Background(), msg, uuid.Nil, uuid.Nil))
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
	// P6: real-sender path binds the sender uuid (not nil) to
	// last_message_sender_id, mirroring messages.sender_id.
	mock.ExpectExec(`UPDATE conversations\s+SET updated_at\s+= \$2,\s+last_message_seq\s+= \$3,\s+last_message_content_preview = LEFT\(\$4, 100\),\s+last_message_at\s+= \$2,\s+last_message_sender_id\s+= \$5\s+WHERE id = \$1`).
		WithArgs(convID, now, 7, "hello", senderID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	// Same legacy path — uuid.Nil tenant args.
	require.NoError(t, repo.CreateMessage(context.Background(), msg, uuid.Nil, uuid.Nil))
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// P6 — denormalize last_message_* on conversations
//
// These tests pin three invariants of the new denormalized write path:
//
//   1. createMessageInTx fires a SINGLE UPDATE that bumps updated_at
//      AND maintains last_message_seq / preview / at / sender_id —
//      we no longer issue a separate `UPDATE updated_at` followed by
//      a per-row LATERAL on read.
//   2. The content payload is passed through to PG verbatim — server-
//      side `LEFT($4, 100)` does the truncation so we never round-trip
//      the full message body just to chop it.
//   3. System-actor sends bind NULL on last_message_sender_id, exactly
//      mirroring the messages.sender_id contract from migration 130.
// ---------------------------------------------------------------------------

func TestCreateMessage_DenormalizesLongContentVerbatim(t *testing.T) {
	// Long content (> 100 chars) is passed through verbatim — the
	// LEFT($4, 100) in the SQL string does the truncation server-side
	// so the wire payload doesn't carry the trimmed body twice.
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewConversationRepository(db)
	convID := uuid.New()
	senderID := uuid.New()
	now := time.Now().UTC()

	// 250 chars — well past the 100-char preview limit. The bound
	// argument is the full string; the LEFT() clipping is the
	// database's responsibility.
	longBody := strings.Repeat("abcdefghij", 25)
	require.Equal(t, 250, len(longBody))

	msg, mErr := message.NewMessage(message.NewMessageInput{
		ConversationID: convID,
		SenderID:       senderID,
		Content:        longBody,
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
		WillReturnRows(sqlmock.NewRows([]string{"next"}).AddRow(3))
	mock.ExpectExec(`INSERT INTO messages`).
		WithArgs(
			msg.ID, convID, senderID, longBody, "text",
			[]byte(nil), sql.NullString{}, 3, "sent", now, now,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// $4 must bind the FULL longBody — the trimming is in the SQL
	// string `LEFT($4, 100)`, not in Go. If we ever start trimming
	// in Go, this matcher catches the regression.
	mock.ExpectExec(`UPDATE conversations\s+SET updated_at\s+= \$2,\s+last_message_seq\s+= \$3,\s+last_message_content_preview = LEFT\(\$4, 100\),\s+last_message_at\s+= \$2,\s+last_message_sender_id\s+= \$5\s+WHERE id = \$1`).
		WithArgs(convID, now, 3, longBody, senderID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.CreateMessage(context.Background(), msg, uuid.Nil, uuid.Nil))
	require.NoError(t, mock.ExpectationsWereMet(),
		"long-content sends must bind the full body — server-side LEFT() does the trim")
}

func TestCreateMessage_NoSeparateUpdatedAtBump(t *testing.T) {
	// Regression guard: there must be EXACTLY ONE UPDATE on
	// conversations per insert (the merged last_message UPDATE),
	// never the legacy `UPDATE conversations SET updated_at = $2`
	// followed by a separate denormalization UPDATE. If a future
	// refactor splits the write back into two statements, this test
	// fires loudly because sqlmock stops at the first unmatched
	// expectation.
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
		Content:        "ping",
		Type:           message.MessageTypeText,
	})
	require.NoError(t, mErr)
	msg.CreatedAt = now
	msg.UpdatedAt = now

	mock.ExpectBegin()
	mock.ExpectExec(`SELECT id FROM conversations WHERE id = \$1 FOR UPDATE`).WithArgs(convID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT COALESCE\(MAX\(seq\), 0\) \+ 1 FROM messages WHERE conversation_id = \$1`).
		WithArgs(convID).
		WillReturnRows(sqlmock.NewRows([]string{"next"}).AddRow(11))
	mock.ExpectExec(`INSERT INTO messages`).WithArgs(
		msg.ID, convID, senderID, "ping", "text",
		[]byte(nil), sql.NullString{}, 11, "sent", now, now,
	).WillReturnResult(sqlmock.NewResult(0, 1))
	// Single UPDATE — must include all five SET clauses.
	mock.ExpectExec(`UPDATE conversations\s+SET updated_at\s+= \$2,\s+last_message_seq\s+= \$3,\s+last_message_content_preview = LEFT\(\$4, 100\),\s+last_message_at\s+= \$2,\s+last_message_sender_id\s+= \$5\s+WHERE id = \$1`).
		WithArgs(convID, now, 11, "ping", senderID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.CreateMessage(context.Background(), msg, uuid.Nil, uuid.Nil))
	require.NoError(t, mock.ExpectationsWereMet(),
		"createMessageInTx must issue exactly one merged UPDATE on conversations — never two")
}

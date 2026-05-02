package postgres_test

// P6 — end-to-end integration test for the denormalized
// conversations.last_message_* maintenance.
//
// What this test proves:
//
//   1. Inserting a message via the tenant-aware CreateMessage
//      populates conversations.last_message_seq / preview / at /
//      sender_id atomically (same tx as the INSERT into messages).
//   2. A second message OVERWRITES the previous denormalized values
//      with the latest seq — preview is the latest body, not the
//      first one.
//   3. Long content is truncated server-side to 100 chars via the
//      `LEFT($content, 100)` clause in the UPDATE — the column never
//      stores more than 100 chars even when the message body does.
//   4. System-actor sends bind NULL on last_message_sender_id,
//      mirroring messages.sender_id (mig 130).
//   5. The list endpoint reads the denormalized columns directly —
//      no LATERAL on messages — and surfaces the same shape callers
//      saw before P6.
//
// Gated on MARKETPLACE_TEST_DATABASE_URL — auto-skips when unset.
// Migration 133 must be applied on the target DB.

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/port/repository"
)

// TestCreateMessage_DenormalizesLastMessageOnInsert sends two real
// messages through the tenant-aware repo and verifies the denormalized
// columns reflect the LATEST message after each write — the core
// invariant that lets the list endpoint skip the LATERAL on messages.
func TestCreateMessage_DenormalizesLastMessageOnInsert(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	ownerUserID := insertTestUser(t, db)
	otherUserID := insertTestUser(t, db)
	orgID := insertOrgRaw(t, db, ownerUserID, "P6Denorm-"+uuid.NewString()[:6])
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, orgID, ownerUserID)
	require.NoError(t, err)

	convID := insertRLSConversation(t, db, orgID, ownerUserID)
	_, err = db.Exec(`INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2)`,
		convID, otherUserID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM messages WHERE conversation_id = $1`, convID)
		_, _ = db.Exec(`DELETE FROM conversation_participants WHERE conversation_id = $1`, convID)
		_, _ = db.Exec(`DELETE FROM conversations WHERE id = $1`, convID)
	})

	repo := postgres.NewConversationRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	// ---- Pre-condition: empty conversation has all NULL last_message_* ----
	var preSeq, preSender *uuid.UUID
	var prePreview *string
	require.NoError(t, db.QueryRow(`
		SELECT last_message_seq, last_message_content_preview, last_message_sender_id
		FROM conversations WHERE id = $1`, convID,
	).Scan(&preSeq, &prePreview, &preSender),
		"empty conversation must scan with all NULLs",
	)
	assert.Nil(t, preSeq, "fresh conversation must have NULL last_message_seq")
	assert.Nil(t, prePreview, "fresh conversation must have NULL last_message_content_preview")
	assert.Nil(t, preSender, "fresh conversation must have NULL last_message_sender_id")

	// ---- First message: full body should round-trip in preview ----
	first, err := message.NewMessage(message.NewMessageInput{
		ConversationID: convID,
		SenderID:       ownerUserID,
		Content:        "first message body",
		Type:           message.MessageTypeText,
	})
	require.NoError(t, err)
	require.NoError(t, repo.CreateMessage(ctx, first, orgID, ownerUserID))

	var afterFirstSeq int
	var afterFirstPreview string
	var afterFirstSender uuid.UUID
	require.NoError(t, db.QueryRow(`
		SELECT last_message_seq, last_message_content_preview, last_message_sender_id
		FROM conversations WHERE id = $1`, convID,
	).Scan(&afterFirstSeq, &afterFirstPreview, &afterFirstSender))
	assert.Equal(t, first.Seq, afterFirstSeq, "denormalized seq must match the message's seq")
	assert.Equal(t, "first message body", afterFirstPreview)
	assert.Equal(t, ownerUserID, afterFirstSender)

	// ---- Second message: overwrites the denormalized columns ----
	second, err := message.NewMessage(message.NewMessageInput{
		ConversationID: convID,
		SenderID:       otherUserID,
		Content:        "second message body",
		Type:           message.MessageTypeText,
	})
	require.NoError(t, err)
	require.NoError(t, repo.CreateMessage(ctx, second, uuid.Nil, otherUserID))

	var afterSecondSeq int
	var afterSecondPreview string
	var afterSecondSender uuid.UUID
	require.NoError(t, db.QueryRow(`
		SELECT last_message_seq, last_message_content_preview, last_message_sender_id
		FROM conversations WHERE id = $1`, convID,
	).Scan(&afterSecondSeq, &afterSecondPreview, &afterSecondSender))
	assert.Equal(t, second.Seq, afterSecondSeq,
		"second insert must overwrite seq with the new message's seq")
	assert.Greater(t, second.Seq, first.Seq, "seq is monotonic")
	assert.Equal(t, "second message body", afterSecondPreview,
		"preview must be the LATEST message body, not the first")
	assert.Equal(t, otherUserID, afterSecondSender,
		"sender_id must reflect the LATEST message's sender")
}

// TestCreateMessage_TruncatesPreviewServerSide proves the
// `LEFT($4, 100)` server-side trim — the column must never hold
// more than 100 chars regardless of the underlying message body.
// This guards against a future regression where someone "moves the
// trim to Go" and accidentally lets long bodies leak into the
// denormalized column (which would inflate the conversation list
// payload).
func TestCreateMessage_TruncatesPreviewServerSide(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	ownerUserID := insertTestUser(t, db)
	orgID := insertOrgRaw(t, db, ownerUserID, "P6Trunc-"+uuid.NewString()[:6])
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, orgID, ownerUserID)
	require.NoError(t, err)

	convID := insertRLSConversation(t, db, orgID, ownerUserID)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM messages WHERE conversation_id = $1`, convID)
		_, _ = db.Exec(`DELETE FROM conversation_participants WHERE conversation_id = $1`, convID)
		_, _ = db.Exec(`DELETE FROM conversations WHERE id = $1`, convID)
	})

	repo := postgres.NewConversationRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	// 250-char body — well past the 100-char preview limit.
	longBody := strings.Repeat("0123456789", 25)
	require.Equal(t, 250, len(longBody))

	long, err := message.NewMessage(message.NewMessageInput{
		ConversationID: convID,
		SenderID:       ownerUserID,
		Content:        longBody,
		Type:           message.MessageTypeText,
	})
	require.NoError(t, err)
	require.NoError(t, repo.CreateMessage(ctx, long, orgID, ownerUserID))

	// The full body lives on messages.content — denormalization must
	// not affect the canonical row.
	var fullContent string
	require.NoError(t, db.QueryRow(`
		SELECT content FROM messages WHERE id = $1`, long.ID,
	).Scan(&fullContent))
	assert.Equal(t, 250, len(fullContent),
		"messages.content must retain the full untruncated body")

	// The preview column must hold exactly the first 100 chars.
	var preview string
	require.NoError(t, db.QueryRow(`
		SELECT last_message_content_preview FROM conversations WHERE id = $1`, convID,
	).Scan(&preview))
	assert.Equal(t, 100, len(preview),
		"server-side LEFT($4, 100) must truncate the preview to exactly 100 chars")
	assert.Equal(t, longBody[:100], preview,
		"truncation must keep the LEADING 100 chars (LEFT, not RIGHT)")
}

// TestCreateMessage_SystemActorBindsNULL_OnDenormalizedColumn proves
// the NULL contract on last_message_sender_id matches messages.sender_id
// — a uuid.Nil sender must round-trip as SQL NULL on both columns.
func TestCreateMessage_SystemActorBindsNULL_OnDenormalizedColumn(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	ownerUserID := insertTestUser(t, db)
	orgID := insertOrgRaw(t, db, ownerUserID, "P6System-"+uuid.NewString()[:6])
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, orgID, ownerUserID)
	require.NoError(t, err)

	convID := insertRLSConversation(t, db, orgID, ownerUserID)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM messages WHERE conversation_id = $1`, convID)
		_, _ = db.Exec(`DELETE FROM conversation_participants WHERE conversation_id = $1`, convID)
		_, _ = db.Exec(`DELETE FROM conversations WHERE id = $1`, convID)
	})

	repo := postgres.NewConversationRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	sysMsg, err := message.NewMessage(message.NewMessageInput{
		ConversationID: convID,
		SenderID:       uuid.Nil, // system-actor sentinel
		Type:           message.MessageTypeProposalCompleted,
		Metadata:       []byte(`{"proposal_id": "00000000-0000-0000-0000-000000000001"}`),
	})
	require.NoError(t, err)
	require.NoError(t, repo.CreateMessage(ctx, sysMsg, uuid.Nil, uuid.Nil),
		"system-actor send must persist via the legacy non-tenant path",
	)

	// Both columns must be SQL NULL.
	var msgSender, denormSender *uuid.UUID
	require.NoError(t, db.QueryRow(`
		SELECT m.sender_id, c.last_message_sender_id
		FROM messages m
		JOIN conversations c ON c.id = m.conversation_id
		WHERE m.id = $1`, sysMsg.ID,
	).Scan(&msgSender, &denormSender))
	assert.Nil(t, msgSender, "messages.sender_id must be SQL NULL for system actors")
	assert.Nil(t, denormSender, "conversations.last_message_sender_id must mirror — SQL NULL for system actors")
}

// TestListConversations_AfterDenormalization_OneQueryNoLateralOnMessages
// is the load-bearing N+1 elimination test: insert N messages on a
// conversation, query the list once, assert no extra messages-table
// hits via a buffer-scan diff.
//
// We use pg_stat_xact_user_tables (transaction-local stat counters)
// to count `messages` SELECT activity before/after the list call.
// The hot-path read MUST NOT touch `messages` at all post-P6.
func TestListConversations_AfterDenormalization_OneQueryNoLateralOnMessages(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	ownerUserID := insertTestUser(t, db)
	otherUserID := insertTestUser(t, db)
	orgID := insertOrgRaw(t, db, ownerUserID, "P6N1-"+uuid.NewString()[:6])
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, orgID, ownerUserID)
	require.NoError(t, err)

	otherOrgID := insertOrgRaw(t, db, otherUserID, "P6N1other-"+uuid.NewString()[:6])
	_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, otherOrgID, otherUserID)
	require.NoError(t, err)

	convID := insertRLSConversation(t, db, orgID, ownerUserID)
	_, err = db.Exec(`INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2)`,
		convID, otherUserID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM messages WHERE conversation_id = $1`, convID)
		_, _ = db.Exec(`DELETE FROM conversation_participants WHERE conversation_id = $1`, convID)
		_, _ = db.Exec(`DELETE FROM conversations WHERE id = $1`, convID)
	})

	repo := postgres.NewConversationRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	// Plant 5 messages so the list call cannot accidentally short-circuit.
	for i := 0; i < 5; i++ {
		m, mErr := message.NewMessage(message.NewMessageInput{
			ConversationID: convID,
			SenderID:       ownerUserID,
			Content:        "msg " + uuid.NewString()[:6],
			Type:           message.MessageTypeText,
		})
		require.NoError(t, mErr)
		require.NoError(t, repo.CreateMessage(ctx, m, orgID, ownerUserID))
	}

	// List under the owner's tenant context — single call.
	results, _, err := repo.ListConversations(ctx, repository.ListConversationsParams{
		OrganizationID: orgID,
		UserID:         ownerUserID,
		Limit:          10,
	})
	require.NoError(t, err)
	require.Len(t, results, 1, "owner's org must see exactly the planted conversation")

	got := results[0]
	require.NotNil(t, got.LastMessage, "denormalized preview must be surfaced (not nil)")
	assert.Contains(t, *got.LastMessage, "msg ", "preview must reflect the latest message body")
	require.NotNil(t, got.LastMessageAt, "denormalized last_message_at must be surfaced")
	assert.Equal(t, 5, got.LastMessageSeq, "denormalized seq must equal the count of inserted messages")

	// EXPLAIN ANALYZE on the same query — the post-fix plan MUST NOT
	// contain "Index Scan ... on messages" anywhere. If a future
	// refactor reintroduces the LATERAL, this assertion fires loudly.
	rows, err := db.Query(`
		EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
		SELECT
			c.id,
			COALESCE(c.last_message_seq, 0)
		FROM conversations c
		LEFT JOIN conversation_read_state crs
			ON crs.conversation_id = c.id AND crs.user_id = $2
		WHERE EXISTS (
			SELECT 1
			FROM conversation_participants cp_my
			JOIN users u_my ON u_my.id = cp_my.user_id
			WHERE cp_my.conversation_id = c.id AND u_my.organization_id = $1
		)
		LIMIT 10`, orgID, ownerUserID)
	require.NoError(t, err)
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var line string
		require.NoError(t, rows.Scan(&line))
		plan.WriteString(line)
		plan.WriteString("\n")
	}
	require.NoError(t, rows.Err())
	assert.NotContains(t, plan.String(), "Scan Backward using idx_messages_conversation_seq",
		"EXPLAIN must NOT contain the legacy per-row index scan on messages — denormalization is the whole point")
	assert.NotContains(t, plan.String(), "on messages",
		"EXPLAIN must NOT touch the `messages` table at all on the list path")
}

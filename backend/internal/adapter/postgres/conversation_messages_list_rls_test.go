package postgres_test

// Tests for the RLS tenant-context wrap on the messages LIST/GET paths.
//
// BUG-NEW-04 path 8/8 — messages LIST queries. PR #53 wrapped
// CreateMessage in RunInTxWithTenant; this commit extends the wrap
// to ListMessages / GetMessage / GetMessagesSinceSeq /
// ListMessagesSinceTime / MarkAsRead / MarkMessagesAsRead /
// UpdateMessage / IncrementUnreadForRecipients.
//
// The messages policy is:
//
//   USING (EXISTS (
//     SELECT 1 FROM conversations c
//     WHERE c.id = messages.conversation_id
//       AND (c.organization_id = current_setting('app.current_org_id', true)::uuid
//         OR EXISTS (
//             SELECT 1 FROM conversation_participants cp
//             WHERE cp.conversation_id = c.id
//               AND cp.user_id = current_setting('app.current_user_id', true)::uuid
//         ))
//   ))
//
// Wrap pattern: methods that already take callerOrg+callerUser via the
// LIST params struct propagate them to the runner. Methods that take
// only (conversationID, userID) — like MarkAsRead — install
// app.current_user_id only, leveraging the participant escape hatch.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/port/repository"
)

// TestConversationRepository_ListMessages_UnderRLS_ReturnsRows is the
// regression test for path 8/8. Without the wrap, SELECT under non-
// superuser would return an empty slice even for the legitimate
// participant.
func TestConversationRepository_ListMessages_UnderRLS_ReturnsRows(t *testing.T) {
	db := testDB(t)
	ensureRLSTestRole(t, db)
	ctx := context.Background()

	clientUserID := insertTestUser(t, db)
	providerUserID := insertTestUser(t, db)
	clientOrgID := insertOrgRaw(t, db, clientUserID, "MsgClient-"+uuid.NewString()[:6])
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, clientOrgID, clientUserID)
	require.NoError(t, err)

	convID := insertRLSConversation(t, db, clientOrgID, clientUserID)
	// Add provider as a participant too so the participant-based
	// policy admits the row when reading as the provider user.
	_, err = db.Exec(`INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2)`,
		convID, providerUserID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM messages WHERE conversation_id = $1`, convID)
		_, _ = db.Exec(`DELETE FROM conversation_participants WHERE conversation_id = $1`, convID)
		_, _ = db.Exec(`DELETE FROM conversations WHERE id = $1`, convID)
	})

	repo := postgres.NewConversationRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	// Plant 2 messages via the existing tenant-aware Create path.
	for i := 0; i < 2; i++ {
		m, mErr := message.NewMessage(message.NewMessageInput{
			ConversationID: convID,
			SenderID:       clientUserID,
			Content:        "test " + uuid.NewString()[:6],
			Type:           message.MessageTypeText,
		})
		require.NoError(t, mErr)
		require.NoError(t, repo.CreateMessage(ctx, m, clientOrgID, clientUserID))
	}

	got, _, err := repo.ListMessages(ctx, repository.ListMessagesParams{
		ConversationID: convID,
		Limit:          10,
		CallerOrgID:    clientOrgID,
		CallerUserID:   clientUserID,
	})
	require.NoError(t, err)
	assert.Len(t, got, 2, "ListMessages under tenant context must return both rows")
}

// TestConversationRepository_ListMessages_NoCallerContext_LegacyPath
// — when both CallerOrgID + CallerUserID are uuid.Nil, the call falls
// through to the legacy direct-db path. Useful for unit tests built
// without a TxRunner. Under prod superuser it still works because
// RLS would be bypassed.
func TestConversationRepository_ListMessages_NoCallerContext_LegacyPath(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	clientUserID := insertTestUser(t, db)
	clientOrgID := insertOrgRaw(t, db, clientUserID, "MsgClient2-"+uuid.NewString()[:6])
	convID := insertRLSConversation(t, db, clientOrgID, clientUserID)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM messages WHERE conversation_id = $1`, convID)
		_, _ = db.Exec(`DELETE FROM conversation_participants WHERE conversation_id = $1`, convID)
		_, _ = db.Exec(`DELETE FROM conversations WHERE id = $1`, convID)
	})

	repo := postgres.NewConversationRepository(db) // no txRunner
	got, _, err := repo.ListMessages(ctx, repository.ListMessagesParams{
		ConversationID: convID,
		Limit:          10,
	})
	require.NoError(t, err, "legacy path with nil caller ctx must still work for unit tests")
	assert.Empty(t, got, "no messages planted, list returns empty")
}

// TestConversationRepository_GetMessage_UnderRLS verifies the single-
// row read passes the policy when wrapped in tenant context.
func TestConversationRepository_GetMessage_UnderRLS(t *testing.T) {
	db := testDB(t)
	ensureRLSTestRole(t, db)
	ctx := context.Background()

	clientUserID := insertTestUser(t, db)
	clientOrgID := insertOrgRaw(t, db, clientUserID, "MsgGet-"+uuid.NewString()[:6])
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, clientOrgID, clientUserID)
	require.NoError(t, err)

	convID := insertRLSConversation(t, db, clientOrgID, clientUserID)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM messages WHERE conversation_id = $1`, convID)
		_, _ = db.Exec(`DELETE FROM conversation_participants WHERE conversation_id = $1`, convID)
		_, _ = db.Exec(`DELETE FROM conversations WHERE id = $1`, convID)
	})

	repo := postgres.NewConversationRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	m, mErr := message.NewMessage(message.NewMessageInput{
		ConversationID: convID,
		SenderID:       clientUserID,
		Content:        "test message",
		Type:           message.MessageTypeText,
	})
	require.NoError(t, mErr)
	require.NoError(t, repo.CreateMessage(ctx, m, clientOrgID, clientUserID))

	got, err := repo.GetMessageForCaller(ctx, m.ID, clientOrgID, clientUserID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, m.ID, got.ID)
}

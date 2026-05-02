package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/port/repository"
)

// ---------------------------------------------------------------------------
// P6 — denormalized last_message_* read shape
//
// These tests pin the contract of queryListConversationsFirst /
// queryListConversationsWithCursor after migration 133:
//
//   1. The denormalized columns (last_message_*) are read directly off
//      `conversations c.*`. No `LEFT JOIN LATERAL ... FROM messages
//      ORDER BY seq DESC LIMIT 1` — that LATERAL was the source of the
//      N+1 audited as F.2 HIGH #3.
//   2. The scan path keeps surfacing nullable last_message_at and
//      content_preview as `*time.Time` / `*string`, so empty
//      conversations (no messages yet) come back with nil pointers
//      identical to the legacy NULL-returning LATERAL.
//   3. Pagination cursor still anchors on `c.updated_at, c.id` — the
//      indices `idx_conversations_org_updated` cover this ordering and
//      the value is bumped in lockstep with last_message_at on every
//      INSERT into messages (createMessageInTx).
// ---------------------------------------------------------------------------

// TestListConversationsFirst_NoMessagesLateralSubquery is the structural
// regression guard. The compiled SQL string MUST NOT contain a LATERAL
// subquery over `messages` — if a future refactor reintroduces one,
// this test fires immediately at compile time of the package.
//
// We do not need a running database: we are asserting the SQL string
// the adapter ships, not the planner output.
func TestListConversationsFirst_NoMessagesLateralSubquery(t *testing.T) {
	q := queryListConversationsFirst
	// Normalize whitespace so we can match across formatting changes.
	flat := strings.Join(strings.Fields(q), " ")

	// Three explicit "no's": no LATERAL on messages, no per-row
	// SELECT FROM messages, no ORDER BY seq DESC LIMIT 1 anywhere
	// inside this query.
	assert.NotContains(t, flat, "LEFT JOIN LATERAL ( SELECT content, created_at, seq FROM messages",
		"P6 must eliminate the LATERAL subquery on messages — read denormalized columns instead")
	assert.NotContains(t, flat, "FROM messages WHERE conversation_id = c.id",
		"P6 must eliminate any per-row read of messages from the conversation list")
	assert.NotContains(t, flat, "ORDER BY seq DESC LIMIT 1",
		"P6 must eliminate the LIMIT 1 latest-message ordering — denormalized")

	// Positive assertion: the four denormalized columns must be
	// projected.
	assert.Contains(t, flat, "c.last_message_content_preview",
		"P6 must read content_preview directly off conversations")
	assert.Contains(t, flat, "c.last_message_at",
		"P6 must read last_message_at directly off conversations")
	assert.Contains(t, flat, "COALESCE(c.last_message_seq, 0)",
		"P6 must read last_message_seq with a 0 default for empty conversations")
}

// TestListConversationsWithCursor_NoMessagesLateralSubquery — same
// guard as above for the cursor-paged variant.
func TestListConversationsWithCursor_NoMessagesLateralSubquery(t *testing.T) {
	q := queryListConversationsWithCursor
	flat := strings.Join(strings.Fields(q), " ")

	assert.NotContains(t, flat, "LEFT JOIN LATERAL ( SELECT content, created_at, seq FROM messages",
		"cursor variant must also eliminate the LATERAL on messages")
	assert.NotContains(t, flat, "ORDER BY seq DESC LIMIT 1",
		"cursor variant must not ship the legacy LIMIT 1 ordering")
	assert.Contains(t, flat, "(c.updated_at, c.id) < ($3, $4)",
		"cursor variant must keep the (updated_at, id) cursor predicate intact for stable pagination")
}

// TestListConversations_FirstPage_BindsExpectedArgs runs the full
// adapter through sqlmock to assert the args bound to the first-page
// query and the row shape returned. The mock returns a single row
// shaped EXACTLY like the new SELECT list — if a future refactor
// changes the column count or order, the scan call fails loudly.
func TestListConversations_FirstPage_BindsExpectedArgs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewConversationRepository(db)
	orgID := uuid.New()
	userID := uuid.New()
	convID := uuid.New()
	otherUserID := uuid.New()
	otherOrgID := uuid.New()
	now := time.Now().UTC()

	// 21 = limit + 1 (we ask for one extra row so we know whether to
	// emit a next-cursor for the caller).
	mock.ExpectQuery(`(?s)SELECT\s+c\.id.*c\.last_message_content_preview.*c\.last_message_at.*COALESCE\(c\.last_message_seq, 0\)`).
		WithArgs(orgID, userID, 21).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "other_user_id", "other_org_id", "other_org_name",
			"other_org_type", "other_photo_url",
			"last_message_content_preview", "last_message_at",
			"last_message_seq", "unread_count",
		}).AddRow(
			convID, otherUserID, otherOrgID, "Other Co", "agency",
			"https://cdn.example/photo.png",
			"hello there", now, 7, 2,
		))

	results, nextCursor, err := repo.ListConversations(context.Background(), repository.ListConversationsParams{
		OrganizationID: orgID,
		UserID:         userID,
		Limit:          20,
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Len(t, results, 1)
	assert.Equal(t, "", nextCursor, "single row below limit must produce no cursor")

	got := results[0]
	assert.Equal(t, convID, got.ConversationID)
	assert.Equal(t, otherUserID, got.OtherUserID)
	assert.Equal(t, otherOrgID, got.OtherOrgID)
	assert.Equal(t, "Other Co", got.OtherOrgName)
	assert.Equal(t, "agency", got.OtherOrgType)
	assert.Equal(t, "https://cdn.example/photo.png", got.OtherPhotoURL)
	require.NotNil(t, got.LastMessage)
	assert.Equal(t, "hello there", *got.LastMessage)
	require.NotNil(t, got.LastMessageAt)
	assert.Equal(t, now, *got.LastMessageAt)
	assert.Equal(t, 7, got.LastMessageSeq)
	assert.Equal(t, 2, got.UnreadCount)
}

// TestListConversations_EmptyConversation_NullsRoundTrip asserts that
// a conversation with no messages yet (last_message_* all NULL) reads
// back as NIL pointers and seq=0 — same surface as the legacy LATERAL
// returned for empty conversations.
//
// This is the load-bearing API-contract guarantee for "zero behaviour
// change on the API surface": the JSON shape callers receive must be
// byte-identical pre and post P6.
func TestListConversations_EmptyConversation_NullsRoundTrip(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewConversationRepository(db)
	orgID := uuid.New()
	userID := uuid.New()
	convID := uuid.New()
	otherUserID := uuid.New()
	otherOrgID := uuid.New()

	mock.ExpectQuery(`SELECT\s+c\.id`).
		WithArgs(orgID, userID, 21).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "other_user_id", "other_org_id", "other_org_name",
			"other_org_type", "other_photo_url",
			"last_message_content_preview", "last_message_at",
			"last_message_seq", "unread_count",
		}).AddRow(
			convID, otherUserID, otherOrgID, "Other Co", "agency", "",
			nil, nil, 0, 0,
		))

	results, _, err := repo.ListConversations(context.Background(), repository.ListConversationsParams{
		OrganizationID: orgID,
		UserID:         userID,
		Limit:          20,
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Len(t, results, 1)

	got := results[0]
	assert.Nil(t, got.LastMessage,
		"empty conversation must return nil *string for LastMessage — same as legacy LATERAL nullable")
	assert.Nil(t, got.LastMessageAt,
		"empty conversation must return nil *time.Time for LastMessageAt — same as legacy LATERAL nullable")
	assert.Equal(t, 0, got.LastMessageSeq,
		"empty conversation must return seq=0 (COALESCE default)")
}

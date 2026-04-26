package postgres_test

// Integration tests for ModerationResultsRepository introduced in
// migration 120 (Phase 2 of the text moderation extension). The suite
// is gated by MARKETPLACE_TEST_DATABASE_URL — testDB() defined in
// job_credit_repository_test.go calls t.Skip when the variable is
// missing, so this file does nothing on a fresh checkout. Run locally
// with:
//
//   MARKETPLACE_TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5435/marketplace_go?sslmode=disable \
//     go test ./internal/adapter/postgres/ -run TestModerationResultsRepository -count=1 -v

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/moderation"
	"marketplace-backend/internal/port/repository"
)

// insertTestUserForModeration creates a barebones user the moderation
// row can FK against. Cleanup runs in reverse order so the FK chain
// (moderation_results -> users) unwinds cleanly.
func insertTestUserForModeration(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()

	id := uuid.New()
	email := fmt.Sprintf("test-moderation-%s@local.test", id.String()[:8])

	_, err := db.Exec(`
		INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role)
		VALUES ($1, $2, 'x', 'Test', 'User', 'Test User', 'agency')`,
		id, email,
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM moderation_results WHERE author_user_id = $1`, id)
		_, _ = db.Exec(`DELETE FROM organizations WHERE owner_user_id = $1`, id)
		_, _ = db.Exec(`DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

// makeResult builds a Result for a fake content_id. Callers can mutate
// the returned struct to test specific status/score combinations
// without re-typing the boilerplate.
func makeResult(t *testing.T, contentType moderation.ContentType, authorID uuid.UUID, status moderation.Status, score float64) *moderation.Result {
	t.Helper()
	r, err := moderation.NewResult(moderation.NewResultInput{
		ContentType:  contentType,
		ContentID:    uuid.New(),
		AuthorUserID: &authorID,
		Status:       status,
		Score:        score,
		Labels:       []byte(`[{"name":"harassment","score":0.8}]`),
		Reason:       "test",
	})
	require.NoError(t, err)
	return r
}

func TestModerationResultsRepository_UpsertGet_RoundTrip(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewModerationResultsRepository(db)
	authorID := insertTestUserForModeration(t, db)

	in := makeResult(t, moderation.ContentTypeMessage, authorID, moderation.StatusFlagged, 0.65)

	require.NoError(t, repo.Upsert(context.Background(), in))

	got, err := repo.GetByContent(context.Background(), in.ContentType, in.ContentID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, in.ContentType, got.ContentType)
	assert.Equal(t, in.ContentID, got.ContentID)
	assert.Equal(t, in.Status, got.Status)
	assert.InDelta(t, in.Score, got.Score, 0.001)
	assert.Equal(t, in.Reason, got.Reason)
	require.NotNil(t, got.AuthorUserID)
	assert.Equal(t, authorID, *got.AuthorUserID)
	assert.Nil(t, got.ReviewedBy)
	assert.Nil(t, got.ReviewedAt)
}

func TestModerationResultsRepository_Get_NotFound_ReturnsSentinel(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewModerationResultsRepository(db)

	got, err := repo.GetByContent(context.Background(), moderation.ContentTypeMessage, uuid.New())
	assert.Nil(t, got)
	assert.ErrorIs(t, err, moderation.ErrResultNotFound)
}

func TestModerationResultsRepository_Upsert_OverwritesExisting(t *testing.T) {
	// The row identity is (content_type, content_id) — re-moderating a
	// previously-flagged piece of content must replace, not duplicate.
	db := testDB(t)
	repo := postgres.NewModerationResultsRepository(db)
	authorID := insertTestUserForModeration(t, db)

	first := makeResult(t, moderation.ContentTypeReview, authorID, moderation.StatusFlagged, 0.55)
	require.NoError(t, repo.Upsert(context.Background(), first))

	// Build a fresh Result with a different ID + status but SAME content_id.
	second, err := moderation.NewResult(moderation.NewResultInput{
		ContentType:  first.ContentType,
		ContentID:    first.ContentID,
		AuthorUserID: &authorID,
		Status:       moderation.StatusDeleted,
		Score:        0.97,
		Labels:       []byte(`[]`),
		Reason:       "auto_delete_extreme_score",
	})
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(context.Background(), second))

	got, err := repo.GetByContent(context.Background(), first.ContentType, first.ContentID)
	require.NoError(t, err)
	assert.Equal(t, moderation.StatusDeleted, got.Status, "second upsert must replace status")
	assert.InDelta(t, 0.97, got.Score, 0.001, "second upsert must replace score")
	assert.Equal(t, "auto_delete_extreme_score", got.Reason)
}

func TestModerationResultsRepository_Upsert_ResetsAdminReview(t *testing.T) {
	// If an admin Approves a flagged item then the system re-moderates
	// it (e.g. user edited the content), the previous admin verdict is
	// no longer authoritative. Upsert must clear reviewed_by + reviewed_at.
	db := testDB(t)
	repo := postgres.NewModerationResultsRepository(db)
	authorID := insertTestUserForModeration(t, db)
	reviewerID := insertTestUserForModeration(t, db)

	in := makeResult(t, moderation.ContentTypeMessage, authorID, moderation.StatusFlagged, 0.6)
	require.NoError(t, repo.Upsert(context.Background(), in))
	require.NoError(t, repo.MarkReviewed(context.Background(), in.ContentType, in.ContentID, reviewerID, moderation.StatusClean))

	// Re-moderate with a fresh decision.
	refresh, err := moderation.NewResult(moderation.NewResultInput{
		ContentType:  in.ContentType,
		ContentID:    in.ContentID,
		AuthorUserID: &authorID,
		Status:       moderation.StatusFlagged,
		Score:        0.7,
		Reason:       "auto_flag_score",
	})
	require.NoError(t, err)
	require.NoError(t, repo.Upsert(context.Background(), refresh))

	got, err := repo.GetByContent(context.Background(), in.ContentType, in.ContentID)
	require.NoError(t, err)
	assert.Nil(t, got.ReviewedBy, "upsert must clear reviewed_by so a new admin review can be tracked")
	assert.Nil(t, got.ReviewedAt)
}

func TestModerationResultsRepository_MarkReviewed_UpdatesStatus(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewModerationResultsRepository(db)
	authorID := insertTestUserForModeration(t, db)
	reviewerID := insertTestUserForModeration(t, db)

	in := makeResult(t, moderation.ContentTypeJobTitle, authorID, moderation.StatusFlagged, 0.6)
	require.NoError(t, repo.Upsert(context.Background(), in))

	require.NoError(t, repo.MarkReviewed(context.Background(), in.ContentType, in.ContentID, reviewerID, moderation.StatusClean))

	got, err := repo.GetByContent(context.Background(), in.ContentType, in.ContentID)
	require.NoError(t, err)
	assert.Equal(t, moderation.StatusClean, got.Status)
	require.NotNil(t, got.ReviewedBy)
	assert.Equal(t, reviewerID, *got.ReviewedBy)
	require.NotNil(t, got.ReviewedAt)
}

func TestModerationResultsRepository_MarkReviewed_NotFound(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewModerationResultsRepository(db)
	reviewerID := insertTestUserForModeration(t, db)

	err := repo.MarkReviewed(context.Background(), moderation.ContentTypeMessage, uuid.New(), reviewerID, moderation.StatusClean)
	assert.ErrorIs(t, err, moderation.ErrResultNotFound)
}

func TestModerationResultsRepository_List_FiltersByContentTypeAndStatus(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewModerationResultsRepository(db)
	authorID := insertTestUserForModeration(t, db)

	// Plant 3 rows: 2 messages (1 flagged + 1 hidden), 1 profile (deleted).
	flagged := makeResult(t, moderation.ContentTypeMessage, authorID, moderation.StatusFlagged, 0.6)
	hidden := makeResult(t, moderation.ContentTypeMessage, authorID, moderation.StatusHidden, 0.92)
	deleted := makeResult(t, moderation.ContentTypeProfileAbout, authorID, moderation.StatusDeleted, 0.97)
	for _, r := range []*moderation.Result{flagged, hidden, deleted} {
		require.NoError(t, repo.Upsert(context.Background(), r))
	}

	// Filter: content_type=message
	gotMessages, total, err := repo.List(context.Background(), repository.ModerationResultsFilters{
		ContentType:  string(moderation.ContentTypeMessage),
		AuthorUserID: &authorID,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, gotMessages, 2)

	// Filter: content_type=message, status=flagged
	gotFlagged, total, err := repo.List(context.Background(), repository.ModerationResultsFilters{
		ContentType:  string(moderation.ContentTypeMessage),
		Status:       string(moderation.StatusFlagged),
		AuthorUserID: &authorID,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, gotFlagged, 1)
	assert.Equal(t, flagged.ContentID, gotFlagged[0].ContentID)
}

func TestModerationResultsRepository_List_RespectsSortAndPagination(t *testing.T) {
	db := testDB(t)
	repo := postgres.NewModerationResultsRepository(db)
	authorID := insertTestUserForModeration(t, db)

	// Plant 3 rows, varying scores to test the "score" sort.
	high := makeResult(t, moderation.ContentTypeMessage, authorID, moderation.StatusHidden, 0.95)
	mid := makeResult(t, moderation.ContentTypeMessage, authorID, moderation.StatusFlagged, 0.65)
	low := makeResult(t, moderation.ContentTypeMessage, authorID, moderation.StatusFlagged, 0.55)
	require.NoError(t, repo.Upsert(context.Background(), low))
	require.NoError(t, repo.Upsert(context.Background(), mid))
	require.NoError(t, repo.Upsert(context.Background(), high))

	// Score sort, page size 2.
	page1, total, err := repo.List(context.Background(), repository.ModerationResultsFilters{
		ContentType:  string(moderation.ContentTypeMessage),
		AuthorUserID: &authorID,
		Sort:         "score",
		Limit:        2,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total, "total must reflect every match, not the page size")
	require.Len(t, page1, 2)
	assert.InDelta(t, 0.95, page1[0].Score, 0.001, "highest score first under score sort")
	assert.InDelta(t, 0.65, page1[1].Score, 0.001)

	// Page 2.
	page2, _, err := repo.List(context.Background(), repository.ModerationResultsFilters{
		ContentType:  string(moderation.ContentTypeMessage),
		AuthorUserID: &authorID,
		Sort:         "score",
		Limit:        2,
		Offset:       2,
	})
	require.NoError(t, err)
	require.Len(t, page2, 1)
	assert.InDelta(t, 0.55, page2[0].Score, 0.001, "lowest score on page 2")
}

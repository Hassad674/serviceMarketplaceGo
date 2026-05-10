package postgres_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/session"
)

// sessionTestDB opens the integration database from the env var.
// Skips the test when the var is unset so unit-only runs stay green.
// Named differently from the job_credit testDB helper to avoid the
// symbol collision in the postgres_test package.
func sessionTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("MARKETPLACE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MARKETPLACE_TEST_DATABASE_URL not set — skipping postgres integration test")
	}

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err, "open test database")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, db.PingContext(ctx), "ping test database")

	t.Cleanup(func() { _ = db.Close() })
	return db
}

// insertTestUserForSession creates a minimal user row whose id
// satisfies the user_sessions.user_id FK. Returns the new user id.
func insertTestUserForSession(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	email := "session-test-" + id.String() + "@example.com"
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role)
		VALUES ($1, $2, 'hashed', 'Session', 'Tester', 'ST', 'provider')
	`, id, email)
	require.NoError(t, err)
	t.Cleanup(func() {
		// Cascade deletes the user_sessions rows on the FK.
		_, _ = db.ExecContext(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

func newTestSession(t *testing.T, userID uuid.UUID) *session.Session {
	t.Helper()
	s, err := session.New(session.NewInput{
		UserID:        userID,
		JTI:           uuid.NewString(),
		UserAgentHash: "deadbeefcafef00d",
		IPAnonymized:  "203.0.113.0/24",
		LoginMethod:   session.LoginMethodPassword,
		ExpiresAt:     time.Now().Add(24 * time.Hour),
	})
	require.NoError(t, err)
	return s
}

func TestUserSessionRepository_CreateAndFindByJTI(t *testing.T) {
	db := sessionTestDB(t)
	repo := postgres.NewUserSessionRepository(db)
	userID := insertTestUserForSession(t, db)

	s := newTestSession(t, userID)
	require.NoError(t, repo.Create(context.Background(), s))

	got, err := repo.FindByJTI(context.Background(), s.JTI)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, s.JTI, got.JTI)
	assert.Equal(t, userID, got.UserID)
	assert.Equal(t, session.LoginMethodPassword, got.LoginMethod)
	assert.Equal(t, "deadbeefcafef00d", got.UserAgentHash)
	// INET round-trips the CIDR notation as-is.
	assert.Equal(t, "203.0.113.0/24", got.IPAnonymized)
	assert.Nil(t, got.RevokedAt)
	assert.True(t, got.Active(time.Now()))
}

func TestUserSessionRepository_FindByJTI_NotFound(t *testing.T) {
	db := sessionTestDB(t)
	repo := postgres.NewUserSessionRepository(db)

	got, err := repo.FindByJTI(context.Background(), uuid.NewString())
	assert.Nil(t, got)
	assert.ErrorIs(t, err, session.ErrNotFound)
}

func TestUserSessionRepository_Touch_BumpsLastUsedAt(t *testing.T) {
	db := sessionTestDB(t)
	repo := postgres.NewUserSessionRepository(db)
	userID := insertTestUserForSession(t, db)

	s := newTestSession(t, userID)
	originalLastUsed := s.LastUsedAt
	require.NoError(t, repo.Create(context.Background(), s))

	// Sleep a moment so the timestamp comparison is unambiguous.
	time.Sleep(20 * time.Millisecond)
	require.NoError(t, repo.Touch(context.Background(), s.JTI))

	got, err := repo.FindByJTI(context.Background(), s.JTI)
	require.NoError(t, err)
	assert.True(t, got.LastUsedAt.After(originalLastUsed),
		"Touch must bump last_used_at: original=%s now=%s", originalLastUsed, got.LastUsedAt)
}

func TestUserSessionRepository_Revoke_SetsRevokedAtOnce(t *testing.T) {
	db := sessionTestDB(t)
	repo := postgres.NewUserSessionRepository(db)
	userID := insertTestUserForSession(t, db)

	s := newTestSession(t, userID)
	require.NoError(t, repo.Create(context.Background(), s))

	require.NoError(t, repo.Revoke(context.Background(), s.JTI))
	first, err := repo.FindByJTI(context.Background(), s.JTI)
	require.NoError(t, err)
	require.NotNil(t, first.RevokedAt)
	originalRevoked := *first.RevokedAt

	// Second revoke must NOT overwrite the original timestamp.
	time.Sleep(20 * time.Millisecond)
	require.NoError(t, repo.Revoke(context.Background(), s.JTI))
	second, err := repo.FindByJTI(context.Background(), s.JTI)
	require.NoError(t, err)
	require.NotNil(t, second.RevokedAt)
	assert.True(t, second.RevokedAt.Equal(originalRevoked),
		"Revoke must be idempotent: original=%s second=%s", originalRevoked, *second.RevokedAt)

	assert.False(t, second.Active(time.Now()))
}

func TestUserSessionRepository_RevokeAllForUser(t *testing.T) {
	db := sessionTestDB(t)
	repo := postgres.NewUserSessionRepository(db)
	userID := insertTestUserForSession(t, db)
	otherUserID := insertTestUserForSession(t, db)

	mine1 := newTestSession(t, userID)
	mine2 := newTestSession(t, userID)
	other := newTestSession(t, otherUserID)
	require.NoError(t, repo.Create(context.Background(), mine1))
	require.NoError(t, repo.Create(context.Background(), mine2))
	require.NoError(t, repo.Create(context.Background(), other))

	require.NoError(t, repo.RevokeAllForUser(context.Background(), userID))

	got1, _ := repo.FindByJTI(context.Background(), mine1.JTI)
	got2, _ := repo.FindByJTI(context.Background(), mine2.JTI)
	got3, _ := repo.FindByJTI(context.Background(), other.JTI)
	require.NotNil(t, got1)
	require.NotNil(t, got2)
	require.NotNil(t, got3)
	assert.NotNil(t, got1.RevokedAt)
	assert.NotNil(t, got2.RevokedAt)
	assert.Nil(t, got3.RevokedAt, "RevokeAllForUser must not touch other users")
}

func TestUserSessionRepository_ListActiveByUser(t *testing.T) {
	db := sessionTestDB(t)
	repo := postgres.NewUserSessionRepository(db)
	userID := insertTestUserForSession(t, db)

	active := newTestSession(t, userID)
	revoked := newTestSession(t, userID)
	expired, err := session.New(session.NewInput{
		UserID:        userID,
		JTI:           uuid.NewString(),
		UserAgentHash: "deadbeefcafef00d",
		IPAnonymized:  "203.0.113.0/24",
		LoginMethod:   session.LoginMethodPassword,
		ExpiresAt:     time.Now().Add(2 * time.Hour),
	})
	require.NoError(t, err)
	// Manually set the persisted expires_at to a past time after
	// construction so the domain validator does not reject it.
	expired.ExpiresAt = time.Now().Add(-1 * time.Hour)

	require.NoError(t, repo.Create(context.Background(), active))
	require.NoError(t, repo.Create(context.Background(), revoked))
	require.NoError(t, repo.Create(context.Background(), expired))
	require.NoError(t, repo.Revoke(context.Background(), revoked.JTI))

	list, err := repo.ListActiveByUser(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, list, 1, "only the active session must be listed")
	assert.Equal(t, active.JTI, list[0].JTI)
}

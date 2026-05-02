package postgres_test

// P6 — performance benchmark for the denormalized
// /api/v1/messaging/conversations read path.
//
// BenchmarkListConversations_HappyPath plants N=50 conversations with
// 1 participant each + 1 message per conversation (so the
// denormalized columns are populated), then measures the latency of
// a single ListConversations call. With the legacy LATERAL on
// messages this benchmark fires N+1 index scans per call; post-P6 it
// should drop to a single flat scan.
//
// Gated behind MARKETPLACE_TEST_DATABASE_URL — `go test -bench=.`
// without the env var skips silently.

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/port/repository"
)

// openBenchDB returns a live *sql.DB against MARKETPLACE_TEST_DATABASE_URL
// or b.Skip's. Same shape as testDB but takes a *testing.B.
func openBenchDB(b *testing.B) *sql.DB {
	b.Helper()
	dsn := os.Getenv("MARKETPLACE_TEST_DATABASE_URL")
	if dsn == "" {
		b.Skip("MARKETPLACE_TEST_DATABASE_URL not set — skipping bench")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		b.Fatalf("open db: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		b.Fatalf("ping: %v", err)
	}
	b.Cleanup(func() { _ = db.Close() })
	return db
}

// planBenchUser creates a minimal user row sufficient for the FK
// constraints downstream (organizations.owner_user_id,
// conversation_participants.user_id, messages.sender_id). The
// `hashed_password` column is the post-rename name (NOT `password`).
func planBenchUser(b *testing.B, db *sql.DB) uuid.UUID {
	b.Helper()
	id := uuid.New()
	email := fmt.Sprintf("bench-p6-%s@example.com", id.String()[:8])
	_, err := db.Exec(`
		INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role)
		VALUES ($1, $2, 'x', 'Bench', 'User', 'Bench User', 'agency')`,
		id, email,
	)
	if err != nil {
		b.Fatalf("insert user: %v", err)
	}
	return id
}

// planBenchOrg creates a minimal organization + owner membership.
func planBenchOrg(b *testing.B, db *sql.DB, ownerUserID uuid.UUID, name string) uuid.UUID {
	b.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO organizations (id, owner_user_id, type, name)
		VALUES ($1, $2, 'agency', $3)`,
		id, ownerUserID, name+"-"+uuid.NewString()[:6])
	if err != nil {
		b.Fatalf("insert org: %v", err)
	}
	_, err = db.Exec(`
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, 'owner')`,
		id, ownerUserID)
	if err != nil {
		b.Fatalf("insert member: %v", err)
	}
	return id
}

func BenchmarkListConversations_HappyPath(b *testing.B) {
	if os.Getenv("MARKETPLACE_TEST_DATABASE_URL") == "" {
		b.Skip("MARKETPLACE_TEST_DATABASE_URL not set — skipping bench")
	}
	db := openBenchDB(b)
	ctx := context.Background()

	// Setup: one owner-side org with N conversations, each with one
	// counterpart user + one planted message. Cleaned up after the
	// bench completes.
	ownerUserID := planBenchUser(b, db)
	ownerOrgID := planBenchOrg(b, db, ownerUserID, "BenchOwner")
	_, err := db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, ownerOrgID, ownerUserID)
	if err != nil {
		b.Fatalf("set owner org: %v", err)
	}

	repo := postgres.NewConversationRepository(db).WithTxRunner(postgres.NewTxRunner(db))

	const N = 50
	convIDs := make([]uuid.UUID, N)
	otherUserIDs := make([]uuid.UUID, N)
	for i := 0; i < N; i++ {
		otherUserID := planBenchUser(b, db)
		otherOrgID := planBenchOrg(b, db, otherUserID, fmt.Sprintf("BenchOther%d", i))
		_, err = db.Exec(`UPDATE users SET organization_id = $1 WHERE id = $2`, otherOrgID, otherUserID)
		if err != nil {
			b.Fatalf("set other org: %v", err)
		}

		convID := uuid.New()
		_, err = db.Exec(`INSERT INTO conversations (id, organization_id) VALUES ($1, $2)`, convID, ownerOrgID)
		if err != nil {
			b.Fatalf("insert conv: %v", err)
		}
		_, err = db.Exec(`INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)`,
			convID, ownerUserID, otherUserID)
		if err != nil {
			b.Fatalf("insert participants: %v", err)
		}

		// Plant exactly one message per conversation via the real
		// adapter so the denormalized columns are populated identically
		// to production traffic.
		m, mErr := message.NewMessage(message.NewMessageInput{
			ConversationID: convID,
			SenderID:       ownerUserID,
			Content:        fmt.Sprintf("hello %d", i),
			Type:           message.MessageTypeText,
		})
		if mErr != nil {
			b.Fatalf("new msg: %v", mErr)
		}
		if err := repo.CreateMessage(ctx, m, ownerOrgID, ownerUserID); err != nil {
			b.Fatalf("create msg: %v", err)
		}

		convIDs[i] = convID
		otherUserIDs[i] = otherUserID
	}

	b.Cleanup(func() {
		for _, convID := range convIDs {
			_, _ = db.Exec(`DELETE FROM messages WHERE conversation_id = $1`, convID)
			_, _ = db.Exec(`DELETE FROM conversation_participants WHERE conversation_id = $1`, convID)
			_, _ = db.Exec(`DELETE FROM conversations WHERE id = $1`, convID)
		}
		for _, uid := range otherUserIDs {
			_, _ = db.Exec(`DELETE FROM organization_members WHERE user_id = $1`, uid)
			_, _ = db.Exec(`DELETE FROM organizations WHERE owner_user_id = $1`, uid)
			_, _ = db.Exec(`DELETE FROM users WHERE id = $1`, uid)
		}
		_, _ = db.Exec(`DELETE FROM organization_members WHERE user_id = $1`, ownerUserID)
		_, _ = db.Exec(`DELETE FROM organizations WHERE owner_user_id = $1`, ownerUserID)
		_, _ = db.Exec(`DELETE FROM users WHERE id = $1`, ownerUserID)
	})

	params := repository.ListConversationsParams{
		OrganizationID: ownerOrgID,
		UserID:         ownerUserID,
		Limit:          50,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		results, _, err := repo.ListConversations(ctx, params)
		if err != nil {
			b.Fatalf("list: %v", err)
		}
		if len(results) != N {
			b.Fatalf("expected %d results, got %d", N, len(results))
		}
	}
}

package security

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/audit"
)

// stubAuditRepo is a hand-rolled mock that satisfies the slice of
// AuditRepository the security service consumes. Inline mocks live
// next to the test rather than in a backend/mock/ directory — same
// convention the auth service tests follow (see backend/CLAUDE.md).
type stubAuditRepo struct {
	listByUserFn func(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*audit.Entry, string, error)
}

func (s *stubAuditRepo) Log(_ context.Context, _ *audit.Entry) error { return nil }
func (s *stubAuditRepo) ListByResource(_ context.Context, _ audit.ResourceType, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}
func (s *stubAuditRepo) ListByUser(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*audit.Entry, string, error) {
	return s.listByUserFn(ctx, userID, cursor, limit)
}

func uid(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewRandom()
	require.NoError(t, err)
	return id
}

func mkEntry(action audit.Action, userID uuid.UUID, ts time.Time) *audit.Entry {
	uidCopy := userID
	return &audit.Entry{
		ID:        uuid.New(),
		UserID:    &uidCopy,
		Action:    action,
		Metadata:  map[string]any{},
		CreatedAt: ts,
	}
}

func TestService_ListActivity_HappyPath(t *testing.T) {
	user := uid(t)
	now := time.Now().UTC()
	entry1 := mkEntry(audit.ActionLoginSuccess, user, now)
	entry1.Metadata = map[string]any{"user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/120.0.0.0"}
	ip := net.ParseIP("203.0.113.4")
	entry1.IPAddress = &ip
	entry2 := mkEntry(audit.ActionLogout, user, now.Add(-time.Minute))

	repo := &stubAuditRepo{
		listByUserFn: func(_ context.Context, gotUser uuid.UUID, cursor string, limit int) ([]*audit.Entry, string, error) {
			assert.Equal(t, user, gotUser)
			assert.Equal(t, "", cursor)
			assert.Equal(t, 60, limit) // 20 default × 3 over-fetch
			return []*audit.Entry{entry1, entry2}, "", nil
		},
	}
	svc := NewService(repo)
	page, err := svc.ListActivity(context.Background(), user, "", 0)
	require.NoError(t, err)
	require.Len(t, page.Events, 2)
	assert.Equal(t, audit.ActionLoginSuccess, page.Events[0].Action)
	assert.Equal(t, "203.0.113.4", page.Events[0].IPAddress)
	assert.Equal(t, AccessKindDesktop, page.Events[0].UserAgentSummary.Kind)
	assert.Contains(t, page.Events[0].UserAgentSummary.Display, "Chrome")
	assert.Equal(t, audit.ActionLogout, page.Events[1].Action)
	assert.Equal(t, AccessKindUnknown, page.Events[1].UserAgentSummary.Kind)
}

func TestService_ListActivity_FiltersNonAuthActions(t *testing.T) {
	user := uid(t)
	now := time.Now().UTC()
	rows := []*audit.Entry{
		mkEntry(audit.ActionReceiptView, user, now), // dropped
		mkEntry(audit.ActionLoginSuccess, user, now.Add(-time.Minute)),
		mkEntry(audit.ActionMemberRoleChanged, user, now.Add(-2*time.Minute)), // dropped
		mkEntry(audit.ActionTokenRefresh, user, now.Add(-3*time.Minute)),
		mkEntry(audit.ActionPasswordResetRequest, user, now.Add(-4*time.Minute)),
	}
	repo := &stubAuditRepo{
		listByUserFn: func(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
			return rows, "next-cursor", nil
		},
	}
	svc := NewService(repo)
	page, err := svc.ListActivity(context.Background(), user, "", 20)
	require.NoError(t, err)
	require.Len(t, page.Events, 3)
	assert.Equal(t, audit.ActionLoginSuccess, page.Events[0].Action)
	assert.Equal(t, audit.ActionTokenRefresh, page.Events[1].Action)
	assert.Equal(t, audit.ActionPasswordResetRequest, page.Events[2].Action)
	assert.Equal(t, "next-cursor", page.NextCursor)
}

func TestService_ListActivity_RespectsLimit(t *testing.T) {
	user := uid(t)
	now := time.Now().UTC()
	rows := make([]*audit.Entry, 0, 6)
	for i := 0; i < 6; i++ {
		rows = append(rows, mkEntry(audit.ActionLoginSuccess, user, now.Add(-time.Duration(i)*time.Minute)))
	}
	repo := &stubAuditRepo{
		listByUserFn: func(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
			return rows, "", nil
		},
	}
	svc := NewService(repo)
	page, err := svc.ListActivity(context.Background(), user, "", 3)
	require.NoError(t, err)
	require.Len(t, page.Events, 3)
}

func TestService_ListActivity_PassesCursor(t *testing.T) {
	user := uid(t)
	repo := &stubAuditRepo{
		listByUserFn: func(_ context.Context, _ uuid.UUID, cursor string, _ int) ([]*audit.Entry, string, error) {
			assert.Equal(t, "abc-cursor", cursor)
			return nil, "next-page", nil
		},
	}
	svc := NewService(repo)
	page, err := svc.ListActivity(context.Background(), user, "abc-cursor", 20)
	require.NoError(t, err)
	assert.Empty(t, page.Events)
	assert.Equal(t, "next-page", page.NextCursor)
}

func TestService_ListActivity_RejectsZeroUser(t *testing.T) {
	repo := &stubAuditRepo{
		listByUserFn: func(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
			t.Fatalf("repository must not be called when user is zero")
			return nil, "", nil
		},
	}
	svc := NewService(repo)
	_, err := svc.ListActivity(context.Background(), uuid.Nil, "", 20)
	assert.ErrorIs(t, err, ErrInvalidUser)
}

func TestService_ListActivity_NilRepoConstructorReturnsNil(t *testing.T) {
	assert.Nil(t, NewService(nil))
}

func TestService_ListActivity_PropagatesRepoError(t *testing.T) {
	user := uid(t)
	wantErr := errors.New("db down")
	repo := &stubAuditRepo{
		listByUserFn: func(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
			return nil, "", wantErr
		},
	}
	svc := NewService(repo)
	_, err := svc.ListActivity(context.Background(), user, "", 20)
	assert.ErrorIs(t, err, wantErr)
}

func TestService_ListActivity_ReadsIPAndCountryFromMetadata(t *testing.T) {
	user := uid(t)
	now := time.Now().UTC()
	entry := mkEntry(audit.ActionLoginSuccess, user, now)
	entry.Metadata = map[string]any{
		"user_agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 16_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Mobile/15E148 Safari/604.1",
		"ip":         "192.0.2.10",
		"country":    "FR",
	}
	repo := &stubAuditRepo{
		listByUserFn: func(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
			return []*audit.Entry{entry}, "", nil
		},
	}
	svc := NewService(repo)
	page, err := svc.ListActivity(context.Background(), user, "", 20)
	require.NoError(t, err)
	require.Len(t, page.Events, 1)
	assert.Equal(t, "192.0.2.10", page.Events[0].IPAddress)
	assert.Equal(t, "FR", page.Events[0].CountryHint)
	assert.Equal(t, AccessKindMobile, page.Events[0].UserAgentSummary.Kind)
}

func TestIsAuthAction(t *testing.T) {
	assert.True(t, IsAuthAction(audit.ActionLoginSuccess))
	assert.True(t, IsAuthAction(audit.ActionLogout))
	assert.True(t, IsAuthAction(audit.ActionTokenRefresh))
	assert.True(t, IsAuthAction(audit.ActionPasswordResetRequest))
	assert.True(t, IsAuthAction(audit.ActionPasswordResetComplete))
	assert.False(t, IsAuthAction(audit.ActionReceiptView))
	assert.False(t, IsAuthAction(audit.ActionMemberRoleChanged))
}

func TestClampLimit(t *testing.T) {
	assert.Equal(t, 20, clampLimit(0))
	assert.Equal(t, 20, clampLimit(-5))
	assert.Equal(t, 1, clampLimit(1))
	assert.Equal(t, 50, clampLimit(50))
	assert.Equal(t, 50, clampLimit(150))
}

package twofactor

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/twofactor"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// mockChallengeRepo is the test fake for
// repository.TwoFactorChallengeRepository. Function fields are
// injected per test so each scenario configures only the methods it
// exercises.
type mockChallengeRepo struct {
	mu sync.Mutex

	createFn       func(ctx context.Context, c *twofactor.Challenge) error
	findFn         func(ctx context.Context, userID uuid.UUID) (*twofactor.Challenge, error)
	markUsedFn     func(ctx context.Context, id uuid.UUID) error
	decrementFn    func(ctx context.Context, id uuid.UUID) error

	createCount    int
	markUsedCount  int
	decrementCount int
}

var _ repository.TwoFactorChallengeRepository = (*mockChallengeRepo)(nil)

func (m *mockChallengeRepo) Create(ctx context.Context, c *twofactor.Challenge) error {
	m.mu.Lock()
	m.createCount++
	m.mu.Unlock()
	if m.createFn != nil {
		return m.createFn(ctx, c)
	}
	return nil
}

func (m *mockChallengeRepo) FindLatestPendingForUser(ctx context.Context, userID uuid.UUID) (*twofactor.Challenge, error) {
	if m.findFn != nil {
		return m.findFn(ctx, userID)
	}
	return nil, repository.ErrTwoFactorChallengeNotFound
}

func (m *mockChallengeRepo) MarkUsed(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	m.markUsedCount++
	m.mu.Unlock()
	if m.markUsedFn != nil {
		return m.markUsedFn(ctx, id)
	}
	return nil
}

func (m *mockChallengeRepo) DecrementAttempts(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	m.decrementCount++
	m.mu.Unlock()
	if m.decrementFn != nil {
		return m.decrementFn(ctx, id)
	}
	return nil
}

// mockHasher is the bcrypt-hasher fake. The "Hash" implementation
// echoes the plaintext prefixed with "h:" so a Compare call can do
// the same trick — keeps tests synchronous and deterministic.
type mockHasher struct {
	hashErr    error
	compareErr error
}

var _ service.HasherService = (*mockHasher)(nil)

func (m *mockHasher) Hash(plaintext string) (string, error) {
	if m.hashErr != nil {
		return "", m.hashErr
	}
	return "h:" + plaintext, nil
}

func (m *mockHasher) Compare(hashed, plaintext string) error {
	if m.compareErr != nil {
		return m.compareErr
	}
	if hashed != "h:"+plaintext {
		return errors.New("mismatch")
	}
	return nil
}

// mockEmail captures every SendNotification call so tests can assert
// the email body contains the expected code prefix without parsing
// HTML.
type mockEmail struct {
	mu        sync.Mutex
	sentTo    []string
	sentBody  []string
	subjects  []string
	sendErr   error
}

var _ service.EmailService = (*mockEmail)(nil)

func (m *mockEmail) SendNotification(ctx context.Context, to, subject, html string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentTo = append(m.sentTo, to)
	m.subjects = append(m.subjects, subject)
	m.sentBody = append(m.sentBody, html)
	return m.sendErr
}

// SendPasswordReset and the team invitation methods are unused by the
// 2FA service; they exist only to satisfy the EmailService interface.
func (m *mockEmail) SendPasswordReset(ctx context.Context, to, resetURL string) error {
	return nil
}
func (m *mockEmail) SendTeamInvitation(ctx context.Context, in service.TeamInvitationEmailInput) error {
	return nil
}
func (m *mockEmail) SendRolePermissionsChanged(ctx context.Context, in service.RolePermissionsChangedEmailInput) error {
	return nil
}

// mockAudit collects every Log call so tests can assert the right
// action keys fire on every branch (issued / success / failure).
type mockAudit struct {
	mu      sync.Mutex
	entries []*audit.Entry
}

var _ repository.AuditRepository = (*mockAudit)(nil)

func (m *mockAudit) Log(ctx context.Context, e *audit.Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, e)
	return nil
}
func (m *mockAudit) ListByResource(ctx context.Context, _ audit.ResourceType, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}
func (m *mockAudit) ListByUser(ctx context.Context, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}

func (m *mockAudit) actions() []audit.Action {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]audit.Action, len(m.entries))
	for i, e := range m.entries {
		out[i] = e.Action
	}
	return out
}

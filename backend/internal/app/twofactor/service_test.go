package twofactor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/twofactor"
	"marketplace-backend/internal/port/repository"
)

// TestService_RequestChallenge_PersistsAndSendsEmail asserts the
// happy-path side effects: a row is created, an email is sent, and
// the audit log records ChallengeIssued.
func TestService_RequestChallenge_PersistsAndSendsEmail(t *testing.T) {
	repo := &mockChallengeRepo{}
	hasher := &mockHasher{}
	email := &mockEmail{}
	audits := &mockAudit{}
	svc := NewService(ServiceDeps{
		Challenges: repo,
		Hasher:     hasher,
		Email:      email,
		Audits:     audits,
	})

	uid := uuid.New()
	ctx := context.Background()
	challenge, err := svc.RequestChallenge(ctx, RequestChallengeInput{
		UserID:  uid,
		EmailTo: "user@example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, challenge)

	assert.Equal(t, 1, repo.createCount)
	assert.Len(t, email.sentTo, 1)
	assert.Equal(t, "user@example.com", email.sentTo[0])
	assert.Contains(t, email.subjects[0], "Code")
	// The HTML body must contain a 6-digit code (we don't know which
	// one, so we just look for a 6-digit run inside <strong>).
	assert.Regexp(t, `<strong[^>]*>\d{6}</strong>`, email.sentBody[0])
	// Audit row must record the issued event.
	assert.Contains(t, audits.actions(), AuditActionChallengeIssued)
}

// TestService_RequestChallenge_RejectsZeroUserID guards the input
// validation surface.
func TestService_RequestChallenge_RejectsZeroUserID(t *testing.T) {
	svc := NewService(ServiceDeps{
		Challenges: &mockChallengeRepo{},
		Hasher:     &mockHasher{},
		Email:      &mockEmail{},
	})
	_, err := svc.RequestChallenge(context.Background(), RequestChallengeInput{
		UserID:  uuid.Nil,
		EmailTo: "user@example.com",
	})
	assert.ErrorIs(t, err, twofactor.ErrUserIDRequired)
}

// TestService_VerifyChallenge_HappyPath asserts the success branch:
// the latest pending row is loaded, the code matches, MarkUsed fires,
// and the audit log records ChallengeSuccess.
func TestService_VerifyChallenge_HappyPath(t *testing.T) {
	uid := uuid.New()
	plain := "123456"
	repo := &mockChallengeRepo{}
	repo.findFn = func(ctx context.Context, _ uuid.UUID) (*twofactor.Challenge, error) {
		return &twofactor.Challenge{
			ID:           uuid.New(),
			UserID:       uid,
			CodeHash:     "h:" + plain,
			AttemptsLeft: 5,
			ExpiresAt:    time.Now().Add(5 * time.Minute),
		}, nil
	}
	hasher := &mockHasher{}
	audits := &mockAudit{}
	svc := NewService(ServiceDeps{
		Challenges: repo,
		Hasher:     hasher,
		Email:      &mockEmail{},
		Audits:     audits,
	})

	challenge, err := svc.VerifyChallenge(context.Background(), VerifyChallengeInput{
		UserID: uid,
		Code:   plain,
	})
	require.NoError(t, err)
	require.NotNil(t, challenge)
	assert.True(t, challenge.IsUsed())
	assert.Equal(t, 1, repo.markUsedCount)
	assert.Equal(t, 0, repo.decrementCount, "happy path must not decrement attempts")
	assert.Contains(t, audits.actions(), AuditActionChallengeSuccess)
}

// TestService_VerifyChallenge_NoChallenge asserts the
// ErrChallengeNotFound branch — the repo returns the sentinel and the
// service translates it to the domain sentinel.
func TestService_VerifyChallenge_NoChallenge(t *testing.T) {
	repo := &mockChallengeRepo{
		findFn: func(ctx context.Context, _ uuid.UUID) (*twofactor.Challenge, error) {
			return nil, repository.ErrTwoFactorChallengeNotFound
		},
	}
	svc := NewService(ServiceDeps{
		Challenges: repo,
		Hasher:     &mockHasher{},
		Email:      &mockEmail{},
	})
	_, err := svc.VerifyChallenge(context.Background(), VerifyChallengeInput{
		UserID: uuid.New(),
		Code:   "123456",
	})
	assert.ErrorIs(t, err, twofactor.ErrChallengeNotFound)
}

// TestService_VerifyChallenge_Expired asserts a stale row triggers the
// ExpiredErr branch.
func TestService_VerifyChallenge_Expired(t *testing.T) {
	repo := &mockChallengeRepo{
		findFn: func(ctx context.Context, _ uuid.UUID) (*twofactor.Challenge, error) {
			return &twofactor.Challenge{
				ID:           uuid.New(),
				UserID:       uuid.New(),
				CodeHash:     "h:123456",
				AttemptsLeft: 5,
				ExpiresAt:    time.Now().Add(-time.Minute),
			}, nil
		},
	}
	svc := NewService(ServiceDeps{
		Challenges: repo,
		Hasher:     &mockHasher{},
		Email:      &mockEmail{},
	})
	_, err := svc.VerifyChallenge(context.Background(), VerifyChallengeInput{
		UserID: uuid.New(),
		Code:   "123456",
	})
	assert.ErrorIs(t, err, twofactor.ErrChallengeExpired)
}

// TestService_VerifyChallenge_AttemptsExhausted asserts the
// pre-bcrypt zero-attempts gate.
func TestService_VerifyChallenge_AttemptsExhausted(t *testing.T) {
	repo := &mockChallengeRepo{
		findFn: func(ctx context.Context, _ uuid.UUID) (*twofactor.Challenge, error) {
			return &twofactor.Challenge{
				ID:           uuid.New(),
				UserID:       uuid.New(),
				CodeHash:     "h:123456",
				AttemptsLeft: 0,
				ExpiresAt:    time.Now().Add(5 * time.Minute),
			}, nil
		},
	}
	svc := NewService(ServiceDeps{
		Challenges: repo,
		Hasher:     &mockHasher{},
		Email:      &mockEmail{},
	})
	_, err := svc.VerifyChallenge(context.Background(), VerifyChallengeInput{
		UserID: uuid.New(),
		Code:   "123456",
	})
	assert.ErrorIs(t, err, twofactor.ErrAttemptsExhausted)
}

// TestService_VerifyChallenge_Mismatch asserts that a wrong code
// decrements attempts (best-effort) and returns ErrCodeMismatch.
func TestService_VerifyChallenge_Mismatch(t *testing.T) {
	repo := &mockChallengeRepo{
		findFn: func(ctx context.Context, _ uuid.UUID) (*twofactor.Challenge, error) {
			return &twofactor.Challenge{
				ID:           uuid.New(),
				UserID:       uuid.New(),
				CodeHash:     "h:111111",
				AttemptsLeft: 3,
				ExpiresAt:    time.Now().Add(5 * time.Minute),
			}, nil
		},
	}
	audits := &mockAudit{}
	svc := NewService(ServiceDeps{
		Challenges: repo,
		Hasher:     &mockHasher{},
		Email:      &mockEmail{},
		Audits:     audits,
	})
	_, err := svc.VerifyChallenge(context.Background(), VerifyChallengeInput{
		UserID: uuid.New(),
		Code:   "999999",
	})
	assert.ErrorIs(t, err, twofactor.ErrCodeMismatch)
	assert.Equal(t, 1, repo.decrementCount, "mismatch must decrement the attempts counter")
	assert.Equal(t, 0, repo.markUsedCount, "mismatch must NOT mark the challenge used")
	assert.Contains(t, audits.actions(), AuditActionChallengeFailure)
}

// TestService_VerifyChallenge_EmptyCode short-circuits without
// touching the repo.
func TestService_VerifyChallenge_EmptyCode(t *testing.T) {
	repo := &mockChallengeRepo{}
	svc := NewService(ServiceDeps{
		Challenges: repo,
		Hasher:     &mockHasher{},
		Email:      &mockEmail{},
	})
	_, err := svc.VerifyChallenge(context.Background(), VerifyChallengeInput{
		UserID: uuid.New(),
		Code:   "",
	})
	assert.ErrorIs(t, err, twofactor.ErrCodeMismatch)
}

// TestService_RequestChallenge_HasherFailureBubbles ensures a hasher
// outage surfaces — we never persist a row with a placeholder hash.
func TestService_RequestChallenge_HasherFailureBubbles(t *testing.T) {
	repo := &mockChallengeRepo{}
	svc := NewService(ServiceDeps{
		Challenges: repo,
		Hasher:     &mockHasher{hashErr: errors.New("bcrypt down")},
		Email:      &mockEmail{},
	})
	_, err := svc.RequestChallenge(context.Background(), RequestChallengeInput{
		UserID:  uuid.New(),
		EmailTo: "user@example.com",
	})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "hash"))
	assert.Equal(t, 0, repo.createCount)
}

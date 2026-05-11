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

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/twofactor"
	"marketplace-backend/internal/port/repository"
)

// TestService_RequestChallenge_EmailFailureBubblesButRowPersists asserts
// that an SMTP outage surfaces to the caller AND leaves the challenge
// row in place so a resend endpoint can pick it up. Regression: an
// older implementation silently swallowed the email error.
func TestService_RequestChallenge_EmailFailureBubblesButRowPersists(t *testing.T) {
	repo := &mockChallengeRepo{}
	svc := NewService(ServiceDeps{
		Challenges: repo,
		Hasher:     &mockHasher{},
		Email:      &mockEmail{sendErr: errors.New("smtp down")},
	})

	challenge, err := svc.RequestChallenge(context.Background(), RequestChallengeInput{
		UserID:  uuid.New(),
		EmailTo: "user@example.com",
	})
	require.Error(t, err)
	require.NotNil(t, challenge, "row must persist so the user can resend")
	assert.Equal(t, 1, repo.createCount, "challenge row must be persisted before the email")
	assert.True(t, strings.Contains(err.Error(), "send"))
}

// TestService_RequestChallenge_NoEmailSendWhenEmailToEmpty asserts the
// service skips the SMTP round-trip when EmailTo is empty. This is the
// branch used by integration tests that don't want to wire an SMTP
// adapter and by future flows that deliver the code through SMS.
func TestService_RequestChallenge_NoEmailSendWhenEmailToEmpty(t *testing.T) {
	repo := &mockChallengeRepo{}
	mail := &mockEmail{}
	svc := NewService(ServiceDeps{
		Challenges: repo,
		Hasher:     &mockHasher{},
		Email:      mail,
	})

	challenge, err := svc.RequestChallenge(context.Background(), RequestChallengeInput{
		UserID:  uuid.New(),
		EmailTo: "",
	})
	require.NoError(t, err)
	require.NotNil(t, challenge)
	assert.Equal(t, 1, repo.createCount)
	assert.Empty(t, mail.sentTo, "no email must be sent when EmailTo is empty")
}

// TestService_RequestChallenge_RepoCreateFailureBubbles asserts the
// service surfaces a persistence error. The hasher already ran so the
// plaintext is gone; reporting an error to the caller is the only
// correct behaviour.
func TestService_RequestChallenge_RepoCreateFailureBubbles(t *testing.T) {
	repo := &mockChallengeRepo{
		createFn: func(_ context.Context, _ *twofactor.Challenge) error {
			return errors.New("conflict")
		},
	}
	svc := NewService(ServiceDeps{
		Challenges: repo,
		Hasher:     &mockHasher{},
		Email:      &mockEmail{},
	})

	_, err := svc.RequestChallenge(context.Background(), RequestChallengeInput{
		UserID:  uuid.New(),
		EmailTo: "user@example.com",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "persist")
}

// TestService_RequestChallenge_AuditOptional asserts the constructor
// is happy with Audits == nil and the issue path doesn't blow up.
// Older versions panicked because they assumed audits was non-nil.
func TestService_RequestChallenge_AuditOptional(t *testing.T) {
	svc := NewService(ServiceDeps{
		Challenges: &mockChallengeRepo{},
		Hasher:     &mockHasher{},
		Email:      &mockEmail{},
		Audits:     nil,
	})
	_, err := svc.RequestChallenge(context.Background(), RequestChallengeInput{
		UserID:  uuid.New(),
		EmailTo: "user@example.com",
	})
	require.NoError(t, err)
}

// TestService_VerifyChallenge_LoadErrorPropagates asserts that a
// non-sentinel repo failure surfaces wrapped (so handlers can log) but
// is not silently mapped to ErrChallengeNotFound — that would mask an
// outage as "user typed the wrong code".
func TestService_VerifyChallenge_LoadErrorPropagates(t *testing.T) {
	repo := &mockChallengeRepo{
		findFn: func(_ context.Context, _ uuid.UUID) (*twofactor.Challenge, error) {
			return nil, errors.New("connection lost")
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
	require.Error(t, err)
	assert.NotErrorIs(t, err, twofactor.ErrChallengeNotFound,
		"non-sentinel load error must NOT collapse to ErrChallengeNotFound")
	assert.Contains(t, err.Error(), "load challenge")
}

// TestService_VerifyChallenge_MarkUsedFailureBubbles asserts that if
// MarkUsed errors after a successful bcrypt match, the failure
// surfaces. We must never report "verified" while the DB still shows
// the challenge as pending.
func TestService_VerifyChallenge_MarkUsedFailureBubbles(t *testing.T) {
	plain := "246810"
	repo := &mockChallengeRepo{
		findFn: func(_ context.Context, uid uuid.UUID) (*twofactor.Challenge, error) {
			return &twofactor.Challenge{
				ID:           uuid.New(),
				UserID:       uid,
				CodeHash:     "h:" + plain,
				AttemptsLeft: 5,
				ExpiresAt:    time.Now().Add(5 * time.Minute),
			}, nil
		},
		markUsedFn: func(_ context.Context, _ uuid.UUID) error {
			return errors.New("disk full")
		},
	}
	svc := NewService(ServiceDeps{
		Challenges: repo,
		Hasher:     &mockHasher{},
		Email:      &mockEmail{},
	})
	_, err := svc.VerifyChallenge(context.Background(), VerifyChallengeInput{
		UserID: uuid.New(),
		Code:   plain,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mark used")
}

// TestService_VerifyChallenge_MismatchDecrementErrorSwallowed asserts
// that a flaky DB during DecrementAttempts does not change the
// user-visible outcome (still ErrCodeMismatch). The decrement is
// best-effort — failing it must not flip the verdict to success.
func TestService_VerifyChallenge_MismatchDecrementErrorSwallowed(t *testing.T) {
	repo := &mockChallengeRepo{
		findFn: func(_ context.Context, uid uuid.UUID) (*twofactor.Challenge, error) {
			return &twofactor.Challenge{
				ID:           uuid.New(),
				UserID:       uid,
				CodeHash:     "h:111111",
				AttemptsLeft: 3,
				ExpiresAt:    time.Now().Add(5 * time.Minute),
			}, nil
		},
		decrementFn: func(_ context.Context, _ uuid.UUID) error {
			return errors.New("flaky")
		},
	}
	svc := NewService(ServiceDeps{
		Challenges: repo,
		Hasher:     &mockHasher{},
		Email:      &mockEmail{},
	})
	_, err := svc.VerifyChallenge(context.Background(), VerifyChallengeInput{
		UserID: uuid.New(),
		Code:   "999999",
	})
	// Outcome must remain ErrCodeMismatch despite the decrement
	// failure — otherwise a flaky DB would let an attacker brute-force
	// the code by triggering decrement errors.
	assert.ErrorIs(t, err, twofactor.ErrCodeMismatch)
}

// TestService_VerifyChallenge_AuditFailuresSwallowed asserts that even
// when the audit repository errors on every call, the verify path
// still returns the correct verdict. Audit completeness is best-effort
// — refusing the login because audit insert failed would degrade UX.
func TestService_VerifyChallenge_AuditFailuresSwallowed(t *testing.T) {
	plain := "654321"
	repo := &mockChallengeRepo{
		findFn: func(_ context.Context, uid uuid.UUID) (*twofactor.Challenge, error) {
			return &twofactor.Challenge{
				ID:           uuid.New(),
				UserID:       uid,
				CodeHash:     "h:" + plain,
				AttemptsLeft: 5,
				ExpiresAt:    time.Now().Add(5 * time.Minute),
			}, nil
		},
	}
	svc := NewService(ServiceDeps{
		Challenges: repo,
		Hasher:     &mockHasher{},
		Email:      &mockEmail{},
		Audits:     &erroringAudit{},
	})
	c, err := svc.VerifyChallenge(context.Background(), VerifyChallengeInput{
		UserID: uuid.New(),
		Code:   plain,
	})
	require.NoError(t, err)
	require.NotNil(t, c)
}

// TestService_VerifyChallenge_AuditFailures_NotFoundBranch covers the
// audit-on-not-found path so the audit error-swallow logic runs on
// each branch.
func TestService_VerifyChallenge_AuditFailures_NotFoundBranch(t *testing.T) {
	repo := &mockChallengeRepo{
		findFn: func(_ context.Context, _ uuid.UUID) (*twofactor.Challenge, error) {
			return nil, repository.ErrTwoFactorChallengeNotFound
		},
	}
	svc := NewService(ServiceDeps{
		Challenges: repo,
		Hasher:     &mockHasher{},
		Email:      &mockEmail{},
		Audits:     &erroringAudit{},
	})
	_, err := svc.VerifyChallenge(context.Background(), VerifyChallengeInput{
		UserID: uuid.New(),
		Code:   "123456",
	})
	// Still must report not-found despite the audit insert failure.
	assert.ErrorIs(t, err, twofactor.ErrChallengeNotFound)
}

// TestService_RequestChallenge_AuditBuildFailureSwallowed asserts the
// logAudit helper tolerates an audit.NewEntry validation failure. We
// craft an input that the audit constructor rejects to exercise the
// fallback branch.
func TestService_RequestChallenge_AuditBuildFailureSwallowed(t *testing.T) {
	// Audit.NewEntry rejects unknown ResourceType. Service always
	// passes a valid one, so we can't easily force the error through
	// the public API; but we can directly exercise the logAudit code
	// path by calling it with an invalid input via reflection-free
	// access (logAudit is a method). We use the test's own visibility
	// of the type to exercise the branch.
	s := NewService(ServiceDeps{
		Challenges: &mockChallengeRepo{},
		Hasher:     &mockHasher{},
		Email:      &mockEmail{},
		Audits:     &mockAudit{},
	})
	// Calling logAudit with an invalid action key forces the
	// audit.NewEntry path to error and exercises the swallow branch.
	s.logAudit(context.Background(), audit.NewEntryInput{
		Action:       audit.Action(""), // empty Action triggers audit.ErrActionRequired
		ResourceType: audit.ResourceTypeUser,
	})
	// No panic, no error returned — fire-and-forget contract upheld.
}

// erroringAudit is an audit fake that always returns an error from Log.
// We use it to make sure the swallow logic in the service holds.
type erroringAudit struct{}

func (e *erroringAudit) Log(_ context.Context, _ *audit.Entry) error {
	return errors.New("audit table read-only")
}
func (e *erroringAudit) ListByResource(_ context.Context, _ audit.ResourceType, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}
func (e *erroringAudit) ListByUser(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*audit.Entry, string, error) {
	return nil, "", nil
}

// TestChallenge_TTLIsTenMinutes asserts the documented default TTL
// (10 min) matches the value the email body promises. Drift between
// the email copy and the domain constant breaks user trust.
func TestChallenge_TTLIsTenMinutes(t *testing.T) {
	assert.Equal(t, 10*time.Minute, twofactor.DefaultTTL,
		"DefaultTTL must remain 10 minutes — the email body literally tells the user 10 minutes")
}

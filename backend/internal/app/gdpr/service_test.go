package gdpr

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domaingdpr "marketplace-backend/internal/domain/gdpr"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/service"
)

// ---------------------------------------------------------------------
// Stubs
// ---------------------------------------------------------------------

type stubGDPRRepo struct {
	loadExportFn       func(ctx context.Context, id uuid.UUID) (*domaingdpr.Export, error)
	softDeleteFn       func(ctx context.Context, id uuid.UUID, at time.Time) (time.Time, error)
	cancelFn           func(ctx context.Context, id uuid.UUID) (bool, error)
	findBlockingFn     func(ctx context.Context, id uuid.UUID) ([]domaingdpr.BlockedOrg, error)
	listPurgeableFn    func(ctx context.Context, before time.Time, limit int) ([]uuid.UUID, error)
	purgeFn            func(ctx context.Context, id uuid.UUID, before time.Time, salt string) (bool, error)
}

func (s *stubGDPRRepo) LoadExport(ctx context.Context, id uuid.UUID) (*domaingdpr.Export, error) {
	return s.loadExportFn(ctx, id)
}
func (s *stubGDPRRepo) SoftDelete(ctx context.Context, id uuid.UUID, at time.Time) (time.Time, error) {
	return s.softDeleteFn(ctx, id, at)
}
func (s *stubGDPRRepo) CancelDeletion(ctx context.Context, id uuid.UUID) (bool, error) {
	return s.cancelFn(ctx, id)
}
func (s *stubGDPRRepo) FindOwnedOrgsBlockingDeletion(ctx context.Context, id uuid.UUID) ([]domaingdpr.BlockedOrg, error) {
	return s.findBlockingFn(ctx, id)
}
func (s *stubGDPRRepo) ListPurgeable(ctx context.Context, before time.Time, limit int) ([]uuid.UUID, error) {
	return s.listPurgeableFn(ctx, before, limit)
}
func (s *stubGDPRRepo) PurgeUser(ctx context.Context, id uuid.UUID, before time.Time, salt string) (bool, error) {
	return s.purgeFn(ctx, id, before, salt)
}

type stubUserRepo struct {
	getFn func(ctx context.Context, id uuid.UUID) (*user.User, error)
	stubMissingMethods
}

func (s *stubUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	return s.getFn(ctx, id)
}

type stubHasher struct {
	compareErr error
}

func (s *stubHasher) Hash(p string) (string, error)        { return "hash:" + p, nil }
func (s *stubHasher) Compare(hashed, password string) error { return s.compareErr }

type stubEmail struct {
	calls []emailCall
	err   error
}

type emailCall struct {
	to      string
	subject string
	html    string
}

func (s *stubEmail) SendNotification(_ context.Context, to, subject, html string) error {
	s.calls = append(s.calls, emailCall{to, subject, html})
	return s.err
}
func (s *stubEmail) SendPasswordReset(_ context.Context, _, _ string) error { return nil }
func (s *stubEmail) SendTeamInvitation(_ context.Context, _ service.TeamInvitationEmailInput) error {
	return nil
}
func (s *stubEmail) SendRolePermissionsChanged(_ context.Context, _ service.RolePermissionsChangedEmailInput) error {
	return nil
}

// Compile-time check that stubEmail implements service.EmailService.
var _ service.EmailService = (*stubEmail)(nil)

type recordingSigner struct {
	signed string
	parseFn func(token string, claims jwt.Claims) error
}

func (r *recordingSigner) Sign(claims jwt.Claims) (string, error) {
	mc, _ := claims.(jwt.MapClaims)
	if mc != nil {
		// store sub for assertion
		if sub, ok := mc["sub"].(string); ok {
			r.signed = sub + "|" + mc["purpose"].(string)
		}
	}
	return "stub.token.value", nil
}

func (r *recordingSigner) Parse(token string, claims jwt.Claims) error {
	if r.parseFn != nil {
		return r.parseFn(token, claims)
	}
	return nil
}

// ---------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------

func newServiceForTest(t *testing.T, deps ServiceDeps) *Service {
	t.Helper()
	if deps.Hasher == nil {
		deps.Hasher = &stubHasher{}
	}
	if deps.FrontendURL == "" {
		deps.FrontendURL = "https://app.test"
	}
	if deps.Clock == nil {
		now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
		deps.Clock = func() time.Time { return now }
	}
	if deps.Email == nil {
		deps.Email = &stubEmail{}
	}
	if deps.Signer == nil {
		deps.Signer = &recordingSigner{}
	}
	return NewService(deps)
}


func makeUser(t *testing.T, deleted bool) *user.User {
	t.Helper()
	u := &user.User{
		ID:             uuid.New(),
		Email:          "alice@example.com",
		FirstName:      "Alice",
		LastName:       "Doe",
		Role:           user.RoleProvider,
		HashedPassword: "hash:correct",
		Status:         user.StatusActive,
	}
	if deleted {
		now := time.Now().UTC()
		u.DeletedAt = &now
	}
	return u
}

// ---------------------------------------------------------------------
// ExportData
// ---------------------------------------------------------------------

func TestExportData_HappyPath(t *testing.T) {
	u := makeUser(t, false)
	expectedExport := &domaingdpr.Export{
		UserID:    u.ID,
		Email:     u.Email,
		Profile:   []map[string]any{{"id": u.ID.String()}},
		Timestamp: time.Now(),
	}
	repo := &stubGDPRRepo{
		loadExportFn: func(_ context.Context, id uuid.UUID) (*domaingdpr.Export, error) {
			require.Equal(t, u.ID, id)
			return expectedExport, nil
		},
	}
	users := &stubUserRepo{getFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
		return u, nil
	}}

	svc := newServiceForTest(t, ServiceDeps{Repo: repo, Users: users})
	got, err := svc.ExportData(context.Background(), u.ID)
	require.NoError(t, err)
	assert.Equal(t, "fr", got.Locale, "default locale should be fr")
	assert.Equal(t, expectedExport.Email, got.Email)
}

func TestExportData_RefusesSoftDeletedUser(t *testing.T) {
	u := makeUser(t, true)
	users := &stubUserRepo{getFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return u, nil
	}}
	svc := newServiceForTest(t, ServiceDeps{Users: users, Repo: &stubGDPRRepo{}})
	_, err := svc.ExportData(context.Background(), u.ID)
	assert.ErrorIs(t, err, user.ErrAccountScheduledForDeletion)
}

func TestExportData_RejectsEmptyExport(t *testing.T) {
	u := makeUser(t, false)
	repo := &stubGDPRRepo{
		loadExportFn: func(_ context.Context, _ uuid.UUID) (*domaingdpr.Export, error) {
			return &domaingdpr.Export{UserID: u.ID}, nil
		},
	}
	users := &stubUserRepo{getFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) { return u, nil }}
	svc := newServiceForTest(t, ServiceDeps{Repo: repo, Users: users})
	_, err := svc.ExportData(context.Background(), u.ID)
	assert.ErrorIs(t, err, domaingdpr.ErrEmptyExport)
}

func TestExportData_PropagatesUserNotFound(t *testing.T) {
	users := &stubUserRepo{getFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return nil, user.ErrUserNotFound
	}}
	svc := newServiceForTest(t, ServiceDeps{Users: users, Repo: &stubGDPRRepo{}})
	_, err := svc.ExportData(context.Background(), uuid.New())
	assert.ErrorIs(t, err, user.ErrUserNotFound)
}

// ---------------------------------------------------------------------
// RequestDeletion
// ---------------------------------------------------------------------

func TestRequestDeletion_HappyPath_SendsEmail(t *testing.T) {
	u := makeUser(t, false)
	stubMail := &stubEmail{}
	users := &stubUserRepo{getFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) { return u, nil }}
	repo := &stubGDPRRepo{
		findBlockingFn: func(_ context.Context, _ uuid.UUID) ([]domaingdpr.BlockedOrg, error) {
			return nil, nil
		},
	}
	svc := newServiceForTest(t, ServiceDeps{
		Repo:   repo,
		Users:  users,
		Email:  stubMail,
		Hasher: &stubHasher{},
	})
	res, err := svc.RequestDeletion(context.Background(), RequestDeletionInput{
		UserID:   u.ID,
		Password: "correct",
	})
	require.NoError(t, err)
	assert.Equal(t, u.Email, res.EmailSentTo)
	assert.Len(t, stubMail.calls, 1)
	assert.Contains(t, stubMail.calls[0].html, "/account/confirm-deletion?token=")
}

func TestRequestDeletion_RefusesWrongPassword(t *testing.T) {
	u := makeUser(t, false)
	users := &stubUserRepo{getFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) { return u, nil }}
	svc := newServiceForTest(t, ServiceDeps{
		Repo:   &stubGDPRRepo{},
		Users:  users,
		Hasher: &stubHasher{compareErr: user.ErrInvalidCredentials},
	})
	_, err := svc.RequestDeletion(context.Background(), RequestDeletionInput{
		UserID:   u.ID,
		Password: "wrong",
	})
	assert.ErrorIs(t, err, user.ErrInvalidCredentials)
}

func TestRequestDeletion_RefusesOrgOwnerWithMembers(t *testing.T) {
	u := makeUser(t, false)
	orgID := uuid.New()
	users := &stubUserRepo{getFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) { return u, nil }}
	repo := &stubGDPRRepo{
		findBlockingFn: func(_ context.Context, _ uuid.UUID) ([]domaingdpr.BlockedOrg, error) {
			return []domaingdpr.BlockedOrg{
				{
					OrgID:       orgID,
					OrgName:     "Acme",
					MemberCount: 3,
					Actions:     []domaingdpr.RemediationAction{domaingdpr.ActionTransferOwnership, domaingdpr.ActionDissolveOrg},
				},
			}, nil
		},
	}
	svc := newServiceForTest(t, ServiceDeps{Repo: repo, Users: users})
	_, err := svc.RequestDeletion(context.Background(), RequestDeletionInput{
		UserID:   u.ID,
		Password: "correct",
	})
	var blocked *domaingdpr.OwnerBlockedError
	require.ErrorAs(t, err, &blocked)
	require.Len(t, blocked.Orgs, 1)
	assert.Equal(t, "Acme", blocked.Orgs[0].OrgName)
}

func TestRequestDeletion_AlreadyScheduledIsIdempotent(t *testing.T) {
	u := makeUser(t, true)
	stubMail := &stubEmail{}
	users := &stubUserRepo{getFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) { return u, nil }}
	svc := newServiceForTest(t, ServiceDeps{
		Repo:  &stubGDPRRepo{},
		Users: users,
		Email: stubMail,
	})
	// No password check, no blocking check — re-sends email directly.
	res, err := svc.RequestDeletion(context.Background(), RequestDeletionInput{
		UserID:   u.ID,
		Password: "anything",
	})
	require.NoError(t, err)
	assert.Equal(t, u.Email, res.EmailSentTo)
	assert.Len(t, stubMail.calls, 1, "second request should re-send the email")
}

func TestRequestDeletion_PropagatesUserNotFound(t *testing.T) {
	users := &stubUserRepo{getFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return nil, user.ErrUserNotFound
	}}
	svc := newServiceForTest(t, ServiceDeps{Users: users, Repo: &stubGDPRRepo{}})
	_, err := svc.RequestDeletion(context.Background(), RequestDeletionInput{UserID: uuid.New(), Password: "x"})
	assert.ErrorIs(t, err, user.ErrUserNotFound)
}

// ---------------------------------------------------------------------
// ConfirmDeletion
// ---------------------------------------------------------------------

func TestConfirmDeletion_HappyPath(t *testing.T) {
	u := makeUser(t, false)
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	signer := &recordingSigner{
		parseFn: func(_ string, claims jwt.Claims) error {
			mc := claims.(jwt.MapClaims)
			mc["sub"] = u.ID.String()
			mc["purpose"] = domaingdpr.ConfirmationTokenPurpose
			mc["exp"] = float64(now.Add(time.Hour).Unix())
			return nil
		},
	}
	users := &stubUserRepo{getFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) { return u, nil }}
	softCalled := false
	repo := &stubGDPRRepo{
		softDeleteFn: func(_ context.Context, id uuid.UUID, at time.Time) (time.Time, error) {
			softCalled = true
			require.Equal(t, u.ID, id)
			return at, nil
		},
	}

	svc := newServiceForTest(t, ServiceDeps{
		Repo:   repo,
		Users:  users,
		Signer: signer,
		Clock:  func() time.Time { return now },
	})

	got, err := svc.ConfirmDeletion(context.Background(), "stub.token")
	require.NoError(t, err)
	assert.True(t, softCalled)
	assert.Equal(t, u.ID, got.UserID)
	assert.Equal(t, now.Add(domaingdpr.PurgeWindow), got.HardDeleteAt)
}

func TestConfirmDeletion_RejectsTokenWithWrongPurpose(t *testing.T) {
	signer := &recordingSigner{
		parseFn: func(_ string, claims jwt.Claims) error {
			mc := claims.(jwt.MapClaims)
			mc["sub"] = uuid.New().String()
			mc["purpose"] = "password_reset"
			return nil
		},
	}
	svc := newServiceForTest(t, ServiceDeps{
		Repo:   &stubGDPRRepo{},
		Users:  &stubUserRepo{},
		Signer: signer,
	})
	_, err := svc.ConfirmDeletion(context.Background(), "tok")
	assert.ErrorIs(t, err, user.ErrUnauthorized)
}

func TestConfirmDeletion_RejectsInvalidToken(t *testing.T) {
	signer := &recordingSigner{
		parseFn: func(_ string, _ jwt.Claims) error {
			return errors.New("invalid signature")
		},
	}
	svc := newServiceForTest(t, ServiceDeps{
		Repo:   &stubGDPRRepo{},
		Users:  &stubUserRepo{},
		Signer: signer,
	})
	_, err := svc.ConfirmDeletion(context.Background(), "tok")
	assert.ErrorIs(t, err, user.ErrUnauthorized)
}

func TestConfirmDeletion_RejectsBadSubUUID(t *testing.T) {
	signer := &recordingSigner{
		parseFn: func(_ string, claims jwt.Claims) error {
			mc := claims.(jwt.MapClaims)
			mc["sub"] = "not-a-uuid"
			mc["purpose"] = domaingdpr.ConfirmationTokenPurpose
			return nil
		},
	}
	svc := newServiceForTest(t, ServiceDeps{
		Repo:   &stubGDPRRepo{},
		Users:  &stubUserRepo{},
		Signer: signer,
	})
	_, err := svc.ConfirmDeletion(context.Background(), "tok")
	assert.ErrorIs(t, err, user.ErrUnauthorized)
}

// ---------------------------------------------------------------------
// CancelDeletion
// ---------------------------------------------------------------------

func TestCancelDeletion_HappyPath(t *testing.T) {
	id := uuid.New()
	repo := &stubGDPRRepo{
		cancelFn: func(_ context.Context, got uuid.UUID) (bool, error) {
			require.Equal(t, id, got)
			return true, nil
		},
	}
	svc := newServiceForTest(t, ServiceDeps{Repo: repo, Users: &stubUserRepo{}})
	res, err := svc.CancelDeletion(context.Background(), id)
	require.NoError(t, err)
	assert.False(t, res.NoOp)
}

func TestCancelDeletion_NoOpWhenNotScheduled(t *testing.T) {
	repo := &stubGDPRRepo{
		cancelFn: func(_ context.Context, _ uuid.UUID) (bool, error) { return false, nil },
	}
	svc := newServiceForTest(t, ServiceDeps{Repo: repo, Users: &stubUserRepo{}})
	res, err := svc.CancelDeletion(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.True(t, res.NoOp)
}

// ---------------------------------------------------------------------
// PurgeOnce — race + happy path
// ---------------------------------------------------------------------

func TestPurgeOnce_RunsOneRound(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	repo := &stubGDPRRepo{
		listPurgeableFn: func(_ context.Context, before time.Time, _ int) ([]uuid.UUID, error) {
			assert.True(t, before.Before(time.Now().UTC()), "cutoff must be in the past")
			return []uuid.UUID{id1, id2}, nil
		},
		purgeFn: func(_ context.Context, _ uuid.UUID, _ time.Time, salt string) (bool, error) {
			require.Equal(t, "salt-1", salt)
			return true, nil
		},
	}
	svc := newServiceForTest(t, ServiceDeps{Repo: repo, Users: &stubUserRepo{}})
	res, err := svc.PurgeOnce(context.Background(), "salt-1", 10)
	require.NoError(t, err)
	assert.Equal(t, 2, res.Examined)
	assert.Equal(t, 2, res.Purged)
	assert.Empty(t, res.Errors)
}

func TestPurgeOnce_IgnoresCancelRace(t *testing.T) {
	id := uuid.New()
	repo := &stubGDPRRepo{
		listPurgeableFn: func(_ context.Context, _ time.Time, _ int) ([]uuid.UUID, error) {
			return []uuid.UUID{id}, nil
		},
		purgeFn: func(_ context.Context, _ uuid.UUID, _ time.Time, _ string) (bool, error) {
			// Simulate the cancel race: row was cancelled in the
			// interval between ListPurgeable and PurgeUser.
			return false, nil
		},
	}
	svc := newServiceForTest(t, ServiceDeps{Repo: repo, Users: &stubUserRepo{}})
	res, err := svc.PurgeOnce(context.Background(), "s", 10)
	require.NoError(t, err)
	assert.Equal(t, 1, res.Examined)
	assert.Equal(t, 0, res.Purged, "cancel won the race so no row was purged")
}

func TestPurgeOnce_RecordsPerRowErrors(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	repo := &stubGDPRRepo{
		listPurgeableFn: func(_ context.Context, _ time.Time, _ int) ([]uuid.UUID, error) {
			return []uuid.UUID{id1, id2}, nil
		},
		purgeFn: func(_ context.Context, id uuid.UUID, _ time.Time, _ string) (bool, error) {
			if id == id1 {
				return false, errors.New("transient db error")
			}
			return true, nil
		},
	}
	svc := newServiceForTest(t, ServiceDeps{Repo: repo, Users: &stubUserRepo{}})
	res, err := svc.PurgeOnce(context.Background(), "s", 10)
	require.NoError(t, err)
	assert.Equal(t, 2, res.Examined)
	assert.Equal(t, 1, res.Purged, "second row must still be purged after the first errored")
	assert.Len(t, res.Errors, 1)
}

func TestPurgeOnce_RejectsEmptySalt(t *testing.T) {
	svc := newServiceForTest(t, ServiceDeps{Repo: &stubGDPRRepo{}, Users: &stubUserRepo{}})
	_, err := svc.PurgeOnce(context.Background(), "", 10)
	assert.ErrorIs(t, err, domaingdpr.ErrSaltRequired)
}

// ---------------------------------------------------------------------
// Signer (real HS256)
// ---------------------------------------------------------------------

func TestHS256Signer_RoundTrip(t *testing.T) {
	signer, err := NewHS256Signer("supersecret-test-key-min-32-bytes-XX")
	require.NoError(t, err)

	now := time.Now()
	claims := jwt.MapClaims{
		"sub":     uuid.New().String(),
		"purpose": domaingdpr.ConfirmationTokenPurpose,
		"iat":     now.Unix(),
		"exp":     now.Add(time.Hour).Unix(),
	}
	token, err := signer.Sign(claims)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	parsed := jwt.MapClaims{}
	require.NoError(t, signer.Parse(token, &parsed))
	assert.Equal(t, claims["sub"], parsed["sub"])
	assert.Equal(t, claims["purpose"], parsed["purpose"])
}

func TestHS256Signer_RejectsTamperedToken(t *testing.T) {
	signer, err := NewHS256Signer("supersecret-test-key-min-32-bytes-XX")
	require.NoError(t, err)
	token, err := signer.Sign(jwt.MapClaims{"sub": "x"})
	require.NoError(t, err)
	parsed := jwt.MapClaims{}
	err = signer.Parse(token+"tampered", &parsed)
	assert.Error(t, err)
}

func TestHS256Signer_EmptySecretFails(t *testing.T) {
	_, err := NewHS256Signer("")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------
// Email templates — both languages render expected strings
// ---------------------------------------------------------------------

func TestRenderConfirmationEmail_FrenchDefault(t *testing.T) {
	subject, html := renderConfirmationEmail("fr", confirmEmailParams{
		FirstName:  "Alice",
		ConfirmURL: "https://example.com/confirm?t=abc",
		ExpiresAt:  time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	})
	assert.Contains(t, subject, "Confirmez la suppression")
	assert.Contains(t, html, "Bonjour Alice")
	assert.Contains(t, html, "https://example.com/confirm?t=abc")
	assert.Contains(t, html, "30 jours")
}

func TestRenderConfirmationEmail_English(t *testing.T) {
	subject, html := renderConfirmationEmail("en", confirmEmailParams{
		FirstName:  "Bob",
		ConfirmURL: "https://example.com/confirm?t=abc",
		ExpiresAt:  time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	})
	assert.Contains(t, subject, "Confirm")
	assert.Contains(t, html, "Hello Bob")
	assert.Contains(t, html, "30 days")
}

func TestRenderReminderEmail_BothLanguagesIncludeCancelURL(t *testing.T) {
	for _, lang := range []string{"fr", "en"} {
		_, html := renderReminderEmail(lang, reminderEmailParams{
			FirstName:    "Alice",
			HardDeleteAt: time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC),
			CancelURL:    "https://example.com/cancel?t=zz",
		})
		assert.Contains(t, html, "https://example.com/cancel?t=zz", "lang=%s", lang)
	}
}

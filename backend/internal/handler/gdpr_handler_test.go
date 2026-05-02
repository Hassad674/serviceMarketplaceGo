package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gdprapp "marketplace-backend/internal/app/gdpr"
	domaingdpr "marketplace-backend/internal/domain/gdpr"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ---------------------------------------------------------------------
// Test stubs (mirror app/gdpr/stubs but located here so we don't expose
// them outside _test.go)
// ---------------------------------------------------------------------

type fakeGDPRRepo struct {
	loadExportFn    func(context.Context, uuid.UUID) (*domaingdpr.Export, error)
	softDeleteFn    func(context.Context, uuid.UUID, time.Time) (time.Time, error)
	cancelFn        func(context.Context, uuid.UUID) (bool, error)
	findBlockingFn  func(context.Context, uuid.UUID) ([]domaingdpr.BlockedOrg, error)
	listPurgeableFn func(context.Context, time.Time, int) ([]uuid.UUID, error)
	purgeFn         func(context.Context, uuid.UUID, time.Time, string) (bool, error)
}

func (f *fakeGDPRRepo) LoadExport(ctx context.Context, id uuid.UUID) (*domaingdpr.Export, error) {
	return f.loadExportFn(ctx, id)
}
func (f *fakeGDPRRepo) SoftDelete(ctx context.Context, id uuid.UUID, t time.Time) (time.Time, error) {
	return f.softDeleteFn(ctx, id, t)
}
func (f *fakeGDPRRepo) CancelDeletion(ctx context.Context, id uuid.UUID) (bool, error) {
	return f.cancelFn(ctx, id)
}
func (f *fakeGDPRRepo) FindOwnedOrgsBlockingDeletion(ctx context.Context, id uuid.UUID) ([]domaingdpr.BlockedOrg, error) {
	if f.findBlockingFn == nil {
		return nil, nil
	}
	return f.findBlockingFn(ctx, id)
}
func (f *fakeGDPRRepo) ListPurgeable(ctx context.Context, before time.Time, limit int) ([]uuid.UUID, error) {
	return f.listPurgeableFn(ctx, before, limit)
}
func (f *fakeGDPRRepo) PurgeUser(ctx context.Context, id uuid.UUID, before time.Time, salt string) (bool, error) {
	return f.purgeFn(ctx, id, before, salt)
}

type fakeUserRepo struct {
	getFn func(context.Context, uuid.UUID) (*user.User, error)
}

func (f *fakeUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	return f.getFn(ctx, id)
}
func (f *fakeUserRepo) Create(context.Context, *user.User) error                { return nil }
func (f *fakeUserRepo) GetByEmail(context.Context, string) (*user.User, error)  { return nil, nil }
func (f *fakeUserRepo) Update(context.Context, *user.User) error                { return nil }
func (f *fakeUserRepo) Delete(context.Context, uuid.UUID) error                 { return nil }
func (f *fakeUserRepo) ExistsByEmail(context.Context, string) (bool, error)     { return false, nil }
func (f *fakeUserRepo) ListAdmin(context.Context, repository.AdminUserFilters) ([]*user.User, string, error) {
	return nil, "", nil
}
func (f *fakeUserRepo) CountAdmin(context.Context, repository.AdminUserFilters) (int, error) {
	return 0, nil
}
func (f *fakeUserRepo) CountByRole(context.Context) (map[string]int, error)   { return nil, nil }
func (f *fakeUserRepo) CountByStatus(context.Context) (map[string]int, error) { return nil, nil }
func (f *fakeUserRepo) RecentSignups(context.Context, int) ([]*user.User, error) {
	return nil, nil
}
func (f *fakeUserRepo) BumpSessionVersion(context.Context, uuid.UUID) (int, error) { return 0, nil }
func (f *fakeUserRepo) GetSessionVersion(context.Context, uuid.UUID) (int, error)  { return 0, nil }
func (f *fakeUserRepo) UpdateEmailNotificationsEnabled(context.Context, uuid.UUID, bool) error {
	return nil
}
func (f *fakeUserRepo) TouchLastActive(context.Context, uuid.UUID) error { return nil }

type fakeHasher struct{ err error }

func (f *fakeHasher) Hash(s string) (string, error)         { return "h:" + s, nil }
func (f *fakeHasher) Compare(_ string, _ string) error      { return f.err }

type fakeEmail struct {
	calls []struct{ to, subject, html string }
	err   error
}

func (f *fakeEmail) SendPasswordReset(context.Context, string, string) error { return nil }
func (f *fakeEmail) SendNotification(_ context.Context, to, subject, html string) error {
	f.calls = append(f.calls, struct{ to, subject, html string }{to, subject, html})
	return f.err
}
func (f *fakeEmail) SendTeamInvitation(context.Context, service.TeamInvitationEmailInput) error {
	return nil
}
func (f *fakeEmail) SendRolePermissionsChanged(context.Context, service.RolePermissionsChangedEmailInput) error {
	return nil
}

type fakeSigner struct {
	parseErr     error
	claimsToFill map[string]any
}

func (f *fakeSigner) Sign(jwt.Claims) (string, error) { return "stub.token", nil }
func (f *fakeSigner) Parse(_ string, claims jwt.Claims) error {
	if f.parseErr != nil {
		return f.parseErr
	}
	if mc, ok := claims.(jwt.MapClaims); ok && f.claimsToFill != nil {
		for k, v := range f.claimsToFill {
			mc[k] = v
		}
	}
	return nil
}

func newGDPRTestHandler(t *testing.T, repo *fakeGDPRRepo, users *fakeUserRepo, mail *fakeEmail, hasher *fakeHasher, signer *fakeSigner) *GDPRHandler {
	t.Helper()
	if hasher == nil {
		hasher = &fakeHasher{}
	}
	if mail == nil {
		mail = &fakeEmail{}
	}
	if signer == nil {
		signer = &fakeSigner{}
	}
	svc := gdprapp.NewService(gdprapp.ServiceDeps{
		Repo:        repo,
		Users:       users,
		Hasher:      hasher,
		Email:       mail,
		Signer:      signer,
		FrontendURL: "https://app.test",
		Clock: func() time.Time {
			return time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
		},
	})
	return NewGDPRHandler(svc)
}

func ctxWithUser(t *testing.T, id uuid.UUID) context.Context {
	t.Helper()
	return context.WithValue(context.Background(), middleware.ContextKeyUserID, id)
}

// ---------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------

func TestGDPRHandler_Export_HappyPath_ZIPContents(t *testing.T) {
	uid := uuid.New()
	export := &domaingdpr.Export{
		UserID:    uid,
		Email:     "alice@example.com",
		Locale:    "en",
		Timestamp: time.Now().UTC(),
		Profile:   []map[string]any{{"id": uid.String(), "email": "alice@example.com"}},
		Proposals: []map[string]any{},
		Messages:  []map[string]any{},
	}
	repo := &fakeGDPRRepo{
		loadExportFn: func(_ context.Context, _ uuid.UUID) (*domaingdpr.Export, error) {
			return export, nil
		},
	}
	users := &fakeUserRepo{getFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return &user.User{ID: uid, Email: "alice@example.com"}, nil
	}}
	h := newGDPRTestHandler(t, repo, users, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/export", nil)
	req = req.WithContext(ctxWithUser(t, uid))
	rec := httptest.NewRecorder()
	h.Export(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/zip", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "marketplace-export-")

	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	require.NoError(t, err)

	files := map[string]bool{}
	for _, f := range zr.File {
		files[f.Name] = true
	}
	assert.True(t, files["manifest.json"], "manifest.json must be in the zip")
	assert.True(t, files["README.txt"], "README.txt must be in the zip")
	assert.True(t, files["profile.json"], "profile.json must be in the zip")
	assert.True(t, files["proposals.json"], "proposals.json must be in the zip")
	assert.True(t, files["messages.json"], "messages.json must be in the zip")
	assert.True(t, files["audit_logs.json"], "audit_logs.json must be in the zip")

	// Manifest content
	mf := openZIP(t, zr, "manifest.json")
	var manifest map[string]any
	require.NoError(t, json.Unmarshal(mf, &manifest))
	assert.Equal(t, domaingdpr.ExportVersion, manifest["version"])
	assert.Equal(t, uid.String(), manifest["user_id"])
}

func TestGDPRHandler_Export_RefusesWithoutAuth(t *testing.T) {
	h := newGDPRTestHandler(t, &fakeGDPRRepo{}, &fakeUserRepo{}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/export", nil)
	rec := httptest.NewRecorder()
	h.Export(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGDPRHandler_Export_RefusesScheduledForDeletion(t *testing.T) {
	uid := uuid.New()
	now := time.Now().UTC()
	users := &fakeUserRepo{getFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return &user.User{ID: uid, Email: "x@y.com", DeletedAt: &now}, nil
	}}
	h := newGDPRTestHandler(t, &fakeGDPRRepo{}, users, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/export", nil).
		WithContext(ctxWithUser(t, uid))
	rec := httptest.NewRecorder()
	h.Export(rec, req)
	assert.Equal(t, http.StatusGone, rec.Code)
}

func TestGDPRHandler_Export_404OnUserNotFound(t *testing.T) {
	uid := uuid.New()
	users := &fakeUserRepo{getFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return nil, user.ErrUserNotFound
	}}
	h := newGDPRTestHandler(t, &fakeGDPRRepo{}, users, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/export", nil).
		WithContext(ctxWithUser(t, uid))
	rec := httptest.NewRecorder()
	h.Export(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------
// RequestDeletion
// ---------------------------------------------------------------------

func TestGDPRHandler_RequestDeletion_HappyPath(t *testing.T) {
	uid := uuid.New()
	repo := &fakeGDPRRepo{
		findBlockingFn: func(_ context.Context, _ uuid.UUID) ([]domaingdpr.BlockedOrg, error) {
			return nil, nil
		},
	}
	users := &fakeUserRepo{getFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return &user.User{ID: uid, Email: "alice@example.com", FirstName: "Alice"}, nil
	}}
	mail := &fakeEmail{}
	h := newGDPRTestHandler(t, repo, users, mail, nil, nil)

	body := strings.NewReader(`{"password":"correct","confirm":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/account/request-deletion", body).
		WithContext(ctxWithUser(t, uid))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.RequestDeletion(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, mail.calls, 1, "confirmation email must be sent")
	var payload map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&payload))
	assert.Equal(t, "alice@example.com", payload["email_sent_to"])
}

func TestGDPRHandler_RequestDeletion_RefusesWithoutConfirm(t *testing.T) {
	uid := uuid.New()
	h := newGDPRTestHandler(t, &fakeGDPRRepo{}, &fakeUserRepo{}, nil, nil, nil)
	body := strings.NewReader(`{"password":"x","confirm":false}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/account/request-deletion", body).
		WithContext(ctxWithUser(t, uid))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.RequestDeletion(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "confirm_required")
}

func TestGDPRHandler_RequestDeletion_RefusesEmptyPassword(t *testing.T) {
	uid := uuid.New()
	h := newGDPRTestHandler(t, &fakeGDPRRepo{}, &fakeUserRepo{}, nil, nil, nil)
	body := strings.NewReader(`{"password":"","confirm":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/account/request-deletion", body).
		WithContext(ctxWithUser(t, uid))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.RequestDeletion(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGDPRHandler_RequestDeletion_401OnWrongPassword(t *testing.T) {
	uid := uuid.New()
	users := &fakeUserRepo{getFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return &user.User{ID: uid, Email: "x@y.com"}, nil
	}}
	hasher := &fakeHasher{err: user.ErrInvalidCredentials}
	h := newGDPRTestHandler(t, &fakeGDPRRepo{
		findBlockingFn: func(_ context.Context, _ uuid.UUID) ([]domaingdpr.BlockedOrg, error) {
			return nil, nil
		},
	}, users, nil, hasher, nil)
	body := strings.NewReader(`{"password":"wrong","confirm":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/account/request-deletion", body).
		WithContext(ctxWithUser(t, uid))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.RequestDeletion(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_password")
}

func TestGDPRHandler_RequestDeletion_409OnOrgOwnerWithMembers(t *testing.T) {
	uid := uuid.New()
	orgID := uuid.New()
	repo := &fakeGDPRRepo{
		findBlockingFn: func(_ context.Context, _ uuid.UUID) ([]domaingdpr.BlockedOrg, error) {
			return []domaingdpr.BlockedOrg{
				{
					OrgID:       orgID,
					OrgName:     "Acme",
					MemberCount: 4,
					Admins: []domaingdpr.AvailableAdmin{
						{UserID: uuid.New(), Email: "admin1@acme.test"},
					},
					Actions: []domaingdpr.RemediationAction{
						domaingdpr.ActionTransferOwnership,
						domaingdpr.ActionDissolveOrg,
					},
				},
			}, nil
		},
	}
	users := &fakeUserRepo{getFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return &user.User{ID: uid, Email: "x@y.com"}, nil
	}}
	h := newGDPRTestHandler(t, repo, users, nil, nil, nil)
	body := strings.NewReader(`{"password":"correct","confirm":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/account/request-deletion", body).
		WithContext(ctxWithUser(t, uid))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.RequestDeletion(rec, req)
	require.Equal(t, http.StatusConflict, rec.Code)

	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Details struct {
				BlockedOrgs []struct {
					OrgID       string `json:"org_id"`
					OrgName     string `json:"org_name"`
					MemberCount int    `json:"member_count"`
				} `json:"blocked_orgs"`
			} `json:"details"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&payload))
	assert.Equal(t, "owner_must_transfer_or_dissolve", payload.Error.Code)
	require.Len(t, payload.Error.Details.BlockedOrgs, 1)
	assert.Equal(t, "Acme", payload.Error.Details.BlockedOrgs[0].OrgName)
	assert.Equal(t, 4, payload.Error.Details.BlockedOrgs[0].MemberCount)
}

func TestGDPRHandler_RequestDeletion_RefusesWithoutAuth(t *testing.T) {
	h := newGDPRTestHandler(t, &fakeGDPRRepo{}, &fakeUserRepo{}, nil, nil, nil)
	body := strings.NewReader(`{"password":"x","confirm":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/account/request-deletion", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.RequestDeletion(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGDPRHandler_RequestDeletion_RejectsMalformedBody(t *testing.T) {
	uid := uuid.New()
	h := newGDPRTestHandler(t, &fakeGDPRRepo{}, &fakeUserRepo{}, nil, nil, nil)
	body := strings.NewReader(`{not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/account/request-deletion", body).
		WithContext(ctxWithUser(t, uid))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.RequestDeletion(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ---------------------------------------------------------------------
// ConfirmDeletion
// ---------------------------------------------------------------------

func TestGDPRHandler_ConfirmDeletion_HappyPath(t *testing.T) {
	uid := uuid.New()
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	users := &fakeUserRepo{getFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return &user.User{ID: uid, Email: "x@y.com"}, nil
	}}
	repo := &fakeGDPRRepo{
		softDeleteFn: func(_ context.Context, id uuid.UUID, _ time.Time) (time.Time, error) {
			require.Equal(t, uid, id)
			return now, nil
		},
	}
	signer := &fakeSigner{
		claimsToFill: map[string]any{
			"sub":     uid.String(),
			"purpose": domaingdpr.ConfirmationTokenPurpose,
		},
	}
	h := newGDPRTestHandler(t, repo, users, nil, nil, signer)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/account/confirm-deletion?token=valid", nil)
	rec := httptest.NewRecorder()
	h.ConfirmDeletion(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var payload map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&payload))
	assert.Equal(t, uid.String(), payload["user_id"])
	assert.NotEmpty(t, payload["deleted_at"])
	assert.NotEmpty(t, payload["hard_delete_at"])
}

func TestGDPRHandler_ConfirmDeletion_400OnMissingToken(t *testing.T) {
	h := newGDPRTestHandler(t, &fakeGDPRRepo{}, &fakeUserRepo{}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/account/confirm-deletion", nil)
	rec := httptest.NewRecorder()
	h.ConfirmDeletion(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGDPRHandler_ConfirmDeletion_401OnInvalidToken(t *testing.T) {
	signer := &fakeSigner{parseErr: errors.New("invalid signature")}
	h := newGDPRTestHandler(t, &fakeGDPRRepo{}, &fakeUserRepo{}, nil, nil, signer)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/account/confirm-deletion?token=bad", nil)
	rec := httptest.NewRecorder()
	h.ConfirmDeletion(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_token")
}

func TestGDPRHandler_ConfirmDeletion_401OnWrongPurpose(t *testing.T) {
	signer := &fakeSigner{
		claimsToFill: map[string]any{
			"sub":     uuid.New().String(),
			"purpose": "password_reset",
		},
	}
	h := newGDPRTestHandler(t, &fakeGDPRRepo{}, &fakeUserRepo{}, nil, nil, signer)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/account/confirm-deletion?token=tok", nil)
	rec := httptest.NewRecorder()
	h.ConfirmDeletion(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGDPRHandler_ConfirmDeletion_404WhenUserGone(t *testing.T) {
	uid := uuid.New()
	users := &fakeUserRepo{getFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
		return nil, user.ErrUserNotFound
	}}
	signer := &fakeSigner{
		claimsToFill: map[string]any{
			"sub":     uid.String(),
			"purpose": domaingdpr.ConfirmationTokenPurpose,
		},
	}
	h := newGDPRTestHandler(t, &fakeGDPRRepo{}, users, nil, nil, signer)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/account/confirm-deletion?token=tok", nil)
	rec := httptest.NewRecorder()
	h.ConfirmDeletion(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------
// CancelDeletion
// ---------------------------------------------------------------------

func TestGDPRHandler_CancelDeletion_HappyPath(t *testing.T) {
	uid := uuid.New()
	repo := &fakeGDPRRepo{
		cancelFn: func(_ context.Context, id uuid.UUID) (bool, error) {
			require.Equal(t, uid, id)
			return true, nil
		},
	}
	h := newGDPRTestHandler(t, repo, &fakeUserRepo{}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/account/cancel-deletion", nil).
		WithContext(ctxWithUser(t, uid))
	rec := httptest.NewRecorder()
	h.CancelDeletion(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var payload map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&payload))
	assert.Equal(t, true, payload["cancelled"])
}

func TestGDPRHandler_CancelDeletion_NoOp(t *testing.T) {
	uid := uuid.New()
	repo := &fakeGDPRRepo{
		cancelFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return false, nil
		},
	}
	h := newGDPRTestHandler(t, repo, &fakeUserRepo{}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/account/cancel-deletion", nil).
		WithContext(ctxWithUser(t, uid))
	rec := httptest.NewRecorder()
	h.CancelDeletion(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var payload map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&payload))
	assert.Equal(t, false, payload["cancelled"])
}

func TestGDPRHandler_CancelDeletion_RefusesWithoutAuth(t *testing.T) {
	h := newGDPRTestHandler(t, &fakeGDPRRepo{}, &fakeUserRepo{}, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/account/cancel-deletion", nil)
	rec := httptest.NewRecorder()
	h.CancelDeletion(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ---------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------

func openZIP(t *testing.T, zr *zip.Reader, name string) []byte {
	t.Helper()
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			require.NoError(t, err)
			defer rc.Close()
			b, err := io.ReadAll(rc)
			require.NoError(t, err)
			return b
		}
	}
	t.Fatalf("file %q not found in zip", name)
	return nil
}

// Compile-time assertions
var (
	_ repository.GDPRRepository = (*fakeGDPRRepo)(nil)
	_ repository.UserRepository = (*fakeUserRepo)(nil)
	_ service.HasherService     = (*fakeHasher)(nil)
	_ service.EmailService      = (*fakeEmail)(nil)
)

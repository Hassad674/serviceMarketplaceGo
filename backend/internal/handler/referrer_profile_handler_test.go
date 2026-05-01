package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	referrerprofileapp "marketplace-backend/internal/app/referrerprofile"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/profile"
	domainreferrer "marketplace-backend/internal/domain/referrerprofile"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
)

type mockReferrerProfileRepo struct {
	getOrCreateFn func(ctx context.Context, orgID uuid.UUID) (*repository.ReferrerProfileView, error)
	updateCoreFn  func(ctx context.Context, orgID uuid.UUID, title, about, videoURL string) error
	updateAvailFn func(ctx context.Context, orgID uuid.UUID, status profile.AvailabilityStatus) error
	updateExpFn   func(ctx context.Context, orgID uuid.UUID, domains []string) error
	updateVideoFn func(ctx context.Context, orgID uuid.UUID, videoURL string) error
	getVideoFn    func(ctx context.Context, orgID uuid.UUID) (string, error)
}

func (m *mockReferrerProfileRepo) GetOrCreateByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.ReferrerProfileView, error) {
	if m.getOrCreateFn != nil {
		return m.getOrCreateFn(ctx, orgID)
	}
	return newReferrerStubView(orgID), nil
}
func (m *mockReferrerProfileRepo) UpdateCore(ctx context.Context, orgID uuid.UUID, title, about, videoURL string) error {
	if m.updateCoreFn != nil {
		return m.updateCoreFn(ctx, orgID, title, about, videoURL)
	}
	return nil
}
func (m *mockReferrerProfileRepo) UpdateAvailability(ctx context.Context, orgID uuid.UUID, status profile.AvailabilityStatus) error {
	if m.updateAvailFn != nil {
		return m.updateAvailFn(ctx, orgID, status)
	}
	return nil
}
func (m *mockReferrerProfileRepo) UpdateExpertiseDomains(ctx context.Context, orgID uuid.UUID, domains []string) error {
	if m.updateExpFn != nil {
		return m.updateExpFn(ctx, orgID, domains)
	}
	return nil
}
func (m *mockReferrerProfileRepo) UpdateVideo(ctx context.Context, orgID uuid.UUID, videoURL string) error {
	if m.updateVideoFn != nil {
		return m.updateVideoFn(ctx, orgID, videoURL)
	}
	return nil
}
func (m *mockReferrerProfileRepo) GetVideoURL(ctx context.Context, orgID uuid.UUID) (string, error) {
	if m.getVideoFn != nil {
		return m.getVideoFn(ctx, orgID)
	}
	return "", nil
}

func newReferrerHandler(repo *mockReferrerProfileRepo) *ReferrerProfileHandler {
	return NewReferrerProfileHandler(referrerprofileapp.NewService(repo))
}

func newReferrerStubView(orgID uuid.UUID) *repository.ReferrerProfileView {
	return &repository.ReferrerProfileView{
		Profile: &domainreferrer.Profile{
			ID:                 uuid.New(),
			OrganizationID:     orgID,
			Title:              "Top Apporteur",
			About:              "Finds deals.",
			AvailabilityStatus: profile.AvailabilityNow,
			ExpertiseDomains:   []string{},
		},
		Shared: repository.OrganizationSharedProfile{
			PhotoURL:                "https://example.com/r.png",
			City:                    "Lyon",
			CountryCode:             "FR",
			WorkMode:                []string{"remote"},
			LanguagesProfessional:   []string{},
			LanguagesConversational: []string{},
		},
	}
}

func TestReferrerProfileHandler_GetMy_AutoCreatesOnFirstRead(t *testing.T) {
	orgID := uuid.New()
	calls := 0
	repo := &mockReferrerProfileRepo{
		getOrCreateFn: func(ctx context.Context, id uuid.UUID) (*repository.ReferrerProfileView, error) {
			calls++
			return newReferrerStubView(id), nil
		},
	}
	h := newReferrerHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrer-profile", nil)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyOrganizationID, orgID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetMy(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, calls, "GetOrCreate must be called exactly once per GetMy")
}

func TestReferrerProfileHandler_UpdateMyAvailability_AcceptsValidValue(t *testing.T) {
	orgID := uuid.New()
	captured := profile.AvailabilityStatus("")
	repo := &mockReferrerProfileRepo{
		updateAvailFn: func(ctx context.Context, id uuid.UUID, status profile.AvailabilityStatus) error {
			captured = status
			return nil
		},
	}
	h := newReferrerHandler(repo)

	body := `{"availability_status":"available_soon"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/referrer-profile/availability", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.ContextKeyOrganizationID, orgID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UpdateMyAvailability(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, profile.AvailabilitySoon, captured)
}

func TestReferrerProfileHandler_GetMy_Unauthenticated(t *testing.T) {
	h := newReferrerHandler(&mockReferrerProfileRepo{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrer-profile", nil)
	rec := httptest.NewRecorder()
	h.GetMy(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ─── GetReputation ─────────────────────────────────────────────────────────

// stubOrgOwnerLookup is a minimal OrgOwnerLookup implementation that
// returns a fixed user_id (or a fixed error) per orgID. Used by the
// GetReputation tests below to isolate the handler's empty-state and
// error-mapping logic from the real organization repository.
type stubOrgOwnerLookup struct {
	userIDByOrg map[uuid.UUID]uuid.UUID
	errByOrg    map[uuid.UUID]error
}

func (s *stubOrgOwnerLookup) OwnerUserIDForOrg(_ context.Context, orgID uuid.UUID) (uuid.UUID, error) {
	if err, ok := s.errByOrg[orgID]; ok {
		return uuid.Nil, err
	}
	if id, ok := s.userIDByOrg[orgID]; ok {
		return id, nil
	}
	return uuid.Nil, organization.ErrOrgNotFound
}

// newReputationRequest wires the orgID into the chi route context the
// handler reads via chi.URLParam("orgID"). httptest.NewRequest alone
// would leave that param empty.
func newReputationRequest(orgID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orgID", orgID)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrer-profiles/"+orgID+"/reputation", nil)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// TestReferrerProfileHandler_GetReputation_ZeroReferralsReturns200 is
// the regression guard for the production bug observed on
// /fr/referrers/{uuid}: a freshly-enabled apporteur with zero
// attributed projects MUST receive a 200 + empty `history`, never a
// 404 / 500 — otherwise the React Query call resolves to isError=true
// and the section renders the "Impossible de charger les projets
// apportés" toast on a perfectly valid blank profile.
func TestReferrerProfileHandler_GetReputation_ZeroReferralsReturns200(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()
	repo := &mockReferrerProfileRepo{}
	// No reputation deps wired — service returns an empty aggregate.
	h := NewReferrerProfileHandler(referrerprofileapp.NewService(repo)).
		WithOrgOwnerLookup(&stubOrgOwnerLookup{
			userIDByOrg: map[uuid.UUID]uuid.UUID{orgID: ownerID},
		})

	req := newReputationRequest(orgID.String())
	rec := httptest.NewRecorder()
	h.GetReputation(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "zero referrals must be 200, got body=%s", rec.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	// `history` must be a non-nil empty array, not null — the JS
	// client does Array.isArray() / .length checks that would fail
	// silently on null and trigger the same load-error UI.
	hist, ok := body["history"].([]any)
	require.True(t, ok, "history field must be a JSON array, got %T", body["history"])
	assert.Empty(t, hist)
	assert.Equal(t, float64(0), body["rating_avg"])
	assert.Equal(t, float64(0), body["review_count"])
	assert.Equal(t, false, body["has_more"])
}

// TestReferrerProfileHandler_GetReputation_OrgNotFoundReturns200 makes
// the handler's empty-state contract explicit for the "the org id in
// the URL points to nothing in the DB" branch. Returning 404 here was
// the previous behaviour and the source of the production load-error
// regression — public profile + reputation surfaces must stay
// symmetrical.
func TestReferrerProfileHandler_GetReputation_OrgNotFoundReturns200(t *testing.T) {
	orgID := uuid.New()
	repo := &mockReferrerProfileRepo{}
	h := NewReferrerProfileHandler(referrerprofileapp.NewService(repo)).
		WithOrgOwnerLookup(&stubOrgOwnerLookup{
			errByOrg: map[uuid.UUID]error{orgID: organization.ErrOrgNotFound},
		})

	req := newReputationRequest(orgID.String())
	rec := httptest.NewRecorder()
	h.GetReputation(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "ErrOrgNotFound is the natural empty state — body=%s", rec.Body.String())
}

// TestReferrerProfileHandler_GetReputation_UnknownLookupErrorIs500
// keeps the differentiation between "no such org" (200 empty) and
// "infra glitch" (500) explicit. A transient DB error should be
// visible in the browser network tab so it can be alerted on,
// instead of silently rendering an empty "Projets apportés" section
// that hides the underlying outage.
func TestReferrerProfileHandler_GetReputation_UnknownLookupErrorIs500(t *testing.T) {
	orgID := uuid.New()
	repo := &mockReferrerProfileRepo{}
	h := NewReferrerProfileHandler(referrerprofileapp.NewService(repo)).
		WithOrgOwnerLookup(&stubOrgOwnerLookup{
			errByOrg: map[uuid.UUID]error{orgID: errors.New("connection refused")},
		})

	req := newReputationRequest(orgID.String())
	rec := httptest.NewRecorder()
	h.GetReputation(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// TestReferrerProfileHandler_GetReputation_BadOrgIDReturns400 keeps
// the validation envelope on the URL parameter consistent with the
// other persona endpoints — a malformed UUID must surface as 400, not
// 500 / 404.
func TestReferrerProfileHandler_GetReputation_BadOrgIDReturns400(t *testing.T) {
	repo := &mockReferrerProfileRepo{}
	h := NewReferrerProfileHandler(referrerprofileapp.NewService(repo))

	req := newReputationRequest("not-a-uuid")
	rec := httptest.NewRecorder()
	h.GetReputation(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// TestReferrerProfileHandler_GetReputation_NoOrgOwnerWiringReturns200
// guards the handler's "feature is fully removable" contract: when
// the orgOwner lookup is not wired (zero-config boot path), the
// endpoint must return an empty aggregate rather than 500 — otherwise
// removing the lookup would silently break the public profile page.
func TestReferrerProfileHandler_GetReputation_NoOrgOwnerWiringReturns200(t *testing.T) {
	orgID := uuid.New()
	repo := &mockReferrerProfileRepo{}
	h := NewReferrerProfileHandler(referrerprofileapp.NewService(repo))

	req := newReputationRequest(orgID.String())
	rec := httptest.NewRecorder()
	h.GetReputation(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

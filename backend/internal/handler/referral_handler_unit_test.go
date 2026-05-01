package handler

// Unit (mocked) tests for ReferralHandler that complement the DB-gated
// integration tests. Goal: cover the HTTP-layer error mappings and
// auth-context branches without requiring MARKETPLACE_TEST_DATABASE_URL.
//
// The strategy is to wire a real referralapp.Service against in-memory
// stubs (a minimal fakeReferralRepo + fakeUserRepo) so the handler's
// branches (no auth, invalid uuid, invalid body, invalid action,
// non-party 403) are exercised end-to-end through real domain logic.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	referralapp "marketplace-backend/internal/app/referral"
	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// ─── Tiny in-memory fakes (handler-test scope only) ───────────────────

type unitReferralRepo struct {
	mu              sync.Mutex
	rows            map[uuid.UUID]*referral.Referral
	negotiations    []*referral.Negotiation
	attributions    map[uuid.UUID]*referral.Attribution
	commissions     map[string]*referral.Commission
}

var _ repository.ReferralRepository = (*unitReferralRepo)(nil)

func newUnitReferralRepo() *unitReferralRepo {
	return &unitReferralRepo{
		rows:         map[uuid.UUID]*referral.Referral{},
		attributions: map[uuid.UUID]*referral.Attribution{},
		commissions:  map[string]*referral.Commission{},
	}
}

func (f *unitReferralRepo) Create(_ context.Context, r *referral.Referral) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, existing := range f.rows {
		if existing.ProviderID == r.ProviderID && existing.ClientID == r.ClientID && existing.Status.LocksCouple() {
			return referral.ErrCoupleLocked
		}
	}
	cp := *r
	f.rows[r.ID] = &cp
	return nil
}
func (f *unitReferralRepo) GetByID(_ context.Context, id uuid.UUID) (*referral.Referral, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.rows[id]
	if !ok {
		return nil, referral.ErrNotFound
	}
	cp := *r
	return &cp, nil
}
func (f *unitReferralRepo) Update(_ context.Context, r *referral.Referral) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *r
	f.rows[r.ID] = &cp
	return nil
}
func (f *unitReferralRepo) FindActiveByCouple(_ context.Context, p, c uuid.UUID) (*referral.Referral, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rows {
		if r.ProviderID == p && r.ClientID == c && r.Status.LocksCouple() {
			cp := *r
			return &cp, nil
		}
	}
	return nil, referral.ErrNotFound
}
func (f *unitReferralRepo) ListByReferrer(_ context.Context, refID uuid.UUID, _ repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*referral.Referral{}
	for _, r := range f.rows {
		if r.ReferrerID == refID {
			cp := *r
			out = append(out, &cp)
		}
	}
	return out, "", nil
}
func (f *unitReferralRepo) ListIncomingForProvider(_ context.Context, p uuid.UUID, _ repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*referral.Referral{}
	for _, r := range f.rows {
		if r.ProviderID == p {
			cp := *r
			out = append(out, &cp)
		}
	}
	return out, "", nil
}
func (f *unitReferralRepo) ListIncomingForClient(_ context.Context, c uuid.UUID, _ repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*referral.Referral{}
	for _, r := range f.rows {
		if r.ClientID == c {
			cp := *r
			out = append(out, &cp)
		}
	}
	return out, "", nil
}
func (f *unitReferralRepo) AppendNegotiation(_ context.Context, n *referral.Negotiation) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *n
	f.negotiations = append(f.negotiations, &cp)
	return nil
}
func (f *unitReferralRepo) ListNegotiations(_ context.Context, refID uuid.UUID) ([]*referral.Negotiation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []*referral.Negotiation{}
	for _, n := range f.negotiations {
		if n.ReferralID == refID {
			cp := *n
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (f *unitReferralRepo) CreateAttribution(_ context.Context, _ *referral.Attribution) error {
	return nil
}
func (f *unitReferralRepo) FindAttributionByProposal(_ context.Context, _ uuid.UUID) (*referral.Attribution, error) {
	return nil, referral.ErrAttributionNotFound
}
func (f *unitReferralRepo) FindAttributionByID(_ context.Context, _ uuid.UUID) (*referral.Attribution, error) {
	return nil, referral.ErrAttributionNotFound
}
func (f *unitReferralRepo) ListAttributionsByReferral(_ context.Context, _ uuid.UUID) ([]*referral.Attribution, error) {
	return nil, nil
}
func (f *unitReferralRepo) ListAttributionsByReferralIDs(_ context.Context, _ []uuid.UUID) ([]*referral.Attribution, error) {
	return nil, nil
}
func (f *unitReferralRepo) CreateCommission(_ context.Context, _ *referral.Commission) error {
	return nil
}
func (f *unitReferralRepo) UpdateCommission(_ context.Context, _ *referral.Commission) error {
	return nil
}
func (f *unitReferralRepo) FindCommissionByMilestone(_ context.Context, _ uuid.UUID) (*referral.Commission, error) {
	return nil, referral.ErrCommissionNotFound
}
func (f *unitReferralRepo) ListCommissionsByReferral(_ context.Context, _ uuid.UUID) ([]*referral.Commission, error) {
	return nil, nil
}
func (f *unitReferralRepo) ListRecentCommissionsByReferrer(_ context.Context, _ uuid.UUID, _ int) ([]*referral.Commission, error) {
	return nil, nil
}
func (f *unitReferralRepo) ListPendingKYCByReferrer(_ context.Context, _ uuid.UUID) ([]*referral.Commission, error) {
	return nil, nil
}
func (f *unitReferralRepo) ListExpiringIntros(_ context.Context, _ time.Time, _ int) ([]*referral.Referral, error) {
	return nil, nil
}
func (f *unitReferralRepo) ListExpiringActives(_ context.Context, _ time.Time, _ int) ([]*referral.Referral, error) {
	return nil, nil
}
func (f *unitReferralRepo) CountByReferrer(_ context.Context, _ uuid.UUID) (map[referral.Status]int, error) {
	return nil, nil
}
func (f *unitReferralRepo) SumCommissionsByReferrer(_ context.Context, _ uuid.UUID) (map[referral.CommissionStatus]int64, error) {
	return nil, nil
}

// ─── unitUserRepo ─────────────────────────────────────────────────────

type unitUserRepo struct {
	users map[uuid.UUID]*user.User
}

var _ repository.UserRepository = (*unitUserRepo)(nil)

func (u *unitUserRepo) add(id uuid.UUID, role user.Role, refEnabled bool) {
	if u.users == nil {
		u.users = map[uuid.UUID]*user.User{}
	}
	u.users[id] = &user.User{ID: id, Role: role, ReferrerEnabled: refEnabled}
}
func (u *unitUserRepo) Create(_ context.Context, _ *user.User) error { return nil }
func (u *unitUserRepo) GetByID(_ context.Context, id uuid.UUID) (*user.User, error) {
	if x, ok := u.users[id]; ok {
		return x, nil
	}
	return nil, user.ErrUserNotFound
}
func (u *unitUserRepo) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (u *unitUserRepo) Update(_ context.Context, _ *user.User) error    { return nil }
func (u *unitUserRepo) Delete(_ context.Context, _ uuid.UUID) error     { return nil }
func (u *unitUserRepo) ExistsByEmail(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (u *unitUserRepo) ListAdmin(_ context.Context, _ repository.AdminUserFilters) ([]*user.User, string, error) {
	return nil, "", nil
}
func (u *unitUserRepo) CountAdmin(_ context.Context, _ repository.AdminUserFilters) (int, error) {
	return 0, nil
}
func (u *unitUserRepo) CountByRole(_ context.Context) (map[string]int, error)   { return nil, nil }
func (u *unitUserRepo) CountByStatus(_ context.Context) (map[string]int, error) { return nil, nil }
func (u *unitUserRepo) RecentSignups(_ context.Context, _ int) ([]*user.User, error) {
	return nil, nil
}
func (u *unitUserRepo) BumpSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (u *unitUserRepo) GetSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (u *unitUserRepo) UpdateEmailNotificationsEnabled(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}
func (u *unitUserRepo) TouchLastActive(_ context.Context, _ uuid.UUID) error { return nil }

// ─── stub dependencies ────────────────────────────────────────────────

type unitMessageSender struct{}

func (s *unitMessageSender) FindOrCreateConversation(_ context.Context, _ portservice.FindOrCreateConversationInput) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (s *unitMessageSender) SendSystemMessage(_ context.Context, _ portservice.SystemMessageInput) error {
	return nil
}

type unitNotifier struct{}

func (n *unitNotifier) Send(_ context.Context, _ portservice.NotificationInput) error { return nil }

// ─── fixture builder ──────────────────────────────────────────────────

type unitFixture struct {
	repo     *unitReferralRepo
	users    *unitUserRepo
	handler  *ReferralHandler
}

func newUnitFixture(t *testing.T) *unitFixture {
	t.Helper()
	repo := newUnitReferralRepo()
	users := &unitUserRepo{}
	svc := referralapp.NewService(referralapp.ServiceDeps{
		Referrals:        repo,
		Users:            users,
		Messages:         &unitMessageSender{},
		Notifications:    &unitNotifier{},
		SnapshotProfiles: referralapp.NewThinSnapshotLoader(nil),
		StripeAccounts:   referralapp.NewOrgStripeAccountResolver(nil),
	})
	return &unitFixture{
		repo:    repo,
		users:   users,
		handler: NewReferralHandler(svc),
	}
}

func (f *unitFixture) seed(t *testing.T) (referrer, provider, client uuid.UUID) {
	t.Helper()
	referrer = uuid.New()
	provider = uuid.New()
	client = uuid.New()
	f.users.add(referrer, user.RoleProvider, true)
	f.users.add(provider, user.RoleProvider, false)
	f.users.add(client, user.RoleEnterprise, false)
	return
}

func unitWithUser(req *http.Request, userID uuid.UUID) *http.Request {
	if userID == uuid.Nil {
		return req
	}
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, userID)
	return req.WithContext(ctx)
}

func unitWithChiID(req *http.Request, idStr string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", idStr)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

// ─── Create ───────────────────────────────────────────────────────────

func TestReferralHandler_Create_NoAuth_401(t *testing.T) {
	f := newUnitFixture(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/referrals", bytes.NewReader([]byte(`{}`)))
	rec := httptest.NewRecorder()
	f.handler.Create(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestReferralHandler_Create_InvalidBody_400(t *testing.T) {
	f := newUnitFixture(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/referrals", bytes.NewReader([]byte("garbage")))
	req = unitWithUser(req, uuid.New())
	rec := httptest.NewRecorder()
	f.handler.Create(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestReferralHandler_Create_InvalidProviderUUID_400(t *testing.T) {
	f := newUnitFixture(t)
	body, _ := json.Marshal(map[string]any{
		"provider_id":            "not-uuid",
		"client_id":              uuid.New().String(),
		"rate_pct":               5.0,
		"duration_months":        6,
		"intro_message_provider": "x",
		"intro_message_client":   "y",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/referrals", bytes.NewReader(body))
	req = unitWithUser(req, uuid.New())
	rec := httptest.NewRecorder()
	f.handler.Create(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestReferralHandler_Create_HappyPath_201(t *testing.T) {
	f := newUnitFixture(t)
	referrer, provider, client := f.seed(t)
	body, _ := json.Marshal(map[string]any{
		"provider_id":            provider.String(),
		"client_id":              client.String(),
		"rate_pct":               5.0,
		"duration_months":        6,
		"intro_message_provider": "pitch",
		"intro_message_client":   "pitch c",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/referrals", bytes.NewReader(body))
	req = unitWithUser(req, referrer)
	rec := httptest.NewRecorder()
	f.handler.Create(rec, req)
	require.Equalf(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
}

func TestReferralHandler_Create_InvalidRate_400(t *testing.T) {
	f := newUnitFixture(t)
	referrer, provider, client := f.seed(t)
	body, _ := json.Marshal(map[string]any{
		"provider_id":            provider.String(),
		"client_id":              client.String(),
		"rate_pct":               150.0, // out of [0, 100]
		"duration_months":        6,
		"intro_message_provider": "x",
		"intro_message_client":   "y",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/referrals", bytes.NewReader(body))
	req = unitWithUser(req, referrer)
	rec := httptest.NewRecorder()
	f.handler.Create(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code,
		"validator must reject rate_pct > 100")
}

// ─── Get ──────────────────────────────────────────────────────────────

func TestReferralHandler_Get_NoAuth_401(t *testing.T) {
	f := newUnitFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrals/x", nil)
	req = unitWithChiID(req, uuid.New().String())
	rec := httptest.NewRecorder()
	f.handler.Get(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestReferralHandler_Get_InvalidUUID_400(t *testing.T) {
	f := newUnitFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrals/abc", nil)
	req = unitWithChiID(req, "abc")
	req = unitWithUser(req, uuid.New())
	rec := httptest.NewRecorder()
	f.handler.Get(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestReferralHandler_Get_NotFound_404(t *testing.T) {
	f := newUnitFixture(t)
	missing := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrals/"+missing.String(), nil)
	req = unitWithChiID(req, missing.String())
	req = unitWithUser(req, uuid.New())
	rec := httptest.NewRecorder()
	f.handler.Get(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ─── ListMine / ListIncoming ─────────────────────────────────────────

func TestReferralHandler_ListMine_NoAuth_401(t *testing.T) {
	f := newUnitFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrals/me", nil)
	rec := httptest.NewRecorder()
	f.handler.ListMine(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestReferralHandler_ListMine_OK(t *testing.T) {
	f := newUnitFixture(t)
	uid := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrals/me", nil)
	req = unitWithUser(req, uid)
	rec := httptest.NewRecorder()
	f.handler.ListMine(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestReferralHandler_ListIncoming_NoAuth_401(t *testing.T) {
	f := newUnitFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrals/incoming", nil)
	rec := httptest.NewRecorder()
	f.handler.ListIncoming(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestReferralHandler_ListIncoming_OK(t *testing.T) {
	f := newUnitFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrals/incoming", nil)
	req = unitWithUser(req, uuid.New())
	rec := httptest.NewRecorder()
	f.handler.ListIncoming(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ─── Respond branches ─────────────────────────────────────────────────

func TestReferralHandler_Respond_NoAuth_401(t *testing.T) {
	f := newUnitFixture(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/referrals/x/respond", bytes.NewReader([]byte(`{}`)))
	req = unitWithChiID(req, uuid.New().String())
	rec := httptest.NewRecorder()
	f.handler.Respond(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestReferralHandler_Respond_InvalidUUID_400(t *testing.T) {
	f := newUnitFixture(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/referrals/abc/respond", bytes.NewReader([]byte(`{"action":"accept"}`)))
	req = unitWithChiID(req, "abc")
	req = unitWithUser(req, uuid.New())
	rec := httptest.NewRecorder()
	f.handler.Respond(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestReferralHandler_Respond_InvalidBody_400(t *testing.T) {
	f := newUnitFixture(t)
	id := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/referrals/"+id.String()+"/respond", bytes.NewReader([]byte("garbage")))
	req = unitWithChiID(req, id.String())
	req = unitWithUser(req, uuid.New())
	rec := httptest.NewRecorder()
	f.handler.Respond(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestReferralHandler_Respond_NotFound_404(t *testing.T) {
	f := newUnitFixture(t)
	missing := uuid.New()
	body, _ := json.Marshal(map[string]any{"action": "accept"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/referrals/"+missing.String()+"/respond", bytes.NewReader(body))
	req = unitWithChiID(req, missing.String())
	req = unitWithUser(req, uuid.New())
	rec := httptest.NewRecorder()
	f.handler.Respond(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ─── ListNegotiations / ListAttributions / ListCommissions ───────────

func TestReferralHandler_ListNegotiations_NoAuth_401(t *testing.T) {
	f := newUnitFixture(t)
	id := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrals/"+id.String()+"/negotiations", nil)
	req = unitWithChiID(req, id.String())
	rec := httptest.NewRecorder()
	f.handler.ListNegotiations(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestReferralHandler_ListNegotiations_InvalidUUID_400(t *testing.T) {
	f := newUnitFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrals/abc/negotiations", nil)
	req = unitWithChiID(req, "abc")
	req = unitWithUser(req, uuid.New())
	rec := httptest.NewRecorder()
	f.handler.ListNegotiations(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestReferralHandler_ListAttributions_NoAuth_401(t *testing.T) {
	f := newUnitFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrals/x/attributions", nil)
	req = unitWithChiID(req, uuid.New().String())
	rec := httptest.NewRecorder()
	f.handler.ListAttributions(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestReferralHandler_ListAttributions_NotFound_404(t *testing.T) {
	f := newUnitFixture(t)
	missing := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrals/"+missing.String()+"/attributions", nil)
	req = unitWithChiID(req, missing.String())
	req = unitWithUser(req, uuid.New())
	rec := httptest.NewRecorder()
	f.handler.ListAttributions(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestReferralHandler_ListCommissions_NoAuth_401(t *testing.T) {
	f := newUnitFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrals/x/commissions", nil)
	req = unitWithChiID(req, uuid.New().String())
	rec := httptest.NewRecorder()
	f.handler.ListCommissions(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestReferralHandler_ListCommissions_InvalidUUID_400(t *testing.T) {
	f := newUnitFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/referrals/abc/commissions", nil)
	req = unitWithChiID(req, "abc")
	req = unitWithUser(req, uuid.New())
	rec := httptest.NewRecorder()
	f.handler.ListCommissions(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestReferralHandler_ListCommissions_ClientForbidden_403(t *testing.T) {
	// Set up a real referral and try to read commissions as the client
	// — must be 403 (clients are blocked from commission visibility).
	f := newUnitFixture(t)
	referrer, provider, client := f.seed(t)
	body, _ := json.Marshal(map[string]any{
		"provider_id":            provider.String(),
		"client_id":              client.String(),
		"rate_pct":               5.0,
		"duration_months":        6,
		"intro_message_provider": "p",
		"intro_message_client":   "c",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/referrals", bytes.NewReader(body))
	req = unitWithUser(req, referrer)
	rec := httptest.NewRecorder()
	f.handler.Create(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	req = httptest.NewRequest(http.MethodGet, "/api/v1/referrals/"+created.ID.String()+"/commissions", nil)
	req = unitWithChiID(req, created.ID.String())
	req = unitWithUser(req, client)
	rec = httptest.NewRecorder()
	f.handler.ListCommissions(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code,
		"the client must NEVER read commission rows — Modèle A confidentiality")
}

// ─── handleReferralError mapping ──────────────────────────────────────

func TestHandleReferralError_NotFound_404(t *testing.T) {
	rec := httptest.NewRecorder()
	handleReferralError(rec, referral.ErrNotFound)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleReferralError_AttributionNotFound_404(t *testing.T) {
	rec := httptest.NewRecorder()
	handleReferralError(rec, referral.ErrAttributionNotFound)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleReferralError_CommissionNotFound_404(t *testing.T) {
	rec := httptest.NewRecorder()
	handleReferralError(rec, referral.ErrCommissionNotFound)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleReferralError_NotAuthorized_403(t *testing.T) {
	rec := httptest.NewRecorder()
	handleReferralError(rec, referral.ErrNotAuthorized)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandleReferralError_CoupleLocked_409(t *testing.T) {
	rec := httptest.NewRecorder()
	handleReferralError(rec, referral.ErrCoupleLocked)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestHandleReferralError_ValidationErrors_400(t *testing.T) {
	cases := []error{
		referral.ErrSelfReferral,
		referral.ErrSameOrganization,
		referral.ErrInvalidProviderRole,
		referral.ErrInvalidClientRole,
		referral.ErrReferrerRequired,
		referral.ErrRateOutOfRange,
		referral.ErrDurationOutOfRange,
		referral.ErrEmptyMessage,
		referral.ErrMessageTooLong,
		referral.ErrSnapshotInvalid,
	}
	for _, e := range cases {
		rec := httptest.NewRecorder()
		handleReferralError(rec, e)
		assert.Equalf(t, http.StatusBadRequest, rec.Code, "expected 400 for %v", e)
	}
}

func TestHandleReferralError_TransitionErrors_409(t *testing.T) {
	for _, e := range []error{referral.ErrInvalidTransition, referral.ErrAlreadyTerminal} {
		rec := httptest.NewRecorder()
		handleReferralError(rec, e)
		assert.Equalf(t, http.StatusConflict, rec.Code, "expected 409 for %v", e)
	}
}

func TestHandleReferralError_Unknown_500(t *testing.T) {
	rec := httptest.NewRecorder()
	handleReferralError(rec, errBoomTest)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ─── filterFromQuery / negotiationActionFromString ────────────────────

func TestFilterFromQuery_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/x", nil)
	got := filterFromQuery(req)
	assert.Empty(t, got.Cursor)
	assert.Empty(t, got.Statuses)
}

func TestFilterFromQuery_WithStatuses(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/x?status=active&status=cancelled&cursor=xx", nil)
	got := filterFromQuery(req)
	assert.Equal(t, "xx", got.Cursor)
	require.Len(t, got.Statuses, 2)
}

func TestNegotiationActionFromString(t *testing.T) {
	cases := map[string]referral.NegotiationAction{
		"accept":    referral.NegoActionAccepted,
		"reject":    referral.NegoActionRejected,
		"negotiate": referral.NegoActionCountered,
		"counter":   referral.NegoActionCountered,
		"propose":   referral.NegoActionProposed,
	}
	for input, want := range cases {
		got, ok := negotiationActionFromString(input)
		require.True(t, ok, "%q must be a recognised action", input)
		assert.Equal(t, want, got)
	}
}

func TestNegotiationActionFromString_Unknown(t *testing.T) {
	_, ok := negotiationActionFromString("garbage")
	assert.False(t, ok)
}
